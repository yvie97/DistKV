# DistKV - Distributed Key-Value Store

DistKV is a highly available, scalable distributed key-value store inspired by Amazon's Dynamo and Facebook's Cassandra. It demonstrates core distributed systems concepts including consistent hashing, quorum consensus, vector clocks, and gossip protocols.

## 🚀 Features

- **High Availability**: 99.9%+ availability through data replication and gossip-based failure detection
- **Horizontal Scalability**: Add nodes dynamically to scale throughput and storage
- **Tunable Consistency**: Configurable N/R/W quorum parameters for consistency vs availability trade-offs
- **Persistent Storage**: LSM-tree storage engine with MemTables, SSTables, and compaction
- **Gossip Protocol**: Complete network-based gossip implementation for cluster coordination and failure detection
- **Consistent Hashing**: Virtual node-based partitioning with minimal data movement when scaling
- **Vector Clocks**: Conflict detection and causality tracking for concurrent updates

## 📋 System Requirements

- Go 1.19 or later
- Protocol Buffers compiler (protoc)
- Make (optional, for build automation)

## 🛠️ Quick Start

### 1. Clone the Repository

```bash
git clone <https://github.com/yvie97/DistKV.git>
cd DistKV
```

### 2. Install Prerequisites

**Windows Users:**
```cmd
# Automated build with prerequisite checks
scripts\build.bat
```

**Linux/Mac Users:**
```bash
# Option 1: Automated installation
./scripts/install-prerequisites.sh && make all

# Option 2: Manual installation
# - Go 1.19+: https://golang.org/dl/
# - protoc: https://github.com/protocolbuffers/protobuf/releases
# - make (usually pre-installed on Linux/Mac)
```

### 3. Build the Project

**Using Make (Linux/Mac - Recommended):**
```bash
make all
```

**Using Build Scripts:**
```bash
# Linux/Mac
./scripts/build.sh

# Windows  
scripts\build.bat
```

**Manual Build:**
```bash
go mod tidy
./scripts/generate-proto.sh  # Generates proto/*.pb.go files
make build
```

> **📝 Note**: The protobuf files (`proto/distkv.pb.go` and `proto/distkv_grpc.pb.go`) are auto-generated during build and required for compilation. They are not committed to version control.

### 4. Start a Single Node

```bash
# Start server
./build/distkv-server -node-id=node1 -address=localhost:8080 -data-dir=./data

# In another terminal, use the client
./build/distkv-client put user:123 "John Doe"
./build/distkv-client get user:123
./build/distkv-client status
```

### 5. Start a 3-Node Development Cluster

**Using Make (Linux/Mac):**
```bash
# Start the cluster (runs in background)
make dev-cluster

# Test the cluster
make test-cluster

# Stop the cluster
make stop-cluster
```

**Using Windows Batch Scripts:**
```cmd
# Start the cluster (opens 3 separate windows)
scripts\dev-cluster.bat

# Test the cluster
scripts\test-cluster.bat

# Stop the cluster
scripts\stop-cluster.bat
```

**Manual Testing:**
```bash
# Use the client to interact with any node  
./build/distkv-client --server=localhost:8080 put key1 "value1"
./build/distkv-client --server=localhost:8081 get key1
./build/distkv-client --server=localhost:8082 get key1
```

## 🏗️ Architecture

### System Components

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │     │   Client    │     │   Client    │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────────────┴───────────────────┘
                           │
                    ┌──────▼──────┐
                    │ Coordinator │
                    │   Nodes     │
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
   ┌────▼────┐       ┌────▼────┐       ┌────▼────┐
   │Storage  │       │Storage  │       │Storage  │
   │Node A   │◄─────►│Node B   │◄─────►│Node C   │
   └─────────┘       └─────────┘       └─────────┘
```

### Key Technologies

- **Storage Engine**: LSM-tree with MemTables and SSTables with Bloom filters
  - **Level-based compaction**: Organizes SSTables into 7 levels with exponential growth for optimal read performance
  - **Configurable Bloom filters**: Target false positive rates (e.g., 1%) with automatic parameter calculation
  - Complete iterator interface for range queries and compaction
  - Production-ready compaction with tombstone garbage collection
  - Concurrent-safe operations with proper resource management
  - **Memory management**: Configurable limits (2GB default) with pressure monitoring and automatic GC tuning
- **Partitioning**: Consistent hashing with configurable virtual nodes
- **Replication**: Configurable quorum-based consensus (default: N=3, R=2, W=2)
- **Conflict Resolution**: Vector clocks for causality tracking
- **Failure Detection**: Network-based gossip protocol with heartbeat monitoring
- **Communication**: gRPC with optimized connection pooling
  - Connection pool with health monitoring and automatic reconnection
  - Idle connection cleanup (5 min timeout)
  - Maximum pool size limits (100 connections)

## 🔧 Configuration

### Server Configuration

```bash
distkv-server [options]

Options:
  -node-id string          Unique node identifier (required)
  -address string          Server listen address (default: localhost:8080)
  -data-dir string         Directory for data storage (default: ./data)
  -seed-nodes string       Comma-separated list of seed nodes for cluster joining
  -replicas int           Number of replicas N (default: 3)
  -read-quorum int        Read quorum size R (default: 2)
  -write-quorum int       Write quorum size W (default: 2)
  -virtual-nodes int      Virtual nodes for consistent hashing (default: 150)
```

### Client Configuration

```bash
distkv-client [options] <command> [args...]

Options:
  -server string          Server address (default: localhost:8080)
  -timeout duration       Request timeout (default: 5s)
  -consistency string     Consistency level: one, quorum, all (default: quorum)

Commands:
  put <key> <value>       Store a key-value pair
  get <key>               Retrieve value for a key
  delete <key>            Delete a key-value pair
  batch <k1> <v1> ...     Store multiple key-value pairs
  status                  Show cluster status
```

## 🧪 Testing

### Unit Tests
The project includes comprehensive unit tests organized in the `tests/unit/` directory:

```bash
# Run all unit tests
make test

# Run specific component tests
go test ./tests/unit/consensus/     # Vector clock tests
go test ./tests/unit/storage/       # Storage engine tests
go test ./tests/unit/errors/        # Error handling tests
go test ./tests/unit/logging/       # Logging system tests
go test ./tests/unit/metrics/       # Metrics collection tests
go test ./tests/unit/partition/     # Consistent hashing tests
```

**Test Coverage:**
- **Consensus (Vector Clocks)**: Causality tracking, conflict detection, merging, distributed scenarios (17 tests)
- **Storage Engine**: LSM-tree operations, MemTable, SSTables, compaction, concurrent access (28+ tests)
  - MemTable: CRUD operations, sorting, concurrency, read-only mode (14 tests)
  - Iterators: MemTable, SSTable, merge iteration, tombstone handling (8 tests)
  - Engine: Integration, compaction, statistics, error handling (6 tests)
- **Error Handling**: Error creation, wrapping, context, retryability, all error codes (18 tests)
- **Logging**: Log levels, filtering, structured fields, concurrent logging (15 tests)
- **Metrics**: All metrics categories, snapshots, concurrent access, latency tracking (11 tests)
- **Partition**: Consistent hashing, virtual nodes, distribution, consistency (22 tests)

### Integration Testing
```bash
# Start test cluster
make dev-cluster

# Run integration tests
go test ./tests/integration/...
go test ./tests/chaos/...          # Chaos engineering tests

# Stop cluster
make stop-cluster
```

### Performance Testing
```bash
# Basic throughput test
for i in {1..1000}; do
  ./build/distkv-client put "key$i" "value$i"
done

# Measure read latency
time ./build/distkv-client get key500
```

## 🔍 Consistency Models

### Strong Consistency (W + R > N)
```bash
# Configuration: N=3, W=2, R=2 (default)
# Guarantees: Reads always return the latest write
./build/distkv-client -consistency=quorum put key value
```

### Eventual Consistency (W + R ≤ N)
```bash
# Configuration: N=3, W=1, R=1
# Guarantees: High availability, eventual consistency
./build/distkv-client -consistency=one put key value
```

### Linearizable (W=N, R=1)
```bash
# Configuration: All replicas must acknowledge writes
# Guarantees: Strongest consistency, lower availability
./build/distkv-client -consistency=all put key value
```

## 📊 Monitoring & Observability

### Production-Ready Features

**Structured Logging:**
- Component-based loggers with contextual fields
- Multiple log levels (DEBUG, INFO, WARN, ERROR, FATAL)
- Automatic caller information tracking

**Comprehensive Metrics:**
- **Storage**: Read/write ops, latencies, cache hit rates, compaction stats
- **Replication**: Quorum success rates, conflict resolution tracking
- **Gossip**: Node health, message counts, state transitions
- **Network**: Connections, bytes transferred, timeout tracking

**Error Handling:**
- Structured errors with error codes and context
- Stack trace capture for debugging
- Retryable error classification

**Graceful Shutdown:**
- Proper resource cleanup and data flushing
- Ordered component shutdown with timeouts
- Comprehensive shutdown metrics reporting

### Cluster Status
```bash
./build/distkv-client status
```

### Example Metrics Output
```
=== Cluster Status ===
Health: 3 total nodes, 3 alive, 0 dead (100.0% availability)

=== Nodes ===
  node1 (localhost:8080) - ALIVE - Last seen: 2025-09-05T22:45:29-07:00
  node2 (localhost:8081) - ALIVE - Last seen: 2025-09-05T22:45:28-07:00
  node3 (localhost:8082) - ALIVE - Last seen: 2025-09-05T22:45:27-07:00

=== Storage Metrics ===
Total reads: 5234 (errors: 12)
Total writes: 3421 (errors: 3)
Cache hit rate: 87.3%
SSTable count: 8 (Level 0: 2, Level 1: 3, Level 2: 3)
Compaction count: 15
Memory usage: 1.2GB / 2.0GB (60%)
Heap usage: 856MB
```

## 🐳 Docker Support

### Build Docker Image
```bash
make docker-build
```

### Run Single Node
```bash
make docker-run
```

### Docker Compose Cluster
```bash
docker-compose up -d
```

## 🏗️ Implementation Details

### Storage Engine Features

**Complete LSM-tree Implementation:**
```go
// Unified iterator interface across all storage components
type Iterator interface {
    Valid() bool
    Key() string
    Value() *Entry
    Next()
    Close() error
}
```

**Key Features:**
- ✅ **Complete Iterator Implementation** - Full range query support with MemTable, SSTable, and merge iterators
- ✅ **Production-Ready Compaction** - Level-based compaction with smart overlapping range selection
- ✅ **Optimized Bloom Filters** - Configurable false positive rates with automatic parameter calculation
- ✅ **Memory Management** - Configurable limits, pressure monitoring, and automatic GC tuning
- ✅ **Connection Pooling** - Efficient gRPC connection reuse with health monitoring
- ✅ **Concurrent Safety** - Thread-safe operations across all storage components
- ✅ **Comprehensive Testing** - 111+ unit tests with extensive coverage of all components

**Performance Characteristics:**
- **Iterator**: O(1) MemTable init, O(log n) SSTable lookup, O(k log k) merge (k = sources)
- **Compaction**: O(n log n) with level-based strategy - 2-10x faster reads
- **Memory**: Bounded resource usage with automatic pressure mitigation
- **Network**: 10-100x less overhead with connection pooling

**Storage Configuration:**
```go
type StorageConfig struct {
    // Compaction Strategy
    CompactionStrategy  CompactionStrategy // Level-based (default), Simple, or Size-tiered
    CompactionThreshold int                // Trigger level (default: 4 SSTables)
    LevelSizeMultiplier int                // Size multiplier between levels (default: 10)
    MaxLevels           int                // Maximum number of levels (default: 7)

    // Bloom Filter Optimization
    BloomFilterFPR     float64       // Target false positive rate (default: 0.01 = 1%)
    BloomFilterBits    int           // Alternative: bits per key (default: 10)

    // Memory Management
    MaxMemoryUsage     int64         // Total memory limit (default: 2GB)
    MaxMemTableMemory  int64         // MemTable memory limit (default: 512MB)
    MaxCacheMemory     int64         // Cache memory limit (default: 512MB)

    // Basic Settings
    TombstoneTTL       time.Duration // Garbage collection TTL (default: 3 hours)
    MemTableMaxSize    int64         // Flush threshold (default: 64MB)
}
```

### Advanced Storage Features

**Level-Based Compaction:**
- Organizes SSTables into 7 levels with 10x growth per level
- Smart overlapping range selection for efficient compaction
- 2-10x faster reads compared to simple compaction strategies

**Optimized Bloom Filters:**
- Configure by target false positive rate (e.g., 1%)
- Automatic calculation of optimal parameters
- Fine-grained control over read performance vs memory tradeoff

**Memory Management:**
- Configurable memory limits with automatic pressure detection
- Auto-GC triggering on high memory pressure
- Detailed memory statistics and monitoring

**gRPC Connection Pooling:**
- Intelligent connection reuse with health monitoring
- Automatic cleanup of idle connections (5 min timeout)
- LRU eviction at max pool size (100 connections)

### Project Structure Optimization

**Clean Architecture:**
- **`pkg/` directory**: Contains only production code (no test files for cleaner structure)
- **`tests/unit/` directory**: Organized unit tests with proper package structure using `package_test` pattern
- **Separation of Concerns**: Clear distinction between production code and testing infrastructure
- **Comprehensive Test Coverage**: 111+ unit tests covering all core packages with 100% of critical paths tested

**Test Organization Benefits:**
- **Code Readability**: Clean `pkg/` directory focused on core business logic
- **Test Isolation**: Unified test management with clear hierarchical structure
- **Go Best Practices**: Follows Go community standards with `package_test` naming
- **Quality Assurance**: Extensive test coverage ensures reliability and catches regressions early

## 🔧 Development

### Code Structure
```
DistKV/
├── cmd/                         # Application entry points
│   ├── server/                 # DistKV server implementation
│   │   ├── main.go            # Server entry point and configuration
│   │   ├── services.go        # gRPC service implementations
│   │   ├── node_selector.go   # Node selection and routing logic
│   │   └── replica_client.go  # Inter-node communication client
│   └── client/                # Command-line client
│       └── main.go            # Client CLI implementation
├── pkg/                        # Core distributed systems packages
│   ├── consensus/             # Vector clocks for conflict resolution
│   │   └── vector_clock.go    # Causality tracking implementation
│   ├── errors/               # Comprehensive error handling
│   │   └── errors.go         # Structured errors with codes and context
│   ├── logging/              # Centralized structured logging
│   │   └── logger.go         # Component-based logging with levels
│   ├── metrics/              # Production-ready metrics collection
│   │   └── metrics.go        # Storage, replication, gossip, network metrics
│   ├── gossip/               # Network-based failure detection
│   │   ├── gossip.go         # Gossip protocol implementation
│   │   ├── connection_pool.go # gRPC connection pooling with health monitoring
│   │   └── node_info.go      # Node health and metadata
│   ├── partition/            # Data distribution
│   │   └── consistent_hash.go # Consistent hashing with virtual nodes
│   ├── replication/          # Quorum-based data replication
│   │   └── quorum.go         # N/R/W quorum consensus implementation
│   └── storage/              # LSM-tree storage engine (production-ready)
│       ├── engine.go         # Main storage engine with level-based compaction
│       ├── memtable.go       # In-memory write buffer
│       ├── sstable.go        # Sorted string table implementation
│       ├── iterator.go       # Complete iterator interface (range queries)
│       ├── bloom_filter.go   # Optimized Bloom filters with configurable FPR
│       ├── memory_monitor.go # Memory management and pressure monitoring
│       ├── types.go          # Storage data types and interfaces
│       └── errors.go         # Storage-specific error types
├── proto/                      # Protocol buffer definitions
│   ├── distkv.proto           # gRPC service and message definitions
│   ├── distkv.pb.go          # Generated protobuf code (auto-generated)
│   └── distkv_grpc.pb.go     # Generated gRPC code (auto-generated)
├── scripts/                    # Build automation and utilities
│   ├── build.bat             # Windows build script
│   ├── build.sh              # Linux/Mac build script
│   ├── dev-cluster.bat       # Windows cluster startup
│   ├── stop-cluster.bat      # Windows cluster shutdown
│   ├── test-cluster.bat      # Windows cluster testing
│   ├── generate-proto.sh     # Protobuf code generation
│   └── install-prerequisites.sh # Dependency installation
├── tests/                      # Comprehensive test suites (organized structure)
│   ├── unit/                 # Unit tests for individual components
│   │   ├── consensus/       # Vector clock tests (17 tests)
│   │   ├── storage/         # Storage engine tests (28+ tests)
│   │   ├── errors/          # Error handling tests (18 tests)
│   │   ├── logging/         # Logging system tests (15 tests)
│   │   ├── metrics/         # Metrics collection tests (11 tests)
│   │   └── partition/       # Consistent hashing tests (22 tests)
│   ├── integration/          # Multi-node integration tests
│   └── chaos/                # Fault injection and chaos testing
├── deploy/                     # Production deployment configurations
│   ├── docker/               # Docker deployment files
│   └── k8s/                  # Kubernetes manifests and configs
├── docs/                       # Additional documentation
│   ├── api.md               # API documentation
│   └── operations.md        # Operational guides
├── Dockerfile                  # Container build configuration
├── docker-compose.yml          # Multi-node Docker deployment
├── Makefile                    # Build automation for Unix systems
├── kvstore-design-doc.md       # System design documentation
└── go.mod                      # Go module dependencies
```


### Code Quality
```bash
make fmt      # Format code
make lint     # Run linters
make test     # Run test suite
```

## 🔧 Troubleshooting

### Build Issues

**Problem**: `protoc: command not found`
```bash
# Linux/Ubuntu
sudo apt install protobuf-compiler

# macOS
brew install protobuf

# Windows
# Download from https://github.com/protocolbuffers/protobuf/releases
# Extract and add to PATH
```

**Problem**: `go: command not found`
```bash
# Install Go from https://golang.org/dl/
# Add to PATH: export PATH=$PATH:/usr/local/go/bin
```

**Problem**: Missing `.pb.go` files
```bash
# Generate protobuf files manually
./scripts/generate-proto.sh

# Or build with make (generates automatically)
make all
```

**Problem**: Permission denied on scripts (Linux/Mac)
```bash
chmod +x scripts/*.sh
```

### Runtime Issues

**Problem**: `bind: address already in use`
```bash
# Check what's using the port
lsof -i :8080

# Use different port
./build/distkv-server -address=localhost:8081
```

**Problem**: `connection refused` from client
```bash
# Ensure server is running
./build/distkv-client status

# Check server logs for errors
./build/distkv-server --node-id=debug-node --address=localhost:8080 --data-dir=debug-data
```

### Common Questions

**Q: Why are `.pb.go` files not in the repository?**  
A: These are auto-generated from `.proto` files during build. This keeps the repo clean and ensures compatibility.

**Q: Which consistency level should I use?**  
A: For learning: `quorum` (default). For production: depends on your CAP theorem requirements.

**Q: Can I run this in production?**  
A: This implementation includes production-grade features like persistent storage, replication, and failure detection. However, it's designed for learning distributed systems concepts. For production workloads, consider battle-tested solutions like Cassandra, DynamoDB, or ScyllaDB.

## 📖 Learning Resources

This implementation demonstrates key distributed systems concepts:

- **CAP Theorem**: Choose consistency vs. availability with configurable quorum parameters
- **Consistent Hashing**: Minimize data movement during scaling with virtual nodes
- **Vector Clocks**: Track causality without global coordination for conflict resolution
- **Quorum Consensus**: Balance consistency and availability with N/R/W configuration
- **Gossip Protocols**: Network-based failure detection and cluster coordination
- **LSM-trees**: Write-optimized storage with MemTables, SSTables, and compaction

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `make test`
5. Submit a pull request


## 🙏 Acknowledgments

- Inspired by Amazon's Dynamo paper
- Storage engine design from Cassandra
- Gossip protocol from academic literature
- Vector clock implementation follows standard algorithms

---

**Note**: This is a feature-complete implementation demonstrating distributed systems concepts with production-grade components. For enterprise production use, consider battle-tested solutions like Apache Cassandra, Amazon DynamoDB, or ScyllaDB.