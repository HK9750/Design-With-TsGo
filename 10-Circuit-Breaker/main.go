// Package main demonstrates a Circuit Breaker pattern for fault-tolerant service calls.
// It transitions through Closed → Open → Half-Open states based on failure thresholds
// and timeout windows. Prevents cascading failures in distributed systems.
package main

import (
	"errors"
	"os"
	"sync"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// CircuitState represents the current state of the circuit breaker.
type CircuitState string

const (
	// CircuitClosed is the normal operating state — requests flow through.
	CircuitClosed CircuitState = "closed"
	// CircuitOpen is the failure state — requests are rejected immediately.
	CircuitOpen CircuitState = "open"
	// CircuitHalfOpen is the recovery testing state — a limited number of requests are allowed through.
	CircuitHalfOpen CircuitState = "half-open"
)

// CircuitBreaker implements the circuit breaker pattern to protect downstream services.
// It tracks consecutive failures and opens the circuit when the failure threshold is reached.
// After a timeout period, it transitions to half-open to test if the service has recovered.
type CircuitBreaker struct {
	mu               sync.Mutex
	state            CircuitState
	failures         int
	failureThreshold int
	openTimeout      time.Duration
	openedAt         time.Time
}

// NewCircuitBreaker creates a new circuit breaker with the given failure threshold
// and open timeout duration. Starts in the closed state. Time complexity: O(1).
func NewCircuitBreaker(failureThreshold int, openTimeout time.Duration) *CircuitBreaker {
	log.Debug("Created circuit breaker (threshold=%d, timeout=%v)", failureThreshold, openTimeout)
	return &CircuitBreaker{
		state:            CircuitClosed,
		failureThreshold: failureThreshold,
		openTimeout:      openTimeout,
	}
}

// State returns the current circuit breaker state. If the circuit is open and the
// timeout has elapsed, it transitions to half-open before returning. Time complexity: O(1).
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == CircuitOpen && time.Since(cb.openedAt) >= cb.openTimeout {
		log.Debug("Circuit breaker transitioning from open to half-open (opened_for=%v, timeout=%v)",
			time.Since(cb.openedAt), cb.openTimeout)
		cb.state = CircuitHalfOpen
	}
	return cb.state
}

// Execute runs the given operation through the circuit breaker. If the circuit is open,
// the operation is not executed and an error is returned immediately.
// On operation failure, the failure count is incremented and the circuit may open.
// On success, the failure count is reset and the circuit closes. Time complexity: O(1).
func (cb *CircuitBreaker) Execute(operation func() error) error {
	if cb.State() == CircuitOpen {
		log.Warn("Circuit breaker open — rejecting request")
		return errors.New("circuit breaker is open")
	}
	if err := operation(); err != nil {
		log.Debug("Circuit breaker operation failed: %v", err)
		cb.recordFailure()
		return err
	}
	cb.recordSuccess()
	log.Debug("Circuit breaker operation succeeded")
	return nil
}

// recordSuccess resets the failure count and transitions the circuit to closed.
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	if cb.state != CircuitClosed {
		log.Debug("Circuit breaker reset — transitioning to closed")
	}
	cb.state = CircuitClosed
}

// recordFailure increments the failure count. If the threshold is reached or the circuit
// is in half-open state, the circuit opens and records the open time.
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	if cb.state == CircuitHalfOpen || cb.failures >= cb.failureThreshold {
		log.Warn("Circuit breaker opened (failures=%d, threshold=%d)", cb.failures, cb.failureThreshold)
		cb.state = CircuitOpen
		cb.openedAt = time.Now()
	}
}

// main demonstrates the circuit breaker protecting a downstream payment service
// from cascading failures. When the payment service fails repeatedly, the circuit
// opens to prevent wasted calls, then tests recovery after a timeout.
func main() {
	defer log.Operation("main", "Running Circuit Breaker demo")()

	logger.Section("Circuit Breaker — Protecting Downstream Payment Service")
	breaker := NewCircuitBreaker(3, 2*time.Second)
	logger.KeyValue("failure_threshold", 3)
	logger.KeyValue("open_timeout", "2s")

	logger.Section("Healthy service — all requests succeed")
	for i := 0; i < 3; i++ {
		err := breaker.Execute(func() error {
			log.Debug("Payment processed successfully")
			return nil
		})
		log.Info("Request %d: err=%v state=%s", i+1, err, breaker.State())
	}

	logger.Section("Service degradation — consecutive failures trigger circuit open")
	var failCount int
	for i := 0; i < 5; i++ {
		err := breaker.Execute(func() error {
			failCount++
			return errors.New("payment gateway timeout")
		})
		if err != nil {
			log.Warn("Request %d: FAILED — %v | state=%s", i+1, err, breaker.State())
		}
	}

	logger.Section("Circuit open — requests fast-fail without calling downstream")
	for i := 0; i < 3; i++ {
		err := breaker.Execute(func() error {
			return nil
		})
		if err != nil {
			log.Warn("Request fast-failed: %v | state=%s", err, breaker.State())
		}
	}

	logger.Section("Waiting for recovery timeout...")
	time.Sleep(2 * time.Second)
	log.Info("Timeout elapsed — circuit transitions to half-open")

	logger.Section("Half-open — testing with a probe request")
	err := breaker.Execute(func() error {
		log.Debug("Probe request to payment service")
		return nil
	})
	if err == nil {
		log.Info("Probe succeeded! Circuit reset to closed. state=%s", breaker.State())
	} else {
		log.Warn("Probe failed. Circuit remains open. state=%s", breaker.State())
	}

	logger.Section("Service recovered — normal operations resume")
	for i := 0; i < 5; i++ {
		err := breaker.Execute(func() error {
			return nil
		})
		log.Info("Request %d: err=%v state=%s", i+1, err, breaker.State())
	}

	logger.Section("Full cycle demonstration — failure → open → timeout → half-open → closed")
	log.Debug("Resetting breaker for full cycle demo")
	breaker2 := NewCircuitBreaker(2, 500*time.Millisecond)
	log.Info("Initial state: %s", breaker2.State())
	breaker2.Execute(func() error { return errors.New("fail 1") })
	log.Info("After 1st failure: failures=1 state=%s", breaker2.State())
	breaker2.Execute(func() error { return errors.New("fail 2") })
	log.Info("After 2nd failure (threshold=2): state=%s", breaker2.State())
	time.Sleep(600 * time.Millisecond)
	log.Info("After timeout: state=%s", breaker2.State())
	breaker2.Execute(func() error { return nil })
	log.Info("After successful probe: state=%s", breaker2.State())

	logger.Section("Final Stats")
	log.Info("Circuit breaker prevented cascading failures during outage")
	log.Info("All operations completed — states: Closed → Open → Half-Open → Closed")
}
