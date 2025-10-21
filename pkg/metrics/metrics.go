// Package metrics provides comprehensive monitoring and metrics collection for DistKV.
// It tracks performance, health, and operational metrics across all components.
package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector is the central metrics collection system
type MetricsCollector struct {
	storage     *StorageMetrics
	replication *ReplicationMetrics
	gossip      *GossipMetrics
	network     *NetworkMetrics
	system      *SystemMetrics
	mutex       sync.RWMutex
	startTime   time.Time
}

// StorageMetrics tracks storage engine performance
type StorageMetrics struct {
	// Read metrics
	ReadOps       atomic.Uint64
	ReadErrors    atomic.Uint64
	ReadLatencyNs atomic.Uint64 // Average in nanoseconds
	CacheHits     atomic.Uint64
	CacheMisses   atomic.Uint64

	// Write metrics
	WriteOps       atomic.Uint64
	WriteErrors    atomic.Uint64
	WriteLatencyNs atomic.Uint64

	// Storage metrics
	MemTableSize      atomic.Int64
	MemTableCount     atomic.Int64
	SSTableCount      atomic.Int64
	TotalDataSize     atomic.Int64
	BloomFilterHits   atomic.Uint64
	BloomFilterMisses atomic.Uint64

	// Compaction metrics
	CompactionCount      atomic.Uint64
	CompactionErrors     atomic.Uint64
	CompactionDurationNs atomic.Uint64
	TombstonesCollected  atomic.Uint64

	// Flush metrics
	FlushCount      atomic.Uint64
	FlushErrors     atomic.Uint64
	FlushDurationNs atomic.Uint64
}

// ReplicationMetrics tracks replication performance
type ReplicationMetrics struct {
	// Quorum write metrics
	QuorumWriteOps       atomic.Uint64
	QuorumWriteSuccess   atomic.Uint64
	QuorumWriteFailed    atomic.Uint64
	QuorumWriteLatencyNs atomic.Uint64

	// Quorum read metrics
	QuorumReadOps       atomic.Uint64
	QuorumReadSuccess   atomic.Uint64
	QuorumReadFailed    atomic.Uint64
	QuorumReadLatencyNs atomic.Uint64

	// Replica metrics
	ReplicaWriteOps    atomic.Uint64
	ReplicaWriteErrors atomic.Uint64
	ReplicaReadOps     atomic.Uint64
	ReplicaReadErrors  atomic.Uint64

	// Conflict resolution
	ConflictsDetected atomic.Uint64
	ConflictsResolved atomic.Uint64

	// Hinted handoff
	HintedHandoffWrites atomic.Uint64
	HintedHandoffReads  atomic.Uint64
}

// GossipMetrics tracks gossip protocol metrics
type GossipMetrics struct {
	// Node tracking
	TotalNodes   atomic.Int64
	AliveNodes   atomic.Int64
	SuspectNodes atomic.Int64
	DeadNodes    atomic.Int64

	// Gossip messages
	MessagesSent     atomic.Uint64
	MessagesReceived atomic.Uint64
	MessageErrors    atomic.Uint64
	MessagesDropped  atomic.Uint64

	// State changes
	NodeJoined    atomic.Uint64
	NodeLeft      atomic.Uint64
	NodeFailed    atomic.Uint64
	NodeRecovered atomic.Uint64

	// Heartbeats
	HeartbeatsSent     atomic.Uint64
	HeartbeatsReceived atomic.Uint64
	HeartbeatsMissed   atomic.Uint64
}

// NetworkMetrics tracks network performance
type NetworkMetrics struct {
	// Connection metrics
	ActiveConnections atomic.Int64
	TotalConnections  atomic.Uint64
	FailedConnections atomic.Uint64
	ConnectionErrors  atomic.Uint64

	// Data transfer
	BytesSent       atomic.Uint64
	BytesReceived   atomic.Uint64
	PacketsSent     atomic.Uint64
	PacketsReceived atomic.Uint64

	// Request metrics
	RequestsInProgress atomic.Int64
	TotalRequests      atomic.Uint64
	FailedRequests     atomic.Uint64
	RequestLatencyNs   atomic.Uint64

	// Timeout metrics
	TimeoutCount atomic.Uint64
	RetryCount   atomic.Uint64
}

// SystemMetrics tracks system-level metrics
type SystemMetrics struct {
	// Health status
	Healthy  atomic.Bool
	ReadOnly atomic.Bool

	// Resource usage (simplified - would integrate with OS metrics in production)
	GoroutineCount  atomic.Int64
	MemoryUsageMB   atomic.Int64
	CPUUsagePercent atomic.Int64

	// Uptime
	UptimeSeconds atomic.Uint64
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		storage:     &StorageMetrics{},
		replication: &ReplicationMetrics{},
		gossip:      &GossipMetrics{},
		network:     &NetworkMetrics{},
		system:      &SystemMetrics{},
		startTime:   time.Now(),
	}

	// Initialize healthy state
	mc.system.Healthy.Store(true)

	// Start background metric updater
	go mc.updateSystemMetrics()

	return mc
}

// Storage returns storage metrics
func (mc *MetricsCollector) Storage() *StorageMetrics {
	return mc.storage
}

// Replication returns replication metrics
func (mc *MetricsCollector) Replication() *ReplicationMetrics {
	return mc.replication
}

// Gossip returns gossip metrics
func (mc *MetricsCollector) Gossip() *GossipMetrics {
	return mc.gossip
}

// Network returns network metrics
func (mc *MetricsCollector) Network() *NetworkMetrics {
	return mc.network
}

// System returns system metrics
func (mc *MetricsCollector) System() *SystemMetrics {
	return mc.system
}

// updateSystemMetrics periodically updates system-level metrics
func (mc *MetricsCollector) updateSystemMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Update uptime
		uptime := uint64(time.Since(mc.startTime).Seconds())
		mc.system.UptimeSeconds.Store(uptime)

		// Update goroutine count
		// Note: In production, we'd use runtime.NumGoroutine() here
		// For now, this is a placeholder
	}
}

// Snapshot returns a complete snapshot of all metrics
func (mc *MetricsCollector) Snapshot() *MetricsSnapshot {
	return &MetricsSnapshot{
		Timestamp:   time.Now(),
		Storage:     mc.snapshotStorage(),
		Replication: mc.snapshotReplication(),
		Gossip:      mc.snapshotGossip(),
		Network:     mc.snapshotNetwork(),
		System:      mc.snapshotSystem(),
	}
}

// MetricsSnapshot is a point-in-time snapshot of all metrics
type MetricsSnapshot struct {
	Timestamp   time.Time
	Storage     StorageSnapshot
	Replication ReplicationSnapshot
	Gossip      GossipSnapshot
	Network     NetworkSnapshot
	System      SystemSnapshot
}

type StorageSnapshot struct {
	ReadOps            uint64
	ReadErrors         uint64
	AvgReadLatencyMs   float64
	CacheHitRate       float64
	WriteOps           uint64
	WriteErrors        uint64
	AvgWriteLatencyMs  float64
	MemTableSize       int64
	MemTableCount      int64
	SSTableCount       int64
	TotalDataSize      int64
	CompactionCount    uint64
	FlushCount         uint64
	BloomFilterHitRate float64
}

type ReplicationSnapshot struct {
	QuorumWriteOps          uint64
	QuorumWriteSuccess      uint64
	QuorumWriteSuccessRate  float64
	AvgQuorumWriteLatencyMs float64
	QuorumReadOps           uint64
	QuorumReadSuccess       uint64
	QuorumReadSuccessRate   float64
	AvgQuorumReadLatencyMs  float64
	ConflictsDetected       uint64
	ConflictsResolved       uint64
}

type GossipSnapshot struct {
	TotalNodes       int64
	AliveNodes       int64
	SuspectNodes     int64
	DeadNodes        int64
	MessagesSent     uint64
	MessagesReceived uint64
	MessageErrors    uint64
	NodeJoined       uint64
	NodeFailed       uint64
}

type NetworkSnapshot struct {
	ActiveConnections   int64
	FailedConnections   uint64
	BytesSent           uint64
	BytesReceived       uint64
	TotalRequests       uint64
	FailedRequests      uint64
	AvgRequestLatencyMs float64
	TimeoutCount        uint64
}

type SystemSnapshot struct {
	Healthy         bool
	ReadOnly        bool
	UptimeSeconds   uint64
	GoroutineCount  int64
	MemoryUsageMB   int64
	CPUUsagePercent int64
}

func (mc *MetricsCollector) snapshotStorage() StorageSnapshot {
	s := mc.storage

	readOps := s.ReadOps.Load()
	writeOps := s.WriteOps.Load()
	cacheHits := s.CacheHits.Load()
	cacheMisses := s.CacheMisses.Load()
	bloomHits := s.BloomFilterHits.Load()
	bloomMisses := s.BloomFilterMisses.Load()

	var cacheHitRate, bloomHitRate float64
	if total := cacheHits + cacheMisses; total > 0 {
		cacheHitRate = float64(cacheHits) / float64(total)
	}
	if total := bloomHits + bloomMisses; total > 0 {
		bloomHitRate = float64(bloomHits) / float64(total)
	}

	return StorageSnapshot{
		ReadOps:            readOps,
		ReadErrors:         s.ReadErrors.Load(),
		AvgReadLatencyMs:   float64(s.ReadLatencyNs.Load()) / 1e6,
		CacheHitRate:       cacheHitRate,
		WriteOps:           writeOps,
		WriteErrors:        s.WriteErrors.Load(),
		AvgWriteLatencyMs:  float64(s.WriteLatencyNs.Load()) / 1e6,
		MemTableSize:       s.MemTableSize.Load(),
		MemTableCount:      s.MemTableCount.Load(),
		SSTableCount:       s.SSTableCount.Load(),
		TotalDataSize:      s.TotalDataSize.Load(),
		CompactionCount:    s.CompactionCount.Load(),
		FlushCount:         s.FlushCount.Load(),
		BloomFilterHitRate: bloomHitRate,
	}
}

func (mc *MetricsCollector) snapshotReplication() ReplicationSnapshot {
	r := mc.replication

	writeOps := r.QuorumWriteOps.Load()
	writeSuccess := r.QuorumWriteSuccess.Load()
	readOps := r.QuorumReadOps.Load()
	readSuccess := r.QuorumReadSuccess.Load()

	var writeSuccessRate, readSuccessRate float64
	if writeOps > 0 {
		writeSuccessRate = float64(writeSuccess) / float64(writeOps)
	}
	if readOps > 0 {
		readSuccessRate = float64(readSuccess) / float64(readOps)
	}

	return ReplicationSnapshot{
		QuorumWriteOps:          writeOps,
		QuorumWriteSuccess:      writeSuccess,
		QuorumWriteSuccessRate:  writeSuccessRate,
		AvgQuorumWriteLatencyMs: float64(r.QuorumWriteLatencyNs.Load()) / 1e6,
		QuorumReadOps:           readOps,
		QuorumReadSuccess:       readSuccess,
		QuorumReadSuccessRate:   readSuccessRate,
		AvgQuorumReadLatencyMs:  float64(r.QuorumReadLatencyNs.Load()) / 1e6,
		ConflictsDetected:       r.ConflictsDetected.Load(),
		ConflictsResolved:       r.ConflictsResolved.Load(),
	}
}

func (mc *MetricsCollector) snapshotGossip() GossipSnapshot {
	g := mc.gossip

	return GossipSnapshot{
		TotalNodes:       g.TotalNodes.Load(),
		AliveNodes:       g.AliveNodes.Load(),
		SuspectNodes:     g.SuspectNodes.Load(),
		DeadNodes:        g.DeadNodes.Load(),
		MessagesSent:     g.MessagesSent.Load(),
		MessagesReceived: g.MessagesReceived.Load(),
		MessageErrors:    g.MessageErrors.Load(),
		NodeJoined:       g.NodeJoined.Load(),
		NodeFailed:       g.NodeFailed.Load(),
	}
}

func (mc *MetricsCollector) snapshotNetwork() NetworkSnapshot {
	n := mc.network

	totalReqs := n.TotalRequests.Load()
	failedReqs := n.FailedRequests.Load()

	return NetworkSnapshot{
		ActiveConnections:   n.ActiveConnections.Load(),
		FailedConnections:   n.FailedConnections.Load(),
		BytesSent:           n.BytesSent.Load(),
		BytesReceived:       n.BytesReceived.Load(),
		TotalRequests:       totalReqs,
		FailedRequests:      failedReqs,
		AvgRequestLatencyMs: float64(n.RequestLatencyNs.Load()) / 1e6,
		TimeoutCount:        n.TimeoutCount.Load(),
	}
}

func (mc *MetricsCollector) snapshotSystem() SystemSnapshot {
	s := mc.system

	return SystemSnapshot{
		Healthy:         s.Healthy.Load(),
		ReadOnly:        s.ReadOnly.Load(),
		UptimeSeconds:   s.UptimeSeconds.Load(),
		GoroutineCount:  s.GoroutineCount.Load(),
		MemoryUsageMB:   s.MemoryUsageMB.Load(),
		CPUUsagePercent: s.CPUUsagePercent.Load(),
	}
}

// Global metrics collector
var globalMetrics *MetricsCollector
var metricsOnce sync.Once

// GetGlobalMetrics returns the global metrics collector
func GetGlobalMetrics() *MetricsCollector {
	metricsOnce.Do(func() {
		globalMetrics = NewMetricsCollector()
	})
	return globalMetrics
}

// Helper functions for tracking operations with automatic metrics

// TrackOperation tracks an operation with start/end timing
func TrackOperation(name string, operation func() error) error {
	start := time.Now()
	err := operation()
	duration := time.Since(start)

	// This is a simplified version - in production you'd route to specific metrics
	_ = duration
	_ = name

	return err
}

// LatencyTracker helps track operation latency
type LatencyTracker struct {
	start time.Time
}

// NewLatencyTracker creates a new latency tracker
func NewLatencyTracker() *LatencyTracker {
	return &LatencyTracker{start: time.Now()}
}

// Finish returns the elapsed time in nanoseconds
func (lt *LatencyTracker) Finish() uint64 {
	return uint64(time.Since(lt.start).Nanoseconds())
}
