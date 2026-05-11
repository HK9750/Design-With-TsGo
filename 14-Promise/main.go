package main

import (
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// Future represents an asynchronous computation that will yield a value of type T
// or an error. Call Await to block until the result is ready.
type Future[T any] struct{ done chan result[T] }

// result holds the outcome of an async operation — either a value or an error.
type result[T any] struct {
	value T
	err   error
}

// Async executes fn in a new goroutine and returns a Future that will hold its
// result. Time complexity: O(1) launch, O(fn) resolution.
func Async[T any](fn func() (T, error)) *Future[T] {
	future := &Future[T]{done: make(chan result[T], 1)}
	log.Debug("Async operation launched")
	go func() {
		value, err := fn()
		future.done <- result[T]{value: value, err: err}
	}()
	return future
}

// Await blocks until the future resolves, then returns its value and error.
// Time complexity: O(fn) — depends on the underlying computation.
func (f *Future[T]) Await() (T, error) {
	res := <-f.done
	if res.err != nil {
		log.Warn("Future resolved with error: %v", res.err)
	} else {
		log.Debug("Future resolved with value: %v", res.value)
	}
	return res.value, res.err
}

// Then chains another async operation that receives the resolved value of future.
// If the first future fails, the second is never executed and the error is
// propagated. Time complexity: O(first + next).
func Then[T any, U any](future *Future[T], next func(T) (U, error)) *Future[U] {
	return Async(func() (U, error) {
		log.Debug("Chaining: awaiting previous future")
		value, err := future.Await()
		if err != nil {
			var zero U
			return zero, err
		}
		return next(value)
	})
}

func main() {
	defer log.Operation("main", "Running Promise/Future demo")()

	logger.Section("Scenario: User Profile Fetch Pipeline")
	log.Info("Simulating async microservice calls: fetch user -> fetch profile")

	start := time.Now()

	future := Async(func() (int, error) {
		log.Step("Calling User Service...")
		time.Sleep(50 * time.Millisecond)
		return 10042, nil // user ID
	})

	chained := Then(future, func(userID int) (string, error) {
		log.Step("Calling Profile Service for user %d...", userID)
		time.Sleep(30 * time.Millisecond)
		return "John Doe <john@example.com>", nil
	})

	profile, err := chained.Await()
	elapsed := time.Since(start)

	log.Info("Profile fetched: %s", profile)

	logger.Section("Final Stats Summary")
	logger.KeyValue("profile", profile)
	logger.KeyValue("error", err)
	logger.KeyValue("totalTime", elapsed)
	log.Info("Pipeline completed in %v", elapsed)
}
