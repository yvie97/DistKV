package replication

import (
	"context"
	"distkv/pkg/consensus"
	"distkv/pkg/storage"
	"errors"
	"testing"
	"time"
)

// MockNodeSelector implements NodeSelector for testing
type MockNodeSelector struct {
	replicas      []ReplicaInfo
	aliveReplicas []ReplicaInfo
}

func (m *MockNodeSelector) GetReplicas(key string, count int) []ReplicaInfo {
	if len(m.replicas) < count {
		return m.replicas
	}
	return m.replicas[:count]
}

func (m *MockNodeSelector) GetAliveReplicas(key string, count int) []ReplicaInfo {
	if len(m.aliveReplicas) < count {
		return m.aliveReplicas
	}
	return m.aliveReplicas[:count]
}

// MockReplicaClient implements ReplicaClient for testing
type MockReplicaClient struct {
	writeResponses map[string]*ReplicaResponse
	readResponses  map[string]*ReplicaResponse
	writeDelay     time.Duration
	readDelay      time.Duration
}

func (m *MockReplicaClient) WriteReplica(ctx context.Context, nodeID string, key string, value []byte, vectorClock *consensus.VectorClock) (*ReplicaResponse, error) {
	if m.writeDelay > 0 {
		time.Sleep(m.writeDelay)
	}

	if resp, exists := m.writeResponses[nodeID]; exists {
		return resp, resp.Error
	}

	return &ReplicaResponse{
		NodeID:      nodeID,
		Success:     true,
		VectorClock: vectorClock,
	}, nil
}

func (m *MockReplicaClient) ReadReplica(ctx context.Context, nodeID string, key string) (*ReplicaResponse, error) {
	if m.readDelay > 0 {
		time.Sleep(m.readDelay)
	}

	if resp, exists := m.readResponses[nodeID]; exists {
		return resp, resp.Error
	}

	return &ReplicaResponse{
		NodeID:  nodeID,
		Success: true,
		Value:   []byte("test-value"),
	}, nil
}

// MockStorageEngine implements StorageEngine for testing
type MockStorageEngine struct {
	data map[string]*storage.Entry
}

func NewMockStorageEngine() *MockStorageEngine {
	return &MockStorageEngine{
		data: make(map[string]*storage.Entry),
	}
}

func (m *MockStorageEngine) Put(key string, value []byte, vectorClock *consensus.VectorClock) error {
	m.data[key] = &storage.Entry{
		Key:         key,
		Value:       value,
		VectorClock: vectorClock,
		Deleted:     false,
	}
	return nil
}

func (m *MockStorageEngine) Get(key string) (*storage.Entry, error) {
	entry, exists := m.data[key]
	if !exists {
		return nil, nil
	}
	return entry, nil
}

func (m *MockStorageEngine) Delete(key string, vectorClock *consensus.VectorClock) error {
	if entry, exists := m.data[key]; exists {
		entry.Deleted = true
		entry.VectorClock = vectorClock
	}
	return nil
}

// TestDefaultQuorumConfig tests the default configuration
func TestDefaultQuorumConfig(t *testing.T) {
	config := DefaultQuorumConfig()

	if config.N != 3 {
		t.Errorf("Expected N=3, got %d", config.N)
	}

	if config.R != 2 {
		t.Errorf("Expected R=2, got %d", config.R)
	}

	if config.W != 2 {
		t.Errorf("Expected W=2, got %d", config.W)
	}

	if !config.IsStrongConsistency() {
		t.Error("Default config should provide strong consistency (W+R > N)")
	}
}

// TestQuorumConfigValidate tests configuration validation
func TestQuorumConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *QuorumConfig
		shouldError bool
	}{
		{
			name: "Valid config",
			config: &QuorumConfig{N: 3, R: 2, W: 2, RequestTimeout: 5 * time.Second,
				RetryAttempts: 3, RetryDelay: 100 * time.Millisecond},
			shouldError: false,
		},
		{
			name: "Invalid N (zero)",
			config: &QuorumConfig{N: 0, R: 1, W: 1, RequestTimeout: 5 * time.Second,
				RetryAttempts: 3, RetryDelay: 100 * time.Millisecond},
			shouldError: true,
		},
		{
			name: "Invalid R (zero)",
			config: &QuorumConfig{N: 3, R: 0, W: 2, RequestTimeout: 5 * time.Second,
				RetryAttempts: 3, RetryDelay: 100 * time.Millisecond},
			shouldError: true,
		},
		{
			name: "Invalid W (greater than N)",
			config: &QuorumConfig{N: 3, R: 2, W: 5, RequestTimeout: 5 * time.Second,
				RetryAttempts: 3, RetryDelay: 100 * time.Millisecond},
			shouldError: true,
		},
		{
			name: "Invalid R (greater than N)",
			config: &QuorumConfig{N: 3, R: 5, W: 2, RequestTimeout: 5 * time.Second,
				RetryAttempts: 3, RetryDelay: 100 * time.Millisecond},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.shouldError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// TestIsStrongConsistency tests strong consistency detection
func TestIsStrongConsistency(t *testing.T) {
	tests := []struct {
		name     string
		N, R, W  int
		expected bool
	}{
		{"Strong (W+R > N)", 3, 2, 2, true},
		{"Strong (W+R > N) variant", 5, 3, 3, true},
		{"Weak (W+R = N)", 3, 2, 1, false},
		{"Weak (W+R < N)", 5, 2, 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &QuorumConfig{N: tt.N, R: tt.R, W: tt.W,
				RequestTimeout: 5 * time.Second, RetryAttempts: 3, RetryDelay: 100 * time.Millisecond}
			if config.IsStrongConsistency() != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, config.IsStrongConsistency())
			}
		})
	}
}

// TestNewQuorumManager tests creating a quorum manager
func TestNewQuorumManager(t *testing.T) {
	config := DefaultQuorumConfig()
	selector := &MockNodeSelector{}
	client := &MockReplicaClient{
		writeResponses: make(map[string]*ReplicaResponse),
		readResponses:  make(map[string]*ReplicaResponse),
	}
	storageEngine := NewMockStorageEngine()

	qm, err := NewQuorumManager(config, selector, client, storageEngine)

	if err != nil {
		t.Fatalf("Failed to create quorum manager: %v", err)
	}

	if qm == nil {
		t.Fatal("QuorumManager is nil")
	}

	if qm.config.N != 3 {
		t.Errorf("Expected N=3, got %d", qm.config.N)
	}
}

// TestNewQuorumManagerWithInvalidConfig tests creating manager with invalid config
func TestNewQuorumManagerWithInvalidConfig(t *testing.T) {
	config := &QuorumConfig{N: 0, R: 1, W: 1, RequestTimeout: 5 * time.Second,
		RetryAttempts: 3, RetryDelay: 100 * time.Millisecond}
	selector := &MockNodeSelector{}
	client := &MockReplicaClient{
		writeResponses: make(map[string]*ReplicaResponse),
		readResponses:  make(map[string]*ReplicaResponse),
	}
	storageEngine := NewMockStorageEngine()

	_, err := NewQuorumManager(config, selector, client, storageEngine)

	if err == nil {
		t.Error("Expected error for invalid config, got nil")
	}
}

// TestWriteSuccess tests successful quorum write
func TestWriteSuccess(t *testing.T) {
	config := &QuorumConfig{N: 3, R: 2, W: 2, RequestTimeout: 5 * time.Second,
		RetryAttempts: 3, RetryDelay: 100 * time.Millisecond}

	replicas := []ReplicaInfo{
		{NodeID: "node1", Address: "addr1", IsAlive: true},
		{NodeID: "node2", Address: "addr2", IsAlive: true},
		{NodeID: "node3", Address: "addr3", IsAlive: true},
	}

	selector := &MockNodeSelector{aliveReplicas: replicas, replicas: replicas}
	client := &MockReplicaClient{
		writeResponses: make(map[string]*ReplicaResponse),
		readResponses:  make(map[string]*ReplicaResponse),
	}
	storageEngine := NewMockStorageEngine()

	qm, _ := NewQuorumManager(config, selector, client, storageEngine)

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	req := &WriteRequest{
		Key:         "test-key",
		Value:       []byte("test-value"),
		VectorClock: vc,
		Context:     context.Background(),
	}

	resp, err := qm.Write(req)

	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected successful write")
	}

	if resp.ReplicasWritten < 2 {
		t.Errorf("Expected at least 2 replicas written, got %d", resp.ReplicasWritten)
	}
}

// TestWriteInsufficientReplicas tests write with insufficient replicas
func TestWriteInsufficientReplicas(t *testing.T) {
	config := &QuorumConfig{N: 3, R: 2, W: 2, RequestTimeout: 5 * time.Second,
		RetryAttempts: 3, RetryDelay: 100 * time.Millisecond}

	// Only 1 alive replica, need 2
	replicas := []ReplicaInfo{
		{NodeID: "node1", Address: "addr1", IsAlive: true},
	}

	selector := &MockNodeSelector{aliveReplicas: replicas, replicas: replicas}
	client := &MockReplicaClient{
		writeResponses: make(map[string]*ReplicaResponse),
		readResponses:  make(map[string]*ReplicaResponse),
	}
	storageEngine := NewMockStorageEngine()

	qm, _ := NewQuorumManager(config, selector, client, storageEngine)

	vc := consensus.NewVectorClock()
	req := &WriteRequest{
		Key:         "test-key",
		Value:       []byte("test-value"),
		VectorClock: vc,
		Context:     context.Background(),
	}

	_, err := qm.Write(req)

	if err == nil {
		t.Error("Expected error for insufficient replicas")
	}
}

// TestWritePartialFailure tests write with some replica failures
func TestWritePartialFailure(t *testing.T) {
	config := &QuorumConfig{N: 3, R: 2, W: 2, RequestTimeout: 5 * time.Second,
		RetryAttempts: 3, RetryDelay: 100 * time.Millisecond}

	replicas := []ReplicaInfo{
		{NodeID: "node1", Address: "addr1", IsAlive: true},
		{NodeID: "node2", Address: "addr2", IsAlive: true},
		{NodeID: "node3", Address: "addr3", IsAlive: true},
	}

	selector := &MockNodeSelector{aliveReplicas: replicas, replicas: replicas}

	// Make node2 fail
	client := &MockReplicaClient{
		writeResponses: map[string]*ReplicaResponse{
			"node2": {NodeID: "node2", Success: false, Error: errors.New("write failed")},
		},
		readResponses: make(map[string]*ReplicaResponse),
	}
	storageEngine := NewMockStorageEngine()

	qm, _ := NewQuorumManager(config, selector, client, storageEngine)

	vc := consensus.NewVectorClock()
	req := &WriteRequest{
		Key:         "test-key",
		Value:       []byte("test-value"),
		VectorClock: vc,
		Context:     context.Background(),
	}

	resp, err := qm.Write(req)

	// Should still succeed with 2/3 replicas
	if err != nil {
		t.Fatalf("Write should succeed with partial failure: %v", err)
	}

	if !resp.Success {
		t.Error("Expected successful write despite one failure")
	}

	if resp.ReplicasWritten < 2 {
		t.Errorf("Expected at least 2 replicas written, got %d", resp.ReplicasWritten)
	}
}

// TestReadSuccess tests successful quorum read
func TestReadSuccess(t *testing.T) {
	config := &QuorumConfig{N: 3, R: 2, W: 2, RequestTimeout: 5 * time.Second,
		RetryAttempts: 3, RetryDelay: 100 * time.Millisecond}

	replicas := []ReplicaInfo{
		{NodeID: "node1", Address: "addr1", IsAlive: true},
		{NodeID: "node2", Address: "addr2", IsAlive: true},
		{NodeID: "node3", Address: "addr3", IsAlive: true},
	}

	selector := &MockNodeSelector{aliveReplicas: replicas, replicas: replicas}

	vc := consensus.NewVectorClock()
	vc.Increment("node1")

	client := &MockReplicaClient{
		writeResponses: make(map[string]*ReplicaResponse),
		readResponses: map[string]*ReplicaResponse{
			"node1": {NodeID: "node1", Success: true, Value: []byte("value1"), VectorClock: vc},
			"node2": {NodeID: "node2", Success: true, Value: []byte("value1"), VectorClock: vc},
		},
	}
	storageEngine := NewMockStorageEngine()

	qm, _ := NewQuorumManager(config, selector, client, storageEngine)

	req := &ReadRequest{
		Key:     "test-key",
		Context: context.Background(),
	}

	resp, err := qm.Read(req)

	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if !resp.Found {
		t.Error("Expected to find value")
	}

	if string(resp.Value) != "value1" {
		t.Errorf("Expected value 'value1', got '%s'", string(resp.Value))
	}

	if resp.ReplicasRead < 2 {
		t.Errorf("Expected at least 2 replicas read, got %d", resp.ReplicasRead)
	}
}

// TestReadConflictResolution tests conflict resolution using vector clocks
func TestReadConflictResolution(t *testing.T) {
	config := &QuorumConfig{N: 3, R: 2, W: 2, RequestTimeout: 5 * time.Second,
		RetryAttempts: 3, RetryDelay: 100 * time.Millisecond}

	replicas := []ReplicaInfo{
		{NodeID: "node1", Address: "addr1", IsAlive: true},
		{NodeID: "node2", Address: "addr2", IsAlive: true},
		{NodeID: "node3", Address: "addr3", IsAlive: true},
	}

	selector := &MockNodeSelector{aliveReplicas: replicas, replicas: replicas}

	// Create two vector clocks, one is newer
	oldVC := consensus.NewVectorClock()
	oldVC.Increment("node1")

	newVC := oldVC.Copy()
	newVC.Increment("node2")

	client := &MockReplicaClient{
		writeResponses: make(map[string]*ReplicaResponse),
		readResponses: map[string]*ReplicaResponse{
			"node1": {NodeID: "node1", Success: true, Value: []byte("old-value"), VectorClock: oldVC},
			"node2": {NodeID: "node2", Success: true, Value: []byte("new-value"), VectorClock: newVC},
		},
	}
	storageEngine := NewMockStorageEngine()

	qm, _ := NewQuorumManager(config, selector, client, storageEngine)

	req := &ReadRequest{
		Key:     "test-key",
		Context: context.Background(),
	}

	resp, err := qm.Read(req)

	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Should return the newer value
	if string(resp.Value) != "new-value" {
		t.Errorf("Expected 'new-value', got '%s'", string(resp.Value))
	}
}

// TestReadInsufficientReplicas tests read with insufficient replicas
func TestReadInsufficientReplicas(t *testing.T) {
	config := &QuorumConfig{N: 3, R: 2, W: 2, RequestTimeout: 5 * time.Second,
		RetryAttempts: 3, RetryDelay: 100 * time.Millisecond}

	// Only 1 alive replica, need 2
	replicas := []ReplicaInfo{
		{NodeID: "node1", Address: "addr1", IsAlive: true},
	}

	selector := &MockNodeSelector{aliveReplicas: replicas, replicas: replicas}
	client := &MockReplicaClient{
		writeResponses: make(map[string]*ReplicaResponse),
		readResponses:  make(map[string]*ReplicaResponse),
	}
	storageEngine := NewMockStorageEngine()

	qm, _ := NewQuorumManager(config, selector, client, storageEngine)

	req := &ReadRequest{
		Key:     "test-key",
		Context: context.Background(),
	}

	_, err := qm.Read(req)

	if err == nil {
		t.Error("Expected error for insufficient replicas")
	}
}

// TestGetConfig tests getting the configuration
func TestGetConfig(t *testing.T) {
	config := &QuorumConfig{N: 5, R: 3, W: 3, RequestTimeout: 5 * time.Second,
		RetryAttempts: 3, RetryDelay: 100 * time.Millisecond}
	selector := &MockNodeSelector{}
	client := &MockReplicaClient{
		writeResponses: make(map[string]*ReplicaResponse),
		readResponses:  make(map[string]*ReplicaResponse),
	}
	storageEngine := NewMockStorageEngine()

	qm, _ := NewQuorumManager(config, selector, client, storageEngine)

	retrievedConfig := qm.GetConfig()

	if retrievedConfig.N != 5 {
		t.Errorf("Expected N=5, got %d", retrievedConfig.N)
	}

	// Modifying returned config shouldn't affect internal config
	retrievedConfig.N = 10
	if qm.config.N != 5 {
		t.Error("GetConfig should return a copy")
	}
}

// TestLocalWriteRead tests local-only write and read for single-node mode
func TestLocalWriteRead(t *testing.T) {
	config := &QuorumConfig{N: 1, R: 1, W: 1, RequestTimeout: 5 * time.Second,
		RetryAttempts: 3, RetryDelay: 100 * time.Millisecond}

	selector := &MockNodeSelector{aliveReplicas: []ReplicaInfo{}, replicas: []ReplicaInfo{}}
	client := &MockReplicaClient{
		writeResponses: make(map[string]*ReplicaResponse),
		readResponses:  make(map[string]*ReplicaResponse),
	}
	storageEngine := NewMockStorageEngine()

	qm, _ := NewQuorumManager(config, selector, client, storageEngine)

	vc := consensus.NewVectorClock()
	vc.Increment("local")

	// Write locally
	writeReq := &WriteRequest{
		Key:         "test-key",
		Value:       []byte("test-value"),
		VectorClock: vc,
		Context:     context.Background(),
	}

	writeResp, err := qm.Write(writeReq)
	if err != nil {
		t.Fatalf("Local write failed: %v", err)
	}

	if !writeResp.Success {
		t.Error("Expected successful local write")
	}

	// Read locally
	readReq := &ReadRequest{
		Key:     "test-key",
		Context: context.Background(),
	}

	readResp, err := qm.Read(readReq)
	if err != nil {
		t.Fatalf("Local read failed: %v", err)
	}

	if !readResp.Found {
		t.Error("Expected to find locally written value")
	}

	if string(readResp.Value) != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", string(readResp.Value))
	}
}
