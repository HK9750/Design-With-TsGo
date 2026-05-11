package main

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// RetryPolicy defines the parameters for exponential backoff retry.
// MaxAttempts: total attempts including the first call.
// InitialDelay: wait time before the first retry.
// MaxDelay: cap on the computed delay.
// Multiplier: exponential factor applied each retry.
// Jitter: if true, randomizes the delay to avoid thundering herd.
type RetryPolicy struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
}

// Retry executes an operation with exponential backoff up to MaxAttempts.
// ctx cancels the retry loop early. Returns nil on first success, or the last
// error after exhausting all attempts. Time complexity: O(MaxAttempts).
func Retry(ctx context.Context, policy RetryPolicy, operation func(attempt int) error) error {
	log.Debug("Retry starting: maxAttempts=%d initialDelay=%v maxDelay=%v multiplier=%.2f jitter=%v",
		policy.MaxAttempts, policy.InitialDelay, policy.MaxDelay, policy.Multiplier, policy.Jitter)

	var lastErr error
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		if err := operation(attempt); err != nil {
			lastErr = err
			log.Warn("Attempt %d/%d failed: %v", attempt, policy.MaxAttempts, err)
			if attempt == policy.MaxAttempts {
				log.Error("All %d attempts exhausted", policy.MaxAttempts)
				break
			}
			delay := delayForAttempt(attempt, policy)
			log.Debug("Waiting %v before next attempt", delay)
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				log.Warn("Retry cancelled by context: %v", ctx.Err())
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		log.Info("Operation succeeded on attempt %d", attempt)
		return nil
	}
	return lastErr
}

// delayForAttempt computes the backoff duration for a given attempt number.
// Formula: min(InitialDelay * Multiplier^(attempt-1), MaxDelay).
// If Jitter is enabled, returns a random duration up to the computed delay.
// Time complexity: O(attempt) due to exponentiation.
func delayForAttempt(attempt int, policy RetryPolicy) time.Duration {
	delay := float64(policy.InitialDelay) * pow(policy.Multiplier, attempt-1)
	if delay > float64(policy.MaxDelay) {
		delay = float64(policy.MaxDelay)
	}
	if policy.Jitter {
		jittered := time.Duration(rand.Int63n(int64(delay) + 1))
		log.Debug("Delay with jitter: %v (max %v)", jittered, time.Duration(delay))
		return jittered
	}
	return time.Duration(delay)
}

// pow computes base^exp using simple iteration. Time complexity: O(exp).
func pow(base float64, exp int) float64 {
	result := 1.0
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}

func main() {
	defer log.Operation("main", "Running Retry Backoff demo")()

	logger.Section("Scenario: Payment Gateway with Transient Failures")
	log.Info("Simulating a payment service that succeeds on the 3rd attempt")

	policy := RetryPolicy{
		MaxAttempts:  4,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		Multiplier:   2,
		Jitter:       true,
	}

	// Simulate an unreliable payment gateway
	callCount := 0
	err := Retry(context.Background(), policy, func(attempt int) error {
		callCount++
		log.Step("Payment attempt %d - contacting gateway...", attempt)
		if attempt < 3 {
			return errors.New("gateway timeout")
		}
		return nil
	})

	logger.Section("Final Stats Summary")
	logger.KeyValue("policy.maxAttempts", policy.MaxAttempts)
	logger.KeyValue("policy.initialDelay", policy.InitialDelay)
	logger.KeyValue("policy.maxDelay", policy.MaxDelay)
	logger.KeyValue("policy.multiplier", policy.Multiplier)
	logger.KeyValue("policy.jitter", policy.Jitter)
	logger.KeyValue("totalAttempts", callCount)
	logger.KeyValue("finalError", err)
}
