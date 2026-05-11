package main

import (
	"design-with-tsgo/pkg/logger"
	"os"
	"time"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// GossipValue holds a key's value and its version number.
// Version is incremented on each Set to resolve conflicts
// during gossip merges.
type GossipValue struct {
	Value   string
	Version int
}

// GossipNode represents a node in a gossip-based cluster.
// Each node maintains its own state map and can exchange
// state with peers via GossipTo, which performs a bidirectional merge.
type GossipNode struct {
	ID    string
	state map[string]GossipValue
}

// NewGossipNode creates a new gossip node with the given ID and an empty state.
// Time complexity: O(1).
func NewGossipNode(id string) *GossipNode {
	defer log.Operation("NewGossipNode", "Creating gossip node id=%s", id)()
	return &GossipNode{ID: id, state: make(map[string]GossipValue)}
}

// Set updates a key to the given value, incrementing the version number.
// Time complexity: O(1).
func (n *GossipNode) Set(key, value string) {
	defer log.Operation("Set", "Node %s setting %s=%s", n.ID, key, value)()
	current := n.state[key]
	n.state[key] = GossipValue{Value: value, Version: current.Version + 1}
	log.Debug("Node %s: %s = %s (version %d)", n.ID, key, value, n.state[key].Version)
}

// GossipTo performs a bidirectional state exchange with a peer.
// Both nodes merge each other's state, resolving conflicts by version.
// Time complexity: O(s + p) where s and p are the state sizes of both nodes.
func (n *GossipNode) GossipTo(peer *GossipNode) {
	defer log.Operation("GossipTo", "Node %s gossiping with %s", n.ID, peer.ID)()
	peer.merge(n.state)
	n.merge(peer.state)
	log.Debug("Bidirectional gossip complete: %s <-> %s", n.ID, peer.ID)
}

// Get retrieves the value for a key from the node's local state.
// Returns the value and a boolean indicating if the key exists.
// Time complexity: O(1).
func (n *GossipNode) Get(key string) (string, bool) {
	value, ok := n.state[key]
	if !ok {
		log.Warn("Key %s not found on node %s", key, n.ID)
	} else {
		log.Debug("Node %s: GET %s = %s (v%d)", n.ID, key, value.Value, value.Version)
	}
	return value.Value, ok
}

// merge incorporates remote state into the local state, keeping the
// entry with the highest version for each key.
// Time complexity: O(r) where r is the size of remote state.
func (n *GossipNode) merge(remote map[string]GossipValue) {
	for key, incoming := range remote {
		if local, ok := n.state[key]; !ok || incoming.Version > local.Version {
			n.state[key] = incoming
			log.Debug("Node %s merged key=%s value=%s version=%d", n.ID, key, incoming.Value, incoming.Version)
		}
	}
}

func main() {
	defer log.Operation("main", "Running Gossip Protocol demo")()

	logger.Section("GOSSIP PROTOCOL — Production Scenario")

	// Simulate a 5-node cluster spreading configuration updates
	log.Info("Bringing up 5-node gossip cluster for config propagation")
	start := time.Now()
	nodes := []*GossipNode{
		NewGossipNode("node-a"),
		NewGossipNode("node-b"),
		NewGossipNode("node-c"),
		NewGossipNode("node-d"),
		NewGossipNode("node-e"),
	}
	log.Info("Cluster initialization completed in %v", time.Since(start))

	logger.Section("SCENARIO: Config Update Propagation via Gossip")
	log.Info("Node-A receives a config update: 'feature_flag' = 'enabled'")
	nodes[0].Set("feature_flag", "enabled")
	nodes[0].Set("max_connections", "1000")

	log.Info("Node-B independently sets 'max_connections' = '500' (stale)")
	nodes[1].Set("max_connections", "500")

	logger.Section("SCENARIO: Gossip Round 1 — A <-> B")
	log.Info("Node-A gossips with Node-B")
	nodes[0].GossipTo(nodes[1])

	// Verify propagation
	val, _ := nodes[1].Get("feature_flag")
	log.Info("Node-B now knows feature_flag = %s (propagated from A)", val)
	val, _ = nodes[1].Get("max_connections")
	log.Info("Node-B now has max_connections = %s (A's version won)", val)

	logger.Section("SCENARIO: Multi-hop Propagation")
	log.Info("Node-B gossips with Node-C (spreading A's data), Node-C with Node-D")
	nodes[1].GossipTo(nodes[2])
	nodes[2].GossipTo(nodes[3])

	// Verify end-to-end propagation
	val, _ = nodes[3].Get("feature_flag")
	log.Info("Node-D knows feature_flag = %s (propagated through B->C->D)", val)

	logger.Section("SCENARIO: Full Mesh Gossip for Convergence")
	log.Info("Running full gossip round across all nodes for convergence")
	meshStart := time.Now()
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			nodes[i].GossipTo(nodes[j])
		}
	}
	log.Info("Full mesh gossip completed in %v", time.Since(meshStart))

	// Final stats summary
	logger.Section("FINAL STATS")
	for _, node := range nodes {
		logger.KeyValue("node_"+node.ID, len(node.state))
	}
	logger.KeyValue("total_nodes", len(nodes))
	totalKeys := 0
	for _, node := range nodes {
		totalKeys += len(node.state)
	}
	logger.KeyValue("converged_state_size", len(nodes[0].state))
	// Verify convergence
	baseFeatureFlag, _ := nodes[0].Get("feature_flag")
	baseMaxConn, _ := nodes[0].Get("max_connections")
	logger.KeyValue("feature_flag", baseFeatureFlag)
	logger.KeyValue("max_connections", baseMaxConn)
	log.Info("Gossip protocol demo completed successfully")
}
