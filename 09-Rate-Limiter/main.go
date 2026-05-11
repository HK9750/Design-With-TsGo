// Package main demonstrates a Token Bucket Rate Limiter for controlling the rate
// of operations (API requests, network calls) with configurable capacity and refill rate.
// Thread-safe via mutex. Suitable for API gateway rate limiting.
package main

import (
	"os"
	"sync"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// TokenBucketRateLimiter implements the token bucket algorithm for rate limiting.
// It has a fixed capacity (burst limit) and refills tokens at a constant rate.
// Thread-safe: all public methods are guarded by a mutex.
type TokenBucketRateLimiter struct {
	mu              sync.Mutex
	capacity        float64
	tokens          float64
	refillPerSecond float64
	lastRefill      time.Time
}

// NewTokenBucketRateLimiter creates a new rate limiter with the given capacity (max tokens/burst)
// and refill rate (tokens added per second). Exits on non-positive parameters.
func NewTokenBucketRateLimiter(capacity int, refillPerSecond float64) *TokenBucketRateLimiter {
	if capacity <= 0 || refillPerSecond <= 0 {
		log.Error("capacity (%d) and refill rate (%.2f) must be positive", capacity, refillPerSecond)
		os.Exit(1)
	}
	log.Debug("Created token bucket rate limiter (capacity=%d, refill_rate=%.2f/s)", capacity, refillPerSecond)
	return &TokenBucketRateLimiter{
		capacity:        float64(capacity),
		tokens:          float64(capacity),
		refillPerSecond: refillPerSecond,
		lastRefill:      time.Now(),
	}
}

// Allow checks if a request with the given token cost can proceed. If enough tokens
// are available, they are consumed and true is returned. If not enough tokens, false
// is returned and no tokens are consumed. A cost of 0 or less is always allowed.
// Time complexity: O(1).
func (l *TokenBucketRateLimiter) Allow(cost float64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill(time.Now())
	if cost <= 0 {
		return true
	}
	if l.tokens < cost {
		log.Debug("Rate limiter reject (tokens=%.2f, cost=%.2f, capacity=%.2f)", l.tokens, cost, l.capacity)
		return false
	}
	l.tokens -= cost
	log.Debug("Rate limiter allow (tokens=%.2f, cost=%.2f)", l.tokens, cost)
	return true
}

// refill adds tokens based on the elapsed time since the last refill, up to capacity.
func (l *TokenBucketRateLimiter) refill(now time.Time) {
	elapsed := now.Sub(l.lastRefill).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	newTokens := mathMin(l.capacity, l.tokens+elapsed*l.refillPerSecond)
	if newTokens > l.tokens {
		log.Debug("Rate limiter refill (tokens=%.2f -> %.2f, elapsed=%.3fs)", l.tokens, newTokens, elapsed)
	}
	l.tokens = newTokens
	l.lastRefill = now
}

// mathMin returns the smaller of two float64 values.
func mathMin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// main demonstrates the token bucket rate limiter protecting an API gateway
// against burst traffic, allowing 5 requests/second with bursts up to 10.
func main() {
	defer log.Operation("main", "Running Token Bucket Rate Limiter demo")()

	logger.Section("Token Bucket Rate Limiter — API Gateway Protection")
	limiter := NewTokenBucketRateLimiter(10, 5)
	logger.KeyValue("capacity", 10)
	logger.KeyValue("refill_rate", "5 tokens/sec")
	logger.KeyValue("max_burst", "10 requests")

	logger.Section("Initial burst — consuming tokens rapidly")
	passed := 0
	rejected := 0
	for i := 0; i < 15; i++ {
		if limiter.Allow(1) {
			passed++
			log.Info("Request %2d: ALLOWED", i+1)
		} else {
			rejected++
			log.Warn("Request %2d: REJECTED (rate limited)", i+1)
		}
	}
	log.Info("Burst result: %d allowed, %d rejected", passed, rejected)

	logger.Section("Waiting for token refill...")
	time.Sleep(1 * time.Second)
	log.Info("Waited 1 second (5 tokens refilled)")

	logger.Section("Post-refill requests")
	for i := 0; i < 8; i++ {
		if limiter.Allow(1) {
			log.Info("Request %2d: ALLOWED", i+1)
		} else {
			log.Warn("Request %2d: REJECTED", i+1)
		}
	}

	logger.Section("Variable cost requests — simulating different API tiers")
	time.Sleep(2 * time.Second)
	log.Info("Waited 2 seconds for full token replenishment")
	log.Info("Light API call (cost=1): %v", limiter.Allow(1))
	log.Info("Heavy API call (cost=3): %v", limiter.Allow(3))
	log.Info("Batch API call (cost=5): %v", limiter.Allow(5))
	log.Info("Another call (cost=2, should fail): %v", limiter.Allow(2))

	logger.Section("Sustained rate simulation — 100 requests over time")
	limiter2 := NewTokenBucketRateLimiter(20, 10)
	start := time.Now()
	allowed := 0
	denied := 0
	for i := 0; i < 100; i++ {
		if limiter2.Allow(1) {
			allowed++
		} else {
			denied++
		}
		time.Sleep(50 * time.Millisecond)
	}
	log.Info("Sustained 100 requests (50ms intervals) in %v: %d allowed, %d denied",
		time.Since(start), allowed, denied)

	logger.Section("Final Stats")
	log.Info("Rate limiter correctly enforces burst and sustained rate limits")
	log.Info("All operations completed — O(1) allow checks with token bucket algorithm")
}
