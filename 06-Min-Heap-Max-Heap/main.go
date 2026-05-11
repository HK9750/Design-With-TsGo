// Package main demonstrates a generic Binary Heap supporting both min-heap and max-heap
// configurations via a custom priority comparator. Heapify runs in O(n), all other
// operations in O(log n). Suitable for a priority job scheduler.
package main

import (
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// BinaryHeap is a generic binary heap that can be configured as a min-heap or max-heap
// by providing a `higherPriority` comparator function. It maintains the heap invariant
// through sift-up on insert and sift-down on extract.
type BinaryHeap[T any] struct {
	data           []T
	higherPriority func(a, b T) bool
}

// NewBinaryHeap creates a new binary heap with the given priority comparator and initial values.
// The initial values are heapified in O(n) time using Floyd's algorithm.
func NewBinaryHeap[T any](higherPriority func(a, b T) bool, values ...T) *BinaryHeap[T] {
	heap := &BinaryHeap[T]{data: append([]T(nil), values...), higherPriority: higherPriority}
	log.Debug("Created binary heap (size=%d)", len(values))
	for i := len(heap.data)/2 - 1; i >= 0; i-- {
		heap.siftDown(i)
	}
	return heap
}

// Size returns the number of elements in the heap. Time complexity: O(1).
func (h *BinaryHeap[T]) Size() int {
	return len(h.data)
}

// Peek returns the top element without removing it. Returns the element and true
// if the heap is non-empty, or the zero value and false otherwise. Time complexity: O(1).
func (h *BinaryHeap[T]) Peek() (T, bool) {
	var zero T
	if len(h.data) == 0 {
		log.Debug("BinaryHeap peek on empty heap")
		return zero, false
	}
	return h.data[0], true
}

// Insert adds a value to the heap and restores the heap invariant in O(log n) time.
func (h *BinaryHeap[T]) Insert(value T) {
	h.data = append(h.data, value)
	h.siftUp(len(h.data) - 1)
	log.Debug("BinaryHeap insert (size=%d)", len(h.data))
}

// Extract removes and returns the top element from the heap. Returns the element and true
// if the heap is non-empty, or the zero value and false otherwise. Time complexity: O(log n).
func (h *BinaryHeap[T]) Extract() (T, bool) {
	var zero T
	if len(h.data) == 0 {
		log.Warn("BinaryHeap extract on empty heap")
		return zero, false
	}
	top := h.data[0]
	last := h.data[len(h.data)-1]
	h.data = h.data[:len(h.data)-1]
	if len(h.data) > 0 {
		h.data[0] = last
		h.siftDown(0)
	}
	log.Debug("BinaryHeap extract (value=%v, remaining=%d)", top, len(h.data))
	return top, true
}

// siftUp moves an element up the heap until the heap invariant is restored.
// Time complexity: O(log n).
func (h *BinaryHeap[T]) siftUp(index int) {
	for index > 0 {
		parent := (index - 1) / 2
		if !h.higherPriority(h.data[index], h.data[parent]) {
			return
		}
		h.data[index], h.data[parent] = h.data[parent], h.data[index]
		index = parent
	}
}

// siftDown moves an element down the heap until the heap invariant is restored.
// Time complexity: O(log n).
func (h *BinaryHeap[T]) siftDown(index int) {
	for {
		left, right, best := index*2+1, index*2+2, index
		if left < len(h.data) && h.higherPriority(h.data[left], h.data[best]) {
			best = left
		}
		if right < len(h.data) && h.higherPriority(h.data[right], h.data[best]) {
			best = right
		}
		if best == index {
			return
		}
		h.data[index], h.data[best] = h.data[best], h.data[index]
		index = best
	}
}

// main demonstrates both min-heap and max-heap configurations as a priority job
// scheduler for a worker system, where jobs have different priority levels.
func main() {
	defer log.Operation("main", "Running Binary Heap demo")()

	logger.Section("Binary Heap — Priority Job Scheduler")

	logger.Section("Min-Heap: Shortest Job First (SJF) Scheduler")
	minHeap := NewBinaryHeap(func(a, b int) bool { return a < b }, 5, 1, 3, 8, 2)
	log.Info("Initial heap (min-heap for job durations): [5, 1, 3, 8, 2]")
	top, _ := minHeap.Peek()
	log.Info("Peek shortest job: %d ms", top)
	minHeap.Insert(2)
	log.Info("Inserted job (duration=2ms)")

	log.Info("Processing jobs in shortest-first order:")
	for minHeap.Size() > 0 {
		job, _ := minHeap.Extract()
		log.Info("  Processing job: %d ms", job)
	}

	logger.Section("Max-Heap: Highest Priority First Scheduler")
	maxHeap := NewBinaryHeap(func(a, b int) bool { return a > b },
		3, 1, 4, 1, 5, 9, 2, 6,
	)
	top, _ = maxHeap.Peek()
	log.Info("Peek highest priority job: p=%d", top)
	log.Info("Processing jobs in priority order:")
	for maxHeap.Size() > 0 {
		job, _ := maxHeap.Extract()
		log.Info("  Processing job priority: %d", job)
	}

	logger.Section("Bulk scheduling stress test — 10,000 jobs")
	start := time.Now()
	bulkHeap := NewBinaryHeap(func(a, b int) bool { return a < b })
	for i := 0; i < 10000; i++ {
		bulkHeap.Insert(i%1000 + 1)
	}
	log.Info("Inserted 10,000 jobs in %v", time.Since(start))

	start = time.Now()
	for bulkHeap.Size() > 0 {
		bulkHeap.Extract()
	}
	log.Info("Processed all 10,000 jobs in %v", time.Since(start))

	logger.Section("Final Stats")
	log.Info("All operations completed — O(log n) insert/extract, O(n) heapify")
}
