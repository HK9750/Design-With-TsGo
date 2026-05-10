package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type RetryPolicy struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
}

func Retry(ctx context.Context, policy RetryPolicy, operation func(attempt int) error) error {
	var lastErr error
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		if err := operation(attempt); err != nil {
			lastErr = err
			if attempt == policy.MaxAttempts {
				break
			}
			timer := time.NewTimer(delayForAttempt(attempt, policy))
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		return nil
	}
	return lastErr
}

func delayForAttempt(attempt int, policy RetryPolicy) time.Duration {
	delay := float64(policy.InitialDelay) * pow(policy.Multiplier, attempt-1)
	if delay > float64(policy.MaxDelay) {
		delay = float64(policy.MaxDelay)
	}
	if policy.Jitter {
		return time.Duration(rand.Int63n(int64(delay) + 1))
	}
	return time.Duration(delay)
}

func pow(base float64, exp int) float64 {
	result := 1.0
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}

func main() {
	err := Retry(context.Background(), RetryPolicy{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Second, Multiplier: 2}, func(attempt int) error {
		if attempt < 2 {
			return errors.New("transient")
		}
		return nil
	})
	fmt.Println(err)
}
