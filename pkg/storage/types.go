// Package storage implements the LSM-tree based storage engine for DistKV.
// LSM-tree (Log-Structured Merge-tree) provides high write throughput
// by writing data to memory first, then flushing to disk in batches.
package storage

import (
	"distkv/pkg/consensus"
	"time"
)

// Entry represents a key-value pair with metadata stored in the system.
// This is the fundamental unit of data in our storage engine.
type Entry struct {
	Key        string                   // The key for this entry
	Value      []byte                   // The actual data (can be nil for deletions)
	VectorClock *consensus.VectorClock  // Version information for conflict resolution
	Timestamp  int64                    // When this entry was created (Unix timestamp)
	Deleted    bool                     // True if this is a deletion marker (tombstone)
}

// NewEntry creates a new entry with current timestamp.
func NewEntry(key string, value []byte, vectorClock *consensus.VectorClock) *Entry {
	return &Entry{
		Key:         key,
		Value:       value,
		VectorClock: vectorClock,
		Timestamp:   time.Now().UnixNano(),
		Deleted:     false,
	}
}

// NewDeleteEntry creates a deletion marker (tombstone) for a key.
// Tombstones are needed in LSM-trees because we can't immediately remove
// data from immutable SSTables - we mark it as deleted instead.
func NewDeleteEntry(key string, vectorClock *consensus.VectorClock) *Entry {
	return &Entry{
		Key:         key,
		Value:       nil,
		VectorClock: vectorClock,
		Timestamp:   time.Now().UnixNano(),
		Deleted:     true,
	}
}

// IsExpired checks if a tombstone is old enough to be garbage collected.
// Tombstones must be kept for a while to ensure deleted data doesn't reappear
// during anti-entropy repairs or when offline nodes come back online.
func (e *Entry) IsExpired(ttl time.Duration) bool {
	if !e.Deleted {
		return false // Only tombstones can expire
	}
	
	expiryTime := time.Unix(0, e.Timestamp).Add(ttl)
	return time.Now().After(expiryTime)
}

// StorageConfig holds configuration parameters for the storage engine.
type StorageConfig struct {
	// MemTable settings
	MemTableMaxSize    int           // Max size of MemTable before flush (bytes)
	MaxMemTables       int           // Max number of MemTables to keep
	
	// SSTable settings  
	SSTableMaxSize     int64         // Max size of individual SSTable files
	BloomFilterBits    int           // Bits per key in Bloom filter
	CompressionEnabled bool          // Whether to compress SSTable blocks
	
	// Compaction settings
	CompactionThreshold int          // Number of SSTables to trigger compaction
	MaxCompactionSize   int64        // Max total size for a compaction operation
	
	// Garbage collection
	TombstoneTTL       time.Duration // How long to keep deletion markers
	GCInterval         time.Duration // How often to run garbage collection
	
	// Performance tuning
	WriteBufferSize    int           // Size of write-ahead log buffer
	CacheSize          int           // Size of block cache
	MaxOpenFiles       int           // Max number of SSTable files to keep open
}

// DefaultStorageConfig returns reasonable default configuration values.
func DefaultStorageConfig() *StorageConfig {
	return &StorageConfig{
		// MemTable: 64MB max, keep up to 2 in memory
		MemTableMaxSize:     64 * 1024 * 1024, // 64MB
		MaxMemTables:        2,
		
		// SSTable: 256MB files, 10 bits per Bloom filter key
		SSTableMaxSize:      256 * 1024 * 1024, // 256MB
		BloomFilterBits:     10,
		CompressionEnabled:  true,
		
		// Compaction: trigger at 4 files, max 1GB per operation
		CompactionThreshold: 4,
		MaxCompactionSize:   1024 * 1024 * 1024, // 1GB
		
		// GC: keep tombstones for 3 hours, run GC every 1 hour
		TombstoneTTL:        3 * time.Hour,
		GCInterval:          1 * time.Hour,
		
		// Performance: 4MB write buffer, 128MB cache, 1000 open files
		WriteBufferSize:     4 * 1024 * 1024,   // 4MB
		CacheSize:          128 * 1024 * 1024,  // 128MB
		MaxOpenFiles:       1000,
	}
}

// ReadResult represents the result of a read operation.
type ReadResult struct {
	Entry *Entry  // The entry if found, nil if not found
	Found bool    // Whether the key was found
	Error error   // Any error that occurred during read
}

// WriteResult represents the result of a write operation.
type WriteResult struct {
	Success bool   // Whether the write was successful
	Error   error  // Any error that occurred during write
}

// Iterator provides a way to scan through key-value pairs in order.
// This is essential for range queries and compaction operations.
type Iterator interface {
	// Valid returns true if the iterator points to a valid entry
	Valid() bool
	
	// Key returns the current key (only valid if Valid() is true)
	Key() string
	
	// Value returns the current entry (only valid if Valid() is true) 
	Value() *Entry
	
	// Next advances the iterator to the next entry
	Next()
	
	// Close releases resources held by the iterator
	Close() error
}

// StorageStats provides metrics about storage engine performance.
// Used for monitoring and debugging.
type StorageStats struct {
	// MemTable stats
	MemTableSize     int64 // Current size of active MemTable
	MemTableCount    int   // Number of MemTables in memory
	
	// SSTable stats
	SSTableCount     int   // Number of SSTable files on disk
	TotalDataSize    int64 // Total size of all data files
	
	// Operation counters
	ReadCount        int64 // Total number of read operations
	WriteCount       int64 // Total number of write operations
	CompactionCount  int64 // Number of compactions performed
	
	// Performance metrics
	AvgReadLatency   time.Duration // Average read latency
	AvgWriteLatency  time.Duration // Average write latency
	CacheHitRate     float64       // Block cache hit rate (0.0 to 1.0)
	
	// Error counters
	ReadErrors       int64 // Number of failed read operations
	WriteErrors      int64 // Number of failed write operations
}