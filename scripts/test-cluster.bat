@echo off
REM Test DistKV development cluster

echo Testing DistKV development cluster...
echo.

REM Build client if needed
if not exist build\distkv-client.exe (
    echo Building client...
    go build -o build\distkv-client.exe .\cmd\client
)

echo Writing test data to node1...
build\distkv-client.exe --server=localhost:8080 put test:cluster "dev-cluster-works"
if %errorlevel% neq 0 (
    echo Error: Failed to write to node1. Is the cluster running?
    pause
    exit /b 1
)

echo.
echo Reading test data from node2...
build\distkv-client.exe --server=localhost:8081 get test:cluster
if %errorlevel% neq 0 (
    echo Error: Failed to read from node2.
    pause
    exit /b 1
)

echo.
echo Reading test data from node3...
build\distkv-client.exe --server=localhost:8082 get test:cluster
if %errorlevel% neq 0 (
    echo Error: Failed to read from node3.
    pause
    exit /b 1
)

echo.
echo ========================================
echo Cluster test completed successfully!
echo The distributed system is working properly.
echo ========================================
echo.
pause