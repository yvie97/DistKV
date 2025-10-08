// Package storage - Main storage engine that coordinates MemTables and SSTables
// This is the main interface that the rest of the system uses for data operations.
package storage

import (
	"distkv/pkg/consensus"
	"distkv/pkg/errors"
	"distkv/pkg/logging"
	"distkv/pkg/metrics"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Engine is the main storage engine that implements the LSM-tree.
// It coordinates between MemTables (memory) and SSTables (disk).
type Engine struct {
	// config holds storage configuration parameters
	config *StorageConfig

	// dataDir is the directory where SSTable files are stored
	dataDir string

	// activeMemTable receives all new writes
	activeMemTable *MemTable

	// flushingMemTables are being written to disk (read-only)
	flushingMemTables []*MemTable

	// sstables contains all disk-based storage files
	sstables []*SSTable

	// mutex protects concurrent access to engine state
	mutex sync.RWMutex

	// flushChan signals when a flush operation is needed
	flushChan chan struct{}

	// compactionChan signals when compaction is needed
	compactionChan chan struct{}

	// stopChan signals shutdown
	stopChan chan struct{}

	// wg tracks background goroutines
	wg sync.WaitGroup

	// stats tracks performance metrics
	stats *StorageStats

	// closed indicates if the engine is shut down
	closed bool

	// logger is the component logger
	logger *logging.Logger

	// metrics collector
	metrics *metrics.MetricsCollector
}

// NewEngine creates a new storage engine with the specified configuration.
func NewEngine(dataDir string, config *StorageConfig) (*Engine, error) {
	logger := logging.WithComponent("storage.engine")

	if config == nil {
		config = DefaultStorageConfig()
		logger.Info("Using default storage configuration")
	}

	// Validate configuration
	if dataDir == "" {
		return nil, errors.NewInvalidConfigError("data directory cannot be empty")
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal,
			"failed to create data directory").WithContext("dataDir", dataDir)
	}

	logger.WithField("dataDir", dataDir).Info("Initializing storage engine")

	engine := &Engine{
		config:            config,
		dataDir:           dataDir,
		activeMemTable:    NewMemTable(),
		flushingMemTables: make([]*MemTable, 0),
		sstables:          make([]*SSTable, 0),
		flushChan:         make(chan struct{}, 1),
		compactionChan:    make(chan struct{}, 1),
		stopChan:          make(chan struct{}),
		stats:             &StorageStats{},
		closed:            false,
		logger:            logger,
		metrics:           metrics.GetGlobalMetrics(),
	}

	// Load existing SSTables from disk
	if err := engine.loadExistingSSTables(); err != nil {
		logger.WithError(err).Error("Failed to load existing SSTables")
		return nil, errors.Wrap(err, errors.ErrCodeStorageCorrupted,
			"failed to load existing SSTables").WithContext("dataDir", dataDir)
	}

	// Start background workers
	engine.startBackgroundWorkers()

	logger.Info("Storage engine initialized successfully")
	return engine, nil
}

// Put stores a key-value pair in the storage engine.
// This is the main write path for the LSM-tree.
func (e *Engine) Put(key string, value []byte, vectorClock *consensus.VectorClock) error {
	// Track latency
	tracker := metrics.NewLatencyTracker()
	defer func() {
		latency := tracker.Finish()
		e.metrics.Storage().WriteLatencyNs.Store(latency)
	}()

	// Validate inputs
	if key == "" {
		e.metrics.Storage().WriteErrors.Add(1)
		return errors.New(errors.ErrCodeInvalidKey, "key cannot be empty")
	}
	if value == nil {
		e.metrics.Storage().WriteErrors.Add(1)
		return errors.New(errors.ErrCodeInvalidValue, "value cannot be nil")
	}

	e.mutex.Lock()

	if e.closed {
		e.mutex.Unlock()
		e.metrics.Storage().WriteErrors.Add(1)
		return errors.NewStorageClosedError()
	}

	// Create entry
	entry := NewEntry(key, value, vectorClock)

	// Add to active MemTable
	if err := e.activeMemTable.Put(*entry); err != nil {
		e.stats.WriteErrors++
		e.metrics.Storage().WriteErrors.Add(1)
		e.mutex.Unlock()
		e.logger.WithError(err).WithField("key", key).Error("Failed to put entry to MemTable")
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to put entry to MemTable")
	}

	// Update stats and metrics
	e.stats.WriteCount++
	e.metrics.Storage().WriteOps.Add(1)

	memTableSize := e.activeMemTable.Size()
	e.metrics.Storage().MemTableSize.Store(memTableSize)

	// Check if MemTable needs to be flushed
	needsFlush := memTableSize >= int64(e.config.MemTableMaxSize)
	if needsFlush {
		e.logger.WithFields(map[string]interface{}{
			"memTableSize": memTableSize,
			"threshold":    e.config.MemTableMaxSize,
		}).Info("MemTable threshold reached, triggering flush")
	}

	// Release the mutex before flushing to avoid deadlock
	e.mutex.Unlock()

	if needsFlush {
		// For testing: force synchronous flush
		e.performFlush()
	}

	return nil
}

// Get retrieves a value by key from the storage engine.
// This implements the LSM-tree read path: MemTable first, then SSTables.
func (e *Engine) Get(key string) (*Entry, error) {
	// Track latency
	tracker := metrics.NewLatencyTracker()
	defer func() {
		latency := tracker.Finish()
		e.metrics.Storage().ReadLatencyNs.Store(latency)
	}()

	// Validate input
	if key == "" {
		e.metrics.Storage().ReadErrors.Add(1)
		return nil, errors.New(errors.ErrCodeInvalidKey, "key cannot be empty")
	}

	e.mutex.RLock()
	defer e.mutex.RUnlock()

	if e.closed {
		e.metrics.Storage().ReadErrors.Add(1)
		return nil, errors.NewStorageClosedError()
	}

	startTime := time.Now()
	defer func() {
		e.stats.ReadCount++
		e.metrics.Storage().ReadOps.Add(1)
		// Update average read latency (simplified)
		e.stats.AvgReadLatency = time.Since(startTime)
	}()

	// First check active MemTable
	if entry := e.activeMemTable.Get(key); entry != nil {
		e.metrics.Storage().CacheHits.Add(1) // MemTable is effectively a cache
		if entry.Deleted {
			return nil, ErrKeyNotFound // Tombstone found
		}
		return entry, nil
	}

	// Check flushing MemTables
	for _, memTable := range e.flushingMemTables {
		if entry := memTable.Get(key); entry != nil {
			e.metrics.Storage().CacheHits.Add(1)
			if entry.Deleted {
				return nil, ErrKeyNotFound // Tombstone found
			}
			return entry, nil
		}
	}

	// Check SSTables (newest first - more likely to have recent data)
	e.metrics.Storage().CacheMisses.Add(1)
	for i := len(e.sstables) - 1; i >= 0; i-- {
		entry, err := e.sstables[i].Get(key)
		if err == nil {
			if entry.Deleted {
				return nil, ErrKeyNotFound // Tombstone found
			}
			return entry, nil
		}
		if err != ErrKeyNotFound {
			e.stats.ReadErrors++
			e.metrics.Storage().ReadErrors.Add(1)
			e.logger.WithError(err).WithField("key", key).
				Error("Error reading from SSTable")
			return nil, errors.Wrap(err, errors.ErrCodeInternal,
				"failed to read from SSTable")
		}
	}

	return nil, ErrKeyNotFound
}

// Delete marks a key as deleted by inserting a tombstone.
func (e *Engine) Delete(key string, vectorClock *consensus.VectorClock) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	if e.closed {
		return ErrStorageClosed
	}
	
	// Create tombstone entry
	entry := NewDeleteEntry(key, vectorClock)
	
	// Add tombstone to active MemTable
	if err := e.activeMemTable.Put(*entry); err != nil {
		e.stats.WriteErrors++
		return fmt.Errorf("failed to put delete entry: %v", err)
	}
	
	// Update stats
	e.stats.WriteCount++
	
	// Check if MemTable needs to be flushed
	if e.activeMemTable.Size() >= int64(e.config.MemTableMaxSize) {
		e.triggerFlush()
	}
	
	return nil
}

// Iterator returns an iterator that scans all key-value pairs.
// This merges data from MemTables and SSTables in sorted order.
func (e *Engine) Iterator() (Iterator, error) {
	return NewEngineIterator(e)
}

// Stats returns current storage engine statistics.
func (e *Engine) Stats() *StorageStats {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	// Update current stats
	e.stats.MemTableSize = e.activeMemTable.Size()
	e.stats.MemTableCount = 1 + len(e.flushingMemTables)
	e.stats.SSTableCount = len(e.sstables)
	
	// Calculate total data size
	var totalSize int64 = e.activeMemTable.Size()
	for _, sst := range e.sstables {
		totalSize += sst.Size()
	}
	e.stats.TotalDataSize = totalSize
	
	// Copy stats to avoid race conditions
	statsCopy := *e.stats
	return &statsCopy
}

// Close shuts down the storage engine gracefully.
func (e *Engine) Close() error {
	e.logger.Info("Initiating graceful shutdown of storage engine")

	e.mutex.Lock()
	if e.closed {
		e.mutex.Unlock()
		e.logger.Warn("Storage engine already closed")
		return nil
	}
	e.closed = true
	e.mutex.Unlock()

	var shutdownErrors []error

	// Signal shutdown to background workers
	e.logger.Debug("Stopping background workers")
	close(e.stopChan)

	// Wait for background workers to finish with timeout
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		e.logger.Debug("Background workers stopped successfully")
	case <-time.After(30 * time.Second):
		e.logger.Error("Timeout waiting for background workers to stop")
		shutdownErrors = append(shutdownErrors,
			errors.New(errors.ErrCodeTimeout, "timeout waiting for background workers"))
	}

	// Flush any remaining data in active MemTable
	e.mutex.Lock()
	if e.activeMemTable.Size() > 0 {
		e.logger.WithField("memTableSize", e.activeMemTable.Size()).
			Info("Flushing remaining MemTable data")
		flushingMemTable := e.activeMemTable
		flushingMemTable.SetReadOnly()
		e.mutex.Unlock()

		// Perform final flush
		sstablesDir := filepath.Join(e.dataDir, "sstables")
		os.MkdirAll(sstablesDir, 0755)
		timestamp := time.Now().UnixNano()
		fileName := fmt.Sprintf("sstable_final_%d.db", timestamp)
		filePath := filepath.Join(sstablesDir, fileName)

		if _, err := CreateSSTable(flushingMemTable, filePath, e.config); err != nil {
			e.logger.WithError(err).Error("Failed to flush final MemTable")
			shutdownErrors = append(shutdownErrors, err)
		} else {
			e.logger.Info("Final MemTable flushed successfully")
		}
	} else {
		e.mutex.Unlock()
	}

	// Close all SSTable files
	e.logger.WithField("sstableCount", len(e.sstables)).Debug("Closing SSTable files")
	e.mutex.Lock()
	sstables := e.sstables
	e.mutex.Unlock()

	for i, sst := range sstables {
		if err := sst.Close(); err != nil {
			e.logger.WithError(err).WithField("sstableIndex", i).
				Error("Failed to close SSTable")
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	if len(shutdownErrors) > 0 {
		e.logger.WithField("errorCount", len(shutdownErrors)).
			Error("Storage engine closed with errors")
		return errors.New(errors.ErrCodeInternal,
			fmt.Sprintf("storage engine shutdown completed with %d errors", len(shutdownErrors)))
	}

	e.logger.Info("Storage engine shutdown completed successfully")
	return nil
}

// triggerFlush signals that a MemTable flush is needed.
func (e *Engine) triggerFlush() {
	select {
	case e.flushChan <- struct{}{}:
	default:
		// Channel full, flush already pending
	}
}

// triggerCompaction signals that compaction is needed.
func (e *Engine) triggerCompaction() {
	select {
	case e.compactionChan <- struct{}{}:
	default:
		// Channel full, compaction already pending
	}
}

// startBackgroundWorkers starts goroutines for flush and compaction.
func (e *Engine) startBackgroundWorkers() {
	// Flush worker
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		for {
			select {
			case <-e.flushChan:
				e.performFlush()
			case <-e.stopChan:
				return
			}
		}
	}()
	
	// Compaction worker  
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		for {
			select {
			case <-e.compactionChan:
				e.performCompaction()
			case <-e.stopChan:
				return
			}
		}
	}()
}

// performFlush flushes the active MemTable to an SSTable.
func (e *Engine) performFlush() {
	tracker := metrics.NewLatencyTracker()
	e.logger.Info("Starting MemTable flush")

	e.mutex.Lock()

	// Move active MemTable to flushing list
	flushingMemTable := e.activeMemTable
	flushingMemTable.SetReadOnly()
	e.flushingMemTables = append(e.flushingMemTables, flushingMemTable)

	// Create new active MemTable
	e.activeMemTable = NewMemTable()
	memTableCount := len(e.flushingMemTables) + 1
	e.metrics.Storage().MemTableCount.Store(int64(memTableCount))

	e.mutex.Unlock()

	// Create sstables subdirectory if it doesn't exist
	sstablesDir := filepath.Join(e.dataDir, "sstables")
	if err := os.MkdirAll(sstablesDir, 0755); err != nil {
		e.logger.WithError(err).Error("Failed to create sstables directory")
		e.metrics.Storage().FlushErrors.Add(1)
		return
	}

	// Generate SSTable file path
	timestamp := time.Now().UnixNano()
	fileName := fmt.Sprintf("sstable_%d.db", timestamp)
	filePath := filepath.Join(sstablesDir, fileName)

	// Create SSTable from MemTable
	e.logger.WithField("path", filePath).Info("Creating SSTable")
	sstable, err := CreateSSTable(flushingMemTable, filePath, e.config)
	if err != nil {
		e.logger.WithError(err).Error("Failed to create SSTable")
		e.metrics.Storage().FlushErrors.Add(1)
		return
	}

	// Add SSTable to engine and remove from flushing list
	e.mutex.Lock()
	e.sstables = append(e.sstables, sstable)
	e.metrics.Storage().SSTableCount.Store(int64(len(e.sstables)))

	// Remove from flushing list
	for i, mt := range e.flushingMemTables {
		if mt == flushingMemTable {
			e.flushingMemTables = append(e.flushingMemTables[:i], e.flushingMemTables[i+1:]...)
			break
		}
	}
	e.mutex.Unlock()

	// Update metrics
	e.metrics.Storage().FlushCount.Add(1)
	e.metrics.Storage().FlushDurationNs.Store(tracker.Finish())
	e.logger.WithFields(map[string]interface{}{
		"path":         filePath,
		"sstableCount": len(e.sstables),
	}).Info("SSTable created successfully")

	// Check if compaction is needed
	if len(e.sstables) >= e.config.CompactionThreshold {
		e.logger.WithField("sstableCount", len(e.sstables)).
			Info("SSTable count exceeded threshold, triggering compaction")
		e.triggerCompaction()
	}
}

// performCompaction merges multiple SSTables into fewer, larger files.
// This is essential for maintaining good read performance in an LSM-tree.
func (e *Engine) performCompaction() {
	tracker := metrics.NewLatencyTracker()
	e.logger.Info("Starting compaction")

	e.mutex.Lock()

	// Check if compaction is needed
	if len(e.sstables) < e.config.CompactionThreshold {
		e.mutex.Unlock()
		e.logger.Debug("Compaction not needed, SSTable count below threshold")
		return
	}

	e.logger.WithFields(map[string]interface{}{
		"sstableCount": len(e.sstables),
		"threshold":    e.config.CompactionThreshold,
	}).Info("Compaction threshold reached")

	// Select SSTables for compaction (simple strategy: compact all)
	// In production, this would use level-based or size-tiered compaction
	tablesToCompact := make([]*SSTable, len(e.sstables))
	copy(tablesToCompact, e.sstables)

	e.mutex.Unlock()

	// Perform compaction without holding the main lock
	newSSTable, err := e.compactSSTables(tablesToCompact)
	if err != nil {
		e.logger.WithError(err).Error("Compaction failed")
		e.metrics.Storage().CompactionErrors.Add(1)
		return
	}

	// Atomically update the engine state
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Replace old SSTables with the new one
	oldSSTables := e.sstables
	e.sstables = []*SSTable{newSSTable}
	e.metrics.Storage().SSTableCount.Store(int64(len(e.sstables)))

	// Update stats and metrics
	e.stats.CompactionCount++
	e.metrics.Storage().CompactionCount.Add(1)
	e.metrics.Storage().CompactionDurationNs.Store(tracker.Finish())

	e.logger.WithFields(map[string]interface{}{
		"oldCount": len(oldSSTables),
		"newCount": 1,
	}).Info("Compaction completed successfully")

	// Close and delete old SSTable files in background
	go e.cleanupOldSSTables(oldSSTables)
}

// compactSSTables merges multiple SSTables into a single new SSTable.
func (e *Engine) compactSSTables(tables []*SSTable) (*SSTable, error) {
	if len(tables) == 0 {
		return nil, errors.New(errors.ErrCodeInternal, "no SSTables to compact")
	}

	e.logger.WithField("tableCount", len(tables)).Debug("Starting SSTable compaction")

	// Create iterators for all SSTables to be compacted
	iterators := make([]Iterator, len(tables))
	for i, table := range tables {
		iterators[i] = NewSSTableIterator(table)
	}

	// Create merge iterator to process entries in sorted order
	mergeIterator := NewMergeIterator(iterators)
	defer mergeIterator.Close()

	// Create temporary compacted SSTable
	timestamp := time.Now().UnixNano()
	compactedPath := filepath.Join(e.dataDir, "sstables", fmt.Sprintf("compacted_%d.db", timestamp))

	// Create a temporary MemTable to collect compacted entries
	tempMemTable := NewMemTable()

	// Process all entries, removing tombstones and duplicates
	processedCount := 0
	tombstonesRemoved := 0

	for mergeIterator.Valid() {
		entry := mergeIterator.Value()

		// Skip expired tombstones (garbage collection)
		if entry.Deleted && entry.IsExpired(e.config.TombstoneTTL) {
			tombstonesRemoved++
			mergeIterator.Next()
			continue
		}

		// Add entry to temporary MemTable
		if err := tempMemTable.Put(*entry); err != nil {
			e.logger.WithError(err).Error("Failed to add entry during compaction")
			return nil, errors.Wrap(err, errors.ErrCodeCompactionFailed,
				"failed to add entry during compaction")
		}

		processedCount++
		mergeIterator.Next()
	}

	e.metrics.Storage().TombstonesCollected.Add(uint64(tombstonesRemoved))
	e.logger.WithFields(map[string]interface{}{
		"processedCount":     processedCount,
		"tombstonesRemoved": tombstonesRemoved,
	}).Info("Compaction processing completed")

	// Create new SSTable from the compacted data
	newSSTable, err := CreateSSTable(tempMemTable, compactedPath, e.config)
	if err != nil {
		e.logger.WithError(err).Error("Failed to create compacted SSTable")
		return nil, errors.Wrap(err, errors.ErrCodeCompactionFailed,
			"failed to create compacted SSTable")
	}

	return newSSTable, nil
}

// cleanupOldSSTables closes and deletes old SSTable files after compaction.
func (e *Engine) cleanupOldSSTables(oldTables []*SSTable) {
	e.logger.WithField("tableCount", len(oldTables)).Debug("Cleaning up old SSTables")

	successCount := 0
	errorCount := 0

	for _, table := range oldTables {
		// Close the SSTable
		if err := table.Close(); err != nil {
			e.logger.WithError(err).WithField("path", table.filePath).
				Warn("Error closing old SSTable")
			errorCount++
		}

		// Delete the file
		if err := os.Remove(table.filePath); err != nil {
			e.logger.WithError(err).WithField("path", table.filePath).
				Warn("Error deleting old SSTable file")
			errorCount++
		} else {
			successCount++
			e.logger.WithField("path", table.filePath).Debug("Deleted old SSTable file")
		}

		// Delete metadata file if it exists
		metadataPath := table.filePath + ".meta"
		if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
			e.logger.WithError(err).WithField("path", metadataPath).
				Warn("Error deleting SSTable metadata file")
		}
	}

	e.logger.WithFields(map[string]interface{}{
		"successCount": successCount,
		"errorCount":   errorCount,
	}).Info("SSTable cleanup completed")
}

// loadExistingSSTables scans the data directory and opens existing SSTables.
func (e *Engine) loadExistingSSTables() error {
	files, err := filepath.Glob(filepath.Join(e.dataDir, "*.db"))
	if err != nil {
		return err
	}
	
	for _, filePath := range files {
		sstable, err := OpenSSTable(filePath)
		if err != nil {
			// Log warning in production, skip corrupted files
			continue
		}
		e.sstables = append(e.sstables, sstable)
	}
	
	return nil
}