package main

import "fmt"

type BoundedQueue[T any] struct{ ch chan T }

func NewBoundedQueue[T any](capacity int) *BoundedQueue[T] {
	if capacity <= 0 {
		panic("capacity must be positive")
	}
	return &BoundedQueue[T]{ch: make(chan T, capacity)}
}

func (q *BoundedQueue[T]) Enqueue(value T) { q.ch <- value }

func (q *BoundedQueue[T]) Dequeue() T { return <-q.ch }

func main() {
	queue := NewBoundedQueue[int](1)
	queue.Enqueue(42)
	fmt.Println(queue.Dequeue())
}
