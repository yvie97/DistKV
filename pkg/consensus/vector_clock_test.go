package consensus

import (
	"testing"
)

// TestNewVectorClock tests creating a new vector clock
func TestNewVectorClock(t *testing.T) {
	vc := NewVectorClock()

	if vc == nil {
		t.Fatal("NewVectorClock returned nil")
	}

	if vc.Size() != 0 {
		t.Errorf("Expected size 0 for new vector clock, got %d", vc.Size())
	}
}

// TestNewVectorClockFromMap tests creating vector clock from existing clocks
func TestNewVectorClockFromMap(t *testing.T) {
	clocks := map[string]uint64{
		"node1": 5,
		"node2": 3,
		"node3": 7,
	}

	vc := NewVectorClockFromMap(clocks)

	if vc.Size() != 3 {
		t.Errorf("Expected size 3, got %d", vc.Size())
	}

	if vc.GetClock("node1") != 5 {
		t.Errorf("Expected node1 clock 5, got %d", vc.GetClock("node1"))
	}

	if vc.GetClock("node2") != 3 {
		t.Errorf("Expected node2 clock 3, got %d", vc.GetClock("node2"))
	}

	// Modifying original map shouldn't affect vector clock
	clocks["node1"] = 10
	if vc.GetClock("node1") != 5 {
		t.Error("NewVectorClockFromMap should create a copy")
	}
}

// TestIncrement tests incrementing clock values
func TestIncrement(t *testing.T) {
	vc := NewVectorClock()

	// First increment initializes to 1
	vc.Increment("node1")
	if vc.GetClock("node1") != 1 {
		t.Errorf("Expected clock 1 after first increment, got %d", vc.GetClock("node1"))
	}

	// Further increments
	vc.Increment("node1")
	vc.Increment("node1")
	if vc.GetClock("node1") != 3 {
		t.Errorf("Expected clock 3 after three increments, got %d", vc.GetClock("node1"))
	}

	// Increment different node
	vc.Increment("node2")
	if vc.GetClock("node2") != 1 {
		t.Errorf("Expected clock 1 for node2, got %d", vc.GetClock("node2"))
	}
}

// TestUpdate tests merging vector clocks
func TestUpdate(t *testing.T) {
	vc1 := NewVectorClock()
	vc1.Increment("node1")
	vc1.Increment("node1")
	vc1.Increment("node2")

	vc2 := NewVectorClock()
	vc2.Increment("node1")
	vc2.Increment("node2")
	vc2.Increment("node2")
	vc2.Increment("node3")

	// Update vc1 with vc2
	vc1.Update(vc2)

	// Should take maximum of each position
	if vc1.GetClock("node1") != 2 {
		t.Errorf("Expected node1 clock 2, got %d", vc1.GetClock("node1"))
	}

	if vc1.GetClock("node2") != 2 {
		t.Errorf("Expected node2 clock 2, got %d", vc1.GetClock("node2"))
	}

	if vc1.GetClock("node3") != 1 {
		t.Errorf("Expected node3 clock 1, got %d", vc1.GetClock("node3"))
	}
}

// TestCopy tests creating a copy of vector clock
func TestCopy(t *testing.T) {
	vc := NewVectorClock()
	vc.Increment("node1")
	vc.Increment("node2")

	copy := vc.Copy()

	if !vc.Equal(copy) {
		t.Error("Copy should be equal to original")
	}

	// Modifying copy shouldn't affect original
	copy.Increment("node1")

	if vc.Equal(copy) {
		t.Error("Modifying copy should not affect original")
	}

	if vc.GetClock("node1") != 1 {
		t.Errorf("Original should still have clock 1 for node1, got %d", vc.GetClock("node1"))
	}
}

// TestIsAfter tests the IsAfter comparison
func TestIsAfter(t *testing.T) {
	vc1 := NewVectorClockFromMap(map[string]uint64{
		"node1": 2,
		"node2": 3,
	})

	vc2 := NewVectorClockFromMap(map[string]uint64{
		"node1": 1,
		"node2": 2,
	})

	if !vc1.IsAfter(vc2) {
		t.Error("vc1 should be after vc2")
	}

	if vc2.IsAfter(vc1) {
		t.Error("vc2 should not be after vc1")
	}

	// Equal clocks
	vc3 := vc1.Copy()
	if vc1.IsAfter(vc3) {
		t.Error("Equal clocks should not be after each other")
	}

	// Concurrent clocks
	vc4 := NewVectorClockFromMap(map[string]uint64{
		"node1": 3,
		"node2": 1,
	})

	if vc1.IsAfter(vc4) {
		t.Error("Concurrent clocks should not be after each other")
	}
}

// TestIsBefore tests the IsBefore comparison
func TestIsBefore(t *testing.T) {
	vc1 := NewVectorClockFromMap(map[string]uint64{
		"node1": 1,
		"node2": 2,
	})

	vc2 := NewVectorClockFromMap(map[string]uint64{
		"node1": 2,
		"node2": 3,
	})

	if !vc1.IsBefore(vc2) {
		t.Error("vc1 should be before vc2")
	}

	if vc2.IsBefore(vc1) {
		t.Error("vc2 should not be before vc1")
	}
}

// TestIsConcurrent tests detecting concurrent updates
func TestIsConcurrent(t *testing.T) {
	vc1 := NewVectorClockFromMap(map[string]uint64{
		"node1": 3,
		"node2": 1,
	})

	vc2 := NewVectorClockFromMap(map[string]uint64{
		"node1": 1,
		"node2": 3,
	})

	if !vc1.IsConcurrent(vc2) {
		t.Error("vc1 and vc2 should be concurrent")
	}

	if !vc2.IsConcurrent(vc1) {
		t.Error("Concurrency should be symmetric")
	}

	// Non-concurrent clocks
	vc3 := NewVectorClockFromMap(map[string]uint64{
		"node1": 4,
		"node2": 2,
	})

	if vc1.IsConcurrent(vc3) {
		t.Error("vc1 and vc3 should not be concurrent")
	}

	// Equal clocks should not be concurrent
	vc4 := vc1.Copy()
	if vc1.IsConcurrent(vc4) {
		t.Error("Equal clocks should not be concurrent")
	}
}

// TestEqual tests equality comparison
func TestEqual(t *testing.T) {
	vc1 := NewVectorClockFromMap(map[string]uint64{
		"node1": 2,
		"node2": 3,
	})

	vc2 := NewVectorClockFromMap(map[string]uint64{
		"node1": 2,
		"node2": 3,
	})

	if !vc1.Equal(vc2) {
		t.Error("vc1 and vc2 should be equal")
	}

	vc3 := NewVectorClockFromMap(map[string]uint64{
		"node1": 2,
		"node2": 4,
	})

	if vc1.Equal(vc3) {
		t.Error("vc1 and vc3 should not be equal")
	}

	// Test equality with implicit zeros
	vc4 := NewVectorClockFromMap(map[string]uint64{
		"node1": 2,
		"node2": 3,
		"node3": 0,
	})

	if !vc1.Equal(vc4) {
		t.Error("Vector clocks with explicit and implicit zeros should be equal")
	}
}

// TestGetClocks tests getting all clock values
func TestGetClocks(t *testing.T) {
	vc := NewVectorClockFromMap(map[string]uint64{
		"node1": 5,
		"node2": 3,
	})

	clocks := vc.GetClocks()

	if len(clocks) != 2 {
		t.Errorf("Expected 2 clocks, got %d", len(clocks))
	}

	if clocks["node1"] != 5 {
		t.Errorf("Expected node1 clock 5, got %d", clocks["node1"])
	}

	// Modifying returned map shouldn't affect vector clock
	clocks["node1"] = 10
	if vc.GetClock("node1") != 5 {
		t.Error("GetClocks should return a copy")
	}
}

// TestMerge tests merging two vector clocks
func TestMerge(t *testing.T) {
	vc1 := NewVectorClockFromMap(map[string]uint64{
		"node1": 5,
		"node2": 2,
	})

	vc2 := NewVectorClockFromMap(map[string]uint64{
		"node1": 3,
		"node2": 4,
		"node3": 1,
	})

	merged := vc1.Merge(vc2)

	// Should have maximum of each position
	if merged.GetClock("node1") != 5 {
		t.Errorf("Expected node1 clock 5, got %d", merged.GetClock("node1"))
	}

	if merged.GetClock("node2") != 4 {
		t.Errorf("Expected node2 clock 4, got %d", merged.GetClock("node2"))
	}

	if merged.GetClock("node3") != 1 {
		t.Errorf("Expected node3 clock 1, got %d", merged.GetClock("node3"))
	}

	// Original clocks should be unchanged
	if vc1.GetClock("node2") != 2 {
		t.Error("Merge should not modify original clocks")
	}
}

// TestString tests string representation
func TestString(t *testing.T) {
	vc := NewVectorClock()
	if vc.String() != "empty" {
		t.Errorf("Expected 'empty', got '%s'", vc.String())
	}

	vc.Increment("node1")
	str := vc.String()
	if str == "" || str == "empty" {
		t.Errorf("Expected non-empty string, got '%s'", str)
	}
}

// TestSize tests the size method
func TestSize(t *testing.T) {
	vc := NewVectorClock()
	if vc.Size() != 0 {
		t.Errorf("Expected size 0, got %d", vc.Size())
	}

	vc.Increment("node1")
	if vc.Size() != 1 {
		t.Errorf("Expected size 1, got %d", vc.Size())
	}

	vc.Increment("node2")
	vc.Increment("node3")
	if vc.Size() != 3 {
		t.Errorf("Expected size 3, got %d", vc.Size())
	}
}

// TestCompare tests the Compare method
func TestCompare(t *testing.T) {
	tests := []struct {
		name     string
		vc1      *VectorClock
		vc2      *VectorClock
		expected CompareResult
	}{
		{
			name:     "Equal",
			vc1:      NewVectorClockFromMap(map[string]uint64{"node1": 2, "node2": 3}),
			vc2:      NewVectorClockFromMap(map[string]uint64{"node1": 2, "node2": 3}),
			expected: Equal,
		},
		{
			name:     "After",
			vc1:      NewVectorClockFromMap(map[string]uint64{"node1": 3, "node2": 4}),
			vc2:      NewVectorClockFromMap(map[string]uint64{"node1": 2, "node2": 3}),
			expected: After,
		},
		{
			name:     "Before",
			vc1:      NewVectorClockFromMap(map[string]uint64{"node1": 1, "node2": 2}),
			vc2:      NewVectorClockFromMap(map[string]uint64{"node1": 2, "node2": 3}),
			expected: Before,
		},
		{
			name:     "Concurrent",
			vc1:      NewVectorClockFromMap(map[string]uint64{"node1": 3, "node2": 1}),
			vc2:      NewVectorClockFromMap(map[string]uint64{"node1": 1, "node2": 3}),
			expected: Concurrent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.vc1.Compare(tt.vc2)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestCausalityChain tests a chain of causal updates
func TestCausalityChain(t *testing.T) {
	// Simulate a series of operations on node1
	vc := NewVectorClock()

	// Operation 1
	vc.Increment("node1")
	snapshot1 := vc.Copy()

	// Operation 2
	vc.Increment("node1")
	snapshot2 := vc.Copy()

	// Operation 3
	vc.Increment("node1")
	snapshot3 := vc.Copy()

	// Verify causality chain
	if !snapshot2.IsAfter(snapshot1) {
		t.Error("snapshot2 should be after snapshot1")
	}

	if !snapshot3.IsAfter(snapshot2) {
		t.Error("snapshot3 should be after snapshot2")
	}

	if !snapshot3.IsAfter(snapshot1) {
		t.Error("snapshot3 should be after snapshot1")
	}

	if snapshot1.IsAfter(snapshot2) {
		t.Error("snapshot1 should not be after snapshot2")
	}
}

// TestDistributedScenario tests a realistic distributed scenario
func TestDistributedScenario(t *testing.T) {
	// Initial state
	node1VC := NewVectorClock()
	node2VC := NewVectorClock()

	// Node1 performs an operation
	node1VC.Increment("node1")

	// Node2 performs an operation (concurrent with node1)
	node2VC.Increment("node2")

	// These should be concurrent
	if !node1VC.IsConcurrent(node2VC) {
		t.Error("Concurrent updates should be detected")
	}

	// Node1 receives update from node2
	node1VC.Update(node2VC)

	// Now node1's clock should be after the original node2 state
	if !node1VC.IsAfter(node2VC) {
		t.Error("After merge, node1 should be ahead")
	}

	// Node1 performs another operation
	node1VC.Increment("node1")

	// Node2 receives the merged state
	node2VC.Update(node1VC)

	// They should be equal now
	if !node2VC.Equal(node1VC) {
		t.Error("After full sync, clocks should be equal")
	}
}

// TestZeroValues tests handling of zero/missing values
func TestZeroValues(t *testing.T) {
	vc := NewVectorClock()

	// Getting clock for non-existent node should return 0
	if vc.GetClock("nonexistent") != 0 {
		t.Error("GetClock for non-existent node should return 0")
	}

	// Update with empty vector clock should be safe
	emptyVC := NewVectorClock()
	vc.Increment("node1")
	vc.Update(emptyVC)

	if vc.GetClock("node1") != 1 {
		t.Error("Update with empty VC should not affect existing values")
	}
}
