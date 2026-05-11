package main

import (
	"os"
	"sync"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// Semaphore is a concurrency limiter that allows at most 'permits' goroutines
// to hold a resource simultaneously. Built on a buffered channel of empty structs.
type Semaphore struct{ permits chan struct{} }

// NewSemaphore creates a semaphore with the given number of permits. Panics if
// permits <= 0. Time complexity: O(1).
func NewSemaphore(permits int) *Semaphore {
	if permits <= 0 {
		log.Error("permits must be positive, got %d", permits)
		os.Exit(1)
	}
	log.Debug("Created semaphore with %d permits", permits)
	return &Semaphore{permits: make(chan struct{}, permits)}
}

// Acquire takes a permit, blocking if none are available. Time complexity: O(1).
func (s *Semaphore) Acquire() {
	s.permits <- struct{}{}
	log.Debug("Permit acquired (available: %d)", cap(s.permits)-len(s.permits))
}

// Release returns a permit to the pool. Time complexity: O(1).
func (s *Semaphore) Release() {
	<-s.permits
	log.Debug("Permit released (available: %d)", cap(s.permits)-len(s.permits))
}

func main() {
	defer log.Operation("main", "Running Semaphore demo")()

	logger.Section("Scenario: Rate-Limited External API Caller")
	log.Info("Simulating 5 concurrent API calls limited to 2 in-flight")

	maxConcurrent := 2
	totalRequests := 5
	semaphore := NewSemaphore(maxConcurrent)

	var wg sync.WaitGroup
	start := time.Now()

	for i := 1; i <= totalRequests; i++ {
		wg.Add(1)
		go func(reqID int) {
			defer wg.Done()

			semaphore.Acquire()
			defer semaphore.Release()

			log.Step("Request %d - calling external API...", reqID)
			time.Sleep(100 * time.Millisecond) // simulated I/O
			log.Info("Request %d completed", reqID)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	logger.Section("Final Stats Summary")
	logger.KeyValue("maxConcurrent", maxConcurrent)
	logger.KeyValue("totalRequests", totalRequests)
	logger.KeyValue("totalTime", elapsed)
	log.Info("All %d requests completed in %v", totalRequests, elapsed)
}
