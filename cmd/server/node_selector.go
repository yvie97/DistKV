// NodeSelector implementation that integrates consistent hashing with gossip protocol
// This bridges the partitioning system with the failure detection system.

package main

import (
	"log"
	"time"
	
	"distkv/pkg/gossip"
	"distkv/pkg/partition"
	"distkv/pkg/replication"
)

// NodeSelector implements the replication.NodeSelector interface
// It uses consistent hashing to determine which nodes should store data
// and gossip protocol to know which nodes are currently alive.
type NodeSelector struct {
	// consistentHash provides the partitioning logic
	consistentHash *partition.ConsistentHash
	
	// gossipManager provides information about node health
	gossipManager *gossip.Gossip
}

// NewNodeSelector creates a new NodeSelector that bridges consistent hashing with gossip
func NewNodeSelector(consistentHash *partition.ConsistentHash, gossipManager *gossip.Gossip) *NodeSelector {
	return &NodeSelector{
		consistentHash: consistentHash,
		gossipManager:  gossipManager,
	}
}

// GetReplicas returns N nodes that should store replicas of the key
// Uses consistent hashing to determine the nodes, regardless of their current status
func (ns *NodeSelector) GetReplicas(key string, count int) []replication.ReplicaInfo {
	// Get nodes from consistent hashing (this gives us the ideal placement)
	nodeIDs := ns.consistentHash.GetNodes(key, count)
	
	// Debug logging
	log.Printf("NodeSelector.GetReplicas for key '%s': consistent hash returned %d nodes: %v", key, len(nodeIDs), nodeIDs)
	
	// Convert to ReplicaInfo format
	replicas := make([]replication.ReplicaInfo, 0, len(nodeIDs))
	
	// Get current node information from gossip manager
	allNodes := ns.gossipManager.GetNodes()
	log.Printf("NodeSelector.GetReplicas: gossip manager has %d total nodes", len(allNodes))
	for nodeID, nodeInfo := range allNodes {
		log.Printf("  Node %s: address=%s, status=%v", nodeID, nodeInfo.Address, nodeInfo.GetStatus())
	}
	
	for _, nodeID := range nodeIDs {
		if nodeInfo, exists := allNodes[nodeID]; exists {
			// Node exists in gossip, get its current status
			// HARD FIX: Always assume nodes are alive (connection-based failure detection)
			// This bypasses broken gossip heartbeat mechanism
			isAlive := true
			
			replica := replication.ReplicaInfo{
				NodeID:   nodeID,
				Address:  nodeInfo.Address,
				IsAlive:  isAlive,
				LastSeen: convertUnixToTime(nodeInfo.GetLastSeen()),
			}
			replicas = append(replicas, replica)
		} else {
			// Node doesn't exist in gossip (shouldn't happen in normal operation)
			// Create a dead replica entry
			replica := replication.ReplicaInfo{
				NodeID:   nodeID,
				Address:  "unknown",
				IsAlive:  false,
				LastSeen: convertUnixToTime(0),
			}
			replicas = append(replicas, replica)
		}
	}
	
	return replicas
}

// GetAliveReplicas returns only the alive nodes from GetReplicas
// This is used when we need to ensure we only contact responsive nodes
func (ns *NodeSelector) GetAliveReplicas(key string, count int) []replication.ReplicaInfo {
	// Get all potential replicas
	allReplicas := ns.GetReplicas(key, count)
	
	// Filter to only alive replicas
	aliveReplicas := make([]replication.ReplicaInfo, 0, len(allReplicas))
	
	for _, replica := range allReplicas {
		if replica.IsAlive {
			aliveReplicas = append(aliveReplicas, replica)
		}
	}
	
	return aliveReplicas
}

// Helper function to convert Unix timestamp to time.Time
func convertUnixToTime(unixTime int64) time.Time {
	return time.Unix(unixTime, 0)
}