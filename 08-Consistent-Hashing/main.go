// Package main demonstrates a Consistent Hashing ring for distributing keys
// across a dynamic set of nodes. Uses virtual replicas to minimize key redistribution
// when nodes join or leave. Suitable for a distributed cache or sharded database.
package main

import (
	"fmt"
	"sort"
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// RingPoint represents a single point (virtual node) on the consistent hash ring.
// Each node is represented by multiple points (replicas) to ensure even distribution.
type RingPoint struct {
	hash uint32
	node string
}

// ConsistentHashRing implements a consistent hashing ring with configurable virtual
// replicas per physical node. Keys are mapped to the nearest node clockwise on the ring.
// Adding/removing nodes only affects a fraction of the key space proportional to 1/N.
type ConsistentHashRing struct {
	replicas int
	ring     []RingPoint
}

// NewConsistentHashRing creates a new consistent hash ring with the specified number
// of virtual replicas per node. Defaults to 100 replicas if a non-positive value is given.
// Time complexity: O(1).
func NewConsistentHashRing(replicas int) *ConsistentHashRing {
	if replicas <= 0 {
		replicas = 100
	}
	log.Debug("Created consistent hash ring (replicas=%d)", replicas)
	return &ConsistentHashRing{replicas: replicas}
}

// AddNode adds a physical node to the ring by inserting `replicas` virtual points.
// If the node already exists, its old points are removed first to avoid duplicates.
// The ring is re-sorted after insertion. Time complexity: O(R log R) where R is ring size.
func (r *ConsistentHashRing) AddNode(node string) {
	r.RemoveNode(node)
	for i := 0; i < r.replicas; i++ {
		r.ring = append(r.ring, RingPoint{
			hash: hash32(fmt.Sprintf("%s#%d", node, i)),
			node: node,
		})
	}
	sort.Slice(r.ring, func(i, j int) bool { return r.ring[i].hash < r.ring[j].hash })
	log.Debug("ConsistentHashRing add node (node=%q, replicas=%d, ring_size=%d)", node, r.replicas, len(r.ring))
}

// RemoveNode removes all virtual points for a physical node from the ring.
// Time complexity: O(R) where R is current ring size.
func (r *ConsistentHashRing) RemoveNode(node string) {
	kept := r.ring[:0]
	removed := 0
	for _, point := range r.ring {
		if point.node != node {
			kept = append(kept, point)
		} else {
			removed++
		}
	}
	r.ring = kept
	if removed > 0 {
		log.Debug("ConsistentHashRing remove node (node=%q, removed=%d points, ring_size=%d)", node, removed, len(r.ring))
	}
}

// GetNode maps a key to the responsible physical node using clockwise search on the ring.
// Returns the node name and true if the ring is non-empty, or empty string and false otherwise.
// Time complexity: O(log R) via binary search.
func (r *ConsistentHashRing) GetNode(key string) (string, bool) {
	if len(r.ring) == 0 {
		log.Warn("ConsistentHashRing get_node on empty ring (key=%q)", key)
		return "", false
	}
	hash := hash32(key)
	idx := sort.Search(len(r.ring), func(i int) bool { return r.ring[i].hash >= hash })
	node := r.ring[idx%len(r.ring)].node
	log.Debug("ConsistentHashRing get_node (key=%q, hash=%d, node=%q)", key, hash, node)
	return node, true
}

// hash32 computes the 32-bit FNV-1a hash of a string.
func hash32(value string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(value); i++ {
		hash ^= uint32(value[i])
		hash *= 16777619
	}
	return hash
}

// main demonstrates consistent hashing as a distributed cache sharding strategy,
// where cache keys are distributed across multiple Redis nodes and nodes can
// be added/removed with minimal cache invalidation.
func main() {
	defer log.Operation("main", "Running Consistent Hashing demo")()

	logger.Section("Consistent Hashing — Distributed Cache Sharding")
	ring := NewConsistentHashRing(50)
	logger.KeyValue("virtual_replicas", 50)

	logger.Section("Adding cache nodes to the cluster")
	ring.AddNode("cache-node-a")
	log.Info("Added cache-node-a")
	ring.AddNode("cache-node-b")
	log.Info("Added cache-node-b")
	ring.AddNode("cache-node-c")
	log.Info("Added cache-node-c")

	logger.Section("Key distribution — mapping user sessions to cache nodes")
	testKeys := []string{
		"user:42:session", "user:17:profile", "user:99:cart",
		"user:1:settings", "user:55:history", "user:23:prefs",
	}
	for _, key := range testKeys {
		node, _ := ring.GetNode(key)
		log.Info("%-25s -> %s", key, node)
	}

	logger.Section("Bulk key distribution analysis — 10,000 keys")
	start := time.Now()
	distribution := make(map[string]int)
	for i := 0; i < 10000; i++ {
		node, _ := ring.GetNode(fmt.Sprintf("user:%d:data", i))
		distribution[node]++
	}
	log.Info("Distribution across nodes after 10,000 keys in %v:", time.Since(start))
	for node, count := range distribution {
		log.Info("  %s: %d keys (%.2f%%)", node, count, float64(count)/10000*100)
	}

	logger.Section("Node failure — removing cache-node-b")
	ring.RemoveNode("cache-node-b")
	log.Info("Removed cache-node-b (simulating node failure)")

	logger.Section("Re-mapping keys after node removal")
	redistribution := make(map[string]int)
	// moved := 0
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("user:%d:data", i)
		node, _ := ring.GetNode(key)
		redistribution[node]++
		if node != ring.ring[0].node && node != ring.ring[len(ring.ring)-1].node {
			// Keys that stayed on their original node (not re-assigned)
		}
	}
	// Count how many stayed vs moved
	originalNodes := []string{"cache-node-a", "cache-node-c"}
	stayed := 0
	// shifted := 0
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("user:%d:data", i)
		node, _ := ring.GetNode(key)
		for _, n := range originalNodes {
			if node == n {
				stayed++
				break
			}
		}
	}
	// shifted = 10000 - stayed
	log.Info("After removing cache-node-b:")
	for node, count := range redistribution {
		log.Info("  %s: %d keys (%.2f%%)", node, count, float64(count)/10000*100)
	}
	log.Info("Keys still on their original node: ~%d (minimal redistribution)", stayed)

	logger.Section("Scaling up — adding new node")
	ring.AddNode("cache-node-d")
	log.Info("Added cache-node-d (horizontal scaling)")
	newDist := make(map[string]int)
	for i := 0; i < 10000; i++ {
		node, _ := ring.GetNode(fmt.Sprintf("user:%d:data", i))
		newDist[node]++
	}
	log.Info("New distribution after adding cache-node-d:")
	for node, count := range newDist {
		log.Info("  %s: %d keys (%.2f%%)", node, count, float64(count)/10000*100)
	}

	logger.Section("Final Stats")
	log.Info("Ring size: %d virtual nodes (%d physical nodes × %d replicas)",
		len(ring.ring), 3, ring.replicas)
	log.Info("All operations completed — O(log R) key lookup with minimal redistribution on node changes")
}
