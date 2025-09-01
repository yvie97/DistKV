// Package gossip - Main gossip protocol implementation
// This implements the gossip-based failure detection system where nodes
// periodically exchange information about cluster membership and health.
package gossip

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Gossip manages the gossip protocol for failure detection in the cluster.
// It maintains a view of all nodes and their health status.
type Gossip struct {
	// localNode is information about this node
	localNode *NodeInfo
	
	// nodes maps node IDs to their information
	nodes map[string]*NodeInfo
	
	// config holds gossip protocol configuration
	config *GossipConfig
	
	// mutex protects concurrent access to the node map
	mutex sync.RWMutex
	
	// stopChan signals shutdown
	stopChan chan struct{}
	
	// wg tracks background goroutines
	wg sync.WaitGroup
	
	// eventCallbacks are called when node status changes
	eventCallbacks []NodeEventCallback
	
	// started indicates if the gossip protocol is running
	started bool
}

// NodeEvent represents a change in node status.
type NodeEvent struct {
	NodeID    string
	Address   string
	OldStatus NodeStatus
	NewStatus NodeStatus
	Timestamp time.Time
}

// NodeEventCallback is called when a node's status changes.
type NodeEventCallback func(event NodeEvent)

// NewGossip creates a new gossip instance for a node.
func NewGossip(nodeID, address string, config *GossipConfig) *Gossip {
	if config == nil {
		config = DefaultGossipConfig()
	}
	
	localNode := NewNodeInfo(nodeID, address)
	
	return &Gossip{
		localNode:      localNode,
		nodes:          make(map[string]*NodeInfo),
		config:         config,
		stopChan:       make(chan struct{}),
		eventCallbacks: make([]NodeEventCallback, 0),
		started:        false,
	}
}

// Start begins the gossip protocol background processes.
func (g *Gossip) Start() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	if g.started {
		return fmt.Errorf("gossip already started")
	}
	
	// Add local node to the cluster view
	g.nodes[g.localNode.NodeID] = g.localNode.Copy()
	
	// Start background workers
	g.startBackgroundWorkers()
	
	g.started = true
	return nil
}

// Stop shuts down the gossip protocol gracefully.
func (g *Gossip) Stop() error {
	g.mutex.Lock()
	if !g.started {
		g.mutex.Unlock()
		return nil
	}
	g.started = false
	g.mutex.Unlock()
	
	// Signal shutdown
	close(g.stopChan)
	
	// Wait for background workers to finish
	g.wg.Wait()
	
	return nil
}

// AddNode adds a new node to the cluster view.
// This is typically called when a node joins the cluster.
func (g *Gossip) AddNode(nodeID, address string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	if _, exists := g.nodes[nodeID]; !exists {
		nodeInfo := NewNodeInfo(nodeID, address)
		g.nodes[nodeID] = nodeInfo
		
		// Notify callbacks about new node
		g.notifyNodeEvent(NodeEvent{
			NodeID:    nodeID,
			Address:   address,
			OldStatus: NodeDead, // Conceptually, it didn't exist before
			NewStatus: NodeAlive,
			Timestamp: time.Now(),
		})
	}
}

// RemoveNode removes a node from the cluster view.
// This is typically called when a node is permanently removed.
func (g *Gossip) RemoveNode(nodeID string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	if nodeInfo, exists := g.nodes[nodeID]; exists {
		oldStatus := nodeInfo.GetStatus()
		delete(g.nodes, nodeID)
		
		// Notify callbacks about node removal
		g.notifyNodeEvent(NodeEvent{
			NodeID:    nodeID,
			Address:   nodeInfo.Address,
			OldStatus: oldStatus,
			NewStatus: NodeDead,
			Timestamp: time.Now(),
		})
	}
}

// GetNodes returns a snapshot of all known nodes.
func (g *Gossip) GetNodes() map[string]*NodeInfo {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	
	nodes := make(map[string]*NodeInfo)
	for nodeID, nodeInfo := range g.nodes {
		nodes[nodeID] = nodeInfo.Copy()
	}
	
	return nodes
}

// GetAliveNodes returns only the nodes that are currently alive.
func (g *Gossip) GetAliveNodes() []*NodeInfo {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	
	aliveNodes := make([]*NodeInfo, 0)
	for _, nodeInfo := range g.nodes {
		if nodeInfo.GetStatus() == NodeAlive {
			aliveNodes = append(aliveNodes, nodeInfo.Copy())
		}
	}
	
	return aliveNodes
}

// RegisterEventCallback registers a callback for node status changes.
func (g *Gossip) RegisterEventCallback(callback NodeEventCallback) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	g.eventCallbacks = append(g.eventCallbacks, callback)
}

// ProcessGossipMessage processes an incoming gossip message from another node.
// This is where the actual gossip information merging happens.
func (g *Gossip) ProcessGossipMessage(senderID string, nodeUpdates []*NodeInfo) []*NodeInfo {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	responses := make([]*NodeInfo, 0)
	
	// Process each node update in the message
	for _, update := range nodeUpdates {
		nodeID := update.NodeID
		
		if existingNode, exists := g.nodes[nodeID]; exists {
			// Node exists, merge information
			oldStatus := existingNode.GetStatus()
			
			if existingNode.MergeFrom(update) {
				// Information was updated, check for status change
				newStatus := existingNode.GetStatus()
				
				if oldStatus != newStatus {
					g.notifyNodeEvent(NodeEvent{
						NodeID:    nodeID,
						Address:   existingNode.Address,
						OldStatus: oldStatus,
						NewStatus: newStatus,
						Timestamp: time.Now(),
					})
				}
			}
			
			// Always include our version in response
			responses = append(responses, existingNode.Copy())
		} else {
			// New node, add it to our view
			g.nodes[nodeID] = update.Copy()
			
			g.notifyNodeEvent(NodeEvent{
				NodeID:    nodeID,
				Address:   update.Address,
				OldStatus: NodeDead, // Conceptually new
				NewStatus: update.Status,
				Timestamp: time.Now(),
			})
			
			responses = append(responses, update.Copy())
		}
	}
	
	// Also send back information about nodes the sender didn't mention
	for nodeID, nodeInfo := range g.nodes {
		found := false
		for _, update := range nodeUpdates {
			if update.NodeID == nodeID {
				found = true
				break
			}
		}
		
		if !found {
			responses = append(responses, nodeInfo.Copy())
		}
	}
	
	return responses
}

// startBackgroundWorkers starts the gossip protocol background tasks.
func (g *Gossip) startBackgroundWorkers() {
	// Heartbeat worker - updates local node's heartbeat
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		ticker := time.NewTicker(g.config.HeartbeatInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				g.updateLocalHeartbeat()
			case <-g.stopChan:
				return
			}
		}
	}()
	
	// Failure detection worker - checks for dead/suspect nodes
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		ticker := time.NewTicker(g.config.SuspectTimeout / 2) // Check twice per timeout period
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				g.checkNodeHealth()
			case <-g.stopChan:
				return
			}
		}
	}()
	
	// Gossip worker - sends gossip messages to random nodes
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		ticker := time.NewTicker(g.config.GossipInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				g.sendGossipMessages()
			case <-g.stopChan:
				return
			}
		}
	}()
}

// updateLocalHeartbeat increments the local node's heartbeat counter.
func (g *Gossip) updateLocalHeartbeat() {
	g.localNode.UpdateHeartbeat()
	
	// Update the local node in our cluster view
	g.mutex.Lock()
	if localNodeCopy, exists := g.nodes[g.localNode.NodeID]; exists {
		localNodeCopy.HeartbeatCounter = g.localNode.GetHeartbeatCounter()
		localNodeCopy.LastSeen = g.localNode.GetLastSeen()
		localNodeCopy.Version = g.localNode.GetVersion()
	}
	g.mutex.Unlock()
}

// checkNodeHealth examines all nodes and updates their status based on timeouts.
func (g *Gossip) checkNodeHealth() {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	for nodeID, nodeInfo := range g.nodes {
		if nodeID == g.localNode.NodeID {
			continue // Skip local node
		}
		
		oldStatus := nodeInfo.GetStatus()
		expectedStatus := nodeInfo.IsExpired(g.config.SuspectTimeout, g.config.DeadTimeout)
		
		if oldStatus != expectedStatus {
			nodeInfo.SetStatus(expectedStatus)
			
			g.notifyNodeEvent(NodeEvent{
				NodeID:    nodeID,
				Address:   nodeInfo.Address,
				OldStatus: oldStatus,
				NewStatus: expectedStatus,
				Timestamp: time.Now(),
			})
		}
	}
}

// sendGossipMessages sends gossip messages to random nodes.
func (g *Gossip) sendGossipMessages() {
	// This is a simplified version - production would use actual network communication
	// For now, we just simulate the selection of nodes to gossip with
	
	aliveNodes := g.GetAliveNodes()
	if len(aliveNodes) <= 1 {
		return // Only us or no one else alive
	}
	
	// Select random nodes to gossip with (fanout)
	fanout := g.config.GossipFanout
	if fanout > len(aliveNodes)-1 {
		fanout = len(aliveNodes) - 1
	}
	
	// In a real implementation, this would:
	// 1. Select random nodes from aliveNodes
	// 2. Create gossip message with node information
	// 3. Send gRPC message to selected nodes
	// 4. Process their responses
	
	// For now, we just track that gossip would occur
}

// notifyNodeEvent sends node status change events to registered callbacks.
func (g *Gossip) notifyNodeEvent(event NodeEvent) {
	for _, callback := range g.eventCallbacks {
		// Run callback in goroutine to avoid blocking
		go callback(event)
	}
}

// GetLocalNode returns information about the local node.
func (g *Gossip) GetLocalNode() *NodeInfo {
	return g.localNode.Copy()
}

// IsAlive returns true if the gossip protocol is running.
func (g *Gossip) IsAlive() bool {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	
	return g.started
}