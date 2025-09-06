@echo off
REM DistKV Build Script for Windows
REM This script builds the DistKV project from source on Windows

echo Building DistKV on Windows...

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Go is not installed. Please install Go 1.19 or later:
    echo    https://golang.org/doc/install
    pause
    exit /b 1
)

echo [OK] Found Go

REM Check if protoc is installed  
protoc --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] protoc not found. Please install Protocol Buffers compiler:
    echo    https://github.com/protocolbuffers/protobuf/releases
    echo    Download protoc-*-win64.zip, extract, and add to PATH
    pause
    exit /b 1
)

echo [OK] Found protoc

REM Step 1: Install dependencies
echo.
echo Installing Go dependencies...
go mod tidy
go mod download

REM Step 2: Install protobuf plugins
echo.
echo Installing compatible protobuf Go plugins...
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

REM Step 3: Generate protobuf files
echo.
echo Generating protobuf Go files...
if not exist proto mkdir proto
protoc --proto_path=proto --go_out=proto --go_opt=paths=source_relative --go-grpc_out=proto --go-grpc_opt=paths=source_relative proto/*.proto

REM Step 4: Build binaries
echo.
echo Building binaries...

REM Create build directory
if not exist build mkdir build

REM Build server
echo    Building server...
go build -o build/distkv-server.exe ./cmd/server
if errorlevel 1 (
    echo [ERROR] Server build failed!
    pause
    exit /b 1
)

REM Build client
echo    Building client...
go build -o build/distkv-client.exe ./cmd/client
if errorlevel 1 (
    echo [ERROR] Client build failed!
    pause
    exit /b 1
)

echo.
echo [SUCCESS] Build completed successfully!
echo.
echo Binaries created:
echo    build/distkv-server.exe  - DistKV server
echo    build/distkv-client.exe  - DistKV client
echo.
echo Quick start:
echo    # Start a single server:
echo    build\distkv-server.exe --node-id=node1 --address=localhost:8080 --data-dir=data1
echo.
echo    # Start a 3-node development cluster:
echo    scripts\dev-cluster.bat
echo.
echo    # Test the cluster:
echo    scripts\test-cluster.bat
echo.
echo    # Stop the cluster:
echo    scripts\stop-cluster.bat
echo.
echo    # Use the client:
echo    build\distkv-client.exe --server=localhost:8080 put hello world
echo    build\distkv-client.exe --server=localhost:8080 get hello
echo.
pause