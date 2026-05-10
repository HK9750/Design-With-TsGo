package main

import (
	"fmt"
	"sync"
)

type ReadWriteLock struct{ mu sync.RWMutex }

func (l *ReadWriteLock) RLock()   { l.mu.RLock() }
func (l *ReadWriteLock) RUnlock() { l.mu.RUnlock() }
func (l *ReadWriteLock) Lock()    { l.mu.Lock() }
func (l *ReadWriteLock) Unlock()  { l.mu.Unlock() }

func main() {
	lock := &ReadWriteLock{}
	lock.RLock()
	lock.RUnlock()
	fmt.Println("read complete")
}
