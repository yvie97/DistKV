// Package replication implements data replication and quorum consensus
// for maintaining consistency across multiple nodes in the distributed system.
package replication

import (
	"context"
	"distkv/pkg/consensus"
	"fmt"
	"sync"
	"time"
)

// QuorumConfig defines the replication parameters for the system.
// These settings control the consistency vs availability trade-offs.
type QuorumConfig struct {
	// N is the total number of replicas for each key
	N int
	
	// R is the number of replicas that must respond for a read operation
	R int
	
	// W is the number of replicas that must acknowledge a write operation  
	W int
	
	// RequestTimeout is the maximum time to wait for replica responses
	RequestTimeout time.Duration
	
	// RetryAttempts is how many times to retry failed operations
	RetryAttempts int
	
	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration
}

// DefaultQuorumConfig returns the default configuration from the design document.
// N=3, W=2, R=2 provides strong consistency when W + R > N
func DefaultQuorumConfig() *QuorumConfig {
	return &QuorumConfig{
		N:              3,                // Store 3 copies of each key
		R:              2,                // Read from 2 replicas (majority)
		W:              2,                // Write to 2 replicas (majority)
		RequestTimeout: 5 * time.Second,  // 5 second timeout for operations
		RetryAttempts:  3,                // Retry failed operations 3 times
		RetryDelay:     100 * time.Millisecond,
	}
}

// Validate checks if the quorum configuration is valid.
func (qc *QuorumConfig) Validate() error {
	if qc.N <= 0 {
		return fmt.Errorf("N must be positive, got %d", qc.N)
	}
	
	if qc.R <= 0 || qc.R > qc.N {
		return fmt.Errorf("R must be between 1 and N, got R=%d, N=%d", qc.R, qc.N)
	}
	
	if qc.W <= 0 || qc.W > qc.N {
		return fmt.Errorf("W must be between 1 and N, got W=%d, N=%d", qc.W, qc.N)
	}
	
	return nil
}

// IsStrongConsistency returns true if the configuration guarantees strong consistency.
// This happens when W + R > N, ensuring read and write quorums overlap.
func (qc *QuorumConfig) IsStrongConsistency() bool {
	return qc.W+qc.R > qc.N
}

// ReplicaInfo contains information about a replica node.
type ReplicaInfo struct {
	NodeID   string    // Unique identifier for the node
	Address  string    // Network address of the node
	IsAlive  bool      // Whether the node is currently reachable
	LastSeen time.Time // When we last heard from this node
}

// WriteRequest represents a request to write data to replicas.
type WriteRequest struct {
	Key         string                   // The key to write
	Value       []byte                   // The value to store
	VectorClock *consensus.VectorClock   // Version information
	Context     context.Context          // Request context for timeouts
}

// WriteResponse represents the response from a write operation.
type WriteResponse struct {
	Success       bool                     // Whether the write succeeded
	VectorClock   *consensus.VectorClock   // Updated vector clock
	ReplicasWritten int                    // Number of replicas that acknowledged
	Errors        []error                  // Any errors that occurred
}

// ReadRequest represents a request to read data from replicas.
type ReadRequest struct {
	Key     string          // The key to read
	Context context.Context // Request context for timeouts
}

// ReadResponse represents the response from a read operation.
type ReadResponse struct {
	Value       []byte                   // The value (nil if not found)
	VectorClock *consensus.VectorClock   // Version information
	Found       bool                     // Whether the key was found
	ReplicasRead int                     // Number of replicas that responded
	Errors      []error                  // Any errors that occurred
}

// ReplicaResponse represents a response from a single replica.
type ReplicaResponse struct {
	NodeID      string                   // Which node responded
	Value       []byte                   // The value from this replica
	VectorClock *consensus.VectorClock   // Vector clock from this replica
	Success     bool                     // Whether the operation succeeded
	Error       error                    // Any error that occurred
}

// QuorumManager handles quorum-based operations across replicas.
type QuorumManager struct {
	config       *QuorumConfig
	nodeSelector NodeSelector // Selects which nodes to use for a key
	client       ReplicaClient // Communicates with replica nodes
	mutex        sync.RWMutex
}

// NodeSelector interface for selecting replica nodes for a given key.
// This will be implemented using consistent hashing.
type NodeSelector interface {
	// GetReplicas returns N nodes that should store replicas of the key
	GetReplicas(key string, count int) []ReplicaInfo
	
	// GetAliveReplicas returns only the alive nodes from GetReplicas
	GetAliveReplicas(key string, count int) []ReplicaInfo
}

// ReplicaClient interface for communicating with replica nodes.
// This will be implemented using gRPC calls.
type ReplicaClient interface {
	// WriteReplica writes a value to a specific replica node
	WriteReplica(ctx context.Context, nodeID string, key string, value []byte, 
		vectorClock *consensus.VectorClock) (*ReplicaResponse, error)
	
	// ReadReplica reads a value from a specific replica node
	ReadReplica(ctx context.Context, nodeID string, key string) (*ReplicaResponse, error)
}

// NewQuorumManager creates a new quorum manager.
func NewQuorumManager(config *QuorumConfig, nodeSelector NodeSelector, client ReplicaClient) (*QuorumManager, error) {
	if config == nil {
		config = DefaultQuorumConfig()
	}
	
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid quorum config: %v", err)
	}
	
	return &QuorumManager{
		config:       config,
		nodeSelector: nodeSelector,
		client:       client,
	}, nil
}

// Write performs a quorum write operation.
// It writes to W replicas and returns success when enough replicas acknowledge.
func (qm *QuorumManager) Write(req *WriteRequest) (*WriteResponse, error) {
	// Get replica nodes for this key
	replicas := qm.nodeSelector.GetAliveReplicas(req.Key, qm.config.N)
	if len(replicas) < qm.config.W {
		return nil, fmt.Errorf("insufficient alive replicas: need %d, have %d", 
			qm.config.W, len(replicas))
	}
	
	// Create context with timeout
	ctx := req.Context
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), qm.config.RequestTimeout)
		defer cancel()
	}
	
	// Send write requests to all available replicas concurrently
	responseChan := make(chan *ReplicaResponse, len(replicas))
	
	for _, replica := range replicas {
		go func(r ReplicaInfo) {
			response, err := qm.client.WriteReplica(ctx, r.NodeID, req.Key, req.Value, req.VectorClock)
			if err != nil {
				responseChan <- &ReplicaResponse{
					NodeID:  r.NodeID,
					Success: false,
					Error:   err,
				}
			} else {
				responseChan <- response
			}
		}(replica)
	}
	
	// Collect responses until we have enough successful writes
	var successfulWrites int
	var errors []error
	var latestVectorClock *consensus.VectorClock
	
	for i := 0; i < len(replicas) && successfulWrites < qm.config.W; i++ {
		select {
		case response := <-responseChan:
			if response.Success {
				successfulWrites++
				// Keep track of the most recent vector clock
				if latestVectorClock == nil || 
				   (response.VectorClock != nil && response.VectorClock.IsAfter(latestVectorClock)) {
					latestVectorClock = response.VectorClock
				}
			} else {
				errors = append(errors, fmt.Errorf("node %s: %v", response.NodeID, response.Error))
			}
		case <-ctx.Done():
			return &WriteResponse{
				Success:         false,
				ReplicasWritten: successfulWrites,
				Errors:          append(errors, ctx.Err()),
			}, ctx.Err()
		}
	}
	
	// Return success if we got enough acknowledgments
	if successfulWrites >= qm.config.W {
		return &WriteResponse{
			Success:         true,
			VectorClock:     latestVectorClock,
			ReplicasWritten: successfulWrites,
			Errors:          errors,
		}, nil
	}
	
	return &WriteResponse{
		Success:         false,
		ReplicasWritten: successfulWrites,
		Errors:          errors,
	}, fmt.Errorf("write quorum failed: got %d acknowledgments, needed %d", 
		successfulWrites, qm.config.W)
}

// Read performs a quorum read operation.
// It reads from R replicas and resolves conflicts using vector clocks.
func (qm *QuorumManager) Read(req *ReadRequest) (*ReadResponse, error) {
	// Get replica nodes for this key
	replicas := qm.nodeSelector.GetAliveReplicas(req.Key, qm.config.N)
	if len(replicas) < qm.config.R {
		return nil, fmt.Errorf("insufficient alive replicas: need %d, have %d", 
			qm.config.R, len(replicas))
	}
	
	// Create context with timeout
	ctx := req.Context
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), qm.config.RequestTimeout)
		defer cancel()
	}
	
	// Send read requests to R replicas concurrently
	responseChan := make(chan *ReplicaResponse, qm.config.R)
	
	for i := 0; i < qm.config.R && i < len(replicas); i++ {
		go func(r ReplicaInfo) {
			response, err := qm.client.ReadReplica(ctx, r.NodeID, req.Key)
			if err != nil {
				responseChan <- &ReplicaResponse{
					NodeID:  r.NodeID,
					Success: false,
					Error:   err,
				}
			} else {
				responseChan <- response
			}
		}(replicas[i])
	}
	
	// Collect responses
	var responses []*ReplicaResponse
	var errors []error
	
	for i := 0; i < qm.config.R; i++ {
		select {
		case response := <-responseChan:
			if response.Success {
				responses = append(responses, response)
			} else {
				errors = append(errors, fmt.Errorf("node %s: %v", response.NodeID, response.Error))
			}
		case <-ctx.Done():
			return &ReadResponse{
				Found:        false,
				ReplicasRead: len(responses),
				Errors:       append(errors, ctx.Err()),
			}, ctx.Err()
		}
	}
	
	// Check if we got enough responses
	if len(responses) < qm.config.R {
		return &ReadResponse{
			Found:        false,
			ReplicasRead: len(responses),
			Errors:       errors,
		}, fmt.Errorf("read quorum failed: got %d responses, needed %d", 
			len(responses), qm.config.R)
	}
	
	// Resolve conflicts and find the most recent version
	return qm.resolveReadConflicts(responses, errors), nil
}

// resolveReadConflicts finds the most recent version among conflicting replicas.
// Uses vector clocks to determine causality and resolve conflicts.
func (qm *QuorumManager) resolveReadConflicts(responses []*ReplicaResponse, errors []error) *ReadResponse {
	if len(responses) == 0 {
		return &ReadResponse{
			Found:        false,
			ReplicasRead: 0,
			Errors:       errors,
		}
	}
	
	// Find the response with the most recent vector clock
	var latestResponse *ReplicaResponse
	
	for _, response := range responses {
		if latestResponse == nil {
			latestResponse = response
			continue
		}
		
		// Compare vector clocks to find the most recent version
		if response.VectorClock != nil && latestResponse.VectorClock != nil {
			if response.VectorClock.IsAfter(latestResponse.VectorClock) {
				latestResponse = response
			} else if response.VectorClock.IsConcurrent(latestResponse.VectorClock) {
				// Concurrent updates - this is a conflict that needs resolution
				// For now, we'll use a simple tie-breaker (node ID)
				// In production, this might trigger read repair
				if response.NodeID > latestResponse.NodeID {
					latestResponse = response
				}
			}
		} else if response.VectorClock != nil && latestResponse.VectorClock == nil {
			latestResponse = response
		}
	}
	
	return &ReadResponse{
		Value:        latestResponse.Value,
		VectorClock:  latestResponse.VectorClock,
		Found:        latestResponse.Value != nil,
		ReplicasRead: len(responses),
		Errors:       errors,
	}
}

// GetConfig returns the current quorum configuration.
func (qm *QuorumManager) GetConfig() *QuorumConfig {
	qm.mutex.RLock()
	defer qm.mutex.RUnlock()
	
	configCopy := *qm.config
	return &configCopy
}