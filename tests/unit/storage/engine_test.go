package storage_test

import (
	"distkv/pkg/consensus"
	"distkv/pkg/storage"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEngine_BasicOperations(t *testing.T) {
	// Create temporary directory for test
	tempDir := filepath.Join(os.TempDir(), "distkv_test_engine")
	defer os.RemoveAll(tempDir)

	// Create engine with small MemTable size to trigger flushes
	config := storage.DefaultStorageConfig()
	config.MemTableMaxSize = 100 // Small size to trigger flushes quickly

	engine, err := storage.NewEngine(tempDir, config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Test Put
	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	err = engine.Put("key1", []byte("value1"), vc)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Test Get
	entry, err := engine.Get("key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if entry == nil {
		t.Fatal("Entry not found")
	}

	if entry.Key != "key1" {
		t.Errorf("Expected key 'key1', got '%s'", entry.Key)
	}

	if string(entry.Value) != "value1" {
		t.Errorf("Expected value 'value1', got '%s'", string(entry.Value))
	}

	// Test non-existent key
	_, err = engine.Get("nonexistent")
	if err != storage.ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

func TestEngine_Delete(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "distkv_test_delete")
	defer os.RemoveAll(tempDir)

	engine, err := storage.NewEngine(tempDir, storage.DefaultStorageConfig())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Put a key
	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	err = engine.Put("key1", []byte("value1"), vc)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify it exists
	entry, err := engine.Get("key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if entry == nil {
		t.Fatal("Entry should exist")
	}

	// Delete the key
	vc.Increment("node1")
	err = engine.Delete("key1", vc)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	_, err = engine.Get("key1")
	if err != storage.ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound after delete, got %v", err)
	}
}

func TestEngine_MemTableFlush(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "distkv_test_flush")
	defer os.RemoveAll(tempDir)

	// Create engine with very small MemTable to force flush
	config := storage.DefaultStorageConfig()
	config.MemTableMaxSize = 50 // Very small to trigger flush

	engine, err := storage.NewEngine(tempDir, config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	// Put enough data to trigger a flush
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d_with_extra_data_to_increase_size", i)

		err = engine.Put(key, []byte(value), vc)
		if err != nil {
			t.Fatalf("Put %s failed: %v", key, err)
		}
		vc.Increment("node1")
	}

	// Give time for background flush to complete
	time.Sleep(200 * time.Millisecond)

	// Verify all data is still accessible
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		expectedValue := fmt.Sprintf("value%d_with_extra_data_to_increase_size", i)

		entry, err := engine.Get(key)
		if err != nil {
			t.Fatalf("Get %s failed after flush: %v", key, err)
		}

		if string(entry.Value) != expectedValue {
			t.Errorf("Expected value '%s', got '%s'", expectedValue, string(entry.Value))
		}
	}

	// Check that SSTables were created
	stats := engine.Stats()
	if stats.SSTableCount == 0 {
		t.Error("Expected SSTables to be created after flush")
	}
}

func TestEngine_Compaction(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "distkv_test_compaction")
	defer os.RemoveAll(tempDir)

	// Create engine with small limits to trigger compaction
	config := storage.DefaultStorageConfig()
	config.MemTableMaxSize = 50
	config.CompactionThreshold = 3 // Trigger compaction after 3 SSTables

	engine, err := storage.NewEngine(tempDir, config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	// Create enough data to trigger multiple flushes and compaction
	for batch := 0; batch < 5; batch++ {
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("batch%d_key%d", batch, i)
			value := fmt.Sprintf("batch%d_value%d_with_data", batch, i)

			err = engine.Put(key, []byte(value), vc)
			if err != nil {
				t.Fatalf("Put failed: %v", err)
			}
			vc.Increment("node1")
		}

		// Wait for automatic flush to be triggered
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for potential compaction
	time.Sleep(300 * time.Millisecond)

	// Verify all data is still accessible after compaction
	for batch := 0; batch < 5; batch++ {
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("batch%d_key%d", batch, i)
			expectedValue := fmt.Sprintf("batch%d_value%d_with_data", batch, i)

			entry, err := engine.Get(key)
			if err != nil {
				t.Fatalf("Get %s failed after compaction: %v", key, err)
			}

			if string(entry.Value) != expectedValue {
				t.Errorf("Expected value '%s', got '%s'", expectedValue, string(entry.Value))
			}
		}
	}

	// Check that compaction occurred
	stats := engine.Stats()
	if stats.CompactionCount == 0 {
		t.Error("Expected compaction to have occurred")
	}
}

func TestEngine_Iterator(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "distkv_test_iterator")
	defer os.RemoveAll(tempDir)

	engine, err := storage.NewEngine(tempDir, storage.DefaultStorageConfig())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	// Put test data
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
		"key4": "value4",
		"key5": "value5",
	}

	for key, value := range testData {
		err = engine.Put(key, []byte(value), vc)
		if err != nil {
			t.Fatalf("Put %s failed: %v", key, err)
		}
		vc.Increment("node1")
	}

	// Test iterator
	iterator, err := engine.Iterator()
	if err != nil {
		t.Fatalf("Failed to create iterator: %v", err)
	}
	defer iterator.Close()

	foundKeys := make(map[string]string)
	for iterator.Valid() {
		key := iterator.Key()
		entry := iterator.Value()
		foundKeys[key] = string(entry.Value)
		iterator.Next()
	}

	// Verify all keys were found
	if len(foundKeys) != len(testData) {
		t.Errorf("Expected %d keys, found %d", len(testData), len(foundKeys))
	}

	for key, expectedValue := range testData {
		if foundValue, exists := foundKeys[key]; !exists {
			t.Errorf("Key %s not found in iterator", key)
		} else if foundValue != expectedValue {
			t.Errorf("Key %s: expected value %s, got %s", key, expectedValue, foundValue)
		}
	}
}

func TestEngine_ConcurrentAccess(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "distkv_test_concurrent")
	defer os.RemoveAll(tempDir)

	engine, err := storage.NewEngine(tempDir, storage.DefaultStorageConfig())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Test concurrent reads and writes
	done := make(chan bool, 2)

	// Writer goroutine
	go func() {
		vc := consensus.NewVectorClock()
		vc.Increment("writer")

		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("concurrent_key_%d", i)
			value := fmt.Sprintf("concurrent_value_%d", i)

			err := engine.Put(key, []byte(value), vc)
			if err != nil {
				t.Errorf("Concurrent put failed: %v", err)
			}
			vc.Increment("writer")
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 50; i++ {
			key := fmt.Sprintf("concurrent_key_%d", i)

			// Try to read - might not exist yet due to concurrency
			_, err := engine.Get(key)
			if err != nil && err != storage.ErrKeyNotFound {
				t.Errorf("Concurrent get failed: %v", err)
			}
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify final state
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("concurrent_key_%d", i)
		expectedValue := fmt.Sprintf("concurrent_value_%d", i)

		entry, err := engine.Get(key)
		if err != nil {
			t.Errorf("Final get %s failed: %v", key, err)
			continue
		}

		if string(entry.Value) != expectedValue {
			t.Errorf("Final value mismatch for %s: expected %s, got %s",
				key, expectedValue, string(entry.Value))
		}
	}
}

func TestEngine_Stats(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "distkv_test_stats")
	defer os.RemoveAll(tempDir)

	engine, err := storage.NewEngine(tempDir, storage.DefaultStorageConfig())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Initial stats
	stats := engine.Stats()
	if stats.ReadCount != 0 {
		t.Errorf("Expected 0 reads initially, got %d", stats.ReadCount)
	}
	if stats.WriteCount != 0 {
		t.Errorf("Expected 0 writes initially, got %d", stats.WriteCount)
	}

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	// Perform some operations
	err = engine.Put("test", []byte("value"), vc)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	_, err = engine.Get("test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	_, err = engine.Get("nonexistent")
	if err != storage.ErrKeyNotFound {
		t.Fatalf("Expected ErrKeyNotFound, got %v", err)
	}

	// Check updated stats
	stats = engine.Stats()
	if stats.WriteCount != 1 {
		t.Errorf("Expected 1 write, got %d", stats.WriteCount)
	}
	if stats.ReadCount != 2 {
		t.Errorf("Expected 2 reads, got %d", stats.ReadCount)
	}
	if stats.MemTableCount != 1 {
		t.Errorf("Expected 1 MemTable, got %d", stats.MemTableCount)
	}
}