# DistKV Makefile
# This Makefile helps build and manage the DistKV distributed key-value store project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
PROTOC=protoc

# Binary names
SERVER_BINARY=distkv-server
CLIENT_BINARY=distkv-client

# Directories
BUILD_DIR=build
PROTO_DIR=proto
CMD_DIR=cmd

# Proto files
PROTO_FILES=$(wildcard $(PROTO_DIR)/*.proto)
PROTO_GO_FILES=$(PROTO_FILES:.proto=.pb.go)

.PHONY: all clean build server client proto test deps help run-server run-client

# Default target
all: deps proto build

# Generate protobuf files
proto: $(PROTO_GO_FILES)

$(PROTO_DIR)/%.pb.go: $(PROTO_DIR)/%.proto
	@echo "Generating protobuf files..."
	$(PROTOC) --proto_path=$(PROTO_DIR) \
		--go_out=$(PROTO_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_DIR) --go-grpc_opt=paths=source_relative \
		$<

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) tidy
	$(GOMOD) download

# Build all binaries
build: server client

# Build server
server:
	@echo "Building server..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(SERVER_BINARY) $(CMD_DIR)/server

# Build client
client:
	@echo "Building client..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(CLIENT_BINARY) $(CMD_DIR)/client

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f $(PROTO_GO_FILES)

# Run server (development)
run-server:
	@echo "Running DistKV server..."
	$(GOBUILD) -o $(BUILD_DIR)/$(SERVER_BINARY) $(CMD_DIR)/server
	$(BUILD_DIR)/$(SERVER_BINARY) -node-id=dev-node1 -address=localhost:8080 -data-dir=./dev-data

# Run client (development)
run-client:
	@echo "Running DistKV client..."
	$(GOBUILD) -o $(BUILD_DIR)/$(CLIENT_BINARY) $(CMD_DIR)/client
	$(BUILD_DIR)/$(CLIENT_BINARY) help

# Development cluster (3 nodes)
dev-cluster:
	@echo "Starting development cluster..."
	@mkdir -p dev-data/node1 dev-data/node2 dev-data/node3
	$(BUILD_DIR)/$(SERVER_BINARY) -node-id=node1 -address=localhost:8080 -data-dir=./dev-data/node1 &
	sleep 2
	$(BUILD_DIR)/$(SERVER_BINARY) -node-id=node2 -address=localhost:8081 -data-dir=./dev-data/node2 -seed-nodes=localhost:8080 &
	sleep 2
	$(BUILD_DIR)/$(SERVER_BINARY) -node-id=node3 -address=localhost:8082 -data-dir=./dev-data/node3 -seed-nodes=localhost:8080 &
	@echo "Development cluster started on ports 8080, 8081, 8082"
	@echo "To stop cluster: make stop-cluster"

# Stop development cluster
stop-cluster:
	@echo "Stopping development cluster..."
	@pkill -f distkv-server || true
	@echo "Development cluster stopped"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Running linters..."
	golangci-lint run

# Generate code coverage report
coverage:
	@echo "Generating coverage report..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Install tools required for development
install-tools:
	@echo "Installing development tools..."
	$(GOGET) google.golang.org/protobuf/cmd/protoc-gen-go
	$(GOGET) google.golang.org/grpc/cmd/protoc-gen-go-grpc
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t distkv:latest .

# Docker run
docker-run:
	@echo "Running DistKV in Docker..."
	docker run -p 8080:8080 distkv:latest

# Help
help:
	@echo "DistKV Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  all          - Build everything (deps + proto + build)"
	@echo "  deps         - Install Go dependencies"
	@echo "  proto        - Generate protobuf files"
	@echo "  build        - Build server and client binaries"
	@echo "  server       - Build server binary only"
	@echo "  client       - Build client binary only"
	@echo "  test         - Run all tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  run-server   - Run server in development mode"
	@echo "  run-client   - Run client with help"
	@echo "  dev-cluster  - Start 3-node development cluster"
	@echo "  stop-cluster - Stop development cluster"
	@echo "  fmt          - Format Go code"
	@echo "  lint         - Run code linters"
	@echo "  coverage     - Generate test coverage report"
	@echo "  install-tools- Install development tools"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run in Docker container"
	@echo "  help         - Show this help"