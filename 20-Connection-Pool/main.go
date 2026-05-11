package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// ConnectionPool manages a bounded set of reusable connections. It lazily creates
// connections via the factory function up to max. Idle connections are validated
// on acquire; invalid ones are discarded. Uses generics for any connection type T.
type ConnectionPool[T any] struct {
	idle     []T
	active   int
	max      int
	factory  func() T
	validate func(T) bool
	mu       sync.Mutex
}

// NewConnectionPool creates a connection pool. max caps the number of concurrent
// connections. factory creates new connections. validate checks health on acquire.
// Panics if max <= 0. Time complexity: O(1).
func NewConnectionPool[T any](max int, factory func() T, validate func(T) bool) *ConnectionPool[T] {
	if max <= 0 {
		log.Error("max must be positive, got %d", max)
		os.Exit(1)
	}
	log.Debug("Created connection pool: max=%d", max)
	return &ConnectionPool[T]{max: max, factory: factory, validate: validate}
}

// Acquire obtains a connection from the pool. Returns an idle connection if one
// passes validation, otherwise creates a new one up to max. Returns an error if
// the pool is exhausted. Time complexity: O(idle) worst case on stale checks.
func (p *ConnectionPool[T]) Acquire() (T, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for len(p.idle) > 0 {
		conn := p.idle[len(p.idle)-1]
		p.idle = p.idle[:len(p.idle)-1]
		if p.validate(conn) {
			p.active++
			log.Debug("Acquired idle connection (active=%d idle=%d max=%d)", p.active, len(p.idle), p.max)
			return conn, nil
		}
		log.Warn("Idle connection failed validation, discarding (active=%d)", p.active)
	}

	var zero T
	if p.active >= p.max {
		log.Warn("Connection pool exhausted: active=%d max=%d", p.active, p.max)
		return zero, fmt.Errorf("connection pool exhausted")
	}

	p.active++
	conn := p.factory()
	log.Debug("Created new connection (active=%d idle=%d max=%d)", p.active, len(p.idle), p.max)
	return conn, nil
}

// Release returns a connection to the pool. If the connection fails validation,
// it is discarded permanently. Time complexity: O(1).
func (p *ConnectionPool[T]) Release(conn T) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.active--
	if p.validate(conn) {
		p.idle = append(p.idle, conn)
		log.Debug("Released connection to idle pool (active=%d idle=%d)", p.active, len(p.idle))
	} else {
		log.Warn("Released connection failed validation, discarding (active=%d)", p.active)
	}
}

func main() {
	defer log.Operation("main", "Running Connection Pool demo")()

	logger.Section("Scenario: Database Connection Pool for Web Service")
	log.Info("Simulating an API server with a max of 3 database connections serving 5 requests")

	connCount := 0
	pool := NewConnectionPool(3,
		func() string {
			connCount++
			return "db-conn-" + string(rune('A'+connCount-1))
		},
		func(s string) bool {
			return len(s) > 0 // all connections valid
		},
	)

	start := time.Now()
	totalRequests := 5
	var wg sync.WaitGroup
	acquired := make([]string, 0, totalRequests)
	var mu sync.Mutex

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(reqID int) {
			defer wg.Done()

			log.Step("Request %d acquiring connection...", reqID)
			conn, err := pool.Acquire()
			if err != nil {
				log.Error("Request %d failed: %v", reqID, err)
				return
			}

			mu.Lock()
			acquired = append(acquired, conn)
			mu.Unlock()

			// Simulate database query
			time.Sleep(80 * time.Millisecond)
			log.Info("Request %d completed using %s", reqID, conn)

			pool.Release(conn)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	logger.Section("Connection Usage")
	for _, conn := range acquired {
		logger.KeyValue("usedConnection", conn)
	}

	logger.Section("Final Stats Summary")
	logger.KeyValue("maxConnections", 3)
	logger.KeyValue("totalRequests", totalRequests)
	logger.KeyValue("connectionsCreated", connCount)
	logger.KeyValue("totalTime", elapsed)
	log.Info("Connection pool handled %d requests in %v (created %d connections)", totalRequests, elapsed, connCount)
}
