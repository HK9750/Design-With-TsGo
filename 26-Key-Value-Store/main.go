package main

import (
	"fmt"
	"sort"
	"time"
)

type KVEntry struct {
	Value     string
	ExpiresAt time.Time
	Version   int64
}

type KeyValueStore struct {
	data    map[string]KVEntry
	version int64
}

func NewKeyValueStore() *KeyValueStore { return &KeyValueStore{data: make(map[string]KVEntry)} }

func (s *KeyValueStore) Put(key, value string, ttl time.Duration) int64 {
	s.version++
	entry := KVEntry{Value: value, Version: s.version}
	if ttl > 0 {
		entry.ExpiresAt = time.Now().Add(ttl)
	}
	s.data[key] = entry
	return entry.Version
}

func (s *KeyValueStore) Get(key string) (string, bool) {
	entry, ok := s.data[key]
	if !ok {
		return "", false
	}
	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		delete(s.data, key)
		return "", false
	}
	return entry.Value, true
}

func (s *KeyValueStore) Delete(key string) bool {
	_, ok := s.data[key]
	delete(s.data, key)
	return ok
}

func (s *KeyValueStore) Range(start, end string) [][2]string {
	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var result [][2]string
	for _, key := range keys {
		if key >= start && key <= end {
			if value, ok := s.Get(key); ok {
				result = append(result, [2]string{key, value})
			}
		}
	}
	return result
}

func main() {
	store := NewKeyValueStore()
	store.Put("a", "1", 0)
	store.Put("b", "2", 0)
	fmt.Println(store.Range("a", "z"))
}
