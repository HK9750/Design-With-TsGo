package main

import (
	"os"
	"sync"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// BoundedQueue is a fixed-capacity thread-safe producer-consumer queue.
// Enqueue blocks when full; Dequeue blocks when empty. Built on a buffered
// channel. Uses generics to support any element type T.
type BoundedQueue[T any] struct{ ch chan T }

// NewBoundedQueue creates a bounded queue with the given capacity. Panics if
// capacity <= 0. Time complexity: O(1).
func NewBoundedQueue[T any](capacity int) *BoundedQueue[T] {
	if capacity <= 0 {
		log.Error("capacity must be positive, got %d", capacity)
		os.Exit(1)
	}
	log.Debug("Created bounded queue with capacity %d", capacity)
	return &BoundedQueue[T]{ch: make(chan T, capacity)}
}

// Enqueue adds a value to the queue, blocking if the queue is full.
// Time complexity: O(1).
func (q *BoundedQueue[T]) Enqueue(value T) {
	q.ch <- value
	log.Debug("Enqueued: %v (len=%d cap=%d)", value, len(q.ch), cap(q.ch))
}

// Dequeue removes and returns the oldest value from the queue, blocking if empty.
// Time complexity: O(1).
func (q *BoundedQueue[T]) Dequeue() T {
	value := <-q.ch
	log.Debug("Dequeued: %v (len=%d cap=%d)", value, len(q.ch), cap(q.ch))
	return value
}

func main() {
	defer log.Operation("main", "Running Producer-Consumer demo")()

	logger.Section("Scenario: Real-Time Event Processing Pipeline")
	log.Info("Simulating a log aggregation service: producer -> queue -> consumer")

	capacity := 3
	totalEvents := 12
	queue := NewBoundedQueue[string](capacity)

	var wg sync.WaitGroup
	produced := 0
	consumed := 0
	var mu sync.Mutex

	start := time.Now()

	// Producer goroutines
	for p := 0; p < 2; p++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()
			for i := 0; i < totalEvents/2; i++ {
				event := "event_" + string(rune('A'+producerID)) + "_" + string(rune('0'+i))
				log.Step("Producer %d enqueuing %s", producerID, event)
				queue.Enqueue(event)
				mu.Lock()
				produced++
				mu.Unlock()
				time.Sleep(20 * time.Millisecond)
			}
			log.Debug("Producer %d finished", producerID)
		}(p)
	}

	// Consumer goroutines
	for c := 0; c < 3; c++ {
		wg.Add(1)
		go func(consumerID int) {
			defer wg.Done()
			for range totalEvents / 3 {
				event := queue.Dequeue()
				log.Step("Consumer %d processing %s", consumerID, event)
				mu.Lock()
				consumed++
				mu.Unlock()
				time.Sleep(25 * time.Millisecond)
			}
			log.Debug("Consumer %d finished", consumerID)
		}(c)
	}

	wg.Wait()
	elapsed := time.Since(start)

	logger.Section("Final Stats Summary")
	logger.KeyValue("queueCapacity", capacity)
	logger.KeyValue("totalEvents", totalEvents)
	logger.KeyValue("eventsProduced", produced)
	logger.KeyValue("eventsConsumed", consumed)
	logger.KeyValue("totalTime", elapsed)
	log.Info("Event pipeline completed in %v (%d produced, %d consumed)", elapsed, produced, consumed)
}
