// Package storage - Main storage engine that coordinates MemTables and SSTables
// This is the main interface that the rest of the system uses for data operations.
package storage

import (
	"distkv/pkg/consensus"
	"fmt"
	"log"
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
}

// NewEngine creates a new storage engine with the specified configuration.
func NewEngine(dataDir string, config *StorageConfig) (*Engine, error) {
	if config == nil {
		config = DefaultStorageConfig()
	}
	
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}
	
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
	}
	
	// Load existing SSTables from disk
	if err := engine.loadExistingSSTables(); err != nil {
		return nil, fmt.Errorf("failed to load existing SSTables: %v", err)
	}
	
	// Start background workers
	engine.startBackgroundWorkers()
	
	return engine, nil
}

// Put stores a key-value pair in the storage engine.
// This is the main write path for the LSM-tree.
func (e *Engine) Put(key string, value []byte, vectorClock *consensus.VectorClock) error {
	e.mutex.Lock()
	
	if e.closed {
		e.mutex.Unlock()
		return ErrStorageClosed
	}
	
	// Create entry
	entry := NewEntry(key, value, vectorClock)
	
	// Add to active MemTable
	if err := e.activeMemTable.Put(*entry); err != nil {
		e.stats.WriteErrors++
		e.mutex.Unlock()
		return fmt.Errorf("failed to put entry: %v", err)
	}
	
	// Update stats
	e.stats.WriteCount++
	
	// Debug: Log current MemTable size
	log.Printf("MemTable current size: %d bytes, threshold: %d bytes", e.activeMemTable.Size(), e.config.MemTableMaxSize)
	
	// Check if MemTable needs to be flushed
	needsFlush := e.activeMemTable.Size() >= int64(e.config.MemTableMaxSize)
	if needsFlush {
		log.Printf("MemTable size %d >= threshold %d, flushing to disk", e.activeMemTable.Size(), e.config.MemTableMaxSize)
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
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	if e.closed {
		return nil, ErrStorageClosed
	}
	
	startTime := time.Now()
	defer func() {
		e.stats.ReadCount++
		// Update average read latency (simplified)
		e.stats.AvgReadLatency = time.Since(startTime)
	}()
	
	// First check active MemTable
	if entry := e.activeMemTable.Get(key); entry != nil {
		return entry, nil
	}
	
	// Check flushing MemTables
	for _, memTable := range e.flushingMemTables {
		if entry := memTable.Get(key); entry != nil {
			return entry, nil
		}
	}
	
	// Check SSTables (newest first - more likely to have recent data)
	for i := len(e.sstables) - 1; i >= 0; i-- {
		entry, err := e.sstables[i].Get(key)
		if err == nil {
			return entry, nil
		}
		if err != ErrKeyNotFound {
			e.stats.ReadErrors++
			return nil, err
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
	// This is a simplified implementation - production would use a merge iterator
	// that efficiently combines multiple sorted sources
	return nil, fmt.Errorf("iterator not implemented in this simplified version")
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
	e.mutex.Lock()
	if e.closed {
		e.mutex.Unlock()
		return nil
	}
	e.closed = true
	e.mutex.Unlock()
	
	// Signal shutdown to background workers
	close(e.stopChan)
	
	// Wait for background workers to finish
	e.wg.Wait()
	
	// Close all SSTable files
	for _, sst := range e.sstables {
		sst.Close()
	}
	
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
	e.mutex.Lock()
	
	// Move active MemTable to flushing list
	flushingMemTable := e.activeMemTable
	flushingMemTable.SetReadOnly()
	e.flushingMemTables = append(e.flushingMemTables, flushingMemTable)
	
	// Create new active MemTable
	e.activeMemTable = NewMemTable()
	
	e.mutex.Unlock()
	
	// Create sstables subdirectory if it doesn't exist
	sstablesDir := filepath.Join(e.dataDir, "sstables")
	os.MkdirAll(sstablesDir, 0755)
	
	// Generate SSTable file path
	timestamp := time.Now().UnixNano()
	fileName := fmt.Sprintf("sstable_%d.db", timestamp)
	filePath := filepath.Join(sstablesDir, fileName)
	
	// Create SSTable from MemTable
	log.Printf("Creating SSTable at path: %s", filePath)
	sstable, err := CreateSSTable(flushingMemTable, filePath, e.config)
	if err != nil {
		log.Printf("Error creating SSTable: %v", err)
		return
	}
	log.Printf("SSTable created successfully: %s", filePath)
	
	// Add SSTable to engine and remove from flushing list
	e.mutex.Lock()
	e.sstables = append(e.sstables, sstable)
	
	// Remove from flushing list
	for i, mt := range e.flushingMemTables {
		if mt == flushingMemTable {
			e.flushingMemTables = append(e.flushingMemTables[:i], e.flushingMemTables[i+1:]...)
			break
		}
	}
	e.mutex.Unlock()
	
	// Check if compaction is needed
	if len(e.sstables) >= e.config.CompactionThreshold {
		e.triggerCompaction()
	}
}

// performCompaction merges multiple SSTables into fewer, larger files.
func (e *Engine) performCompaction() {
	// Simplified compaction - production would be more sophisticated
	e.mutex.Lock()
	if len(e.sstables) < e.config.CompactionThreshold {
		e.mutex.Unlock()
		return
	}
	e.mutex.Unlock()
	
	// In a real implementation, this would:
	// 1. Select SSTables for compaction
	// 2. Merge them in sorted order
	// 3. Remove tombstones and duplicate keys
	// 4. Create new larger SSTables
	// 5. Delete old SSTables
	// 6. Update engine state atomically
	
	// For now, just update compaction counter
	e.mutex.Lock()
	e.stats.CompactionCount++
	e.mutex.Unlock()
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