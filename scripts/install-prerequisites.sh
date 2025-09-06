#!/bin/bash
# Script to install prerequisites for building DistKV
# This helps users who don't have the required tools installed

set -e

echo "🛠️  Installing DistKV Prerequisites"
echo "This script will install Go, protoc, and other required tools."
echo ""

# Detect OS
OS="unknown"
case "$(uname -s)" in
    Linux*)     OS="linux";;
    Darwin*)    OS="mac";;
    MINGW*|CYGWIN*|MSYS*) OS="windows";;
    *)          OS="unknown";;
esac

echo "🖥️  Detected OS: $OS"
echo ""

install_go() {
    echo "📦 Installing Go..."
    
    case $OS in
        "linux")
            if command -v apt &> /dev/null; then
                # Ubuntu/Debian
                sudo apt update
                sudo apt install -y wget
                
                # Download and install Go
                GO_VERSION="1.21.0"
                wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
                sudo rm -rf /usr/local/go
                sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
                rm "go${GO_VERSION}.linux-amd64.tar.gz"
                
                # Add to PATH
                echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
                export PATH=/usr/local/go/bin:$PATH
                
            elif command -v yum &> /dev/null; then
                # CentOS/RHEL/Fedora
                sudo yum install -y golang
            fi
            ;;
        "mac")
            if command -v brew &> /dev/null; then
                brew install go
            else
                echo "Please install Homebrew first: https://brew.sh/"
                echo "Then run: brew install go"
                exit 1
            fi
            ;;
        "windows")
            echo "Please download and install Go from: https://golang.org/dl/"
            echo "Choose the Windows installer (.msi file)"
            exit 1
            ;;
        *)
            echo "Unsupported OS. Please install Go manually from: https://golang.org/dl/"
            exit 1
            ;;
    esac
}

install_protoc() {
    echo "⚡ Installing Protocol Buffers compiler..."
    
    case $OS in
        "linux")
            if command -v apt &> /dev/null; then
                sudo apt update
                sudo apt install -y protobuf-compiler
            elif command -v yum &> /dev/null; then
                sudo yum install -y protobuf-compiler
            fi
            ;;
        "mac")
            if command -v brew &> /dev/null; then
                brew install protobuf
            else
                echo "Please install Homebrew first: https://brew.sh/"
                exit 1
            fi
            ;;
        "windows")
            echo "Please download protoc from: https://github.com/protocolbuffers/protobuf/releases"
            echo "Extract it and add the 'bin' directory to your PATH"
            exit 1
            ;;
        *)
            echo "Please install protoc manually"
            exit 1
            ;;
    esac
}

install_make() {
    echo "🔨 Installing Make..."
    
    case $OS in
        "linux")
            if command -v apt &> /dev/null; then
                sudo apt install -y build-essential
            elif command -v yum &> /dev/null; then
                sudo yum groupinstall -y "Development Tools"
            fi
            ;;
        "mac")
            # Make comes with Xcode Command Line Tools
            if ! command -v make &> /dev/null; then
                echo "Installing Xcode Command Line Tools..."
                xcode-select --install
            fi
            ;;
        "windows")
            echo "On Windows, you can:"
            echo "1. Install Git for Windows (includes make): https://git-scm.com/download/win"
            echo "2. Or install make via Chocolatey: choco install make"
            echo "3. Or use WSL (Windows Subsystem for Linux)"
            exit 1
            ;;
    esac
}

# Check what's already installed
echo "🔍 Checking current installation..."

if command -v go &> /dev/null; then
    echo "✅ Go is already installed: $(go version)"
else
    echo "❌ Go not found"
    INSTALL_GO=true
fi

if command -v protoc &> /dev/null; then
    echo "✅ protoc is already installed: $(protoc --version)"
else
    echo "❌ protoc not found"
    INSTALL_PROTOC=true
fi

if command -v make &> /dev/null; then
    echo "✅ make is already installed"
else
    echo "❌ make not found"
    INSTALL_MAKE=true
fi

echo ""

# Install missing components
if [ "$INSTALL_GO" = true ]; then
    install_go
fi

if [ "$INSTALL_PROTOC" = true ]; then
    install_protoc
fi

if [ "$INSTALL_MAKE" = true ]; then
    install_make
fi

echo ""
echo "🎉 Prerequisites installation completed!"
echo ""
echo "📋 Next steps:"
echo "1. Reload your shell or run: source ~/.bashrc"
echo "2. Verify installation:"
echo "   go version"
echo "   protoc --version" 
echo "   make --version"
echo "3. Build DistKV:"
echo "   bash build.sh"
echo "   # OR"
echo "   make all"