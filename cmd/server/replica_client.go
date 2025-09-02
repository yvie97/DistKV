// ReplicaClient implementation for inter-node communication
// This handles gRPC communication between nodes for replication operations.

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"distkv/pkg/consensus"
	"distkv/pkg/replication"
	"distkv/proto"
)

// ReplicaClient implements the replication.ReplicaClient interface
// It manages gRPC connections to other nodes and handles replica operations
type ReplicaClient struct {
	// connections is a pool of gRPC connections to other nodes
	// Key: nodeID, Value: gRPC connection
	connections map[string]*grpc.ClientConn
	
	// nodeAddresses maps node IDs to their network addresses
	// Key: nodeID, Value: address (e.g., "192.168.1.10:8080")
	nodeAddresses map[string]string
	
	// mutex protects concurrent access to the connection maps
	mutex sync.RWMutex
	
	// connectionTimeout is how long to wait when establishing connections
	connectionTimeout time.Duration
}

// NewReplicaClient creates a new ReplicaClient for inter-node communication
func NewReplicaClient() *ReplicaClient {
	return &ReplicaClient{
		connections:       make(map[string]*grpc.ClientConn),
		nodeAddresses:     make(map[string]string),
		connectionTimeout: 5 * time.Second,
	}
}

// WriteReplica writes a value to a specific replica node
// This is called during quorum write operations
func (rc *ReplicaClient) WriteReplica(ctx context.Context, nodeID string, key string, value []byte, vectorClock *consensus.VectorClock) (*replication.ReplicaResponse, error) {
	// Get gRPC client for the target node
	client, err := rc.getNodeServiceClient(nodeID)
	if err != nil {
		return &replication.ReplicaResponse{
			NodeID:  nodeID,
			Success: false,
			Error:   fmt.Errorf("failed to connect to node %s: %v", nodeID, err),
		}, err
	}
	
	// Convert vector clock to proto format
	protoVectorClock := convertVectorClockToProto(vectorClock)
	
	// Create replication request
	req := &proto.ReplicateRequest{
		Key:         key,
		Value:       value,
		VectorClock: protoVectorClock,
		IsDelete:    value == nil, // nil value means deletion
	}
	
	// Make the gRPC call with timeout
	resp, err := client.Replicate(ctx, req)
	if err != nil {
		return &replication.ReplicaResponse{
			NodeID:  nodeID,
			Success: false,
			Error:   fmt.Errorf("replication failed to node %s: %v", nodeID, err),
		}, err
	}
	
	// Convert response
	var responseVectorClock *consensus.VectorClock
	if resp.VectorClock != nil {
		responseVectorClock = convertVectorClockFromProto(resp.VectorClock)
	}
	
	return &replication.ReplicaResponse{
		NodeID:      nodeID,
		Value:       nil, // Write operations don't return values
		VectorClock: responseVectorClock,
		Success:     resp.Success,
		Error:       parseError(resp.ErrorMessage),
	}, nil
}

// ReadReplica reads a value from a specific replica node
// This is called during quorum read operations
func (rc *ReplicaClient) ReadReplica(ctx context.Context, nodeID string, key string) (*replication.ReplicaResponse, error) {
	// Get gRPC client for the target node
	client, err := rc.getDistKVClient(nodeID)
	if err != nil {
		return &replication.ReplicaResponse{
			NodeID:  nodeID,
			Success: false,
			Error:   fmt.Errorf("failed to connect to node %s: %v", nodeID, err),
		}, err
	}
	
	// Create get request
	req := &proto.GetRequest{
		Key:              key,
		ConsistencyLevel: proto.ConsistencyLevel_ONE, // Single node read
	}
	
	// Make the gRPC call
	resp, err := client.Get(ctx, req)
	if err != nil {
		return &replication.ReplicaResponse{
			NodeID:  nodeID,
			Success: false,
			Error:   fmt.Errorf("read failed from node %s: %v", nodeID, err),
		}, err
	}
	
	// Convert response
	var responseVectorClock *consensus.VectorClock
	if resp.VectorClock != nil {
		responseVectorClock = convertVectorClockFromProto(resp.VectorClock)
	}
	
	var value []byte
	if resp.Found {
		value = resp.Value
	}
	
	return &replication.ReplicaResponse{
		NodeID:      nodeID,
		Value:       value,
		VectorClock: responseVectorClock,
		Success:     resp.Found || resp.ErrorMessage == "", // Success if found or no error
		Error:       parseError(resp.ErrorMessage),
	}, nil
}

// UpdateNodeAddress updates the network address for a node
// This is called when we discover nodes through gossip or configuration
func (rc *ReplicaClient) UpdateNodeAddress(nodeID, address string) {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	
	// Update the address mapping
	rc.nodeAddresses[nodeID] = address
	
	// If we have an existing connection to this node and the address changed,
	// close the old connection so it gets recreated with the new address
	if conn, exists := rc.connections[nodeID]; exists {
		go func() {
			// Close in background to avoid blocking
			conn.Close()
		}()
		delete(rc.connections, nodeID)
	}
}

// Close closes all gRPC connections
func (rc *ReplicaClient) Close() error {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	
	// Close all connections
	for nodeID, conn := range rc.connections {
		if err := conn.Close(); err != nil {
			// Log error but continue closing other connections
			fmt.Printf("Error closing connection to node %s: %v\n", nodeID, err)
		}
	}
	
	// Clear the maps
	rc.connections = make(map[string]*grpc.ClientConn)
	rc.nodeAddresses = make(map[string]string)
	
	return nil
}

// getNodeServiceClient gets a NodeService gRPC client for inter-node operations
func (rc *ReplicaClient) getNodeServiceClient(nodeID string) (proto.NodeServiceClient, error) {
	conn, err := rc.getConnection(nodeID)
	if err != nil {
		return nil, err
	}
	
	return proto.NewNodeServiceClient(conn), nil
}

// getDistKVClient gets a DistKV gRPC client for client-style operations
func (rc *ReplicaClient) getDistKVClient(nodeID string) (proto.DistKVClient, error) {
	conn, err := rc.getConnection(nodeID)
	if err != nil {
		return nil, err
	}
	
	return proto.NewDistKVClient(conn), nil
}

// getConnection gets or creates a gRPC connection to a node
func (rc *ReplicaClient) getConnection(nodeID string) (*grpc.ClientConn, error) {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	
	// Check if we already have a connection
	if conn, exists := rc.connections[nodeID]; exists {
		// Verify the connection is still good
		if conn.GetState().String() != "SHUTDOWN" {
			return conn, nil
		}
		// Connection is shut down, remove it
		delete(rc.connections, nodeID)
	}
	
	// Get the address for this node
	address, exists := rc.nodeAddresses[nodeID]
	if !exists {
		return nil, fmt.Errorf("no address known for node %s", nodeID)
	}
	
	// Create new connection
	ctx, cancel := context.WithTimeout(context.Background(), rc.connectionTimeout)
	defer cancel()
	
	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(), // Wait for connection to be established
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s at %s: %v", nodeID, address, err)
	}
	
	// Cache the connection
	rc.connections[nodeID] = conn
	
	return conn, nil
}

// parseError converts an error message string to an error (or nil if empty)
func parseError(errorMessage string) error {
	if errorMessage == "" {
		return nil
	}
	return fmt.Errorf("%s", errorMessage)
}

// Helper functions for proto conversion (these would normally be in a shared package)

func convertVectorClockToProto(vc *consensus.VectorClock) *proto.VectorClock {
	if vc == nil {
		return &proto.VectorClock{Clocks: make(map[string]uint64)}
	}
	
	return &proto.VectorClock{
		Clocks: vc.GetClocks(),
	}
}

func convertVectorClockFromProto(protoVC *proto.VectorClock) *consensus.VectorClock {
	if protoVC == nil {
		return consensus.NewVectorClock()
	}
	
	return consensus.NewVectorClockFromMap(protoVC.Clocks)
}