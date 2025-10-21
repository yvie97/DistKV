// Package gossip - gRPC Connection Pool implementation
// Provides efficient connection reuse, health checking, and automatic cleanup
package gossip

import (
	"context"
	"distkv/pkg/logging"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// ConnectionPoolConfig holds configuration for the connection pool.
type ConnectionPoolConfig struct {
	MaxConnections    int           // Maximum number of connections to maintain
	MaxIdleTime       time.Duration // Maximum time a connection can be idle
	HealthCheckPeriod time.Duration // How often to check connection health
	DialTimeout       time.Duration // Timeout for establishing new connections
}

// PooledConnection wraps a gRPC connection with metadata.
type PooledConnection struct {
	conn         *grpc.ClientConn
	address      string
	lastUsed     time.Time
	inUse        bool
	createdAt    time.Time
	failureCount int
}

// ConnectionPool manages a pool of gRPC connections.
type ConnectionPool struct {
	config      *ConnectionPoolConfig
	connections map[string]*PooledConnection
	mutex       sync.RWMutex
	stopChan    chan struct{}
	wg          sync.WaitGroup
	logger      *logging.Logger
}

// NewConnectionPool creates a new connection pool.
func NewConnectionPool(config *ConnectionPoolConfig, logger *logging.Logger) *ConnectionPool {
	if config == nil {
		config = &ConnectionPoolConfig{
			MaxConnections:    100,
			MaxIdleTime:       5 * time.Minute,
			HealthCheckPeriod: 30 * time.Second,
			DialTimeout:       5 * time.Second,
		}
	}

	pool := &ConnectionPool{
		config:      config,
		connections: make(map[string]*PooledConnection),
		stopChan:    make(chan struct{}),
		logger:      logger.WithComponent("connection-pool"),
	}

	// Start background maintenance
	pool.startMaintenance()

	return pool
}

// GetConnection gets or creates a connection to the specified address.
func (p *ConnectionPool) GetConnection(address string) (*grpc.ClientConn, error) {
	p.mutex.Lock()

	// Check if we have a healthy connection
	if pooledConn, exists := p.connections[address]; exists {
		state := pooledConn.conn.GetState()

		// Check if connection is healthy
		if state == connectivity.Ready || state == connectivity.Idle {
			pooledConn.lastUsed = time.Now()
			p.mutex.Unlock()
			return pooledConn.conn, nil
		}

		// Connection is bad, remove it
		p.logger.WithFields(map[string]interface{}{
			"address": address,
			"state":   state.String(),
		}).Debug("Removing unhealthy connection")
		pooledConn.conn.Close()
		delete(p.connections, address)
	}

	p.mutex.Unlock()

	// Create new connection
	return p.createConnection(address)
}

// createConnection creates a new gRPC connection.
func (p *ConnectionPool) createConnection(address string) (*grpc.ClientConn, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Double-check if another goroutine created it
	if pooledConn, exists := p.connections[address]; exists {
		state := pooledConn.conn.GetState()
		if state == connectivity.Ready || state == connectivity.Idle {
			pooledConn.lastUsed = time.Now()
			return pooledConn.conn, nil
		}
	}

	// Check pool size limit
	if len(p.connections) >= p.config.MaxConnections {
		p.evictOldestConnection()
	}

	// Create connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), p.config.DialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(10*1024*1024), // 10MB
			grpc.MaxCallSendMsgSize(10*1024*1024), // 10MB
		),
	)
	if err != nil {
		p.logger.WithError(err).WithField("address", address).
			Debug("Failed to create connection")
		return nil, err
	}

	// Add to pool
	pooledConn := &PooledConnection{
		conn:      conn,
		address:   address,
		lastUsed:  time.Now(),
		createdAt: time.Now(),
		inUse:     false,
	}
	p.connections[address] = pooledConn

	p.logger.WithFields(map[string]interface{}{
		"address":  address,
		"poolSize": len(p.connections),
	}).Debug("Created new pooled connection")

	return conn, nil
}

// evictOldestConnection removes the least recently used connection.
func (p *ConnectionPool) evictOldestConnection() {
	var oldestAddress string
	var oldestTime time.Time

	for address, pooledConn := range p.connections {
		if !pooledConn.inUse {
			if oldestAddress == "" || pooledConn.lastUsed.Before(oldestTime) {
				oldestAddress = address
				oldestTime = pooledConn.lastUsed
			}
		}
	}

	if oldestAddress != "" {
		if pooledConn, exists := p.connections[oldestAddress]; exists {
			pooledConn.conn.Close()
			delete(p.connections, oldestAddress)
			p.logger.WithField("address", oldestAddress).Debug("Evicted oldest connection")
		}
	}
}

// startMaintenance starts background maintenance tasks.
func (p *ConnectionPool) startMaintenance() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(p.config.HealthCheckPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				p.cleanupIdleConnections()
			case <-p.stopChan:
				return
			}
		}
	}()
}

// cleanupIdleConnections removes connections that have been idle too long.
func (p *ConnectionPool) cleanupIdleConnections() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	now := time.Now()
	var toRemove []string

	for address, pooledConn := range p.connections {
		// Check if connection is idle for too long
		if !pooledConn.inUse && now.Sub(pooledConn.lastUsed) > p.config.MaxIdleTime {
			toRemove = append(toRemove, address)
			continue
		}

		// Check connection health
		state := pooledConn.conn.GetState()
		if state == connectivity.Shutdown || state == connectivity.TransientFailure {
			toRemove = append(toRemove, address)
		}
	}

	// Remove idle/unhealthy connections
	for _, address := range toRemove {
		if pooledConn, exists := p.connections[address]; exists {
			pooledConn.conn.Close()
			delete(p.connections, address)
			p.logger.WithField("address", address).Debug("Cleaned up idle/unhealthy connection")
		}
	}

	if len(toRemove) > 0 {
		p.logger.WithFields(map[string]interface{}{
			"removedCount": len(toRemove),
			"poolSize":     len(p.connections),
		}).Debug("Cleanup completed")
	}
}

// Close closes all connections and stops maintenance.
func (p *ConnectionPool) Close() error {
	p.logger.Info("Closing connection pool")

	// Stop maintenance
	close(p.stopChan)
	p.wg.Wait()

	// Close all connections
	p.mutex.Lock()
	defer p.mutex.Unlock()

	connectionCount := len(p.connections)
	for address, pooledConn := range p.connections {
		if err := pooledConn.conn.Close(); err != nil {
			p.logger.WithError(err).WithField("address", address).
				Warn("Error closing connection")
		}
	}
	p.connections = make(map[string]*PooledConnection)

	p.logger.WithField("closedCount", connectionCount).Info("Connection pool closed")
	return nil
}

// Stats returns statistics about the connection pool.
func (p *ConnectionPool) Stats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	stats := map[string]interface{}{
		"totalConnections": len(p.connections),
		"maxConnections":   p.config.MaxConnections,
	}

	// Count by state
	stateCount := make(map[string]int)
	for _, pooledConn := range p.connections {
		state := pooledConn.conn.GetState().String()
		stateCount[state]++
	}
	stats["byState"] = stateCount

	return stats
}
