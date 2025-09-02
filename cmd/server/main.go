// DistKV Server - Main entry point for the distributed key-value store server
// This starts a DistKV node that can participate in the distributed cluster.
// 
// The server provides both client-facing API and internal node communication.
// It integrates all the components: storage engine, replication, gossip, etc.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"distkv/pkg/consensus"
	"distkv/pkg/gossip"
	"distkv/pkg/partition"
	"distkv/pkg/replication"
	"distkv/pkg/storage"
	"distkv/proto"
)

// ServerConfig holds all configuration for the DistKV server
type ServerConfig struct {
	// Server identification
	NodeID   string // Unique identifier for this node
	Address  string // Address this server listens on (e.g., "localhost:8080")
	DataDir  string // Directory to store data files
	
	// Cluster configuration
	SeedNodes    []string // List of seed nodes to join the cluster
	VirtualNodes int      // Number of virtual nodes for consistent hashing
	
	// Storage configuration
	StorageConfig *storage.StorageConfig
	
	// Replication configuration
	QuorumConfig *replication.QuorumConfig
	
	// Gossip configuration
	GossipConfig *gossip.GossipConfig
}

// DistKVServer implements the main server logic
type DistKVServer struct {
	// Core configuration
	config *ServerConfig
	
	// Storage layer
	storageEngine *storage.Engine
	
	// Partitioning and replication
	consistentHash *partition.ConsistentHash
	quorumManager  *replication.QuorumManager
	
	// Failure detection
	gossipManager *gossip.Gossip
	
	// gRPC server
	grpcServer *grpc.Server
	
	// Node management
	nodeSelector *NodeSelector
	replicaClient *ReplicaClient
}

// main is the entry point for the DistKV server
func main() {
	// Parse command line flags
	config := parseFlags()
	
	// Validate configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	
	// Create and start the server
	server, err := NewDistKVServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	
	// Start the server
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	
	// Wait for shutdown signal
	waitForShutdown()
	
	// Graceful shutdown
	if err := server.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	
	log.Println("DistKV server shut down successfully")
}

// parseFlags parses command line arguments and returns server configuration
func parseFlags() *ServerConfig {
	var (
		nodeID       = flag.String("node-id", "", "Unique node identifier (required)")
		address      = flag.String("address", "localhost:8080", "Server listen address")
		dataDir      = flag.String("data-dir", "./data", "Directory for data storage")
		seedNodes    = flag.String("seed-nodes", "", "Comma-separated list of seed nodes")
		virtualNodes = flag.Int("virtual-nodes", 150, "Number of virtual nodes for consistent hashing")
		
		// Storage flags
		memTableSize     = flag.Int("mem-table-size", 64*1024*1024, "MemTable size in bytes")
		ssTableSize      = flag.Int64("sstable-size", 256*1024*1024, "SSTable size in bytes")
		bloomFilterBits  = flag.Int("bloom-filter-bits", 10, "Bloom filter bits per key")
		compactionThresh = flag.Int("compaction-threshold", 4, "Number of SSTables to trigger compaction")
		
		// Replication flags
		replicas     = flag.Int("replicas", 3, "Number of replicas (N)")
		readQuorum   = flag.Int("read-quorum", 2, "Read quorum size (R)")
		writeQuorum  = flag.Int("write-quorum", 2, "Write quorum size (W)")
		
		// Gossip flags
		heartbeatInterval = flag.Duration("heartbeat-interval", 1*time.Second, "Heartbeat interval")
		suspectTimeout    = flag.Duration("suspect-timeout", 5*time.Second, "Suspect timeout")
		deadTimeout       = flag.Duration("dead-timeout", 30*time.Second, "Dead timeout")
		gossipInterval    = flag.Duration("gossip-interval", 1*time.Second, "Gossip interval")
		gossipFanout      = flag.Int("gossip-fanout", 3, "Gossip fanout")
	)
	
	flag.Parse()
	
	// Generate node ID if not provided
	if *nodeID == "" {
		*nodeID = generateNodeID(*address)
	}
	
	// Parse seed nodes
	var seedNodesList []string
	if *seedNodes != "" {
		seedNodesList = strings.Split(*seedNodes, ",")
		for i, node := range seedNodesList {
			seedNodesList[i] = strings.TrimSpace(node)
		}
	}
	
	return &ServerConfig{
		NodeID:       *nodeID,
		Address:      *address,
		DataDir:      *dataDir,
		SeedNodes:    seedNodesList,
		VirtualNodes: *virtualNodes,
		
		StorageConfig: &storage.StorageConfig{
			MemTableMaxSize:     *memTableSize,
			MaxMemTables:        2,
			SSTableMaxSize:      *ssTableSize,
			BloomFilterBits:     *bloomFilterBits,
			CompressionEnabled:  true,
			CompactionThreshold: *compactionThresh,
			MaxCompactionSize:   1024 * 1024 * 1024, // 1GB
			TombstoneTTL:        3 * time.Hour,
			GCInterval:          1 * time.Hour,
			WriteBufferSize:     4 * 1024 * 1024,
			CacheSize:          128 * 1024 * 1024,
			MaxOpenFiles:       1000,
		},
		
		QuorumConfig: &replication.QuorumConfig{
			N:              *replicas,
			R:              *readQuorum,
			W:              *writeQuorum,
			RequestTimeout: 5 * time.Second,
			RetryAttempts:  3,
			RetryDelay:     100 * time.Millisecond,
		},
		
		GossipConfig: &gossip.GossipConfig{
			HeartbeatInterval:   *heartbeatInterval,
			SuspectTimeout:      *suspectTimeout,
			DeadTimeout:         *deadTimeout,
			GossipInterval:      *gossipInterval,
			GossipFanout:        *gossipFanout,
			MaxGossipPacketSize: 64 * 1024,
		},
	}
}

// validateConfig validates the server configuration
func validateConfig(config *ServerConfig) error {
	if config.NodeID == "" {
		return fmt.Errorf("node-id is required")
	}
	
	if config.Address == "" {
		return fmt.Errorf("address is required")
	}
	
	if config.DataDir == "" {
		return fmt.Errorf("data-dir is required")
	}
	
	// Validate quorum configuration
	if err := config.QuorumConfig.Validate(); err != nil {
		return fmt.Errorf("invalid quorum config: %v", err)
	}
	
	return nil
}

// generateNodeID generates a node ID based on the address if not provided
func generateNodeID(address string) string {
	// Use address as base and add timestamp for uniqueness
	timestamp := time.Now().Unix()
	return fmt.Sprintf("node-%s-%d", strings.ReplaceAll(address, ":", "-"), timestamp)
}

// NewDistKVServer creates a new DistKV server instance
func NewDistKVServer(config *ServerConfig) (*DistKVServer, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}
	
	// Initialize storage engine
	storageDir := filepath.Join(config.DataDir, "storage")
	storageEngine, err := storage.NewEngine(storageDir, config.StorageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage engine: %v", err)
	}
	
	// Initialize consistent hashing
	consistentHash := partition.NewConsistentHash(config.VirtualNodes)
	
	// Initialize gossip manager
	gossipManager := gossip.NewGossip(config.NodeID, config.Address, config.GossipConfig)
	
	// Create node selector and replica client
	nodeSelector := NewNodeSelector(consistentHash, gossipManager)
	replicaClient := NewReplicaClient()
	
	// Initialize quorum manager
	quorumManager, err := replication.NewQuorumManager(config.QuorumConfig, nodeSelector, replicaClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create quorum manager: %v", err)
	}
	
	return &DistKVServer{
		config:         config,
		storageEngine:  storageEngine,
		consistentHash: consistentHash,
		quorumManager:  quorumManager,
		gossipManager:  gossipManager,
		nodeSelector:   nodeSelector,
		replicaClient:  replicaClient,
	}, nil
}

// Start starts the DistKV server
func (s *DistKVServer) Start() error {
	log.Printf("Starting DistKV server: %s at %s", s.config.NodeID, s.config.Address)
	
	// Start gossip protocol
	if err := s.gossipManager.Start(); err != nil {
		return fmt.Errorf("failed to start gossip manager: %v", err)
	}
	
	// Add self to consistent hash ring
	s.consistentHash.AddNode(s.config.NodeID)
	
	// Join cluster by connecting to seed nodes
	if err := s.joinCluster(); err != nil {
		log.Printf("Warning: failed to join cluster: %v", err)
		// Continue anyway - we can operate as a single-node cluster
	}
	
	// Start gRPC server
	if err := s.startGRPCServer(); err != nil {
		return fmt.Errorf("failed to start gRPC server: %v", err)
	}
	
	log.Printf("DistKV server started successfully")
	return nil
}

// Stop stops the DistKV server gracefully
func (s *DistKVServer) Stop() error {
	log.Printf("Shutting down DistKV server: %s", s.config.NodeID)
	
	// Stop gRPC server
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
	
	// Stop gossip manager
	if s.gossipManager != nil {
		s.gossipManager.Stop()
	}
	
	// Close storage engine
	if s.storageEngine != nil {
		s.storageEngine.Close()
	}
	
	return nil
}

// joinCluster attempts to join the cluster by contacting seed nodes
func (s *DistKVServer) joinCluster() error {
	if len(s.config.SeedNodes) == 0 {
		log.Printf("No seed nodes specified, starting as single-node cluster")
		return nil
	}
	
	log.Printf("Attempting to join cluster via seed nodes: %v", s.config.SeedNodes)
	
	// In a real implementation, this would:
	// 1. Connect to each seed node via gRPC
	// 2. Request current cluster membership
	// 3. Add discovered nodes to gossip manager
	// 4. Add discovered nodes to consistent hash ring
	
	// For now, just add seed nodes to gossip and consistent hash
	for _, seedNode := range s.config.SeedNodes {
		// Extract node ID from address (simplified)
		nodeID := fmt.Sprintf("seed-%s", strings.ReplaceAll(seedNode, ":", "-"))
		
		s.gossipManager.AddNode(nodeID, seedNode)
		s.consistentHash.AddNode(nodeID)
	}
	
	log.Printf("Added %d seed nodes to cluster", len(s.config.SeedNodes))
	return nil
}

// startGRPCServer starts the gRPC server
func (s *DistKVServer) startGRPCServer() error {
	// Create listener
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", s.config.Address, err)
	}
	
	// Create gRPC server
	s.grpcServer = grpc.NewServer()
	
	// Register services
	distkvService := &DistKVServiceImpl{server: s}
	proto.RegisterDistKVServer(s.grpcServer, distkvService)
	
	nodeService := &NodeServiceImpl{server: s}
	proto.RegisterNodeServiceServer(s.grpcServer, nodeService)
	
	adminService := &AdminServiceImpl{server: s}
	proto.RegisterAdminServiceServer(s.grpcServer, adminService)
	
	// Enable reflection for debugging
	reflection.Register(s.grpcServer)
	
	// Start server in background
	go func() {
		log.Printf("gRPC server listening on %s", s.config.Address)
		if err := s.grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()
	
	return nil
}

// waitForShutdown waits for OS signals to shutdown gracefully
func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)
}