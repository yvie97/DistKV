// Package storage - Error definitions for the storage engine
package storage

import "errors"

// Common errors that can occur in the storage engine
var (
	// ErrKeyNotFound is returned when a requested key doesn't exist
	ErrKeyNotFound = errors.New("key not found")

	// ErrMemTableReadOnly is returned when trying to write to a read-only MemTable
	ErrMemTableReadOnly = errors.New("memtable is read-only")

	// ErrSSTableCorrupted is returned when an SSTable file is corrupted
	ErrSSTableCorrupted = errors.New("sstable file is corrupted")

	// ErrInvalidIterator is returned when using an invalid iterator
	ErrInvalidIterator = errors.New("iterator is not valid")

	// ErrStorageClosed is returned when operating on a closed storage engine
	ErrStorageClosed = errors.New("storage engine is closed")

	// ErrCompactionInProgress is returned when a compaction operation conflicts
	ErrCompactionInProgress = errors.New("compaction is already in progress")

	// ErrInvalidConfiguration is returned for invalid storage configuration
	ErrInvalidConfiguration = errors.New("invalid storage configuration")

	// ErrMemoryLimitExceeded is returned when memory usage exceeds configured limits
	ErrMemoryLimitExceeded = errors.New("memory limit exceeded")
)
