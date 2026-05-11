package main

import (
	"design-with-tsgo/pkg/logger"
	"os"
	"time"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// GCounter is a Grow-Only Counter CRDT. Each node maintains its own
// per-node count, and the total value is the sum of all node counts.
// Merges take the max per node. Suitable for scenarios like tracking
// page views, votes, or any monotonically increasing metric.
type GCounter struct{ counts map[string]int }

// NewGCounter creates a new GCounter with all node counts initialized to zero.
// Time complexity: O(1).
func NewGCounter() *GCounter {
	defer log.Operation("NewGCounter", "Creating G-Counter")()
	return &GCounter{counts: make(map[string]int)}
}

// Increment adds the given amount to the named node's counter.
// The amount must be non-negative (grow-only property).
// Time complexity: O(1).
func (c *GCounter) Increment(node string, amount int) {
	if amount < 0 {
		log.Error("GCounter cannot decrement: node=%s amount=%d", node, amount)
		os.Exit(1)
	}
	c.counts[node] += amount
	log.Debug("GCounter: %s += %d (now %d)", node, amount, c.counts[node])
}

// Merge incorporates another GCounter's counts by taking the maximum
// per-node count. This ensures idempotent, commutative merging.
// Time complexity: O(m) where m is the number of nodes in other.
func (c *GCounter) Merge(other *GCounter) {
	defer log.Operation("Merge", "Merging G-Counters")()
	for node, count := range other.counts {
		if count > c.counts[node] {
			log.Debug("GCounter merge: %s: %d -> %d", node, c.counts[node], count)
			c.counts[node] = count
		}
	}
}

// Value returns the total count by summing all per-node counts.
// Time complexity: O(n) where n is the number of distinct nodes.
func (c *GCounter) Value() int {
	total := 0
	for _, count := range c.counts {
		total += count
	}
	return total
}

// PNCounter is a Positive-Negative Counter CRDT that supports both
// increments and decrements. It uses two internal GCounters: one for
// positive increments and one for negative (decrement) amounts.
// The net value is positive.Value() - negative.Value().
type PNCounter struct {
	positive *GCounter
	negative *GCounter
}

// NewPNCounter creates a new PN-Counter with both internal counters
// initialized to zero. Time complexity: O(1).
func NewPNCounter() *PNCounter {
	defer log.Operation("NewPNCounter", "Creating PN-Counter")()
	return &PNCounter{positive: NewGCounter(), negative: NewGCounter()}
}

// Increment adds the given amount to the node's positive counter.
// Time complexity: O(1).
func (c *PNCounter) Increment(node string, amount int) {
	c.positive.Increment(node, amount)
	log.Debug("PNCounter: %s incremented by %d", node, amount)
}

// Decrement records a decrement by adding to the node's negative counter.
// Time complexity: O(1).
func (c *PNCounter) Decrement(node string, amount int) {
	c.negative.Increment(node, amount)
	log.Debug("PNCounter: %s decremented by %d", node, amount)
}

// Merge combines another PN-Counter by merging both internal counters.
// Time complexity: O(m + n) where m, n are node counts in each sub-counter.
func (c *PNCounter) Merge(other *PNCounter) {
	defer log.Operation("Merge", "Merging PN-Counters")()
	c.positive.Merge(other.positive)
	c.negative.Merge(other.negative)
}

// Value returns the net count: sum(positive) - sum(negative).
// Time complexity: O(n + m) where n, m are distinct nodes in each counter.
func (c *PNCounter) Value() int {
	return c.positive.Value() - c.negative.Value()
}

func main() {
	defer log.Operation("main", "Running CRDT Counters demo")()

	logger.Section("CRDT COUNTERS — Production Scenario")

	// Simulate a distributed analytics system tracking user actions
	log.Info("Initializing PN-Counters for 3 edge servers")
	start := time.Now()
	edge1 := NewPNCounter()
	edge2 := NewPNCounter()
	edge3 := NewPNCounter()
	log.Info("Counters initialized in %v", time.Since(start))

	logger.Section("SCENARIO: Concurrent Page Views and Bounces")
	log.Info("Edge-1 records 100 page views and 10 bounces")
	edge1.Increment("edge-1", 100)
	edge1.Decrement("edge-1", 10)

	log.Info("Edge-2 records 50 page views and 5 bounces")
	edge2.Increment("edge-2", 50)
	edge2.Decrement("edge-2", 5)

	log.Info("Edge-3 records 200 page views and 20 bounces")
	edge3.Increment("edge-3", 200)
	edge3.Decrement("edge-3", 20)

	logger.Section("SCENARIO: Gossip-Based Merge for Eventual Consistency")
	log.Info("Edge-1 gossips with Edge-2")
	edge1.Merge(edge2)
	logger.KeyValue("edge1_after_merge_with_2", edge1.Value())

	log.Info("Edge-2 gossips with Edge-3")
	edge2.Merge(edge3)
	logger.KeyValue("edge2_after_merge_with_3", edge2.Value())

	log.Info("Edge-1 gossips with Edge-2 again (full convergence)")
	edge1.Merge(edge2)

	logger.Section("SCENARIO: Independent Increments After Merge")
	log.Info("Edge-1 records 25 additional page views")
	edge1.Increment("edge-1", 25)
	log.Info("Edge-3 records 30 page views and 3 bounces")
	edge3.Increment("edge-3", 30)
	edge3.Decrement("edge-3", 3)

	logger.Section("SCENARIO: Final Merge for Global Totals")
	log.Info("Running final merge across all edges")
	edge1.Merge(edge2)
	edge1.Merge(edge3)

	// Final stats summary
	logger.Section("FINAL STATS")
	logger.KeyValue("edge_1_value", edge1.Value())
	logger.KeyValue("edge_2_value", edge2.Value())
	logger.KeyValue("edge_3_value", edge3.Value())
	expectedTotal := (100 + 50 + 200 + 25 + 30) - (10 + 5 + 20 + 3)
	logger.KeyValue("expected_total", expectedTotal)
	logger.KeyValue("gcounter_test", NewGCounter())
	gc := NewGCounter()
	gc.Increment("a", 5)
	gc.Increment("b", 3)
	other := NewGCounter()
	other.Increment("a", 2)
	other.Increment("c", 7)
	gc.Merge(other)
	logger.KeyValue("gc_merge_test_value", gc.Value())
	log.Info("CRDT counters demo completed successfully")
}
