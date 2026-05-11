// Package main demonstrates a hash table with separate chaining (bucket collision resolution).
// Supports dynamic resizing (grow/shrink), collision handling via linked lists,
// and FNV-1a hashing. Suitable as a key-value store for session management.
package main

import (
	"fmt"
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// FNVOffset32 is the FNV-1a 32-bit offset basis.
const FNVOffset32 uint32 = 2166136261

// FNVPrime32 is the FNV-1a 32-bit prime.
const FNVPrime32 uint32 = 16777619

// FNVOffset64 is the FNV-1a 64-bit offset basis.
const FNVOffset64 uint64 = 14695981039346656037

// FNVPrime64 is the FNV-1a 64-bit prime.
const FNVPrime64 uint64 = 1099511628211

// KeyValuePair represents a single entry in a bucket's linked list chain.
type KeyValuePair struct {
	key   string
	value string
	next  *KeyValuePair
}

// BucketHashTable is a hash table using separate chaining for collision resolution.
// It supports dynamic resizing based on load factor thresholds.
type BucketHashTable struct {
	size             int
	capacity         int
	bucket           []*KeyValuePair
	minimumThreshold float64
	maximumThreshold float64
	minimumCapacity  int
}

// FNVHash32 computes the 32-bit FNV-1a hash of a string key.
func FNVHash32(key string) uint32 {
	hash := FNVOffset32
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= FNVPrime32
	}
	return hash
}

// FNVHash64 computes the 64-bit FNV-1a hash of a string key.
func FNVHash64(key string) uint64 {
	hash := FNVOffset64
	for i := 0; i < len(key); i++ {
		hash ^= uint64(key[i])
		hash *= FNVPrime64
	}
	return hash
}

// NewBucketHashTable creates a new bucket hash table with the given initial capacity.
// Minimum capacity is clamped to 16. Time complexity: O(capacity).
func NewBucketHashTable(capacity int) *BucketHashTable {
	if capacity < 16 {
		capacity = 16
	}
	log.Debug("Created bucket hash table (capacity=%d)", capacity)
	return &BucketHashTable{
		size:             0,
		capacity:         capacity,
		bucket:           make([]*KeyValuePair, capacity),
		minimumThreshold: 0.25,
		maximumThreshold: 0.75,
		minimumCapacity:  16,
	}
}

// Hash returns the bucket index for a key using the specified hash size (32 or 64 bit).
func (ht *BucketHashTable) Hash(key string, size int) int {
	if size == 32 {
		return int(FNVHash32(key)) % ht.capacity
	}
	return int(FNVHash64(key) % uint64(ht.capacity))
}

// shouldGrow returns true if the load factor exceeds the maximum threshold.
func (ht *BucketHashTable) shouldGrow() bool {
	threshold := int(float64(ht.capacity) * ht.maximumThreshold)
	return ht.size > threshold
}

// shouldShrink returns true if the load factor falls below the minimum threshold
// and capacity is above the minimum.
func (ht *BucketHashTable) shouldShrink() bool {
	threshold := int(float64(ht.capacity) * ht.minimumThreshold)
	return ht.capacity > ht.minimumCapacity && ht.size < threshold
}

// Set inserts or updates a key-value pair. Triggers resize (grow) when load factor
// exceeds the maximum threshold. Time complexity: O(1) average, O(n) worst case.
func (ht *BucketHashTable) Set(key string, value string) {
	index := ht.Hash(key, 32)
	entry := ht.bucket[index]

	for entry != nil {
		if entry.key == key {
			log.Debug("BucketHashTable update (key=%q, old=%q, new=%q)", key, entry.value, value)
			entry.value = value
			return
		}
		entry = entry.next
	}

	newEntry := &KeyValuePair{
		key:   key,
		value: value,
		next:  ht.bucket[index],
	}
	ht.bucket[index] = newEntry
	ht.size++
	log.Debug("BucketHashTable insert (key=%q, value=%q, size=%d/%d)", key, value, ht.size, ht.capacity)

	if ht.shouldGrow() {
		log.Debug("BucketHashTable growing (old_capacity=%d, new_capacity=%d)", ht.capacity, ht.capacity*2)
		ht.resize(ht.capacity * 2)
	}
}

// Get retrieves the value for a key. Returns the value and true if found,
// or empty string and false otherwise. Time complexity: O(1) average.
func (ht *BucketHashTable) Get(key string) (string, bool) {
	index := ht.Hash(key, 32)
	entry := ht.bucket[index]
	for entry != nil {
		if entry.key == key {
			log.Debug("BucketHashTable get (key=%q, value=%q)", key, entry.value)
			return entry.value, true
		}
		entry = entry.next
	}
	log.Debug("BucketHashTable miss (key=%q)", key)
	return "", false
}

// Delete removes a key-value pair from the hash table. Returns true if the key was found
// and removed. Triggers resize (shrink) when load factor falls below the minimum threshold.
// Time complexity: O(1) average.
func (ht *BucketHashTable) Delete(key string) bool {
	index := ht.Hash(key, 32)
	entry := ht.bucket[index]
	var prev *KeyValuePair

	for entry != nil {
		if entry.key == key {
			if prev != nil {
				prev.next = entry.next
			} else {
				ht.bucket[index] = entry.next
			}
			ht.size--
			log.Debug("BucketHashTable delete (key=%q, size=%d/%d)", key, ht.size, ht.capacity)

			if ht.shouldShrink() {
				log.Debug("BucketHashTable shrinking (old_capacity=%d, new_capacity=%d)", ht.capacity, ht.capacity/2)
				ht.resize(ht.capacity / 2)
			}
			return true
		}
		prev = entry
		entry = entry.next
	}
	log.Warn("BucketHashTable delete miss (key=%q)", key)
	return false
}

// resize rehashes all entries into a new bucket array of the given capacity.
// Entries are re-inserted via head insertion to preserve order relative to their chains.
// Time complexity: O(n) where n is the number of entries.
func (ht *BucketHashTable) resize(newCapacity int) {
	if newCapacity < ht.minimumCapacity {
		newCapacity = ht.minimumCapacity
	}

	oldBuckets := ht.bucket
	ht.capacity = newCapacity
	ht.bucket = make([]*KeyValuePair, newCapacity)
	ht.size = 0

	for _, head := range oldBuckets {
		entry := head
		for entry != nil {
			index := ht.Hash(entry.key, 32)
			newEntry := &KeyValuePair{
				key:   entry.key,
				value: entry.value,
				next:  ht.bucket[index],
			}
			ht.bucket[index] = newEntry
			ht.size++
			entry = entry.next
		}
	}
}

// main demonstrates the bucket hash table as a session store for a web application,
// handling user sessions with dynamic scaling under load.
func main() {
	defer log.Operation("main", "Running Bucket Hash Table demo")()

	logger.Section("Bucket Hash Table — Web Session Store")
	ht := NewBucketHashTable(16)
	logger.KeyValue("initial_capacity", 16)

	logger.Section("Basic CRUD operations")
	ht.Set("one", "1")
	ht.Set("two", "2")
	ht.Set("three", "3")
	v, ok := ht.Get("one")
	log.Info("one=%q exists=%v", v, ok)
	v, ok = ht.Get("two")
	log.Info("two=%q exists=%v", v, ok)
	v, ok = ht.Get("three")
	log.Info("three=%q exists=%v", v, ok)
	_, ok = ht.Get("four")
	log.Info("four exists: %v", ok)

	logger.Section("Update existing session")
	ht.Set("one", "100")
	v, _ = ht.Get("one")
	log.Info("one updated: %q", v)

	logger.Section("Delete and re-insert")
	ht.Delete("two")
	_, exists := ht.Get("two")
	log.Info("two exists after delete: %v", exists)
	ht.Set("two", "222")
	v, _ = ht.Get("two")
	log.Info("two re-inserted: %q", v)

	logger.Section("Collision handling — 50 sessions with similar key patterns")
	start := time.Now()
	for i := 0; i < 50; i++ {
		k := fmt.Sprintf("collision_%d", i)
		ht.Set(k, fmt.Sprintf("%d", i))
	}
	passed := true
	for i := 0; i < 50; i++ {
		k := fmt.Sprintf("collision_%d", i)
		v, ok := ht.Get(k)
		if !ok || v != fmt.Sprintf("%d", i) {
			passed = false
			break
		}
	}
	log.Info("Collision test passed: %v (completed in %v)", passed, time.Since(start))

	logger.Section("Load test — 10,000 concurrent user sessions")
	start = time.Now()
	largeCount := 10000
	for i := 0; i < largeCount; i++ {
		k := fmt.Sprintf("key_%d", i)
		ht.Set(k, fmt.Sprintf("%d", i))
	}
	valid := true
	for i := 0; i < largeCount; i++ {
		k := fmt.Sprintf("key_%d", i)
		v, ok := ht.Get(k)
		if !ok || v != fmt.Sprintf("%d", i) {
			valid = false
			break
		}
	}
	log.Info("Large insert test: %v (10,000 entries in %v)", valid, time.Since(start))

	logger.Section("Session expiry — bulk delete and shrink")
	for i := 0; i < largeCount-100; i++ {
		ht.Delete(fmt.Sprintf("key_%d", i))
	}
	valid = true
	for i := largeCount - 100; i < largeCount; i++ {
		k := fmt.Sprintf("key_%d", i)
		v, ok := ht.Get(k)
		if !ok || v != fmt.Sprintf("%d", i) {
			valid = false
			break
		}
	}
	log.Info("Shrink integrity test: %v", valid)

	logger.Section("Edge cases")
	log.Info("delete non-existent: %v", ht.Delete("does_not_exist"))
	ht.Set("", "empty")
	v, _ = ht.Get("")
	log.Info("Empty string key: %q", v)
	ht.Set("0", "zero")
	v, _ = ht.Get("0")
	log.Info("Zero key value: %q", v)

	logger.Section("Random stress — 5,000 random sessions")
	start = time.Now()
	randomKeys := make([]string, 0, 5000)
	for i := 0; i < 5000; i++ {
		k := fmt.Sprintf("rand_%d", i)
		randomKeys = append(randomKeys, k)
		ht.Set(k, fmt.Sprintf("%d", i))
	}
	randomValid := true
	for _, k := range randomKeys {
		_, ok := ht.Get(k)
		if !ok {
			randomValid = false
			break
		}
	}
	log.Info("Random stress test: %v (5,000 entries in %v)", randomValid, time.Since(start))

	logger.Section("Cleanup — delete all sessions")
	for _, k := range randomKeys {
		ht.Delete(k)
	}
	allDeleted := true
	for _, k := range randomKeys {
		if _, ok := ht.Get(k); ok {
			allDeleted = false
			break
		}
	}
	log.Info("Delete all test: %v", allDeleted)

	logger.Section("Final Stats")
	log.Info("Final size: %d, capacity: %d", ht.size, ht.capacity)
	log.Info("All tests completed — O(1) average via hashing + chaining + dynamic resizing")
}
