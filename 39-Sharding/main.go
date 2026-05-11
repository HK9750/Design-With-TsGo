package main

import (
	"design-with-tsgo/pkg/logger"
	"os"
	"time"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// ShardedStore distributes key-value pairs across multiple shards using
// a consistent FNV-1a hash function. Each shard is an independent
// in-memory map, enabling horizontal scaling by adding more shards.
type ShardedStore struct{ shards []map[string]string }

// NewShardedStore creates a store with the given number of shards.
// Each shard is initialized as an empty map. Panics if shardCount <= 0.
// Time complexity: O(shardCount).
func NewShardedStore(shardCount int) *ShardedStore {
	defer log.Operation("NewShardedStore", "Creating sharded store with %d shards", shardCount)()
	if shardCount <= 0 {
		log.Error("NewShardedStore: shardCount must be positive, got %d", shardCount)
		os.Exit(1)
	}
	shards := make([]map[string]string, shardCount)
	for i := range shards {
		shards[i] = make(map[string]string)
	}
	return &ShardedStore{shards: shards}
}

// Put stores a key-value pair, routing the key to the correct shard
// based on its hash. Time complexity: O(len(key)) for hashing + O(1) for insertion.
func (s *ShardedStore) Put(key, value string) {
	idx := s.ShardIndex(key)
	s.shards[idx][key] = value
	log.Debug("Put: key=%s value=%s shard=%d/%d", key, value, idx, len(s.shards))
}

// Get retrieves the value for a key from its assigned shard.
// Returns the value and a boolean indicating if the key exists.
// Time complexity: O(len(key)) for hashing + O(1) for lookup.
func (s *ShardedStore) Get(key string) (string, bool) {
	idx := s.ShardIndex(key)
	value, ok := s.shards[idx][key]
	if !ok {
		log.Warn("Key %s not found in shard %d", key, idx)
	} else {
		log.Debug("Get: key=%s value=%s shard=%d", key, value, idx)
	}
	return value, ok
}

// ShardIndex computes which shard a key belongs to using the FNV-1a hash
// of the key modulo the number of shards.
// Time complexity: O(len(key)).
func (s *ShardedStore) ShardIndex(key string) int {
	return int(hash32(key) % uint32(len(s.shards)))
}

// hash32 computes a 32-bit FNV-1a hash of the given string.
// This is a non-cryptographic hash suitable for shard distribution.
// Time complexity: O(n) where n is the length of the string.
func hash32(value string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(value); i++ {
		hash ^= uint32(value[i])
		hash *= 16777619
	}
	return hash
}

func main() {
	defer log.Operation("main", "Running Sharding demo")()

	logger.Section("SHARDING — Production Scenario")

	// Simulate a distributed user session store with 8 shards
	log.Info("Initializing sharded session store with 8 shards")
	start := time.Now()
	store := NewShardedStore(8)
	log.Info("Store initialized in %v", time.Since(start))

	logger.Section("SCENARIO: Distributing Users Across Shards")
	log.Info("Inserting user sessions into the sharded store")
	users := []struct{ id, name string }{
		{"user:1", "Ada"},
		{"user:2", "Bob"},
		{"user:3", "Charlie"},
		{"user:4", "Diana"},
		{"user:5", "Eve"},
		{"user:42", "Zara"},
		{"user:100", "Oscar"},
		{"user:256", "Mallory"},
		{"session:abc", "token-xyz"},
		{"session:def", "token-pqr"},
	}

	writeStart := time.Now()
	for _, u := range users {
		store.Put(u.id, u.name)
		log.Info("User %s -> shard %d", u.id, store.ShardIndex(u.id))
	}
	log.Info("Inserted %d users in %v", len(users), time.Since(writeStart))

	logger.Section("SCENARIO: Reading from Shards")
	log.Info("Looking up specific users")
	for _, u := range users[:3] {
		if val, ok := store.Get(u.id); ok {
			logger.KeyValue(u.id, val)
		}
	}

	logger.Section("SCENARIO: Missing Key Lookup")
	log.Info("Attempting to read a non-existent user")
	if _, ok := store.Get("user:9999"); !ok {
		log.Info("Correctly returned not found for missing key")
	}

	// Shard distribution analysis
	logger.Section("SCENARIO: Shard Distribution Analysis")
	shardCounts := make([]int, len(store.shards))
	for _, u := range users {
		shardCounts[store.ShardIndex(u.id)]++
	}
	for i, count := range shardCounts {
		log.Info("Shard %d: %d keys (%.1f%%)", i, count, float64(count)/float64(len(users))*100)
	}

	// Final stats summary
	logger.Section("FINAL STATS")
	logger.KeyValue("total_shards", len(store.shards))
	logger.KeyValue("total_keys", len(users))
	totalKeys := 0
	maxShard := 0
	minShard := len(users)
	for i := range store.shards {
		s := len(store.shards[i])
		totalKeys += s
		if s > maxShard {
			maxShard = s
		}
		if s < minShard {
			minShard = s
		}
	}
	logger.KeyValue("total_keys_stored", totalKeys)
	logger.KeyValue("max_shard_occupancy", maxShard)
	logger.KeyValue("min_shard_occupancy", minShard)
	logger.KeyValue("distribution_skew", maxShard-minShard)
	log.Info("Sharding demo completed successfully")
}
