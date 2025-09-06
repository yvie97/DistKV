#!/bin/bash
# DistKV Build Script
# This script builds the DistKV project from source

set -e  # Exit on any error

echo "🏗️  Building DistKV..."

# Function to check if a command exists
command_exists() {
    command -v "$1" &> /dev/null
}

# Check prerequisites
echo "📋 Checking prerequisites..."

if ! command_exists go; then
    echo "❌ Go is not installed. Please install Go 1.19 or later:"
    echo "   https://golang.org/doc/install"
    exit 1
fi

GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | head -1)
echo "✅ Found Go version: $GO_VERSION"

# Check Go version (require 1.19+)
GO_MAJOR=$(echo $GO_VERSION | cut -d'.' -f1 | sed 's/go//')
GO_MINOR=$(echo $GO_VERSION | cut -d'.' -f2)

if [ "$GO_MAJOR" -lt 1 ] || ([ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 19 ]); then
    echo "❌ Go 1.19 or later is required. Current version: $GO_VERSION"
    exit 1
fi

if ! command_exists protoc; then
    echo "❌ protoc (Protocol Buffer compiler) not found"
    echo "   Please install protoc:"
    echo "   Ubuntu/Debian: sudo apt install -y protobuf-compiler"
    echo "   macOS: brew install protobuf"
    echo "   Windows: Download from https://github.com/protocolbuffers/protobuf/releases"
    exit 1
fi

echo "✅ Found protoc"

# Step 1: Install dependencies
echo ""
echo "📦 Installing Go dependencies..."
go mod tidy
go mod download

# Step 2: Install protobuf plugins if needed
echo ""
echo "🔧 Installing protobuf Go plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Step 3: Generate protobuf files
echo ""
echo "⚡ Generating protobuf Go files..."
bash generate-proto.sh

# Step 4: Build binaries
echo ""
echo "🔨 Building binaries..."

# Create build directory
mkdir -p build

# Build server
echo "   Building server..."
go build -o build/distkv-server ./cmd/server

# Build client  
echo "   Building client..."
go build -o build/distkv-client ./cmd/client

# Make binaries executable
chmod +x build/distkv-server
chmod +x build/distkv-client

echo ""
echo "🎉 Build completed successfully!"
echo ""
echo "📁 Binaries created:"
echo "   build/distkv-server  - DistKV server"
echo "   build/distkv-client  - DistKV client"
echo ""
echo "🚀 Quick start:"
echo "   # Start a server:"
echo "   ./build/distkv-server -node-id=node1 -address=localhost:8080 -data-dir=./data"
echo ""
echo "   # Use the client:"
echo "   ./build/distkv-client put hello world"
echo "   ./build/distkv-client get hello"
echo "   ./build/distkv-client status"