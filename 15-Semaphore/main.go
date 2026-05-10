package main

import "fmt"

type Semaphore struct{ permits chan struct{} }

func NewSemaphore(permits int) *Semaphore {
	if permits <= 0 {
		panic("permits must be positive")
	}
	return &Semaphore{permits: make(chan struct{}, permits)}
}

func (s *Semaphore) Acquire() { s.permits <- struct{}{} }

func (s *Semaphore) Release() { <-s.permits }

func main() {
	semaphore := NewSemaphore(1)
	semaphore.Acquire()
	semaphore.Release()
	fmt.Println("released")
}
