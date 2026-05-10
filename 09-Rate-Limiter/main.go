package main

import (
	"fmt"
	"sync"
	"time"
)

type TokenBucketRateLimiter struct {
	mu              sync.Mutex
	capacity        float64
	tokens          float64
	refillPerSecond float64
	lastRefill      time.Time
}

func NewTokenBucketRateLimiter(capacity int, refillPerSecond float64) *TokenBucketRateLimiter {
	if capacity <= 0 || refillPerSecond <= 0 {
		panic("capacity and refill rate must be positive")
	}
	return &TokenBucketRateLimiter{capacity: float64(capacity), tokens: float64(capacity), refillPerSecond: refillPerSecond, lastRefill: time.Now()}
}

func (l *TokenBucketRateLimiter) Allow(cost float64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill(time.Now())
	if cost <= 0 {
		return true
	}
	if l.tokens < cost {
		return false
	}
	l.tokens -= cost
	return true
}

func (l *TokenBucketRateLimiter) refill(now time.Time) {
	elapsed := now.Sub(l.lastRefill).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	l.tokens = mathMin(l.capacity, l.tokens+elapsed*l.refillPerSecond)
	l.lastRefill = now
}

func mathMin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func main() {
	limiter := NewTokenBucketRateLimiter(2, 1)
	fmt.Println(limiter.Allow(1), limiter.Allow(1), limiter.Allow(1))
}
