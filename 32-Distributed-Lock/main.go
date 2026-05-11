package main

import (
	"design-with-tsgo/pkg/logger"
	"fmt"
	"os"
	"time"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// Lease represents a granted lock on a resource with an expiration time.
// The Token is a monotonically increasing identifier used to prevent
// accidental releases by stale holders.
type Lease struct {
	Owner     string
	Token     int64
	ExpiresAt time.Time
}

// DistributedLockManager provides a simple in-memory distributed lock
// service with TTL-based leases. It supports Acquire, Release, and
// Validate operations. All operations are O(1).
type DistributedLockManager struct {
	locks     map[string]Lease
	nextToken int64
}

// NewDistributedLockManager creates a new lock manager with no active locks.
// Time complexity: O(1).
func NewDistributedLockManager() *DistributedLockManager {
	defer log.Operation("NewDistributedLockManager", "Initializing lock manager")()
	return &DistributedLockManager{locks: make(map[string]Lease), nextToken: 1}
}

// Acquire attempts to lock a resource for the given owner with a TTL.
// Returns the Lease and true on success, or an empty Lease and false if
// the resource is already locked. Time complexity: O(1).
func (m *DistributedLockManager) Acquire(resource, owner string, ttl time.Duration) (Lease, bool) {
	defer log.Operation("Acquire", "Attempting lock resource=%s owner=%s ttl=%v", resource, owner, ttl)()
	now := time.Now()
	if current, ok := m.locks[resource]; ok && current.ExpiresAt.After(now) {
		log.Warn("Lock contention on %s — held by %s until %v", resource, current.Owner, current.ExpiresAt.Format("15:04:05"))
		return Lease{}, false
	}
	if _, ok := m.locks[resource]; ok {
		log.Debug("Stale lease on %s detected, overriding", resource)
	}
	lease := Lease{Owner: owner, Token: m.nextToken, ExpiresAt: now.Add(ttl)}
	m.nextToken++
	m.locks[resource] = lease
	log.Info("Lock acquired: resource=%s owner=%s token=%d expires=%v", resource, owner, lease.Token, lease.ExpiresAt.Format("15:04:05"))
	return lease, true
}

// Release frees a lock only if the caller matches both the owner and token.
// Returns true if the lock was successfully released. Time complexity: O(1).
func (m *DistributedLockManager) Release(resource, owner string, token int64) bool {
	defer log.Operation("Release", "Releasing lock resource=%s owner=%s token=%d", resource, owner, token)()
	current, ok := m.locks[resource]
	if !ok {
		log.Warn("Release failed: resource %s not locked", resource)
		return false
	}
	if current.Owner != owner {
		log.Warn("Release denied: owner mismatch (expected=%s, got=%s)", current.Owner, owner)
		return false
	}
	if current.Token != token {
		log.Warn("Release denied: token mismatch (expected=%d, got=%d)", current.Token, token)
		return false
	}
	delete(m.locks, resource)
	log.Info("Lock released: resource=%s owner=%s token=%d", resource, owner, token)
	return true
}

// Validate checks whether a given token is still the valid lock holder
// for the resource and the lease has not expired. Time complexity: O(1).
func (m *DistributedLockManager) Validate(resource string, token int64) bool {
	lease, ok := m.locks[resource]
	valid := ok && lease.Token == token && lease.ExpiresAt.After(time.Now())
	log.Debug("Validating lock: resource=%s token=%d valid=%v", resource, token, valid)
	if !ok {
		log.Warn("Validation failed: resource %s not found", resource)
	} else if lease.Token != token {
		log.Warn("Validation failed: token mismatch for %s", resource)
	} else if !lease.ExpiresAt.After(time.Now()) {
		log.Warn("Validation failed: lease expired for %s at %v", resource, lease.ExpiresAt.Format("15:04:05"))
	}
	return valid
}

func main() {
	defer log.Operation("main", "Running Distributed Lock demo")()

	logger.Section("DISTRIBUTED LOCK — Production Scenario")

	// Simulate a payment processing system using distributed locks
	manager := NewDistributedLockManager()

	logger.Section("SCENARIO: Two Workers Competing for Order Processing")
	log.Info("Worker-A acquires lock on order:1 for payment processing")
	leaseA, ok := manager.Acquire("order:1", "worker-a", 2*time.Second)
	if ok {
		logger.KeyValue("worker_a_token", leaseA.Token)
	}

	log.Info("Worker-B attempts to acquire same order:1 (should fail)")
	_, ok = manager.Acquire("order:1", "worker-b", 2*time.Second)
	if !ok {
		log.Info("Worker-B correctly denied — order:1 is locked by worker-a")
	}

	logger.Section("SCENARIO: Lock Validation and Release")
	log.Info("Worker-A validates its lock before processing")
	if manager.Validate("order:1", leaseA.Token) {
		log.Info("Lock valid — safe to process payment")
	} else {
		log.Warn("Lock invalid — aborting payment")
	}

	log.Info("Worker-A completes payment and releases lock")
	if manager.Release("order:1", "worker-a", leaseA.Token) {
		log.Info("Lock released successfully")
	}

	logger.Section("SCENARIO: Lock Expiration and Stale Release Prevention")
	log.Info("Worker-C acquires short-lived lock on inventory:99 (100ms TTL)")
	leaseC, _ := manager.Acquire("inventory:99", "worker-c", 100*time.Millisecond)
	logger.KeyValue("worker_c_token", leaseC.Token)

	log.Info("Waiting for lease to expire...")
	time.Sleep(150 * time.Millisecond)

	log.Info("Worker-D attempts to acquire expired lock on inventory:99")
	leaseD, ok := manager.Acquire("inventory:99", "worker-d", 2*time.Second)
	if ok {
		log.Info("Worker-D successfully acquired (previous lease expired)")
		logger.KeyValue("worker_d_token", leaseD.Token)
	}

	log.Info("Worker-C tries to release with stale token (should fail)")
	if !manager.Release("inventory:99", "worker-c", leaseC.Token) {
		log.Info("Stale release correctly denied — token no longer valid")
	}

	// Final stats summary
	logger.Section("FINAL STATS")
	logger.KeyValue("active_locks", len(manager.locks))
	logger.KeyValue("total_tokens_issued", manager.nextToken-1)
	for resource, lease := range manager.locks {
		logger.KeyValue("lock_"+resource, fmt.Sprintf("owner=%s token=%d expires=%s", lease.Owner, lease.Token, lease.ExpiresAt.Format("15:04:05")))
	}
	log.Info("Distributed lock demo completed successfully")
}
