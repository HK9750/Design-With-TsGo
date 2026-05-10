package main

import "fmt"

type ConnectionPool[T any] struct {
	idle     []T
	active   int
	max      int
	factory  func() T
	validate func(T) bool
}

func NewConnectionPool[T any](max int, factory func() T, validate func(T) bool) *ConnectionPool[T] {
	if max <= 0 {
		panic("max must be positive")
	}
	return &ConnectionPool[T]{max: max, factory: factory, validate: validate}
}

func (p *ConnectionPool[T]) Acquire() (T, error) {
	for len(p.idle) > 0 {
		conn := p.idle[len(p.idle)-1]
		p.idle = p.idle[:len(p.idle)-1]
		if p.validate(conn) {
			p.active++
			return conn, nil
		}
	}
	var zero T
	if p.active >= p.max {
		return zero, fmt.Errorf("connection pool exhausted")
	}
	p.active++
	return p.factory(), nil
}

func (p *ConnectionPool[T]) Release(conn T) {
	p.active--
	if p.validate(conn) {
		p.idle = append(p.idle, conn)
	}
}

func main() {
	pool := NewConnectionPool(2, func() string { return "conn" }, func(string) bool { return true })
	conn, _ := pool.Acquire()
	pool.Release(conn)
	fmt.Println("connection returned")
}
