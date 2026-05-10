package main

import "fmt"

type ChannelPool[T any] struct {
	available chan chan T
	factory   func() chan T
}

func NewChannelPool[T any](size int, factory func() chan T) *ChannelPool[T] {
	if size <= 0 {
		panic("size must be positive")
	}
	pool := &ChannelPool[T]{available: make(chan chan T, size), factory: factory}
	for i := 0; i < size; i++ {
		pool.available <- factory()
	}
	return pool
}

func (p *ChannelPool[T]) Acquire() chan T { return <-p.available }

func (p *ChannelPool[T]) Release(ch chan T) { p.available <- ch }

func main() {
	pool := NewChannelPool(1, func() chan string { return make(chan string, 1) })
	ch := pool.Acquire()
	ch <- "message"
	fmt.Println(<-ch)
	pool.Release(ch)
}
