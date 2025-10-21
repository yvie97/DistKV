# Grafana Configuration for DistKV

This directory contains Grafana provisioning configurations for monitoring DistKV clusters.

## Directory Structure

```
grafana/
├── datasources/           # Datasource configurations
│   └── prometheus.yml    # Prometheus datasource
├── dashboards/           # Dashboard provisioning
│   └── dashboard.yml    # Dashboard provider config
└── README.md            # This file
```

## Datasources

### Prometheus
- **URL**: `http://prometheus:9090`
- **Type**: Prometheus
- **Scrape Interval**: 15s
- **Query Timeout**: 60s

The Prometheus datasource is automatically configured when Grafana starts.

## Dashboards

Currently, this is a placeholder configuration. Custom DistKV dashboards can be added as JSON files in this directory.

### Creating Custom Dashboards

1. **Access Grafana**: http://localhost:3000
   - Default credentials: `admin` / `admin`

2. **Create a dashboard** manually in the UI with panels for:
   - **Storage Metrics**: Read/write operations, latency, cache hit rate
   - **Replication Metrics**: Quorum success rate, conflicts
   - **Gossip Metrics**: Node health, message counts
   - **Network Metrics**: Connections, throughput

3. **Export the dashboard**:
   - Click Dashboard Settings → JSON Model
   - Copy the JSON
   - Save to `dashboards/distkv-overview.json`

4. **Reload Grafana** or wait for auto-reload (10s interval)

## Future Enhancements

When DistKV implements a `/metrics` endpoint with Prometheus format:

### Example Metrics to Monitor

```
# Storage
distkv_storage_read_ops_total
distkv_storage_write_ops_total
distkv_storage_read_latency_seconds
distkv_storage_cache_hit_rate

# Replication
distkv_replication_quorum_success_total
distkv_replication_conflicts_total

# Gossip
distkv_gossip_nodes_alive
distkv_gossip_nodes_dead
distkv_gossip_messages_sent_total

# Network
distkv_network_connections_active
distkv_network_bytes_sent_total
distkv_network_bytes_received_total
```

### Example Dashboard Panels

1. **QPS (Queries Per Second)**
   ```promql
   rate(distkv_storage_read_ops_total[5m]) + rate(distkv_storage_write_ops_total[5m])
   ```

2. **P99 Read Latency**
   ```promql
   histogram_quantile(0.99, rate(distkv_storage_read_latency_seconds_bucket[5m]))
   ```

3. **Cluster Health**
   ```promql
   distkv_gossip_nodes_alive / (distkv_gossip_nodes_alive + distkv_gossip_nodes_dead)
   ```

4. **Cache Hit Rate**
   ```promql
   distkv_storage_cache_hit_rate
   ```

## Usage

This configuration is automatically loaded when using Docker Compose with the monitoring profile:

```bash
docker-compose --profile with-monitoring up -d
```

Access Grafana at http://localhost:3000
