// Package main demonstrates a generic hash table using open addressing with linear probing
// and tombstone deletion. Supports dynamic resizing based on load factor thresholds
// and cleanup of deleted slots.
package main

import (
	"os"

	"design-with-tsgo/pkg/logger"
)

var logoa = logger.New(os.Stdout, logger.DEBUG, true)

// oaFNVOffset32 is the FNV-1a 32-bit offset basis for the open addressing table.
const oaFNVOffset32 uint32 = 2166136261

// oaFNVPrime32 is the FNV-1a 32-bit prime for the open addressing table.
const oaFNVPrime32 uint32 = 16777619

// OAHashEntry represents a single entry in the open addressing hash table.
// The deleted flag marks entries that have been removed (tombstones).
type OAHashEntry[V any] struct {
	key     string
	value   V
	deleted bool
}

// oaProbeResult is returned by findSlot, indicating the target index and whether the key was found.
type oaProbeResult struct {
	index int
	found bool
}

// OpenAddressingHashTable implements a generic hash table with open addressing
// (linear probing) and tombstone deletion for collision resolution.
// Supports dynamic grow/shrink and deleted-slot cleanup.
type OpenAddressingHashTable[V any] struct {
	size             int
	used             int
	capacity         int
	buckets          []*OAHashEntry[V]
	minimumThreshold float64
	maximumThreshold float64
	minimumCapacity  int
}

// NewOpenAddressingHashTable creates a new open addressing hash table with the given
// initial capacity (minimum 16). Time complexity: O(capacity).
func NewOpenAddressingHashTable[V any](capacity int) *OpenAddressingHashTable[V] {
	minimumCapacity := 16
	if capacity < minimumCapacity {
		capacity = minimumCapacity
	}

	log.Debug("Created open addressing hash table (capacity=%d)", capacity)
	return &OpenAddressingHashTable[V]{
		size:             0,
		used:             0,
		capacity:         capacity,
		buckets:          make([]*OAHashEntry[V], capacity),
		minimumThreshold: 0.25,
		maximumThreshold: 0.75,
		minimumCapacity:  minimumCapacity,
	}
}

// oaFNVHash32 computes the 32-bit FNV-1a hash of a string key.
func oaFNVHash32(key string) uint32 {
	hash := oaFNVOffset32
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= oaFNVPrime32
	}
	return hash
}

// Size returns the number of live (non-deleted) entries in the table.
func (ht *OpenAddressingHashTable[V]) Size() int {
	return ht.size
}

// Capacity returns the current bucket array capacity.
func (ht *OpenAddressingHashTable[V]) Capacity() int {
	return ht.capacity
}

// hash returns the bucket index for a key.
func (ht *OpenAddressingHashTable[V]) hash(key string) int {
	return int(oaFNVHash32(key) % uint32(ht.capacity))
}

// shouldGrow returns true if the load factor exceeds the maximum threshold.
func (ht *OpenAddressingHashTable[V]) shouldGrow() bool {
	return float64(ht.size) > float64(ht.capacity)*ht.maximumThreshold
}

// shouldShrink returns true if the load factor falls below the minimum threshold
// and capacity is above the minimum.
func (ht *OpenAddressingHashTable[V]) shouldShrink() bool {
	return ht.capacity > ht.minimumCapacity &&
		float64(ht.size) < float64(ht.capacity)*ht.minimumThreshold
}

// shouldGrowBeforeInsert returns true if inserting one more entry would exceed the maximum load factor.
func (ht *OpenAddressingHashTable[V]) shouldGrowBeforeInsert() bool {
	return float64(ht.size+1) > float64(ht.capacity)*ht.maximumThreshold
}

// shouldCleanDeletedSlotsBeforeInsert returns true if the used slot count (including tombstones)
// would exceed the maximum threshold, requiring a rehash to clean up deletions.
func (ht *OpenAddressingHashTable[V]) shouldCleanDeletedSlotsBeforeInsert() bool {
	return float64(ht.used+1) > float64(ht.capacity)*ht.maximumThreshold
}

// findSlot probes for a key using linear probing. Returns the index where the key exists,
// or the first available slot (nil or deleted). Time complexity: O(1) average.
func (ht *OpenAddressingHashTable[V]) findSlot(key string) oaProbeResult {
	start := ht.hash(key)
	firstDeleted := -1

	for offset := 0; offset < ht.capacity; offset++ {
		index := (start + offset) % ht.capacity
		entry := ht.buckets[index]

		if entry == nil {
			if firstDeleted != -1 {
				return oaProbeResult{index: firstDeleted, found: false}
			}
			return oaProbeResult{index: index, found: false}
		}

		if entry.deleted {
			if firstDeleted == -1 {
				firstDeleted = index
			}
			continue
		}

		if entry.key == key {
			return oaProbeResult{index: index, found: true}
		}
	}

	return oaProbeResult{index: firstDeleted, found: false}
}

// insertRehashedEntry inserts an entry during a resize operation.
// Panics if no slot is available (should never happen after correct resizing).
func (ht *OpenAddressingHashTable[V]) insertRehashedEntry(key string, value V) {
	slot := ht.findSlot(key)
	if slot.index == -1 {
		log.Error("hash table has no available slot during rehash (capacity=%d, size=%d, used=%d)", ht.capacity, ht.size, ht.used)
		panic("hash table has no available slot during rehash")
	}

	ht.buckets[slot.index] = &OAHashEntry[V]{key: key, value: value}
	ht.size++
	ht.used++
}

// Get retrieves the value for a key using linear probing. Returns the value and true
// if found, or the zero value and false otherwise. Time complexity: O(1) average.
func (ht *OpenAddressingHashTable[V]) Get(key string) (V, bool) {
	var zero V
	start := ht.hash(key)

	for offset := 0; offset < ht.capacity; offset++ {
		index := (start + offset) % ht.capacity
		entry := ht.buckets[index]

		if entry == nil {
			log.Debug("OpenAddressingHashTable miss (key=%q)", key)
			return zero, false
		}

		if !entry.deleted && entry.key == key {
			log.Debug("OpenAddressingHashTable get (key=%q)", key)
			return entry.value, true
		}
	}

	return zero, false
}

// Set inserts or updates a key-value pair. Triggers grow, deleted-slot cleanup,
// or rehash as needed. Time complexity: O(1) average.
func (ht *OpenAddressingHashTable[V]) Set(key string, value V) {
	slot := ht.findSlot(key)
	if slot.found {
		ht.buckets[slot.index].value = value
		log.Debug("OpenAddressingHashTable update (key=%q)", key)
		return
	}

	if ht.shouldGrowBeforeInsert() {
		log.Debug("OpenAddressingHashTable growing before insert (old=%d, new=%d)", ht.capacity, ht.capacity*2)
		ht.resize(ht.capacity * 2)
		slot = ht.findSlot(key)
	} else if ht.shouldCleanDeletedSlotsBeforeInsert() {
		log.Debug("OpenAddressingHashTable cleaning deleted slots (capacity=%d)", ht.capacity)
		ht.resize(ht.capacity)
		slot = ht.findSlot(key)
	}

	if slot.index == -1 {
		log.Debug("OpenAddressingHashTable no slot, forcing grow (old=%d, new=%d)", ht.capacity, ht.capacity*2)
		ht.resize(ht.capacity * 2)
		slot = ht.findSlot(key)
	}
	if slot.index == -1 {
		log.Error("hash table has no available slot (capacity=%d, size=%d, used=%d)", ht.capacity, ht.size, ht.used)
		panic("hash table has no available slot")
	}

	if ht.buckets[slot.index] == nil {
		ht.used++
	}

	ht.buckets[slot.index] = &OAHashEntry[V]{key: key, value: value}
	ht.size++
	log.Debug("OpenAddressingHashTable insert (key=%q, size=%d/%d)", key, ht.size, ht.capacity)

	if ht.shouldGrow() {
		log.Debug("OpenAddressingHashTable growing after insert (old=%d, new=%d)", ht.capacity, ht.capacity*2)
		ht.resize(ht.capacity * 2)
	}
}

// Delete removes a key from the hash table. Uses tombstone deletion (marks the entry as deleted).
// Returns true if the key was found and removed. Time complexity: O(1) average.
func (ht *OpenAddressingHashTable[V]) Delete(key string) bool {
	start := ht.hash(key)

	for offset := 0; offset < ht.capacity; offset++ {
		index := (start + offset) % ht.capacity
		entry := ht.buckets[index]

		if entry == nil {
			log.Warn("OpenAddressingHashTable delete miss (key=%q)", key)
			return false
		}

		if !entry.deleted && entry.key == key {
			var zero V
			entry.key = ""
			entry.value = zero
			entry.deleted = true
			ht.size--
			log.Debug("OpenAddressingHashTable delete (key=%q, size=%d/%d)", key, ht.size, ht.capacity)

			if ht.shouldShrink() {
				log.Debug("OpenAddressingHashTable shrinking (old=%d, new=%d)", ht.capacity, ht.capacity/2)
				ht.resize(ht.capacity / 2)
			}

			return true
		}
	}

	log.Warn("OpenAddressingHashTable delete miss (key=%q)", key)
	return false
}

// resize rehashes all non-deleted entries into a new bucket array of the given capacity.
// Time complexity: O(n) where n is the number of live entries.
func (ht *OpenAddressingHashTable[V]) resize(newCapacity int) {
	if newCapacity < ht.minimumCapacity {
		newCapacity = ht.minimumCapacity
	}

	oldBuckets := ht.buckets
	ht.capacity = newCapacity
	ht.buckets = make([]*OAHashEntry[V], newCapacity)
	ht.size = 0
	ht.used = 0

	for _, entry := range oldBuckets {
		if entry != nil && !entry.deleted {
			ht.insertRehashedEntry(entry.key, entry.value)
		}
	}
}
