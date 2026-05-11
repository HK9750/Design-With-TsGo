package main

import (
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// ChannelPool is a fixed-size pool of reusable channels. Useful when channels
// are expensive to create or when you need bounded concurrency with typed
// communication. Uses generics to support any element type T.
type ChannelPool[T any] struct {
	available chan chan T
	factory   func() chan T
}

// NewChannelPool creates a pool of 'size' channels, each created by the factory
// function. Panics if size <= 0. Time complexity: O(size).
func NewChannelPool[T any](size int, factory func() chan T) *ChannelPool[T] {
	if size <= 0 {
		log.Error("size must be positive, got %d", size)
		os.Exit(1)
	}
	pool := &ChannelPool[T]{available: make(chan chan T, size), factory: factory}
	for i := 0; i < size; i++ {
		ch := factory()
		pool.available <- ch
		log.Debug("Pre-allocated channel %d in pool", i)
	}
	log.Info("Channel pool created with %d channels", size)
	return pool
}

// Acquire takes a channel from the pool, blocking if none are available.
// Time complexity: O(1).
func (p *ChannelPool[T]) Acquire() chan T {
	ch := <-p.available
	log.Debug("Channel acquired from pool (remaining: %d)", len(p.available))
	return ch
}

// Release returns a channel to the pool so it can be reused.
// Time complexity: O(1).
func (p *ChannelPool[T]) Release(ch chan T) {
	p.available <- ch
	log.Debug("Channel returned to pool (available: %d)", len(p.available))
}

func main() {
	defer log.Operation("main", "Running Channel Pool demo")()

	logger.Section("Scenario: Message Router with Channel Pool")
	log.Info("Simulating a message broker with 3 pre-allocated channels")

	poolSize := 3
	pool := NewChannelPool(poolSize, func() chan string {
		return make(chan string, 1)
	})

	start := time.Now()

	// Route messages through the pool
	messages := []string{"order.created", "payment.received", "shipment.dispatched", "invoice.generated", "refund.processed"}
	received := make([]string, 0, len(messages))

	for _, msg := range messages {
		ch := pool.Acquire()
		go func(m string, c chan string) {
			c <- m
		}(msg, ch)

		received = append(received, <-ch)
		pool.Release(ch)
	}

	elapsed := time.Since(start)

	logger.Section("Routed Messages")
	for _, m := range received {
		log.Info("Delivered: %s", m)
	}

	logger.Section("Final Stats Summary")
	logger.KeyValue("poolSize", poolSize)
	logger.KeyValue("messagesRouted", len(messages))
	logger.KeyValue("totalTime", elapsed)
	log.Info("Message routing completed in %v", elapsed)
}
