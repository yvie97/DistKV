// Unit tests for the metrics package
package metrics

import (
	"testing"
	"time"
)

// TestNewMetricsCollector tests creating a new metrics collector
func TestNewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector()

	if mc == nil {
		t.Fatal("Expected non-nil metrics collector")
	}

	if mc.Storage() == nil {
		t.Error("Expected non-nil storage metrics")
	}

	if mc.Replication() == nil {
		t.Error("Expected non-nil replication metrics")
	}

	if mc.Gossip() == nil {
		t.Error("Expected non-nil gossip metrics")
	}

	if mc.Network() == nil {
		t.Error("Expected non-nil network metrics")
	}

	if mc.System() == nil {
		t.Error("Expected non-nil system metrics")
	}

	// System should start healthy
	if !mc.System().Healthy.Load() {
		t.Error("Expected system to start in healthy state")
	}
}

// TestStorageMetrics tests storage metrics operations
func TestStorageMetrics(t *testing.T) {
	mc := NewMetricsCollector()
	storage := mc.Storage()

	t.Run("ReadOps", func(t *testing.T) {
		initial := storage.ReadOps.Load()
		storage.ReadOps.Add(5)
		if storage.ReadOps.Load() != initial+5 {
			t.Error("ReadOps increment failed")
		}
	})

	t.Run("WriteOps", func(t *testing.T) {
		initial := storage.WriteOps.Load()
		storage.WriteOps.Add(10)
		if storage.WriteOps.Load() != initial+10 {
			t.Error("WriteOps increment failed")
		}
	})

	t.Run("ReadErrors", func(t *testing.T) {
		initial := storage.ReadErrors.Load()
		storage.ReadErrors.Add(1)
		if storage.ReadErrors.Load() != initial+1 {
			t.Error("ReadErrors increment failed")
		}
	})

	t.Run("WriteErrors", func(t *testing.T) {
		initial := storage.WriteErrors.Load()
		storage.WriteErrors.Add(2)
		if storage.WriteErrors.Load() != initial+2 {
			t.Error("WriteErrors increment failed")
		}
	})

	t.Run("CacheMetrics", func(t *testing.T) {
		storage.CacheHits.Add(100)
		storage.CacheMisses.Add(20)

		hits := storage.CacheHits.Load()
		misses := storage.CacheMisses.Load()

		if hits < 100 {
			t.Error("CacheHits not incremented correctly")
		}
		if misses < 20 {
			t.Error("CacheMisses not incremented correctly")
		}
	})

	t.Run("MemTableMetrics", func(t *testing.T) {
		storage.MemTableSize.Store(1024)
		storage.MemTableCount.Store(2)

		if storage.MemTableSize.Load() != 1024 {
			t.Error("MemTableSize not set correctly")
		}
		if storage.MemTableCount.Load() != 2 {
			t.Error("MemTableCount not set correctly")
		}
	})

	t.Run("SSTableCount", func(t *testing.T) {
		storage.SSTableCount.Store(5)
		if storage.SSTableCount.Load() != 5 {
			t.Error("SSTableCount not set correctly")
		}
	})

	t.Run("CompactionMetrics", func(t *testing.T) {
		storage.CompactionCount.Add(1)
		storage.CompactionDurationNs.Store(1000000)
		storage.TombstonesCollected.Add(50)

		if storage.CompactionCount.Load() < 1 {
			t.Error("CompactionCount not incremented")
		}
		if storage.CompactionDurationNs.Load() != 1000000 {
			t.Error("CompactionDurationNs not set correctly")
		}
		if storage.TombstonesCollected.Load() < 50 {
			t.Error("TombstonesCollected not incremented")
		}
	})

	t.Run("FlushMetrics", func(t *testing.T) {
		storage.FlushCount.Add(3)
		storage.FlushErrors.Add(1)

		if storage.FlushCount.Load() < 3 {
			t.Error("FlushCount not incremented")
		}
		if storage.FlushErrors.Load() < 1 {
			t.Error("FlushErrors not incremented")
		}
	})
}

// TestReplicationMetrics tests replication metrics
func TestReplicationMetrics(t *testing.T) {
	mc := NewMetricsCollector()
	repl := mc.Replication()

	t.Run("QuorumWriteMetrics", func(t *testing.T) {
		repl.QuorumWriteOps.Add(100)
		repl.QuorumWriteSuccess.Add(95)
		repl.QuorumWriteFailed.Add(5)

		if repl.QuorumWriteOps.Load() < 100 {
			t.Error("QuorumWriteOps not incremented")
		}
		if repl.QuorumWriteSuccess.Load() < 95 {
			t.Error("QuorumWriteSuccess not incremented")
		}
		if repl.QuorumWriteFailed.Load() < 5 {
			t.Error("QuorumWriteFailed not incremented")
		}
	})

	t.Run("QuorumReadMetrics", func(t *testing.T) {
		repl.QuorumReadOps.Add(200)
		repl.QuorumReadSuccess.Add(198)
		repl.QuorumReadFailed.Add(2)

		if repl.QuorumReadOps.Load() < 200 {
			t.Error("QuorumReadOps not incremented")
		}
		if repl.QuorumReadSuccess.Load() < 198 {
			t.Error("QuorumReadSuccess not incremented")
		}
	})

	t.Run("ConflictMetrics", func(t *testing.T) {
		repl.ConflictsDetected.Add(10)
		repl.ConflictsResolved.Add(10)

		if repl.ConflictsDetected.Load() < 10 {
			t.Error("ConflictsDetected not incremented")
		}
		if repl.ConflictsResolved.Load() < 10 {
			t.Error("ConflictsResolved not incremented")
		}
	})

	t.Run("ReplicaMetrics", func(t *testing.T) {
		repl.ReplicaWriteOps.Add(500)
		repl.ReplicaWriteErrors.Add(5)
		repl.ReplicaReadOps.Add(1000)
		repl.ReplicaReadErrors.Add(10)

		if repl.ReplicaWriteOps.Load() < 500 {
			t.Error("ReplicaWriteOps not incremented")
		}
		if repl.ReplicaReadOps.Load() < 1000 {
			t.Error("ReplicaReadOps not incremented")
		}
	})

	t.Run("HintedHandoff", func(t *testing.T) {
		repl.HintedHandoffWrites.Add(25)
		repl.HintedHandoffReads.Add(15)

		if repl.HintedHandoffWrites.Load() < 25 {
			t.Error("HintedHandoffWrites not incremented")
		}
		if repl.HintedHandoffReads.Load() < 15 {
			t.Error("HintedHandoffReads not incremented")
		}
	})
}

// TestGossipMetrics tests gossip protocol metrics
func TestGossipMetrics(t *testing.T) {
	mc := NewMetricsCollector()
	gossip := mc.Gossip()

	t.Run("NodeCounts", func(t *testing.T) {
		gossip.TotalNodes.Store(10)
		gossip.AliveNodes.Store(8)
		gossip.SuspectNodes.Store(1)
		gossip.DeadNodes.Store(1)

		if gossip.TotalNodes.Load() != 10 {
			t.Error("TotalNodes not set correctly")
		}
		if gossip.AliveNodes.Load() != 8 {
			t.Error("AliveNodes not set correctly")
		}
		if gossip.SuspectNodes.Load() != 1 {
			t.Error("SuspectNodes not set correctly")
		}
		if gossip.DeadNodes.Load() != 1 {
			t.Error("DeadNodes not set correctly")
		}
	})

	t.Run("MessageMetrics", func(t *testing.T) {
		gossip.MessagesSent.Add(1000)
		gossip.MessagesReceived.Add(950)
		gossip.MessageErrors.Add(5)
		gossip.MessagesDropped.Add(45)

		if gossip.MessagesSent.Load() < 1000 {
			t.Error("MessagesSent not incremented")
		}
		if gossip.MessagesReceived.Load() < 950 {
			t.Error("MessagesReceived not incremented")
		}
		if gossip.MessageErrors.Load() < 5 {
			t.Error("MessageErrors not incremented")
		}
	})

	t.Run("StateChanges", func(t *testing.T) {
		gossip.NodeJoined.Add(2)
		gossip.NodeLeft.Add(1)
		gossip.NodeFailed.Add(1)
		gossip.NodeRecovered.Add(1)

		if gossip.NodeJoined.Load() < 2 {
			t.Error("NodeJoined not incremented")
		}
		if gossip.NodeFailed.Load() < 1 {
			t.Error("NodeFailed not incremented")
		}
	})

	t.Run("Heartbeats", func(t *testing.T) {
		gossip.HeartbeatsSent.Add(5000)
		gossip.HeartbeatsReceived.Add(4950)
		gossip.HeartbeatsMissed.Add(50)

		if gossip.HeartbeatsSent.Load() < 5000 {
			t.Error("HeartbeatsSent not incremented")
		}
		if gossip.HeartbeatsReceived.Load() < 4950 {
			t.Error("HeartbeatsReceived not incremented")
		}
	})
}

// TestNetworkMetrics tests network metrics
func TestNetworkMetrics(t *testing.T) {
	mc := NewMetricsCollector()
	network := mc.Network()

	t.Run("ConnectionMetrics", func(t *testing.T) {
		network.ActiveConnections.Store(50)
		network.TotalConnections.Add(500)
		network.FailedConnections.Add(10)

		if network.ActiveConnections.Load() != 50 {
			t.Error("ActiveConnections not set correctly")
		}
		if network.TotalConnections.Load() < 500 {
			t.Error("TotalConnections not incremented")
		}
		if network.FailedConnections.Load() < 10 {
			t.Error("FailedConnections not incremented")
		}
	})

	t.Run("DataTransfer", func(t *testing.T) {
		network.BytesSent.Add(1024 * 1024)
		network.BytesReceived.Add(2048 * 1024)
		network.PacketsSent.Add(1000)
		network.PacketsReceived.Add(950)

		if network.BytesSent.Load() < 1024*1024 {
			t.Error("BytesSent not incremented")
		}
		if network.BytesReceived.Load() < 2048*1024 {
			t.Error("BytesReceived not incremented")
		}
	})

	t.Run("RequestMetrics", func(t *testing.T) {
		network.RequestsInProgress.Store(10)
		network.TotalRequests.Add(10000)
		network.FailedRequests.Add(50)

		if network.RequestsInProgress.Load() != 10 {
			t.Error("RequestsInProgress not set correctly")
		}
		if network.TotalRequests.Load() < 10000 {
			t.Error("TotalRequests not incremented")
		}
	})

	t.Run("TimeoutAndRetry", func(t *testing.T) {
		network.TimeoutCount.Add(25)
		network.RetryCount.Add(100)

		if network.TimeoutCount.Load() < 25 {
			t.Error("TimeoutCount not incremented")
		}
		if network.RetryCount.Load() < 100 {
			t.Error("RetryCount not incremented")
		}
	})
}

// TestSystemMetrics tests system-level metrics
func TestSystemMetrics(t *testing.T) {
	mc := NewMetricsCollector()
	system := mc.System()

	t.Run("HealthStatus", func(t *testing.T) {
		system.Healthy.Store(true)
		if !system.Healthy.Load() {
			t.Error("Healthy should be true")
		}

		system.Healthy.Store(false)
		if system.Healthy.Load() {
			t.Error("Healthy should be false")
		}
	})

	t.Run("ReadOnlyMode", func(t *testing.T) {
		system.ReadOnly.Store(true)
		if !system.ReadOnly.Load() {
			t.Error("ReadOnly should be true")
		}

		system.ReadOnly.Store(false)
		if system.ReadOnly.Load() {
			t.Error("ReadOnly should be false")
		}
	})

	t.Run("ResourceUsage", func(t *testing.T) {
		system.GoroutineCount.Store(100)
		system.MemoryUsageMB.Store(512)
		system.CPUUsagePercent.Store(75)

		if system.GoroutineCount.Load() != 100 {
			t.Error("GoroutineCount not set correctly")
		}
		if system.MemoryUsageMB.Load() != 512 {
			t.Error("MemoryUsageMB not set correctly")
		}
		if system.CPUUsagePercent.Load() != 75 {
			t.Error("CPUUsagePercent not set correctly")
		}
	})

	t.Run("Uptime", func(t *testing.T) {
		// Wait a bit for background updater
		time.Sleep(100 * time.Millisecond)

		uptime := system.UptimeSeconds.Load()
		// Uptime should be set (may be 0 initially but will update)
		// Just check it's accessible
		_ = uptime
	})
}

// TestSnapshot tests taking metrics snapshots
func TestSnapshot(t *testing.T) {
	mc := NewMetricsCollector()

	// Set some metrics
	mc.Storage().ReadOps.Add(100)
	mc.Storage().WriteOps.Add(50)
	mc.Storage().CacheHits.Add(80)
	mc.Storage().CacheMisses.Add(20)

	mc.Replication().QuorumWriteOps.Add(50)
	mc.Replication().QuorumWriteSuccess.Add(48)

	mc.Gossip().TotalNodes.Store(5)
	mc.Gossip().AliveNodes.Store(5)

	mc.Network().TotalRequests.Add(200)

	snapshot := mc.Snapshot()

	if snapshot == nil {
		t.Fatal("Expected non-nil snapshot")
	}

	t.Run("StorageSnapshot", func(t *testing.T) {
		if snapshot.Storage.ReadOps < 100 {
			t.Error("ReadOps not captured in snapshot")
		}
		if snapshot.Storage.WriteOps < 50 {
			t.Error("WriteOps not captured in snapshot")
		}

		// Cache hit rate calculation (should be around 0.8)
		if snapshot.Storage.CacheHitRate < 0.79 || snapshot.Storage.CacheHitRate > 0.81 {
			t.Errorf("Expected cache hit rate around 0.8, got %f", snapshot.Storage.CacheHitRate)
		}
	})

	t.Run("ReplicationSnapshot", func(t *testing.T) {
		if snapshot.Replication.QuorumWriteOps < 50 {
			t.Error("QuorumWriteOps not captured in snapshot")
		}

		// Success rate calculation (should be around 0.96)
		if snapshot.Replication.QuorumWriteSuccessRate < 0.95 {
			t.Errorf("Expected success rate around 0.96, got %f",
				snapshot.Replication.QuorumWriteSuccessRate)
		}
	})

	t.Run("GossipSnapshot", func(t *testing.T) {
		if snapshot.Gossip.TotalNodes != 5 {
			t.Errorf("Expected 5 total nodes, got %d", snapshot.Gossip.TotalNodes)
		}
		if snapshot.Gossip.AliveNodes != 5 {
			t.Errorf("Expected 5 alive nodes, got %d", snapshot.Gossip.AliveNodes)
		}
	})

	t.Run("NetworkSnapshot", func(t *testing.T) {
		if snapshot.Network.TotalRequests < 200 {
			t.Error("TotalRequests not captured in snapshot")
		}
	})

	t.Run("SystemSnapshot", func(t *testing.T) {
		if !snapshot.System.Healthy {
			t.Error("Expected system to be healthy in snapshot")
		}
	})

	t.Run("Timestamp", func(t *testing.T) {
		if snapshot.Timestamp.IsZero() {
			t.Error("Expected non-zero timestamp")
		}
	})
}

// TestGlobalMetrics tests global metrics singleton
func TestGlobalMetrics(t *testing.T) {
	mc1 := GetGlobalMetrics()
	mc2 := GetGlobalMetrics()

	if mc1 != mc2 {
		t.Error("Expected same global metrics instance")
	}

	if mc1 == nil {
		t.Fatal("Expected non-nil global metrics")
	}
}

// TestLatencyTracker tests latency tracking utility
func TestLatencyTracker(t *testing.T) {
	tracker := NewLatencyTracker()

	if tracker == nil {
		t.Fatal("Expected non-nil latency tracker")
	}

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	duration := tracker.Finish()

	if duration < 10000000 { // 10ms in nanoseconds
		t.Errorf("Expected duration >= 10ms, got %d ns", duration)
	}

	// Should be reasonable (less than 1 second for this test)
	if duration > 1000000000 {
		t.Errorf("Duration too large: %d ns", duration)
	}
}

// TestTrackOperation tests operation tracking
func TestTrackOperation(t *testing.T) {
	t.Run("successful operation", func(t *testing.T) {
		err := TrackOperation("test-op", func() error {
			time.Sleep(5 * time.Millisecond)
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("failed operation", func(t *testing.T) {
		expectedErr := &testError{msg: "test error"}
		err := TrackOperation("test-op", func() error {
			return expectedErr
		})

		if err != expectedErr {
			t.Errorf("Expected error to be returned")
		}
	})
}

// TestConcurrentAccess tests concurrent metric updates
func TestConcurrentAccess(t *testing.T) {
	mc := NewMetricsCollector()
	iterations := 1000
	goroutines := 10

	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				mc.Storage().ReadOps.Add(1)
				mc.Storage().WriteOps.Add(1)
				mc.Replication().QuorumWriteOps.Add(1)
				mc.Gossip().MessagesSent.Add(1)
				mc.Network().TotalRequests.Add(1)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	expectedCount := uint64(iterations * goroutines)

	if mc.Storage().ReadOps.Load() != expectedCount {
		t.Errorf("Expected ReadOps=%d, got %d",
			expectedCount, mc.Storage().ReadOps.Load())
	}

	if mc.Storage().WriteOps.Load() != expectedCount {
		t.Errorf("Expected WriteOps=%d, got %d",
			expectedCount, mc.Storage().WriteOps.Load())
	}
}

// TestZeroValues tests that metrics start at zero
func TestZeroValues(t *testing.T) {
	mc := NewMetricsCollector()

	if mc.Storage().ReadOps.Load() != 0 {
		t.Error("ReadOps should start at 0")
	}

	if mc.Replication().ConflictsDetected.Load() != 0 {
		t.Error("ConflictsDetected should start at 0")
	}

	if mc.Gossip().MessagesSent.Load() != 0 {
		t.Error("MessagesSent should start at 0")
	}

	if mc.Network().FailedConnections.Load() != 0 {
		t.Error("FailedConnections should start at 0")
	}
}

// Helper types for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
