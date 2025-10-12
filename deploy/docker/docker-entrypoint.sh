#!/bin/sh
# DistKV Docker Entrypoint Script
# This script handles initialization, configuration, and graceful startup

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo "${RED}[ERROR]${NC} $1"
}

# Print startup banner
echo "================================================"
echo "  DistKV - Distributed Key-Value Store"
echo "================================================"

# Environment variable defaults
export NODE_ID="${NODE_ID:-docker-node-$(hostname)}"
export ADDRESS="${ADDRESS:-0.0.0.0:8080}"
export DATA_DIR="${DATA_DIR:-/data}"
export LOG_LEVEL="${LOG_LEVEL:-info}"
export REPLICAS="${REPLICAS:-3}"
export READ_QUORUM="${READ_QUORUM:-2}"
export WRITE_QUORUM="${WRITE_QUORUM:-2}"

log_info "Starting DistKV Server"
log_info "Node ID: ${NODE_ID}"
log_info "Address: ${ADDRESS}"
log_info "Data Directory: ${DATA_DIR}"
log_info "Log Level: ${LOG_LEVEL}"

# Validate environment
if [ -z "$NODE_ID" ]; then
    log_error "NODE_ID is required"
    exit 1
fi

if [ -z "$ADDRESS" ]; then
    log_error "ADDRESS is required"
    exit 1
fi

# Create and validate data directory
if [ ! -d "$DATA_DIR" ]; then
    log_warn "Data directory does not exist, creating: $DATA_DIR"
    mkdir -p "$DATA_DIR"
fi

if [ ! -w "$DATA_DIR" ]; then
    log_error "Data directory is not writable: $DATA_DIR"
    exit 1
fi

log_info "Data directory validated: $DATA_DIR"

# Check for existing data
if [ -d "$DATA_DIR/snapshots" ] || [ -d "$DATA_DIR/wal" ]; then
    log_info "Existing data found, performing recovery check"
    DATA_SIZE=$(du -sh "$DATA_DIR" | cut -f1)
    log_info "Current data size: $DATA_SIZE"
fi

# Build command arguments
CMD_ARGS="-node-id=${NODE_ID} -address=${ADDRESS} -data-dir=${DATA_DIR}"

# Add optional arguments
if [ -n "$SEED_NODES" ]; then
    log_info "Seed nodes: ${SEED_NODES}"
    CMD_ARGS="$CMD_ARGS -seed-nodes=${SEED_NODES}"
fi

if [ -n "$REPLICAS" ]; then
    CMD_ARGS="$CMD_ARGS -replicas=${REPLICAS}"
fi

if [ -n "$READ_QUORUM" ]; then
    CMD_ARGS="$CMD_ARGS -read-quorum=${READ_QUORUM}"
fi

if [ -n "$WRITE_QUORUM" ]; then
    CMD_ARGS="$CMD_ARGS -write-quorum=${WRITE_QUORUM}"
fi

if [ -n "$LOG_LEVEL" ]; then
    CMD_ARGS="$CMD_ARGS -log-level=${LOG_LEVEL}"
fi

# Wait for seed nodes to be available
if [ -n "$SEED_NODES" ]; then
    log_info "Waiting for seed nodes to be available..."
    IFS=',' read -ra NODES <<< "$SEED_NODES"

    for NODE in "${NODES[@]}"; do
        HOST=$(echo "$NODE" | cut -d':' -f1)
        PORT=$(echo "$NODE" | cut -d':' -f2)

        log_info "Checking seed node: $HOST:$PORT"

        RETRIES=30
        COUNT=0
        while [ $COUNT -lt $RETRIES ]; do
            if nc -z "$HOST" "$PORT" 2>/dev/null; then
                log_info "Seed node $HOST:$PORT is available"
                break
            fi

            COUNT=$((COUNT + 1))
            if [ $COUNT -eq $RETRIES ]; then
                log_warn "Could not connect to seed node $HOST:$PORT after $RETRIES attempts"
            else
                sleep 2
            fi
        done
    done

    log_info "Seed node availability check complete"
fi

# Trap signals for graceful shutdown
_term() {
    log_info "Received SIGTERM, initiating graceful shutdown..."
    kill -TERM "$child" 2>/dev/null
    wait "$child"
    log_info "Server stopped gracefully"
    exit 0
}

_int() {
    log_info "Received SIGINT, initiating graceful shutdown..."
    kill -INT "$child" 2>/dev/null
    wait "$child"
    log_info "Server stopped gracefully"
    exit 0
}

trap _term SIGTERM
trap _int SIGINT

# Run pre-start hooks (if any)
if [ -d "/docker-entrypoint.d" ]; then
    log_info "Running pre-start initialization scripts..."
    for script in /docker-entrypoint.d/*.sh; do
        if [ -f "$script" ] && [ -x "$script" ]; then
            log_info "Executing: $(basename "$script")"
            "$script"
        fi
    done
fi

# Start the server
log_info "Starting DistKV server with arguments: $CMD_ARGS"
echo "================================================"

# Execute the server command
# shellcheck disable=SC2086
exec ./distkv-server $CMD_ARGS &

child=$!
wait "$child"
