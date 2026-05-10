package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half-open"
)

type CircuitBreaker struct {
	mu               sync.Mutex
	state            CircuitState
	failures         int
	failureThreshold int
	openTimeout      time.Duration
	openedAt         time.Time
}

func NewCircuitBreaker(failureThreshold int, openTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{state: CircuitClosed, failureThreshold: failureThreshold, openTimeout: openTimeout}
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == CircuitOpen && time.Since(cb.openedAt) >= cb.openTimeout {
		cb.state = CircuitHalfOpen
	}
	return cb.state
}

func (cb *CircuitBreaker) Execute(operation func() error) error {
	if cb.State() == CircuitOpen {
		return errors.New("circuit breaker is open")
	}
	if err := operation(); err != nil {
		cb.recordFailure()
		return err
	}
	cb.recordSuccess()
	return nil
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = CircuitClosed
}

func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	if cb.state == CircuitHalfOpen || cb.failures >= cb.failureThreshold {
		cb.state = CircuitOpen
		cb.openedAt = time.Now()
	}
}

func main() {
	breaker := NewCircuitBreaker(2, time.Second)
	fmt.Println(breaker.Execute(func() error { return nil }), breaker.State())
}
