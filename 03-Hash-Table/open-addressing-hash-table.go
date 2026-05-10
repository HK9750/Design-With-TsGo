package main

const (
	oaFNVOffset32 uint32 = 2166136261
	oaFNVPrime32  uint32 = 16777619
)

type OAHashEntry[V any] struct {
	key     string
	value   V
	deleted bool
}

type oaProbeResult struct {
	index int
	found bool
}

type OpenAddressingHashTable[V any] struct {
	size             int
	used             int
	capacity         int
	buckets          []*OAHashEntry[V]
	minimumThreshold float64
	maximumThreshold float64
	minimumCapacity  int
}

func NewOpenAddressingHashTable[V any](capacity int) *OpenAddressingHashTable[V] {
	minimumCapacity := 16
	if capacity < minimumCapacity {
		capacity = minimumCapacity
	}

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

func oaFNVHash32(key string) uint32 {
	hash := oaFNVOffset32
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= oaFNVPrime32
	}
	return hash
}

func (ht *OpenAddressingHashTable[V]) Size() int {
	return ht.size
}

func (ht *OpenAddressingHashTable[V]) Capacity() int {
	return ht.capacity
}

func (ht *OpenAddressingHashTable[V]) hash(key string) int {
	return int(oaFNVHash32(key) % uint32(ht.capacity))
}

func (ht *OpenAddressingHashTable[V]) shouldGrow() bool {
	return float64(ht.size) > float64(ht.capacity)*ht.maximumThreshold
}

func (ht *OpenAddressingHashTable[V]) shouldShrink() bool {
	return ht.capacity > ht.minimumCapacity &&
		float64(ht.size) < float64(ht.capacity)*ht.minimumThreshold
}

func (ht *OpenAddressingHashTable[V]) shouldGrowBeforeInsert() bool {
	return float64(ht.size+1) > float64(ht.capacity)*ht.maximumThreshold
}

func (ht *OpenAddressingHashTable[V]) shouldCleanDeletedSlotsBeforeInsert() bool {
	return float64(ht.used+1) > float64(ht.capacity)*ht.maximumThreshold
}

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

func (ht *OpenAddressingHashTable[V]) insertRehashedEntry(key string, value V) {
	slot := ht.findSlot(key)
	if slot.index == -1 {
		panic("hash table has no available slot during rehash")
	}

	ht.buckets[slot.index] = &OAHashEntry[V]{key: key, value: value}
	ht.size++
	ht.used++
}

func (ht *OpenAddressingHashTable[V]) Get(key string) (V, bool) {
	var zero V
	start := ht.hash(key)

	for offset := 0; offset < ht.capacity; offset++ {
		index := (start + offset) % ht.capacity
		entry := ht.buckets[index]

		if entry == nil {
			return zero, false
		}

		if !entry.deleted && entry.key == key {
			return entry.value, true
		}
	}

	return zero, false
}

func (ht *OpenAddressingHashTable[V]) Set(key string, value V) {
	slot := ht.findSlot(key)
	if slot.found {
		ht.buckets[slot.index].value = value
		return
	}

	if ht.shouldGrowBeforeInsert() {
		ht.resize(ht.capacity * 2)
		slot = ht.findSlot(key)
	} else if ht.shouldCleanDeletedSlotsBeforeInsert() {
		ht.resize(ht.capacity)
		slot = ht.findSlot(key)
	}

	if slot.index == -1 {
		ht.resize(ht.capacity * 2)
		slot = ht.findSlot(key)
	}
	if slot.index == -1 {
		panic("hash table has no available slot")
	}

	if ht.buckets[slot.index] == nil {
		ht.used++
	}

	ht.buckets[slot.index] = &OAHashEntry[V]{key: key, value: value}
	ht.size++

	if ht.shouldGrow() {
		ht.resize(ht.capacity * 2)
	}
}

func (ht *OpenAddressingHashTable[V]) Delete(key string) bool {
	start := ht.hash(key)

	for offset := 0; offset < ht.capacity; offset++ {
		index := (start + offset) % ht.capacity
		entry := ht.buckets[index]

		if entry == nil {
			return false
		}

		if !entry.deleted && entry.key == key {
			var zero V
			entry.key = ""
			entry.value = zero
			entry.deleted = true
			ht.size--

			if ht.shouldShrink() {
				ht.resize(ht.capacity / 2)
			}

			return true
		}
	}

	return false
}

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
