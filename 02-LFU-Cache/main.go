// Package main demonstrates an LFU (Least Frequently Used) Cache implementation
// with O(1) get/put operations, suitable for content delivery network edge caching.
package main

import (
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// ListNode represents a node in the doubly-linked list used within each frequency bucket.
// Each node stores a key-value pair, its access frequency, and list pointers.
type ListNode struct {
	Key   int
	Value int
	Freq  int
	Prev  *ListNode
	Next  *ListNode
}

// DoublyLinkedList is a standard doubly-linked list used to group nodes by frequency.
// Supports add-to-head, remove-node, and remove-last operations in O(1).
type DoublyLinkedList struct {
	Size int
	Head *ListNode
	Tail *ListNode
}

// NewDoublyLinkedList creates an empty doubly-linked list with sentinel head/tail nodes.
// Time complexity: O(1).
func NewDoublyLinkedList() *DoublyLinkedList {
	head := &ListNode{}
	tail := &ListNode{}
	head.Next = tail
	tail.Prev = head
	return &DoublyLinkedList{
		Size: 0,
		Head: head,
		Tail: tail,
	}
}

// addNode inserts a node at the head of the list. Time complexity: O(1).
func (dl *DoublyLinkedList) addNode(node *ListNode) {
	node.Prev = dl.Head
	node.Next = dl.Head.Next
	dl.Head.Next.Prev = node
	dl.Head.Next = node
	dl.Size++
}

// remove detaches a node from the list. Time complexity: O(1).
func (dl *DoublyLinkedList) remove(node *ListNode) {
	node.Prev.Next = node.Next
	node.Next.Prev = node.Prev
	dl.Size--
}

// removeLast removes and returns the last (least recently used) node in this frequency bucket.
// Returns nil if the list is empty. Time complexity: O(1).
func (dl *DoublyLinkedList) removeLast() *ListNode {
	if dl.Size == 0 {
		return nil
	}
	last := dl.Tail.Prev
	dl.remove(last)
	return last
}

// LFUCache implements a fixed-capacity least-frequently-used cache.
// It uses a key-to-node map and frequency-to-DLL map for O(1) operations.
// Evicts the least frequently used entry (ties broken by LRU within same frequency).
type LFUCache struct {
	Capacity   int
	Size       int
	MinFreq    int
	KeyToNodes map[int]*ListNode
	FreqToList map[int]*DoublyLinkedList
}

// NewLFUCache creates a new LFU cache with the specified capacity.
// Time complexity: O(1).
func NewLFUCache(capacity int) *LFUCache {
	log.Debug("Created LFU cache (capacity=%d)", capacity)
	return &LFUCache{
		Capacity:   capacity,
		Size:       0,
		MinFreq:    0,
		KeyToNodes: make(map[int]*ListNode),
		FreqToList: make(map[int]*DoublyLinkedList),
	}
}

// get retrieves the value for a key, returning -1 if not found.
// On a hit, the node's frequency is incremented. Time complexity: O(1).
func (lfu *LFUCache) get(key int) int {
	node, ok := lfu.KeyToNodes[key]
	if !ok {
		log.Debug("LFU cache miss (key=%d)", key)
		return -1
	}
	log.Debug("LFU cache hit (key=%d, value=%d, freq=%d)", key, node.Value, node.Freq)
	lfu.update(node)
	return node.Value
}

// put inserts or updates a key-value pair. If the cache is at capacity, the
// least frequently used entry is evicted. Time complexity: O(1).
func (lfu *LFUCache) put(key int, value int) {
	if lfu.Capacity == 0 {
		log.Warn("LFU cache put ignored (capacity=0)")
		return
	}

	node, ok := lfu.KeyToNodes[key]
	if ok {
		log.Debug("LFU cache update (key=%d, value=%d)", key, value)
		node.Value = value
		lfu.update(node)
		return
	}

	if lfu.Size == lfu.Capacity {
		list := lfu.FreqToList[lfu.MinFreq]
		removed := list.removeLast()
		log.Warn("LFU cache full, evicting key=%d (freq=%d)", removed.Key, removed.Freq)
		delete(lfu.KeyToNodes, removed.Key)
		lfu.Size--
	}

	newNode := &ListNode{Key: key, Value: value, Freq: 1}
	list, ok := lfu.FreqToList[1]
	if !ok {
		list = NewDoublyLinkedList()
		lfu.FreqToList[1] = list
	}
	list.addNode(newNode)
	lfu.KeyToNodes[key] = newNode
	lfu.MinFreq = 1
	lfu.Size++
	log.Debug("LFU cache insert (key=%d, value=%d, size=%d/%d)", key, value, lfu.Size, lfu.Capacity)
}

// update increments the frequency of a node and moves it to the appropriate frequency bucket.
// If the old frequency bucket becomes empty and was the minimum frequency, minFreq is incremented.
func (lfu *LFUCache) update(node *ListNode) {
	freq := node.Freq
	list := lfu.FreqToList[freq]
	list.remove(node)

	if freq == lfu.MinFreq && list.Size == 0 {
		lfu.MinFreq++
	}

	node.Freq++
	newList, ok := lfu.FreqToList[node.Freq]
	if !ok {
		newList = NewDoublyLinkedList()
		lfu.FreqToList[node.Freq] = newList
	}
	newList.addNode(node)
}

// main demonstrates the LFU cache as a CDN edge cache for video chunk popularity,
// where frequently accessed content stays cached and rarely accessed content is evicted.
func main() {
	defer log.Operation("main", "Running LFU Cache demo")()

	logger.Section("LFU Cache — CDN Edge Cache for Video Chunks")
	cache := NewLFUCache(3)
	logger.KeyValue("capacity", cache.Capacity)

	logger.Section("Initializing cache with popular video segments")
	cache.put(1, 10)
	log.Info("Cached video chunk 1 (popular intro)")
	cache.put(2, 20)
	log.Info("Cached video chunk 2")
	cache.put(3, 30)
	log.Info("Cached video chunk 3")

	logger.Section("Simulating access patterns — chunk 1 gets many hits")
	for i := 0; i < 5; i++ {
		cache.get(1)
	}
	log.Info("Chunk 1 accessed 5 times (freq=%d — hot cache entry)", 6)

	logger.Section("Eviction under popularity pressure")
	cache.put(4, 40)
	log.Info("Inserted chunk 4 (least frequent chunk evicted)")

	log.Info("Chunk 1 (hot, freq=6): %d", cache.get(1))
	log.Info("Chunk 2 (evicted): %d", cache.get(2))
	log.Info("Chunk 3 (in cache): %d", cache.get(3))
	log.Info("Chunk 4 (in cache): %d", cache.get(4))

	logger.Section("Bulk access pattern simulation")
	start := time.Now()
	for i := 0; i < 1000; i++ {
		cache.get(1)
		cache.get(3)
		cache.get(4)
	}
	log.Info("1000 access cycles (3 hot keys) completed in %v", time.Since(start))

	logger.Section("Final Stats")
	log.Info("Cache size: %d/%d", cache.Size, cache.Capacity)
	log.Info("Min frequency bucket: %d", cache.MinFreq)
	log.Info("All operations completed — O(1) get/put with LFU eviction")
}
