package main

import "fmt"

type Future[T any] struct{ done chan result[T] }

type result[T any] struct {
	value T
	err   error
}

func Async[T any](fn func() (T, error)) *Future[T] {
	future := &Future[T]{done: make(chan result[T], 1)}
	go func() {
		value, err := fn()
		future.done <- result[T]{value: value, err: err}
	}()
	return future
}

func (f *Future[T]) Await() (T, error) {
	res := <-f.done
	return res.value, res.err
}

func Then[T any, U any](future *Future[T], next func(T) (U, error)) *Future[U] {
	return Async(func() (U, error) {
		value, err := future.Await()
		if err != nil {
			var zero U
			return zero, err
		}
		return next(value)
	})
}

func main() {
	future := Async(func() (int, error) { return 2, nil })
	chained := Then(future, func(n int) (int, error) { return n + 3, nil })
	value, _ := chained.Await()
	fmt.Println(value)
}
