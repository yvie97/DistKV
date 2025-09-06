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

# Binary names (with .exe extension for Windows)
ifeq ($(OS),Windows_NT)
	SERVER_BINARY=distkv-server.exe
	CLIENT_BINARY=distkv-client.exe
else
	SERVER_BINARY=distkv-server
	CLIENT_BINARY=distkv-client
endif

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
	@chmod +x scripts/generate-proto.sh
	@./scripts/generate-proto.sh

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
dev-cluster: build
	@echo "Starting development cluster..."
	@mkdir -p data1 data2 data3
	@echo "Starting node1 on port 8080..."
	start "DistKV Node1" $(BUILD_DIR)/$(SERVER_BINARY) --node-id=node1 --address=localhost:8080 --data-dir=data1
	@timeout /t 3 > nul
	@echo "Starting node2 on port 8081..."
	start "DistKV Node2" $(BUILD_DIR)/$(SERVER_BINARY) --node-id=node2 --address=localhost:8081 --data-dir=data2 --seed-nodes=localhost:8080
	@timeout /t 3 > nul  
	@echo "Starting node3 on port 8082..."
	start "DistKV Node3" $(BUILD_DIR)/$(SERVER_BINARY) --node-id=node3 --address=localhost:8082 --data-dir=data3 --seed-nodes=localhost:8080
	@echo "Development cluster started on ports 8080, 8081, 8082"
	@echo "Each node is running in a separate window"
	@echo "To stop cluster: make stop-cluster"

# Stop development cluster
stop-cluster:
	@echo "Stopping development cluster..."
ifeq ($(OS),Windows_NT)
	@taskkill /F /IM $(SERVER_BINARY) 2>nul || echo "No running nodes found"
else
	@pkill -f distkv-server || true
endif
	@echo "Development cluster stopped"

# Test the development cluster
test-cluster: client
	@echo "Testing development cluster..."
	@echo "Writing test data..."
	$(BUILD_DIR)/$(CLIENT_BINARY) --server=localhost:8080 put test:cluster "make-dev-cluster-works"
	@echo "Reading from different nodes..."
	$(BUILD_DIR)/$(CLIENT_BINARY) --server=localhost:8081 get test:cluster
	$(BUILD_DIR)/$(CLIENT_BINARY) --server=localhost:8082 get test:cluster
	@echo "Cluster test completed successfully!"

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
	@echo "  test-cluster - Test the development cluster"
	@echo "  fmt          - Format Go code"
	@echo "  lint         - Run code linters"
	@echo "  coverage     - Generate test coverage report"
	@echo "  install-tools- Install development tools"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run in Docker container"
	@echo "  help         - Show this help"