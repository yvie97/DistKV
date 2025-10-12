# DistKV Docker Deployment Guide

## File Organization

This project has Docker configurations in two locations:

### Root Directory (Quick Start)
```
DistKV/
├── Dockerfile              # Simple build for quick testing
└── docker-compose.yml      # Basic 3-node cluster
```

**Purpose:** Quick start for developers who want to test DistKV immediately
**Usage:** `docker-compose up` from project root

### deploy/docker/ (Production Ready)
```
deploy/docker/
├── Dockerfile                 # Enhanced with entrypoint script
├── docker-entrypoint.sh       # Initialization & graceful shutdown
├── docker-compose.dev.yml     # Development with hot-reload
├── docker-compose.prod.yml    # Production configuration
├── .dockerignore              # Build optimization
└── DOCKER_GUIDE.md           # This file
```

**Purpose:** Production deployments with advanced features
**Usage:** See configurations below

---

## Quick Comparison

| Feature | Root Files | deploy/docker/ |
|---------|-----------|----------------|
| **Setup Complexity** | Minimal | Moderate |
| **Startup Time** | Fast | Fast |
| **Entrypoint Script** | ❌ No | ✅ Yes |
| **Environment Variables** | Basic | Comprehensive |
| **Graceful Shutdown** | ❌ No | ✅ Yes |
| **Seed Node Checking** | ❌ No | ✅ Yes |
| **Pre-start Hooks** | ❌ No | ✅ Yes |
| **Hot Reload** | ❌ No | ✅ Yes (dev) |
| **Monitoring** | Optional | ✅ Integrated |
| **Load Balancer** | Optional | ✅ Yes (prod) |
| **Best For** | Quick testing | Development & Production |

---

## When to Use Which

### Use Root Files When:
- ✅ You want to quickly test DistKV
- ✅ You're following the main README
- ✅ You need a simple 3-node cluster
- ✅ You don't need advanced features

**Example:**
```bash
cd DistKV
docker-compose up
```

### Use deploy/docker/ When:
- ✅ You need development features (hot-reload, debugging)
- ✅ You're deploying to production
- ✅ You need monitoring and observability
- ✅ You want graceful shutdown handling
- ✅ You need custom initialization scripts

**Examples:**
```bash
# Development
docker-compose -f deploy/docker/docker-compose.dev.yml up

# Production
docker-compose -f deploy/docker/docker-compose.prod.yml up -d
```

---

## Quick Start Options

### Option 1: Minimal Setup (Root)
```bash
# From project root
docker-compose up

# Test the cluster
docker-compose exec distkv-node1 ./distkv-client put test "Hello"
docker-compose exec distkv-node1 ./distkv-client get test

# Clean up
docker-compose down -v
```

### Option 2: Development (deploy/docker/)
```bash
# Development with hot-reload
docker-compose -f deploy/docker/docker-compose.dev.yml --profile hot-reload up

# Development with monitoring
docker-compose -f deploy/docker/docker-compose.dev.yml --profile with-monitoring up

# Interactive development
docker-compose -f deploy/docker/docker-compose.dev.yml exec distkv-dev sh
cd /app && go test ./...
```

### Option 3: Production (deploy/docker/)
```bash
# Build production image
docker build -t distkv:latest -f deploy/docker/Dockerfile .

# Start production cluster
export VERSION=latest
export GRAFANA_ADMIN_PASSWORD=secure_password
docker-compose -f deploy/docker/docker-compose.prod.yml up -d

# Verify cluster
docker-compose -f deploy/docker/docker-compose.prod.yml ps
docker-compose -f deploy/docker/docker-compose.prod.yml logs -f
```

---

## Features Comparison

### Root Dockerfile vs deploy/docker/Dockerfile

**Root Dockerfile:**
- Simple multi-stage build
- Static CMD
- No entrypoint script
- Basic health check

**deploy/docker/Dockerfile:**
- Enhanced multi-stage build
- Custom entrypoint script
- Environment variable support
- Graceful shutdown handling
- Pre-start hooks support
- Seed node health checking

### Root docker-compose.yml vs deploy/docker/docker-compose.*.yml

**Root docker-compose.yml:**
- 3-node cluster
- Basic environment variables
- Optional Prometheus/Grafana (with profiles)
- Suitable for local testing

**deploy/docker/docker-compose.dev.yml:**
- Development optimized
- Hot-reload support
- Source code mounting
- Debug port exposed
- Test client included
- Redis for comparison

**deploy/docker/docker-compose.prod.yml:**
- Production ready
- Nginx load balancer
- Log rotation
- Resource limits
- External volumes
- Full monitoring stack

---

## Migration Guide

### From Root to deploy/docker/

If you're currently using root files and want to switch to deploy/docker/:

**1. Stop current containers:**
```bash
docker-compose down
```

**2. Switch to deploy/docker/ configuration:**
```bash
# For development
docker-compose -f deploy/docker/docker-compose.dev.yml up

# For production
docker build -t distkv:latest -f deploy/docker/Dockerfile .
docker-compose -f deploy/docker/docker-compose.prod.yml up -d
```

**3. Migrate data (if needed):**
```bash
# Backup from root volumes
docker run --rm -v distkv-node1-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/backup.tar.gz -C /data .

# Restore to deploy/docker volumes
docker run --rm -v distkv_prod_node1_data:/data -v $(pwd):/backup alpine \
  tar xzf /backup/backup.tar.gz -C /data
```

---

## Environment Variables Reference

### Root Configuration
Uses basic environment variables defined in services.

### deploy/docker/ Configuration

Full list of supported environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `NODE_ID` | `docker-node-$(hostname)` | Unique node identifier |
| `ADDRESS` | `0.0.0.0:8080` | Server bind address |
| `DATA_DIR` | `/data` | Data storage directory |
| `LOG_LEVEL` | `info` | Logging level |
| `REPLICAS` | `3` | Replication factor |
| `READ_QUORUM` | `2` | Read quorum size |
| `WRITE_QUORUM` | `2` | Write quorum size |
| `SEED_NODES` | - | Comma-separated seed nodes |

Example usage:
```bash
docker run -e NODE_ID=prod-1 -e LOG_LEVEL=debug distkv:latest
```

---

## Common Tasks

### Building Images

**Root:**
```bash
docker build -t distkv:dev .
```

**deploy/docker/:**
```bash
docker build -t distkv:prod -f deploy/docker/Dockerfile .
```

### Running Single Node

**Root:**
```bash
docker run -p 8080:8080 distkv:dev
```

**deploy/docker/:**
```bash
docker run \
  -e NODE_ID=standalone \
  -e LOG_LEVEL=debug \
  -v $(pwd)/data:/data \
  -p 8080:8080 \
  distkv:prod
```

### Viewing Logs

**Root:**
```bash
docker-compose logs -f distkv-node1
```

**deploy/docker/:**
```bash
# Development
docker-compose -f deploy/docker/docker-compose.dev.yml logs -f distkv-dev

# Production
docker-compose -f deploy/docker/docker-compose.prod.yml logs -f distkv-node1
```

---

## Troubleshooting

### "Which configuration should I use?"

**Quick answer:**
- Just testing? → Root files
- Developing features? → `deploy/docker/docker-compose.dev.yml`
- Production deployment? → `deploy/docker/docker-compose.prod.yml`

### Build context issues

Both Dockerfiles expect to be run from project root:

```bash
# Correct
docker build -t distkv -f deploy/docker/Dockerfile .

# Incorrect
cd deploy/docker && docker build -t distkv .
```

### Volume conflicts

Root and deploy/docker/ use different volume names, so they won't conflict:
- Root: `distkv-node1-data`, `distkv-node2-data`, etc.
- deploy/docker: `distkv_prod_node1_data`, `distkv_dev_data`, etc.

---

## Best Practices

1. **Development**: Use `deploy/docker/docker-compose.dev.yml` with hot-reload
2. **Testing**: Use root `docker-compose.yml` for quick validation
3. **Production**: Always use `deploy/docker/docker-compose.prod.yml`
4. **CI/CD**: Build using `deploy/docker/Dockerfile` and tag appropriately
5. **Documentation**: Update this guide when adding new features

---

## See Also

- Main project README for general information
- `deploy/kubernetes/` for Kubernetes deployment
- `deploy/terraform/` for infrastructure provisioning

---

## Summary

**Two sets of Docker files = Two different use cases:**

1. **Root files** = Quick start, minimal complexity
2. **deploy/docker/** = Production ready, full features

Both are valid and serve different purposes. Choose based on your needs!
