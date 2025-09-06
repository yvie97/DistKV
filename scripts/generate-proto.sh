#!/bin/bash
# Script to generate Go protobuf files from .proto definitions
# Run this script when you have protoc and the Go plugins installed

set -e  # Exit on any error

echo "Generating protobuf Go files..."

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc (Protocol Buffer compiler) not found"
    echo "Please install protoc:"
    echo "  Ubuntu/Debian: sudo apt install -y protobuf-compiler"
    echo "  macOS: brew install protobuf"
    echo "  Windows: Download from https://github.com/protocolbuffers/protobuf/releases"
    exit 1
fi

# Install compatible versions of Go protobuf plugins
echo "Installing compatible protobuf Go plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Create output directory if it doesn't exist
mkdir -p proto

# Generate Go files from proto definitions
protoc --proto_path=proto \
    --go_out=proto --go_opt=paths=source_relative \
    --go-grpc_out=proto --go-grpc_opt=paths=source_relative \
    proto/*.proto

echo "✅ Protobuf Go files generated successfully!"
echo "Generated files:"
find proto -name "*.pb.go" -o -name "*_grpc.pb.go"