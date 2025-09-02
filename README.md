# DistKV - Distributed Key-Value Store

DistKV is a highly available, scalable distributed key-value store inspired by Amazon's Dynamo and Facebook's Cassandra. It demonstrates core distributed systems concepts including consistent hashing, quorum consensus, vector clocks, and gossip protocols.

## 🚀 Features

- **High Availability**: 99.9%+ availability through data replication and failure detection
- **Horizontal Scalability**: Add nodes dynamically to scale throughput and storage
- **Tunable Consistency**: Choose between strong consistency and high availability
- **Sub-millisecond Latency**: Optimized read/write paths with intelligent caching
- **LSM-tree Storage**: Write-optimized storage engine with compaction
- **Failure Detection**: Gossip-based protocol for automatic node failure detection
- **Consistent Hashing**: Minimal data movement when adding/removing nodes

## 📋 System Requirements

- Go 1.19 or later
- Protocol Buffers compiler (protoc)
- Make (optional, for build automation)

## 🛠️ Quick Start

### 1. Clone and Build

```bash
# Clone the repository
git clone <repository-url>
cd DistKV

# Install dependencies and build
make all

# Or manually:
go mod tidy
protoc --proto_path=proto --go_out=proto --go-grpc_out=proto proto/*.proto
make build
```

### 2. Start a Single Node

```bash
# Start server
./build/distkv-server -node-id=node1 -address=localhost:8080 -data-dir=./data

# In another terminal, use the client
./build/distkv-client put user:123 "John Doe"
./build/distkv-client get user:123
./build/distkv-client status
```

### 3. Start a 3-Node Cluster

```bash
# Start the cluster (runs in background)
make dev-cluster

# Use the client to interact with the cluster
./build/distkv-client put key1 "value1"
./build/distkv-client get key1

# Stop the cluster
make stop-cluster
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

- **Storage Engine**: LSM-tree with MemTables and SSTables
- **Partitioning**: Consistent hashing with virtual nodes
- **Replication**: Quorum-based consensus (N=3, R=2, W=2)
- **Conflict Resolution**: Vector clocks for causality tracking
- **Failure Detection**: Gossip protocol with configurable timeouts
- **Communication**: gRPC for high-performance inter-node communication

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
```bash
make test
```

### Integration Testing
```bash
# Start test cluster
make dev-cluster

# Run integration tests
go test ./tests/integration/...

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

## 📊 Monitoring

### Cluster Status
```bash
./build/distkv-client status
```

### Metrics Available
- Total requests processed
- Average read/write latency
- Node availability percentage
- Storage utilization per node
- Cache hit rates

### Example Output
```
=== Cluster Status ===
Health: 3 total nodes, 3 alive, 0 dead (100.0% availability)

=== Nodes ===
  node1 (localhost:8080) - ALIVE - Last seen: 2024-01-15T10:30:45Z
  node2 (localhost:8081) - ALIVE - Last seen: 2024-01-15T10:30:44Z
  node3 (localhost:8082) - ALIVE - Last seen: 2024-01-15T10:30:43Z

=== Metrics ===
Total requests: 1543
Average latency: 2.3 ms
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

## 🔧 Development

### Prerequisites
```bash
# Install development tools
make install-tools

# Install protoc (Protocol Buffers compiler)
# On Ubuntu/Debian:
sudo apt install -y protobuf-compiler

# On macOS:
brew install protobuf
```

### Code Structure
```
DistKV/
├── cmd/                    # Application entry points
│   ├── server/            # DistKV server
│   └── client/            # Command-line client
├── pkg/                   # Core packages
│   ├── consensus/         # Vector clocks for conflict resolution
│   ├── gossip/           # Failure detection and node discovery
│   ├── partition/        # Consistent hashing implementation
│   ├── replication/      # Quorum-based replication
│   └── storage/          # LSM-tree storage engine
├── proto/                # Protocol buffer definitions
├── tests/                # Test suites
│   ├── unit/            # Unit tests
│   ├── integration/     # Integration tests
│   └── chaos/           # Chaos engineering tests
└── deploy/              # Deployment configurations
    ├── docker/          # Docker configurations
    └── k8s/             # Kubernetes manifests
```

### Adding New Features

1. **Storage Features**: Modify `pkg/storage/`
2. **Consensus Logic**: Update `pkg/consensus/`
3. **Network Protocols**: Extend `proto/` definitions
4. **Client Features**: Add to `cmd/client/`

### Code Quality
```bash
make fmt      # Format code
make lint     # Run linters
make coverage # Generate coverage report
```

## 📖 Learning Resources

This implementation demonstrates key distributed systems concepts:

- **CAP Theorem**: Choose consistency vs. availability
- **Consistent Hashing**: Minimize data movement during scaling
- **Vector Clocks**: Track causality without global coordination
- **Quorum Consensus**: Balance consistency and availability
- **Gossip Protocols**: Efficient failure detection
- **LSM-trees**: Write-optimized storage for high throughput

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `make test`
5. Submit a pull request

## 📜 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- Inspired by Amazon's Dynamo paper
- Storage engine design from Cassandra
- Gossip protocol from academic literature
- Vector clock implementation follows standard algorithms

---

**Note**: This is an educational implementation demonstrating distributed systems concepts. For production use, consider established solutions like Apache Cassandra, Amazon DynamoDB, or ScyllaDB.