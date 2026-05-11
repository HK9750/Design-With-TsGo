// Package main demonstrates a generic Skip List — a probabilistic alternative to balanced trees
// providing O(log n) average search, insert, and delete operations.
// Suitable for in-memory sorted indexes in a database engine.
package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// SkipNode represents a node in the skip list with a key, value, and forward pointers
// at multiple levels (the "express lanes").
type SkipNode[V any] struct {
	key     int
	value   V
	forward []*SkipNode[V]
}

// SkipList implements a generic skip list data structure. It maintains a sorted
// linked list with multiple levels of "express" pointers for probabilistic O(log n)
// search, insert, and delete operations.
type SkipList[V any] struct {
	head        *SkipNode[V]
	level       int
	maxLevel    int
	probability float64
}

// NewSkipList creates a new skip list with the specified maximum level and promotion probability.
// Defaults to maxLevel=16 and probability=0.5 if invalid values are provided.
// Time complexity: O(maxLevel).
func NewSkipList[V any](maxLevel int, probability float64) *SkipList[V] {
	if maxLevel <= 0 {
		maxLevel = 16
	}
	if probability <= 0 || probability >= 1 {
		probability = 0.5
	}
	log.Debug("Created skip list (max_level=%d, probability=%.2f)", maxLevel, probability)
	return &SkipList[V]{
		head:        &SkipNode[V]{key: -1 << 62, forward: make([]*SkipNode[V], maxLevel+1)},
		maxLevel:    maxLevel,
		probability: probability,
	}
}

// Search looks up a key in the skip list. Returns the associated value and true if found,
// or the zero value and false otherwise. Time complexity: O(log n) average.
func (sl *SkipList[V]) Search(key int) (V, bool) {
	var zero V
	node := sl.head
	for i := sl.level; i >= 0; i-- {
		for node.forward[i] != nil && node.forward[i].key < key {
			node = node.forward[i]
		}
	}
	node = node.forward[0]
	if node != nil && node.key == key {
		log.Debug("SkipList search hit (key=%d)", key)
		return node.value, true
	}
	log.Debug("SkipList search miss (key=%d)", key)
	return zero, false
}

// Insert adds or updates a key-value pair in the skip list. If the key already exists,
// its value is updated. Time complexity: O(log n) average.
func (sl *SkipList[V]) Insert(key int, value V) {
	update := make([]*SkipNode[V], sl.maxLevel+1)
	node := sl.head
	for i := sl.level; i >= 0; i-- {
		for node.forward[i] != nil && node.forward[i].key < key {
			node = node.forward[i]
		}
		update[i] = node
	}
	if existing := node.forward[0]; existing != nil && existing.key == key {
		log.Debug("SkipList update (key=%d)", key)
		existing.value = value
		return
	}

	newLevel := sl.randomLevel()
	if newLevel > sl.level {
		for i := sl.level + 1; i <= newLevel; i++ {
			update[i] = sl.head
		}
		sl.level = newLevel
	}
	created := &SkipNode[V]{key: key, value: value, forward: make([]*SkipNode[V], newLevel+1)}
	for i := 0; i <= newLevel; i++ {
		created.forward[i] = update[i].forward[i]
		update[i].forward[i] = created
	}
	log.Debug("SkipList insert (key=%d, level=%d/%d)", key, newLevel+1, sl.level+1)
}

// Delete removes a key from the skip list. Returns true if the key was found and removed,
// or false if the key was not present. Time complexity: O(log n) average.
func (sl *SkipList[V]) Delete(key int) bool {
	update := make([]*SkipNode[V], sl.maxLevel+1)
	node := sl.head
	for i := sl.level; i >= 0; i-- {
		for node.forward[i] != nil && node.forward[i].key < key {
			node = node.forward[i]
		}
		update[i] = node
	}
	target := node.forward[0]
	if target == nil || target.key != key {
		log.Warn("SkipList delete miss (key=%d)", key)
		return false
	}
	for i := 0; i <= sl.level && update[i].forward[i] == target; i++ {
		if i < len(target.forward) {
			update[i].forward[i] = target.forward[i]
		} else {
			update[i].forward[i] = nil
		}
	}
	for sl.level > 0 && sl.head.forward[sl.level] == nil {
		sl.level--
	}
	log.Debug("SkipList delete (key=%d, new_level=%d)", key, sl.level+1)
	return true
}

// randomLevel generates a random level for a new node using geometric distribution.
// Each level above 0 has probability `probability` of being promoted.
func (sl *SkipList[V]) randomLevel() int {
	level := 0
	for level < sl.maxLevel && rand.Float64() < sl.probability {
		level++
	}
	return level
}

// main demonstrates the skip list as an in-memory sorted index for a database,
// maintaining ordered records with fast range queries and point lookups.
func main() {
	defer log.Operation("main", "Running Skip List demo")()

	logger.Section("Skip List — In-Memory Sorted Index for Database Engine")
	list := NewSkipList[string](16, 0.5)
	logger.KeyValue("max_level", 16)
	logger.KeyValue("promotion_probability", 0.5)

	logger.Section("Building sorted index from database records")
	records := map[int]string{
		3:   "User: Charlie (id=3)",
		1:   "User: Alice (id=1)",
		5:   "User: Eve (id=5)",
		2:   "User: Bob (id=2)",
		4:   "User: Diana (id=4)",
		10:  "User: Jack (id=10)",
		7:   "User: Grace (id=7)",
		100: "User: Zack (id=100)",
	}
	for k, v := range records {
		list.Insert(k, v)
		log.Debug("Indexed record: key=%d -> %s", k, v)
	}
	log.Info("Indexed %d records", len(records))

	logger.Section("Point lookups")
	if value, ok := list.Search(2); ok {
		log.Info("Lookup key=2: %s", value)
	} else {
		log.Warn("Lookup key=2: not found")
	}
	if value, ok := list.Search(7); ok {
		log.Info("Lookup key=7: %s", value)
	}
	if _, ok := list.Search(99); !ok {
		log.Warn("Lookup key=99: not found (as expected)")
	}

	logger.Section("Update existing record")
	list.Insert(3, "User: Charlie (id=3) — UPDATED EMAIL")
	if value, ok := list.Search(3); ok {
		log.Info("Updated record key=3: %s", value)
	}

	logger.Section("Delete expired record")
	list.Delete(10)
	if _, ok := list.Search(10); !ok {
		log.Info("Record key=10 successfully deleted")
	}

	logger.Section("Bulk insert performance test")
	start := time.Now()
	for i := 200; i < 5000; i++ {
		list.Insert(i, fmt.Sprintf("record_%d", i))
	}
	log.Info("Bulk inserted 4,800 records in %v", time.Since(start))

	logger.Section("Bulk lookup performance")
	start = time.Now()
	lookupCount := 2000
	found := 0
	for i := 0; i < lookupCount; i++ {
		if _, ok := list.Search(100 + i); ok {
			found++
		}
	}
	log.Info("Searched %d keys, found %d in %v", lookupCount, found, time.Since(start))

	logger.Section("Final Stats")
	log.Info("Current skip list level: %d", list.level+1)
	log.Info("All operations completed — O(log n) average for search, insert, delete")
}
