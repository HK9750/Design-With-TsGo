package main

import (
	"fmt"
	"time"
)

type Lease struct {
	Owner     string
	Token     int64
	ExpiresAt time.Time
}

type DistributedLockManager struct {
	locks     map[string]Lease
	nextToken int64
}

func NewDistributedLockManager() *DistributedLockManager {
	return &DistributedLockManager{locks: make(map[string]Lease), nextToken: 1}
}

func (m *DistributedLockManager) Acquire(resource, owner string, ttl time.Duration) (Lease, bool) {
	now := time.Now()
	if current, ok := m.locks[resource]; ok && current.ExpiresAt.After(now) {
		return Lease{}, false
	}
	lease := Lease{Owner: owner, Token: m.nextToken, ExpiresAt: now.Add(ttl)}
	m.nextToken++
	m.locks[resource] = lease
	return lease, true
}

func (m *DistributedLockManager) Release(resource, owner string, token int64) bool {
	current, ok := m.locks[resource]
	if !ok || current.Owner != owner || current.Token != token {
		return false
	}
	delete(m.locks, resource)
	return true
}

func (m *DistributedLockManager) Validate(resource string, token int64) bool {
	lease, ok := m.locks[resource]
	return ok && lease.Token == token && lease.ExpiresAt.After(time.Now())
}

func main() {
	manager := NewDistributedLockManager()
	lease, ok := manager.Acquire("order:1", "worker-a", time.Second)
	fmt.Println(lease.Token, ok, manager.Validate("order:1", lease.Token))
}
