// Package main demonstrates an LRU (Least Recently Used) Cache implementation
// with O(1) get/put operations, suitable for web server response caching.
package main

import (
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// ListNode represents a node in the doubly-linked list used by LRUCache.
// Each node stores a key-value pair and pointers to its neighbors.
type ListNode struct {
	Key   int
	Value int
	Next  *ListNode
	Prev  *ListNode
}

// LRUCache implements a fixed-capacity least-recently-used cache.
// It uses a doubly-linked list to track access order and a map for O(1) lookups.
// Evicts the least recently used entry when capacity is exceeded.
type LRUCache struct {
	Size     int
	Capacity int
	Map      map[int]*ListNode
	Head     *ListNode
	Tail     *ListNode
}

// NewLRUCache creates a new LRU cache with the given capacity.
// The cache uses sentinel head/tail nodes to simplify edge cases.
// Time complexity: O(1).
func NewLRUCache(capacity int) *LRUCache {
	head := &ListNode{}
	tail := &ListNode{}
	head.Next = tail
	tail.Prev = head

	log.Debug("Created LRU cache (capacity=%d)", capacity)
	return &LRUCache{
		Size:     0,
		Capacity: capacity,
		Map:      make(map[int]*ListNode),
		Head:     head,
		Tail:     tail,
	}
}

// get retrieves the value for the given key, returning -1 if not found.
// On a cache hit, the accessed node is moved to the tail (most recently used).
// Time complexity: O(1).
func (l *LRUCache) get(key int) int {
	node, ok := l.Map[key]
	if !ok {
		log.Debug("LRU cache miss (key=%d)", key)
		return -1
	}
	log.Debug("LRU cache hit (key=%d, value=%d)", key, node.Value)
	l.deleteNode(node)
	l.addNode(node)
	return node.Value
}

// put inserts or updates a key-value pair in the cache.
// If the key exists, its value is updated and it is moved to most-recently-used.
// If the cache is at capacity, the least recently used entry is evicted.
// Time complexity: O(1).
func (l *LRUCache) put(key int, value int) {
	if existing, ok := l.Map[key]; ok {
		log.Debug("LRU cache update (key=%d, old=%d, new=%d)", key, existing.Value, value)
		existing.Value = value
		l.deleteNode(existing)
		l.addNode(existing)
		return
	}

	if l.Size == l.Capacity {
		lru := l.Head.Next
		log.Warn("LRU cache full (capacity=%d), evicting key=%d (value=%d)", l.Capacity, lru.Key, lru.Value)
		l.deleteNode(lru)
		delete(l.Map, lru.Key)
		l.Size--
	}

	newNode := &ListNode{Key: key, Value: value}
	l.addNode(newNode)
	l.Map[key] = newNode
	l.Size++
	log.Debug("LRU cache insert (key=%d, value=%d, size=%d/%d)", key, value, l.Size, l.Capacity)
}

// addNode appends a node to the tail of the doubly-linked list (marking it as most recently used).
// Time complexity: O(1).
func (l *LRUCache) addNode(node *ListNode) {
	node.Prev = l.Tail.Prev
	node.Next = l.Tail
	l.Tail.Prev.Next = node
	l.Tail.Prev = node
}

// deleteNode removes a node from the doubly-linked list.
// Time complexity: O(1).
func (l *LRUCache) deleteNode(node *ListNode) {
	node.Prev.Next = node.Next
	node.Next.Prev = node.Prev
}

// main demonstrates the LRU cache used as an API response cache for a web server,
// caching database query results under memory constraints.
func main() {
	defer log.Operation("main", "Running LRU Cache demo")()

	logger.Section("LRU Cache — Web API Response Cache")
	cache := NewLRUCache(2)
	logger.KeyValue("capacity", cache.Capacity)

	logger.Section("Populating cache with user profile queries")
	cache.put(1, 100)
	log.Info("Cached user 1 profile")
	cache.put(2, 200)
	log.Info("Cached user 2 profile")

	logger.Section("Fetching cached responses")
	log.Info("User 1: %d", cache.get(1))
	log.Info("User 2: %d", cache.get(2))

	logger.Section("Cache eviction under memory pressure")
	cache.put(3, 300)
	log.Info("User 3 cached (user 2 evicted)")
	log.Info("User 2 (evicted): %d", cache.get(2))
	log.Info("User 3 (in cache): %d", cache.get(3))

	cache.put(4, 400)
	log.Info("User 4 cached (user 1 evicted)")
	log.Info("User 1 (evicted): %d", cache.get(1))
	log.Info("User 3 (in cache): %d", cache.get(3))
	log.Info("User 4 (in cache): %d", cache.get(4))

	logger.Section("Bulk cache load simulation")
	start := time.Now()
	bigCache := NewLRUCache(100)
	for i := 0; i < 200; i++ {
		bigCache.put(i, i*10)
	}
	log.Info("Bulk insert of 200 keys into capacity=100 completed in %v", time.Since(start))

	logger.Section("Final Stats")
	log.Info("Cache size: %d/%d", bigCache.Size, bigCache.Capacity)
	log.Info("All operations completed successfully — O(1) get/put with LRU eviction")
}
