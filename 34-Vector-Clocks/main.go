package main

import (
	"design-with-tsgo/pkg/logger"
	"os"
	"time"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// VectorClock implements a vector clock for tracking causal relationships
// between events in a distributed system. Each node maintains its own
// counter, and clocks are merged to determine happens-before relationships.
type VectorClock struct{ clock map[string]int }

// NewVectorClock creates a new vector clock with all counters initialized to zero.
// Time complexity: O(1).
func NewVectorClock() *VectorClock {
	defer log.Operation("NewVectorClock", "Creating vector clock")()
	return &VectorClock{clock: make(map[string]int)}
}

// Tick increments the logical clock for the given node by 1.
// Should be called before every local event on that node.
// Time complexity: O(1).
func (vc *VectorClock) Tick(node string) {
	vc.clock[node]++
	log.Debug("Vector clock: %s ticked to %d", node, vc.clock[node])
}

// Merge combines another vector clock into this one by taking the
// maximum value for each node. Used when receiving a message from a peer.
// Time complexity: O(k) where k is the number of nodes in the other clock.
func (vc *VectorClock) Merge(other *VectorClock) {
	defer log.Operation("Merge", "Merging vector clocks")()
	for node, value := range other.clock {
		if value > vc.clock[node] {
			log.Debug("Merging node %s: %d -> %d", node, vc.clock[node], value)
			vc.clock[node] = value
		}
	}
}

// Compare determines the causal relationship between this clock and another.
// Returns one of: "before" (this happened-before other), "after" (this happened-after other),
// "concurrent" (unrelated events), or "equal" (same clock state).
// Time complexity: O(n + m) where n, m are the number of entries in each clock.
func (vc *VectorClock) Compare(other *VectorClock) string {
	less, greater := false, false
	seen := make(map[string]bool)
	for node := range vc.clock {
		seen[node] = true
	}
	for node := range other.clock {
		seen[node] = true
	}
	for node := range seen {
		a, b := vc.clock[node], other.clock[node]
		if a < b {
			less = true
		}
		if a > b {
			greater = true
		}
	}
	result := "equal"
	if less && greater {
		result = "concurrent"
	} else if less {
		result = "before"
	} else if greater {
		result = "after"
	}
	log.Debug("Vector compare result: %s (less=%v greater=%v)", result, less, greater)
	return result
}

func main() {
	defer log.Operation("main", "Running Vector Clocks demo")()

	logger.Section("VECTOR CLOCKS — Production Scenario")

	// Simulate a distributed key-value store with 3 replicas tracking causality
	log.Info("Initializing vector clocks for 3 replicas: replica-a, replica-b, replica-c")
	start := time.Now()
	a := NewVectorClock()
	b := NewVectorClock()
	c := NewVectorClock()
	log.Info("Clocks initialized in %v", time.Since(start))

	logger.Section("SCENARIO: Independent Events (Concurrent)")
	log.Info("Replica-A processes a local write: 'user:1' = 'Alice'")
	a.Tick("replica-a")
	log.Info("Replica-B processes a local write: 'user:2' = 'Bob'")
	b.Tick("replica-b")

	relation := a.Compare(b)
	log.Info("a.Compare(b) = %s (expected: concurrent — independent writes)", relation)

	logger.Section("SCENARIO: Causal Relationship (Happens-Before)")
	log.Info("Replica-A sends its state to Replica-C")
	c.Merge(a)
	log.Info("Replica-C processes a local write based on A's data")
	c.Tick("replica-c")

	relation = a.Compare(c)
	log.Info("a.Compare(c) = %s (expected: before — c has all of a's events plus more)", relation)

	relation = c.Compare(a)
	log.Info("c.Compare(a) = %s (expected: after — reverse perspective)", relation)

	logger.Section("SCENARIO: Equal Clocks After Full Sync")
	log.Info("Full synchronization: merging all replicas into one view")
	d := NewVectorClock()
	a.Tick("replica-a") // another tick after initial
	b.Tick("replica-b")
	d.Merge(a)
	d.Merge(b)
	d.Merge(c)

	e := NewVectorClock()
	e.Merge(a)
	e.Merge(b)
	e.Merge(c)

	relation = d.Compare(e)
	log.Info("d.Compare(e) = %s (expected: equal — same merged state)", relation)

	logger.Section("SCENARIO: Detecting Write Conflicts")
	log.Info("Replica-A writes 'key' = 'v1'")
	a.Tick("replica-a")
	log.Info("Replica-B writes 'key' = 'v2'")
	b.Tick("replica-b")

	relation = a.Compare(b)
	log.Info("a.Compare(b) = %s — conflict detected, needs application-level resolution", relation)

	// Final stats summary
	logger.Section("FINAL STATS")
	logger.KeyValue("clock_a_entries", len(a.clock))
	logger.KeyValue("clock_b_entries", len(b.clock))
	logger.KeyValue("clock_c_entries", len(c.clock))
	logger.KeyValue("clock_d_entries", len(d.clock))
	logger.KeyValue("clock_e_entries", len(e.clock))
	for node, v := range a.clock {
		logger.KeyValue("a_"+node, v)
	}
	for node, v := range b.clock {
		logger.KeyValue("b_"+node, v)
	}
	log.Info("Vector clocks demo completed successfully")
}
