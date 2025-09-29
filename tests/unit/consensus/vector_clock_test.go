package consensus_test

import (
	"distkv/pkg/consensus"
	"testing"
)

func TestVectorClock_NewAndBasicOperations(t *testing.T) {
	vc := consensus.NewVectorClock()

	if vc.Size() != 0 {
		t.Errorf("New vector clock should be empty, got size %d", vc.Size())
	}

	// Test increment
	vc.Increment("node1")
	if vc.GetClock("node1") != 1 {
		t.Errorf("Expected clock for node1 to be 1, got %d", vc.GetClock("node1"))
	}

	if vc.Size() != 1 {
		t.Errorf("Expected size 1 after adding node, got %d", vc.Size())
	}

	// Test multiple increments
	vc.Increment("node1")
	vc.Increment("node1")
	if vc.GetClock("node1") != 3 {
		t.Errorf("Expected clock for node1 to be 3, got %d", vc.GetClock("node1"))
	}

	// Test different node
	vc.Increment("node2")
	if vc.GetClock("node2") != 1 {
		t.Errorf("Expected clock for node2 to be 1, got %d", vc.GetClock("node2"))
	}

	if vc.Size() != 2 {
		t.Errorf("Expected size 2 after adding second node, got %d", vc.Size())
	}
}

func TestVectorClock_FromMap(t *testing.T) {
	clocks := map[string]uint64{
		"node1": 5,
		"node2": 3,
		"node3": 7,
	}

	vc := consensus.NewVectorClockFromMap(clocks)

	if vc.Size() != 3 {
		t.Errorf("Expected size 3, got %d", vc.Size())
	}

	for nodeID, expectedClock := range clocks {
		if vc.GetClock(nodeID) != expectedClock {
			t.Errorf("Node %s: expected clock %d, got %d", nodeID, expectedClock, vc.GetClock(nodeID))
		}
	}
}

func TestVectorClock_Update(t *testing.T) {
	vc1 := consensus.NewVectorClock()
	vc1.Increment("node1")
	vc1.Increment("node1")
	vc1.Increment("node2")

	vc2 := consensus.NewVectorClock()
	vc2.Increment("node2")
	vc2.Increment("node2")
	vc2.Increment("node3")

	// Update vc1 with vc2
	vc1.Update(vc2)

	// Check that vc1 now has the maximum of each node's clock
	if vc1.GetClock("node1") != 2 {
		t.Errorf("Expected node1 clock to remain 2, got %d", vc1.GetClock("node1"))
	}

	if vc1.GetClock("node2") != 2 {
		t.Errorf("Expected node2 clock to be max(1,2)=2, got %d", vc1.GetClock("node2"))
	}

	if vc1.GetClock("node3") != 1 {
		t.Errorf("Expected node3 clock to be 1, got %d", vc1.GetClock("node3"))
	}
}

func TestVectorClock_Copy(t *testing.T) {
	original := consensus.NewVectorClock()
	original.Increment("node1")
	original.Increment("node2")

	copy := original.Copy()

	// Verify copy has same values
	if copy.GetClock("node1") != original.GetClock("node1") {
		t.Error("Copy should have same node1 clock as original")
	}

	if copy.GetClock("node2") != original.GetClock("node2") {
		t.Error("Copy should have same node2 clock as original")
	}

	// Verify they're independent
	copy.Increment("node1")
	if copy.GetClock("node1") == original.GetClock("node1") {
		t.Error("Copy and original should be independent")
	}
}

func TestVectorClock_Equal(t *testing.T) {
	vc1 := consensus.NewVectorClock()
	vc1.Increment("node1")
	vc1.Increment("node2")

	vc2 := consensus.NewVectorClock()
	vc2.Increment("node1")
	vc2.Increment("node2")

	if !vc1.Equal(vc2) {
		t.Error("Vector clocks with same values should be equal")
	}

	vc2.Increment("node1")
	if vc1.Equal(vc2) {
		t.Error("Vector clocks with different values should not be equal")
	}

	// Test with different nodes
	vc3 := consensus.NewVectorClock()
	vc3.Increment("node3")

	if vc1.Equal(vc3) {
		t.Error("Vector clocks with different nodes should not be equal")
	}
}

func TestVectorClock_Causality(t *testing.T) {
	// Test happens-before relationship
	vc1 := consensus.NewVectorClock()
	vc1.Increment("node1")

	vc2 := vc1.Copy()
	vc2.Increment("node1")
	vc2.Increment("node2")

	// vc1 should happen before vc2
	if !vc2.IsAfter(vc1) {
		t.Error("vc2 should be after vc1")
	}

	if !vc1.IsBefore(vc2) {
		t.Error("vc1 should be before vc2")
	}

	if vc1.IsAfter(vc2) {
		t.Error("vc1 should not be after vc2")
	}

	if vc2.IsBefore(vc1) {
		t.Error("vc2 should not be before vc1")
	}
}

func TestVectorClock_Concurrent(t *testing.T) {
	// Create concurrent vector clocks
	vc1 := consensus.NewVectorClock()
	vc1.Increment("node1")
	vc1.Increment("node1")

	vc2 := consensus.NewVectorClock()
	vc2.Increment("node2")
	vc2.Increment("node2")

	// They should be concurrent
	if !vc1.IsConcurrent(vc2) {
		t.Error("vc1 and vc2 should be concurrent")
	}

	if !vc2.IsConcurrent(vc1) {
		t.Error("vc2 and vc1 should be concurrent")
	}

	if vc1.IsAfter(vc2) || vc1.IsBefore(vc2) {
		t.Error("Concurrent clocks should not have happens-before relationship")
	}
}

func TestVectorClock_Compare(t *testing.T) {
	vc1 := consensus.NewVectorClock()
	vc1.Increment("node1")

	vc2 := consensus.NewVectorClock()
	vc2.Increment("node1")

	// Equal clocks
	if vc1.Compare(vc2) != consensus.Equal {
		t.Error("Equal clocks should return Equal")
	}

	// After relationship
	vc2.Increment("node1")
	if vc2.Compare(vc1) != consensus.After {
		t.Error("vc2 should be After vc1")
	}

	if vc1.Compare(vc2) != consensus.Before {
		t.Error("vc1 should be Before vc2")
	}

	// Concurrent relationship
	vc3 := consensus.NewVectorClock()
	vc3.Increment("node2")

	if vc1.Compare(vc3) != consensus.Concurrent {
		t.Error("vc1 and vc3 should be Concurrent")
	}
}

func TestVectorClock_Merge(t *testing.T) {
	vc1 := consensus.NewVectorClock()
	vc1.Increment("node1")
	vc1.Increment("node1")
	vc1.Increment("node2")

	vc2 := consensus.NewVectorClock()
	vc2.Increment("node2")
	vc2.Increment("node2")
	vc2.Increment("node3")

	merged := vc1.Merge(vc2)

	// Merged should have maximum of each node's clock
	if merged.GetClock("node1") != 2 {
		t.Errorf("Merged clock for node1 should be 2, got %d", merged.GetClock("node1"))
	}

	if merged.GetClock("node2") != 2 {
		t.Errorf("Merged clock for node2 should be max(1,2)=2, got %d", merged.GetClock("node2"))
	}

	if merged.GetClock("node3") != 1 {
		t.Errorf("Merged clock for node3 should be 1, got %d", merged.GetClock("node3"))
	}

	// Original clocks should be unchanged
	if vc1.GetClock("node3") != 0 {
		t.Error("Original vc1 should not be modified by merge")
	}

	if vc2.GetClock("node1") != 0 {
		t.Error("Original vc2 should not be modified by merge")
	}
}

func TestVectorClock_String(t *testing.T) {
	vc := consensus.NewVectorClock()

	// Empty clock
	if vc.String() != "empty" {
		t.Errorf("Empty clock string should be 'empty', got '%s'", vc.String())
	}

	// Single node
	vc.Increment("node1")
	str := vc.String()
	if str != "node1:1" {
		t.Errorf("Single node string should be 'node1:1', got '%s'", str)
	}

	// Multiple nodes (note: order might vary due to map iteration)
	vc.Increment("node2")
	str = vc.String()

	// Check that both nodes are present
	if !contains(str, "node1:1") {
		t.Errorf("String should contain 'node1:1', got '%s'", str)
	}

	if !contains(str, "node2:1") {
		t.Errorf("String should contain 'node2:1', got '%s'", str)
	}
}

func TestVectorClock_GetClocks(t *testing.T) {
	vc := consensus.NewVectorClock()
	vc.Increment("node1")
	vc.Increment("node2")
	vc.Increment("node1")

	clocks := vc.GetClocks()

	if len(clocks) != 2 {
		t.Errorf("Expected 2 clocks, got %d", len(clocks))
	}

	if clocks["node1"] != 2 {
		t.Errorf("Expected node1 clock to be 2, got %d", clocks["node1"])
	}

	if clocks["node2"] != 1 {
		t.Errorf("Expected node2 clock to be 1, got %d", clocks["node2"])
	}

	// Verify it's a copy (modifying returned map shouldn't affect original)
	clocks["node1"] = 999
	if vc.GetClock("node1") == 999 {
		t.Error("GetClocks should return a copy, not the original map")
	}
}

func TestVectorClock_ComplexScenario(t *testing.T) {
	// Simulate a distributed system scenario with 3 nodes
	node1 := consensus.NewVectorClock()
	node2 := consensus.NewVectorClock()
	node3 := consensus.NewVectorClock()

	// Node1 performs an operation
	node1.Increment("node1")

	// Node1 sends message to Node2
	node2.Update(node1)
	node2.Increment("node2")

	// Node2 sends message to Node3
	node3.Update(node2)
	node3.Increment("node3")

	// Node3 sends message back to Node1
	node1.Update(node3)
	node1.Increment("node1")

	// Verify final state
	// Node1 should have: node1=2, node2=1, node3=1
	if node1.GetClock("node1") != 2 {
		t.Errorf("Node1 should have node1=2, got %d", node1.GetClock("node1"))
	}
	if node1.GetClock("node2") != 1 {
		t.Errorf("Node1 should have node2=1, got %d", node1.GetClock("node2"))
	}
	if node1.GetClock("node3") != 1 {
		t.Errorf("Node1 should have node3=1, got %d", node1.GetClock("node3"))
	}

	// Test causality relationships
	initialNode1 := consensus.NewVectorClock()
	initialNode1.Increment("node1")

	if !node1.IsAfter(initialNode1) {
		t.Error("Final node1 state should be after initial node1 state")
	}

	if !node2.IsAfter(initialNode1) {
		t.Error("Node2 state should be after initial node1 state")
	}

	if !node3.IsAfter(node2) {
		t.Error("Node3 state should be after node2 state")
	}
}

// Helper function for string tests
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}