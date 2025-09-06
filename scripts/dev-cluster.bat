@echo off
REM Start DistKV 3-node development cluster

echo Starting DistKV development cluster...

REM Build the binaries first
echo Building server and client...
go build -o build\distkv-server.exe .\cmd\server
go build -o build\distkv-client.exe .\cmd\client

REM Create data directories
if not exist data1 mkdir data1
if not exist data2 mkdir data2  
if not exist data3 mkdir data3

REM Start node1 (seed node)
echo Starting node1 on port 8080...
start "DistKV Node1" build\distkv-server.exe --node-id=node1 --address=localhost:8080 --data-dir=data1

REM Wait for node1 to start
timeout /t 3 /nobreak > nul

REM Start node2
echo Starting node2 on port 8081...
start "DistKV Node2" build\distkv-server.exe --node-id=node2 --address=localhost:8081 --data-dir=data2 --seed-nodes=localhost:8080

REM Wait for node2 to start
timeout /t 3 /nobreak > nul

REM Start node3  
echo Starting node3 on port 8082...
start "DistKV Node3" build\distkv-server.exe --node-id=node3 --address=localhost:8082 --data-dir=data3 --seed-nodes=localhost:8080

echo.
echo Development cluster started successfully!
echo - Node1: localhost:8080
echo - Node2: localhost:8081  
echo - Node3: localhost:8082
echo.
echo Each node is running in a separate window.
echo To stop the cluster, run: scripts\stop-cluster.bat
echo To test the cluster, run: scripts\test-cluster.bat