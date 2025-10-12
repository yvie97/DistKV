# DistKV Dockerfile
# Multi-stage build for optimal image size

# Build stage
FROM golang:1.19-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make protobuf protobuf-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Install protoc-gen-go and protoc-gen-go-grpc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate protobuf files
RUN protoc --proto_path=proto \
    --go_out=proto --go_opt=paths=source_relative \
    --go-grpc_out=proto --go-grpc_opt=paths=source_relative \
    proto/*.proto

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o distkv-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o distkv-client ./cmd/client

# Runtime stage
FROM alpine:latest

# Install runtime dependencies (including netcat for entrypoint script)
RUN apk --no-cache add ca-certificates tzdata netcat-openbsd bash

# Create non-root user
RUN addgroup -g 1001 distkv && \
    adduser -D -u 1001 -G distkv distkv

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/distkv-server .
COPY --from=builder /app/distkv-client .

# Copy entrypoint script
COPY deploy/docker/docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Create data and hook directories
RUN mkdir -p /data /docker-entrypoint.d && chown -R distkv:distkv /data /app /docker-entrypoint.d

# Switch to non-root user
USER distkv

# Expose default port
EXPOSE 8080

# Set default data directory
ENV DATA_DIR=/data

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ./distkv-client -server=localhost:8080 status || exit 1

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

# Default command (can be overridden)
CMD []

# Labels for metadata
LABEL maintainer="DistKV Team"
LABEL description="DistKV - Distributed Key-Value Store"
LABEL version="1.0"