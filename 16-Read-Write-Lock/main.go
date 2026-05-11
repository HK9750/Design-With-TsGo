package main

import (
	"os"
	"sync"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// ReadWriteLock wraps sync.RWMutex to provide reader/writer mutual exclusion.
// Multiple readers can hold the lock simultaneously, but only one writer at a
// time, and writers block readers.
type ReadWriteLock struct{ mu sync.RWMutex }

// RLock acquires a read lock. Multiple goroutines may hold read locks concurrently.
// Time complexity: O(1).
func (l *ReadWriteLock) RLock() {
	log.Debug("Acquiring read lock")
	l.mu.RLock()
}

// RUnlock releases a read lock. Time complexity: O(1).
func (l *ReadWriteLock) RUnlock() {
	log.Debug("Releasing read lock")
	l.mu.RUnlock()
}

// Lock acquires a write lock. Only one goroutine may hold the write lock, and
// all readers are blocked. Time complexity: O(1).
func (l *ReadWriteLock) Lock() {
	log.Debug("Acquiring write lock")
	l.mu.Lock()
}

// Unlock releases a write lock. Time complexity: O(1).
func (l *ReadWriteLock) Unlock() {
	log.Debug("Releasing write lock")
	l.mu.Unlock()
}

func main() {
	defer log.Operation("main", "Running Read-Write Lock demo")()

	logger.Section("Scenario: In-Memory Cache with Concurrent Reads and Writes")
	log.Info("Simulating a configuration cache: 10 readers, 2 writers")

	lock := &ReadWriteLock{}
	cache := make(map[string]string)
	var wg sync.WaitGroup

	// Pre-populate cache
	cache["db_host"] = "localhost"
	cache["db_port"] = "5432"
	cache["api_key"] = "sk-12345"

	start := time.Now()
	readOps := 0
	writeOps := 0
	var readMu, writeMu sync.Mutex

	// Launch 10 reader goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			lock.RLock()
			_ = cache["db_host"]
			log.Step("Reader %d reading config", id)
			time.Sleep(10 * time.Millisecond)
			readMu.Lock()
			readOps++
			readMu.Unlock()
			lock.RUnlock()
		}(i)
	}

	// Launch 2 writer goroutines
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			lock.Lock()
			cache["updated_by"] = "writer"
			log.Step("Writer %d updating config", id)
			time.Sleep(5 * time.Millisecond)
			writeMu.Lock()
			writeOps++
			writeMu.Unlock()
			lock.Unlock()
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	logger.Section("Final Stats Summary")
	logger.KeyValue("readOperations", readOps)
	logger.KeyValue("writeOperations", writeOps)
	logger.KeyValue("cacheKeys", len(cache))
	logger.KeyValue("totalTime", elapsed)
	log.Info("Cache operations completed in %v (%d reads, %d writes)", elapsed, readOps, writeOps)
}
