package storage_test

import (
	"distkv/pkg/consensus"
	"distkv/pkg/storage"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMemTable_BasicOperations(t *testing.T) {
	memTable := storage.NewMemTable()

	if memTable.Size() != 0 {
		t.Errorf("New MemTable should have size 0, got %d", memTable.Size())
	}

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	entry := storage.NewEntry("test_key", []byte("test_value"), vc)
	err := memTable.Put(*entry)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if memTable.Size() <= 0 {
		t.Error("MemTable size should be greater than 0 after Put")
	}

	retrievedEntry := memTable.Get("test_key")
	if retrievedEntry == nil {
		t.Fatal("Entry not found after Put")
	}

	if retrievedEntry.Key != "test_key" {
		t.Errorf("Expected key 'test_key', got '%s'", retrievedEntry.Key)
	}

	if string(retrievedEntry.Value) != "test_value" {
		t.Errorf("Expected value 'test_value', got '%s'", string(retrievedEntry.Value))
	}

	nonExistentEntry := memTable.Get("non_existent")
	if nonExistentEntry != nil {
		t.Error("Should return nil for non-existent key")
	}
}

func TestMemTable_UpdateExistingKey(t *testing.T) {
	memTable := storage.NewMemTable()

	vc1 := consensus.NewVectorClock()
	vc1.Increment("node1")

	entry1 := storage.NewEntry("key1", []byte("value1"), vc1)
	err := memTable.Put(*entry1)
	if err != nil {
		t.Fatalf("First put failed: %v", err)
	}

	retrievedEntry := memTable.Get("key1")
	if string(retrievedEntry.Value) != "value1" {
		t.Errorf("Expected value 'value1', got '%s'", string(retrievedEntry.Value))
	}

	vc2 := consensus.NewVectorClock()
	vc2.Increment("node1")
	vc2.Increment("node1")

	entry2 := storage.NewEntry("key1", []byte("updated_value"), vc2)
	err = memTable.Put(*entry2)
	if err != nil {
		t.Fatalf("Second put failed: %v", err)
	}

	updatedEntry := memTable.Get("key1")
	if string(updatedEntry.Value) != "updated_value" {
		t.Errorf("Expected updated value 'updated_value', got '%s'", string(updatedEntry.Value))
	}
}

func TestMemTable_MultipleKeys(t *testing.T) {
	memTable := storage.NewMemTable()

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
		"key4": "value4",
		"key5": "value5",
	}

	initialSize := memTable.Size()

	for key, value := range testData {
		entry := storage.NewEntry(key, []byte(value), vc)
		err := memTable.Put(*entry)
		if err != nil {
			t.Fatalf("Put %s failed: %v", key, err)
		}
		vc.Increment("node1")
	}

	if memTable.Size() <= initialSize {
		t.Error("MemTable size should increase after adding entries")
	}

	for key, expectedValue := range testData {
		entry := memTable.Get(key)
		if entry == nil {
			t.Errorf("Key %s not found", key)
			continue
		}

		if string(entry.Value) != expectedValue {
			t.Errorf("Key %s: expected value '%s', got '%s'", key, expectedValue, string(entry.Value))
		}
	}
}

func TestMemTable_DeleteOperations(t *testing.T) {
	memTable := storage.NewMemTable()

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	entry := storage.NewEntry("key1", []byte("value1"), vc)
	err := memTable.Put(*entry)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	retrievedEntry := memTable.Get("key1")
	if retrievedEntry == nil || retrievedEntry.Deleted {
		t.Fatal("Entry should exist and not be deleted")
	}

	vc.Increment("node1")
	deleteEntry := storage.NewDeleteEntry("key1", vc)
	err = memTable.Put(*deleteEntry)
	if err != nil {
		t.Fatalf("Delete entry put failed: %v", err)
	}

	deletedEntry := memTable.Get("key1")
	if deletedEntry == nil {
		t.Fatal("Delete entry should still exist in MemTable")
	}

	if !deletedEntry.Deleted {
		t.Error("Entry should be marked as deleted")
	}
}

func TestMemTable_ReadOnlyMode(t *testing.T) {
	memTable := storage.NewMemTable()

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	entry1 := storage.NewEntry("key1", []byte("value1"), vc)
	err := memTable.Put(*entry1)
	if err != nil {
		t.Fatalf("Put before read-only failed: %v", err)
	}

	memTable.SetReadOnly()

	entry2 := storage.NewEntry("key2", []byte("value2"), vc)
	err = memTable.Put(*entry2)
	if err == nil {
		t.Error("Put should fail when MemTable is read-only")
	}

	retrievedEntry := memTable.Get("key1")
	if retrievedEntry == nil {
		t.Error("Should still be able to read from read-only MemTable")
	}

	if string(retrievedEntry.Value) != "value1" {
		t.Errorf("Expected value 'value1', got '%s'", string(retrievedEntry.Value))
	}
}

func TestMemTable_SizeTracking(t *testing.T) {
	memTable := storage.NewMemTable()

	if memTable.Size() != 0 {
		t.Error("New MemTable should have size 0")
	}

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	baseSize := memTable.Size()

	entry := storage.NewEntry("test", []byte("small"), vc)
	err := memTable.Put(*entry)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	smallEntrySize := memTable.Size()
	if smallEntrySize <= baseSize {
		t.Error("Size should increase after adding entry")
	}

	vc.Increment("node1")
	largeEntry := storage.NewEntry("large", make([]byte, 1000), vc)
	err = memTable.Put(*largeEntry)
	if err != nil {
		t.Fatalf("Put large entry failed: %v", err)
	}

	largeEntrySize := memTable.Size()
	if largeEntrySize <= smallEntrySize {
		t.Error("Size should increase significantly after adding large entry")
	}

	sizeDiff := largeEntrySize - smallEntrySize
	if sizeDiff < 1000 {
		t.Errorf("Size difference should be at least 1000 bytes, got %d", sizeDiff)
	}
}

func TestMemTable_ConcurrentReads(t *testing.T) {
	memTable := storage.NewMemTable()

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		entry := storage.NewEntry(key, []byte(value), vc)
		err := memTable.Put(*entry)
		if err != nil {
			t.Fatalf("Put %s failed: %v", key, err)
		}
		vc.Increment("node1")
	}

	memTable.SetReadOnly()

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key%d", j)
				entry := memTable.Get(key)
				if entry == nil {
					errors <- fmt.Errorf("Worker %d: Key %s not found", workerID, key)
					return
				}
				expectedValue := fmt.Sprintf("value%d", j)
				if string(entry.Value) != expectedValue {
					errors <- fmt.Errorf("Worker %d: Key %s expected '%s', got '%s'",
						workerID, key, expectedValue, string(entry.Value))
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestMemTable_ConcurrentWrites(t *testing.T) {
	memTable := storage.NewMemTable()

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			vc := consensus.NewVectorClock()
			vc.Increment(fmt.Sprintf("node%d", workerID))

			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("worker%d_key%d", workerID, j)
				value := fmt.Sprintf("worker%d_value%d", workerID, j)
				entry := storage.NewEntry(key, []byte(value), vc)
				err := memTable.Put(*entry)
				if err != nil {
					errors <- fmt.Errorf("Worker %d: Put %s failed: %v", workerID, key, err)
					return
				}
				vc.Increment(fmt.Sprintf("node%d", workerID))
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	for workerID := 0; workerID < 10; workerID++ {
		for j := 0; j < 10; j++ {
			key := fmt.Sprintf("worker%d_key%d", workerID, j)
			expectedValue := fmt.Sprintf("worker%d_value%d", workerID, j)

			entry := memTable.Get(key)
			if entry == nil {
				t.Errorf("Key %s not found after concurrent writes", key)
				continue
			}

			if string(entry.Value) != expectedValue {
				t.Errorf("Key %s: expected '%s', got '%s'", key, expectedValue, string(entry.Value))
			}
		}
	}

	expectedEntryCount := 10 * 10
	actualCount := 0
	memTable.SetReadOnly()
	iterator := storage.NewMemTableIterator(memTable)
	defer iterator.Close()

	for iterator.Valid() {
		actualCount++
		iterator.Next()
	}

	if actualCount != expectedEntryCount {
		t.Errorf("Expected %d entries after concurrent writes, got %d", expectedEntryCount, actualCount)
	}
}

func TestMemTable_LargeValues(t *testing.T) {
	memTable := storage.NewMemTable()

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	largeValue := make([]byte, 64*1024)
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}

	entry := storage.NewEntry("large_key", largeValue, vc)
	err := memTable.Put(*entry)
	if err != nil {
		t.Fatalf("Put large value failed: %v", err)
	}

	retrievedEntry := memTable.Get("large_key")
	if retrievedEntry == nil {
		t.Fatal("Large value entry not found")
	}

	if len(retrievedEntry.Value) != len(largeValue) {
		t.Errorf("Expected value length %d, got %d", len(largeValue), len(retrievedEntry.Value))
	}

	for i, expected := range largeValue {
		if i >= len(retrievedEntry.Value) || retrievedEntry.Value[i] != expected {
			t.Errorf("Value mismatch at index %d: expected %d, got %d", i, expected, retrievedEntry.Value[i])
			break
		}
	}
}

func TestMemTable_VectorClockComparison(t *testing.T) {
	memTable := storage.NewMemTable()

	vc1 := consensus.NewVectorClock()
	vc1.Increment("node1")

	entry1 := storage.NewEntry("key1", []byte("value1"), vc1)
	err := memTable.Put(*entry1)
	if err != nil {
		t.Fatalf("First put failed: %v", err)
	}

	vc2 := consensus.NewVectorClock()
	vc2.Increment("node2")

	entry2 := storage.NewEntry("key1", []byte("value2"), vc2)
	err = memTable.Put(*entry2)
	if err != nil {
		t.Fatalf("Second put failed: %v", err)
	}

	retrievedEntry := memTable.Get("key1")
	if retrievedEntry == nil {
		t.Fatal("Entry should exist")
	}

	value := string(retrievedEntry.Value)
	if value != "value1" && value != "value2" {
		t.Errorf("Value should be either 'value1' or 'value2', got '%s'", value)
	}
}

func TestMemTable_Iterator_SortedOrder(t *testing.T) {
	memTable := storage.NewMemTable()

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	keys := []string{"zebra", "alpha", "delta", "beta", "gamma"}
	values := []string{"z_val", "a_val", "d_val", "b_val", "g_val"}

	for i, key := range keys {
		entry := storage.NewEntry(key, []byte(values[i]), vc)
		err := memTable.Put(*entry)
		if err != nil {
			t.Fatalf("Put %s failed: %v", key, err)
		}
		vc.Increment("node1")
	}

	iterator := storage.NewMemTableIterator(memTable)
	defer iterator.Close()

	var foundKeys []string
	for iterator.Valid() {
		foundKeys = append(foundKeys, iterator.Key())
		iterator.Next()
	}

	expectedOrder := []string{"alpha", "beta", "delta", "gamma", "zebra"}
	if len(foundKeys) != len(expectedOrder) {
		t.Fatalf("Expected %d keys, got %d", len(expectedOrder), len(foundKeys))
	}

	for i, expectedKey := range expectedOrder {
		if foundKeys[i] != expectedKey {
			t.Errorf("Position %d: expected key '%s', got '%s'", i, expectedKey, foundKeys[i])
		}
	}
}

func TestMemTable_EmptyKeyHandling(t *testing.T) {
	memTable := storage.NewMemTable()

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	entry := storage.NewEntry("", []byte("empty_key_value"), vc)
	err := memTable.Put(*entry)
	if err != nil {
		t.Fatalf("Put with empty key failed: %v", err)
	}

	retrievedEntry := memTable.Get("")
	if retrievedEntry == nil {
		t.Fatal("Empty key entry not found")
	}

	if string(retrievedEntry.Value) != "empty_key_value" {
		t.Errorf("Expected value 'empty_key_value', got '%s'", string(retrievedEntry.Value))
	}
}

func TestMemTable_EmptyValueHandling(t *testing.T) {
	memTable := storage.NewMemTable()

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	entry := storage.NewEntry("key_with_empty_value", []byte(""), vc)
	err := memTable.Put(*entry)
	if err != nil {
		t.Fatalf("Put with empty value failed: %v", err)
	}

	retrievedEntry := memTable.Get("key_with_empty_value")
	if retrievedEntry == nil {
		t.Fatal("Entry with empty value not found")
	}

	if len(retrievedEntry.Value) != 0 {
		t.Errorf("Expected empty value, got length %d", len(retrievedEntry.Value))
	}
}

func TestMemTable_TimestampOrdering(t *testing.T) {
	memTable := storage.NewMemTable()

	vc1 := consensus.NewVectorClock()
	vc1.Increment("node1")

	time.Sleep(1 * time.Millisecond)

	vc2 := consensus.NewVectorClock()
	vc2.Increment("node1")
	vc2.Increment("node1")

	entry1 := storage.NewEntry("key1", []byte("earlier_value"), vc1)
	entry2 := storage.NewEntry("key1", []byte("later_value"), vc2)

	err := memTable.Put(*entry1)
	if err != nil {
		t.Fatalf("First put failed: %v", err)
	}

	err = memTable.Put(*entry2)
	if err != nil {
		t.Fatalf("Second put failed: %v", err)
	}

	retrievedEntry := memTable.Get("key1")
	if retrievedEntry == nil {
		t.Fatal("Entry should exist")
	}

	value := string(retrievedEntry.Value)
	if value != "later_value" {
		t.Errorf("Expected later value to win, got '%s'", value)
	}
}
