package main

import "fmt"

type ShardedStore struct{ shards []map[string]string }

func NewShardedStore(shardCount int) *ShardedStore {
	if shardCount <= 0 {
		panic("shardCount must be positive")
	}
	shards := make([]map[string]string, shardCount)
	for i := range shards {
		shards[i] = make(map[string]string)
	}
	return &ShardedStore{shards: shards}
}

func (s *ShardedStore) Put(key, value string) { s.shards[s.ShardIndex(key)][key] = value }

func (s *ShardedStore) Get(key string) (string, bool) {
	value, ok := s.shards[s.ShardIndex(key)][key]
	return value, ok
}

func (s *ShardedStore) ShardIndex(key string) int { return int(hash32(key) % uint32(len(s.shards))) }

func hash32(value string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(value); i++ {
		hash ^= uint32(value[i])
		hash *= 16777619
	}
	return hash
}

func main() {
	store := NewShardedStore(4)
	store.Put("user:1", "Ada")
	value, _ := store.Get("user:1")
	fmt.Println(value, store.ShardIndex("user:1"))
}
