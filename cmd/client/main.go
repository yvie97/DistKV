// DistKV Client - Command line client for interacting with DistKV cluster
// This provides a simple CLI interface for testing and interacting with the distributed key-value store.
//
// Usage examples:
//   client put mykey "my value"
//   client get mykey  
//   client delete mykey
//   client status

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"distkv/proto"
)

// ClientConfig holds configuration for the DistKV client
type ClientConfig struct {
	// ServerAddress is the address of any DistKV node to connect to
	ServerAddress string
	
	// Timeout for gRPC operations
	Timeout time.Duration
	
	// ConsistencyLevel for operations
	ConsistencyLevel proto.ConsistencyLevel
}

// DistKVClient wraps the gRPC client with additional functionality
type DistKVClient struct {
	config     *ClientConfig
	conn       *grpc.ClientConn
	distkvClient proto.DistKVClient
	adminClient  proto.AdminServiceClient
}

func main() {
	// Parse command line flags
	config, remainingArgs := parseFlags()
	
	// Create client
	client, err := NewDistKVClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Parse command and execute
	if len(remainingArgs) < 1 {
		printUsage()
		os.Exit(1)
	}
	
	command := remainingArgs[0]
	args := remainingArgs[1:]
	
	switch command {
	case "put":
		if len(args) < 2 {
			fmt.Println("Usage: client put <key> <value>")
			os.Exit(1)
		}
		err = client.Put(args[0], args[1])
	case "get":
		if len(args) < 1 {
			fmt.Println("Usage: client get <key>")
			os.Exit(1)
		}
		err = client.Get(args[0])
	case "delete":
		if len(args) < 1 {
			fmt.Println("Usage: client delete <key>")
			os.Exit(1)
		}
		err = client.Delete(args[0])
	case "batch":
		if len(args) < 2 || len(args)%2 != 0 {
			fmt.Println("Usage: client batch <key1> <value1> <key2> <value2> ...")
			os.Exit(1)
		}
		err = client.BatchPut(args)
	case "status":
		err = client.GetStatus()
	case "help":
		printUsage()
		return
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
	
	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

// parseFlags parses command line arguments and returns config and remaining args
func parseFlags() (*ClientConfig, []string) {
	var (
		serverAddr   = flag.String("server", "localhost:8080", "DistKV server address")
		timeout      = flag.Duration("timeout", 5*time.Second, "Request timeout")
		consistency  = flag.String("consistency", "one", "Consistency level (one, quorum, all)")
	)
	
	// Parse flags and get remaining arguments
	flag.Parse()
	remainingArgs := flag.Args()
	
	// Convert consistency level
	var consistencyLevel proto.ConsistencyLevel
	switch strings.ToLower(*consistency) {
	case "one":
		consistencyLevel = proto.ConsistencyLevel_ONE
	case "quorum":
		consistencyLevel = proto.ConsistencyLevel_QUORUM
	case "all":
		consistencyLevel = proto.ConsistencyLevel_ALL
	default:
		consistencyLevel = proto.ConsistencyLevel_ONE
	}
	
	return &ClientConfig{
		ServerAddress:    *serverAddr,
		Timeout:          *timeout,
		ConsistencyLevel: consistencyLevel,
	}, remainingArgs
}

// NewDistKVClient creates a new DistKV client
func NewDistKVClient(config *ClientConfig) (*DistKVClient, error) {
	// Create gRPC connection
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	
	conn, err := grpc.DialContext(ctx, config.ServerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server %s: %v", config.ServerAddress, err)
	}
	
	return &DistKVClient{
		config:       config,
		conn:         conn,
		distkvClient: proto.NewDistKVClient(conn),
		adminClient:  proto.NewAdminServiceClient(conn),
	}, nil
}

// Put stores a key-value pair
func (c *DistKVClient) Put(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()
	
	req := &proto.PutRequest{
		Key:              key,
		Value:            []byte(value),
		ConsistencyLevel: c.config.ConsistencyLevel,
	}
	
	resp, err := c.distkvClient.Put(ctx, req)
	if err != nil {
		return fmt.Errorf("put failed: %v", err)
	}
	
	if !resp.Success {
		return fmt.Errorf("put failed: %s", resp.ErrorMessage)
	}
	
	fmt.Printf("Successfully stored key '%s'\n", key)
	return nil
}

// Get retrieves a value by key
func (c *DistKVClient) Get(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()
	
	req := &proto.GetRequest{
		Key:              key,
		ConsistencyLevel: c.config.ConsistencyLevel,
	}
	
	resp, err := c.distkvClient.Get(ctx, req)
	if err != nil {
		return fmt.Errorf("get failed: %v", err)
	}
	
	if resp.ErrorMessage != "" {
		return fmt.Errorf("get failed: %s", resp.ErrorMessage)
	}
	
	if !resp.Found {
		fmt.Printf("Key '%s' not found\n", key)
	} else {
		fmt.Printf("Key: %s\nValue: %s\n", key, string(resp.Value))
	}
	
	return nil
}

// Delete removes a key-value pair
func (c *DistKVClient) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()
	
	req := &proto.DeleteRequest{
		Key:              key,
		ConsistencyLevel: c.config.ConsistencyLevel,
	}
	
	resp, err := c.distkvClient.Delete(ctx, req)
	if err != nil {
		return fmt.Errorf("delete failed: %v", err)
	}
	
	if !resp.Success {
		return fmt.Errorf("delete failed: %s", resp.ErrorMessage)
	}
	
	fmt.Printf("Successfully deleted key '%s'\n", key)
	return nil
}

// BatchPut performs multiple put operations
func (c *DistKVClient) BatchPut(args []string) error {
	// Build key-value map from args
	items := make(map[string][]byte)
	for i := 0; i < len(args); i += 2 {
		key := args[i]
		value := args[i+1]
		items[key] = []byte(value)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()
	
	req := &proto.BatchPutRequest{
		Items:            items,
		ConsistencyLevel: c.config.ConsistencyLevel,
	}
	
	resp, err := c.distkvClient.BatchPut(ctx, req)
	if err != nil {
		return fmt.Errorf("batch put failed: %v", err)
	}
	
	if !resp.Success {
		fmt.Printf("Batch put partially failed: %s\n", resp.ErrorMessage)
		if len(resp.FailedKeys) > 0 {
			fmt.Printf("Failed keys: %v\n", resp.FailedKeys)
		}
		return nil // Don't return error for partial failure
	}
	
	fmt.Printf("Successfully stored %d key-value pairs\n", len(items))
	return nil
}

// GetStatus retrieves cluster status
func (c *DistKVClient) GetStatus() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()
	
	req := &proto.ClusterStatusRequest{}
	
	resp, err := c.adminClient.GetClusterStatus(ctx, req)
	if err != nil {
		return fmt.Errorf("get status failed: %v", err)
	}
	
	fmt.Println("=== Cluster Status ===")
	fmt.Printf("Health: %d total nodes, %d alive, %d dead (%.1f%% availability)\n",
		resp.Health.TotalNodes,
		resp.Health.AliveNodes,
		resp.Health.DeadNodes,
		resp.Health.AvailabilityPercentage)
	
	fmt.Println("\n=== Nodes ===")
	for _, node := range resp.Nodes {
		status := "UNKNOWN"
		switch node.Status {
		case proto.NodeStatus_ALIVE:
			status = "ALIVE"
		case proto.NodeStatus_SUSPECT:
			status = "SUSPECT"
		case proto.NodeStatus_DEAD:
			status = "DEAD"
		}
		
		lastSeen := time.Unix(node.LastSeen, 0)
		fmt.Printf("  %s (%s) - %s - Last seen: %s\n",
			node.NodeId, node.Address, status, lastSeen.Format(time.RFC3339))
	}
	
	fmt.Println("\n=== Metrics ===")
	fmt.Printf("Total requests: %d\n", resp.Metrics.TotalRequests)
	fmt.Printf("Average latency: %.2f ms\n", resp.Metrics.AvgLatencyMs)
	
	return nil
}

// Close closes the gRPC connection
func (c *DistKVClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// printUsage prints usage information
func printUsage() {
	fmt.Println("DistKV Client - Command line interface for DistKV")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  client [options] <command> [args...]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -server <address>     Server address (default: localhost:8080)")
	fmt.Println("  -timeout <duration>   Request timeout (default: 5s)")
	fmt.Println("  -consistency <level>  Consistency level: one, quorum, all (default: quorum)")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  put <key> <value>           Store a key-value pair")
	fmt.Println("  get <key>                   Retrieve value for a key")
	fmt.Println("  delete <key>                Delete a key-value pair")
	fmt.Println("  batch <k1> <v1> <k2> <v2>   Store multiple key-value pairs")
	fmt.Println("  status                      Show cluster status")
	fmt.Println("  help                        Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  client put user:123 \"John Doe\"")
	fmt.Println("  client get user:123")
	fmt.Println("  client delete user:123")
	fmt.Println("  client batch key1 val1 key2 val2")
	fmt.Println("  client -consistency=all put important-key \"critical data\"")
}