package main

import "fmt"

const (
	FNVOffset32 uint32 = 2166136261
	FNVPrime32  uint32 = 16777619
	FNVOffset64 uint64 = 14695981039346656037
	FNVPrime64  uint64 = 1099511628211
)

type KeyValuePair struct {
	key   string
	value string
	next  *KeyValuePair
}

type BucketHashTable struct {
	size             int
	capacity         int
	bucket           []*KeyValuePair
	minimumThreshold float64
	maximumThreshold float64
	minimumCapacity  int
}

func FNVHash32(key string) uint32 {
	hash := FNVOffset32
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= FNVPrime32
	}
	return hash
}

func FNVHash64(key string) uint64 {
	hash := FNVOffset64
	for i := 0; i < len(key); i++ {
		hash ^= uint64(key[i])
		hash *= FNVPrime64
	}
	return hash
}

func NewBucketHashTable(capacity int) *BucketHashTable {
	if capacity < 16 {
		capacity = 16
	}
	return &BucketHashTable{
		size:             0,
		capacity:         capacity,
		bucket:           make([]*KeyValuePair, capacity),
		minimumThreshold: 0.25,
		maximumThreshold: 0.75,
		minimumCapacity:  16,
	}
}

func (ht *BucketHashTable) Hash(key string, size int) int {
	if size == 32 {
		return int(FNVHash32(key)) % ht.capacity
	}
	return int(FNVHash64(key) % uint64(ht.capacity))
}

func (ht *BucketHashTable) shouldGrow() bool {
	threshold := int(float64(ht.capacity) * ht.maximumThreshold)
	return ht.size > threshold
}

func (ht *BucketHashTable) shouldShrink() bool {
	threshold := int(float64(ht.capacity) * ht.minimumThreshold)
	return ht.capacity > ht.minimumCapacity && ht.size < threshold
}

func (ht *BucketHashTable) Set(key string, value string) {
	index := ht.Hash(key, 32)
	entry := ht.bucket[index]

	// update existing key
	for entry != nil {
		if entry.key == key {
			entry.value = value
			return
		}
		entry = entry.next
	}

	// insert at head (same as TS)
	newEntry := &KeyValuePair{
		key:   key,
		value: value,
		next:  ht.bucket[index],
	}
	ht.bucket[index] = newEntry
	ht.size++

	if ht.shouldGrow() {
		ht.resize(ht.capacity * 2)
	}
}

func (ht *BucketHashTable) Get(key string) (string, bool) {
	index := ht.Hash(key, 32)
	entry := ht.bucket[index]
	for entry != nil {
		if entry.key == key {
			return entry.value, true
		}
		entry = entry.next
	}
	return "", false
}

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

			if ht.shouldShrink() {
				ht.resize(ht.capacity / 2)
			}
			return true
		}
		prev = entry
		entry = entry.next
	}
	return false
}

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

func main() {
	ht := NewBucketHashTable(16)

	fmt.Println("===== BASIC INSERT TEST =====")
	ht.Set("one", "1")
	ht.Set("two", "2")
	ht.Set("three", "3")
	v, ok := ht.Get("one")
	fmt.Println("one:", v, "exists:", ok)
	v, ok = ht.Get("two")
	fmt.Println("two:", v, "exists:", ok)
	v, ok = ht.Get("three")
	fmt.Println("three:", v, "exists:", ok)
	_, ok = ht.Get("four")
	fmt.Println("four exists:", ok)

	fmt.Println("\n===== UPDATE TEST =====")
	ht.Set("one", "100")
	v, _ = ht.Get("one")
	fmt.Println("one updated:", v)

	fmt.Println("\n===== DELETE TEST =====")
	ht.Delete("two")
	_, exists := ht.Get("two")
	fmt.Println("two exists after delete:", exists)

	fmt.Println("\n===== RE-INSERT AFTER DELETE =====")
	ht.Set("two", "222")
	v, _ = ht.Get("two")
	fmt.Println("two re-inserted:", v)

	fmt.Println("\n===== COLLISION TEST =====")
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
	fmt.Println("Collision test passed:", passed)

	fmt.Println("\n===== LARGE INSERT (GROW TEST) =====")
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
	fmt.Println("Large insert test:", valid)

	fmt.Println("\n===== DELETE MANY (SHRINK TEST) =====")
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
	fmt.Println("Shrink integrity test:", valid)

	fmt.Println("\n===== DELETE NON-EXISTENT =====")
	fmt.Println("delete non-existent:", ht.Delete("does_not_exist"))

	fmt.Println("\n===== RANDOM STRESS TEST =====")
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
	fmt.Println("Random stress test:", randomValid)

	fmt.Println("\n===== DELETE ALL TEST =====")
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
	fmt.Println("Delete all test:", allDeleted)

	fmt.Println("\n===== EDGE CASES =====")
	ht.Set("", "empty")
	v, _ = ht.Get("")
	fmt.Println("Empty string key:", v)
	ht.Set("0", "zero")
	v, _ = ht.Get("0")
	fmt.Println("Zero key value:", v)

	fmt.Println("\n===== FINAL STATUS =====")
	fmt.Printf("Final size: %d, capacity: %d\n", ht.size, ht.capacity)
	fmt.Println("All tests completed. O(1) average-case achieved via hashing + chaining + dynamic resizing.")
}
