package main

import (
	"design-with-tsgo/pkg/logger"
	"os"
	"time"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// PrimaryReplicaStore implements a primary-backup replication strategy.
// The primary stores all writes and synchronously replicates them to
// N replicas. Reads can be served from the primary or the first replica.
type PrimaryReplicaStore struct {
	primary  map[string]string
	replicas []map[string]string
}

// NewPrimaryReplicaStore creates a store with one primary and the given
// number of replicas, each initialized as an empty map.
// Time complexity: O(replicaCount).
func NewPrimaryReplicaStore(replicaCount int) *PrimaryReplicaStore {
	defer log.Operation("NewPrimaryReplicaStore", "Creating store with %d replicas", replicaCount)()
	replicas := make([]map[string]string, replicaCount)
	for i := range replicas {
		replicas[i] = make(map[string]string)
	}
	return &PrimaryReplicaStore{primary: make(map[string]string), replicas: replicas}
}

// Write sets a key-value pair on the primary and synchronously replicates
// it to ALL replicas. Time complexity: O(r) where r is the number of replicas.
func (s *PrimaryReplicaStore) Write(key, value string) {
	defer log.Operation("Write", "Writing key=%s value=%s to primary and %d replicas", key, value, len(s.replicas))()
	s.primary[key] = value
	for i, replica := range s.replicas {
		replica[key] = value
		log.Debug("Replicated to replica-%d: %s=%s", i, key, value)
	}
	log.Info("Wrote %s=%s (replicated to %d replicas)", key, value, len(s.replicas))
}

// Read retrieves a value by key. If fromReplica is true, reads from the
// first replica; otherwise reads from the primary. Returns the value and
// a boolean indicating if the key exists. Time complexity: O(1).
func (s *PrimaryReplicaStore) Read(key string, fromReplica bool) (string, bool) {
	if fromReplica && len(s.replicas) > 0 {
		value, ok := s.replicas[0][key]
		if !ok {
			log.Warn("Key %s not found on replica-0", key)
		} else {
			log.Debug("Read %s=%s from replica-0", key, value)
		}
		return value, ok
	}
	value, ok := s.primary[key]
	if !ok {
		log.Warn("Key %s not found on primary", key)
	} else {
		log.Debug("Read %s=%s from primary", key, value)
	}
	return value, ok
}

// Failover promotes a replica to become the new primary. The specified
// replica's data is copied to a new primary map. Returns false if the
// replica index is out of bounds. Time complexity: O(d) where d is the
// number of keys in the replica.
func (s *PrimaryReplicaStore) Failover(replicaIndex int) bool {
	defer log.Operation("Failover", "Promoting replica-%d to primary", replicaIndex)()
	if replicaIndex < 0 || replicaIndex >= len(s.replicas) {
		log.Warn("Failover failed: replica index %d out of range [0, %d)", replicaIndex, len(s.replicas))
		return false
	}
	s.primary = make(map[string]string)
	for key, value := range s.replicas[replicaIndex] {
		s.primary[key] = value
		log.Debug("Failover: copied %s=%s from replica-%d to new primary", key, value, replicaIndex)
	}
	log.Info("Failover complete: replica-%d promoted to primary (%d keys)", replicaIndex, len(s.primary))
	return true
}

func main() {
	defer log.Operation("main", "Running Primary-Replica Replication demo")()

	logger.Section("PRIMARY-REPLICA REPLICATION — Production Scenario")

	// Simulate a configuration store with 3 replicas
	log.Info("Initializing primary-replica store with 3 replicas")
	start := time.Now()
	store := NewPrimaryReplicaStore(3)
	log.Info("Store initialized in %v", time.Since(start))

	logger.Section("SCENARIO: Bulk Write and Multi-Source Read")
	log.Info("Loading initial configuration into the store")
	writeStart := time.Now()
	configs := map[string]string{
		"db.host":           "db-primary.internal",
		"db.port":           "5432",
		"cache.ttl":         "300",
		"rate.limit":        "1000",
		"feature.new_ui":    "true",
		"queue.max_workers": "10",
	}
	for key, value := range configs {
		store.Write(key, value)
	}
	log.Info("Bulk write of %d keys completed in %v", len(configs), time.Since(writeStart))

	logger.Section("SCENARIO: Reading from Primary vs Replica")
	log.Info("Reading 'db.host' from primary")
	if val, ok := store.Read("db.host", false); ok {
		logger.KeyValue("primary_db_host", val)
	}

	log.Info("Reading 'db.host' from replica-0 (read scale-out)")
	if val, ok := store.Read("db.host", true); ok {
		logger.KeyValue("replica_db_host", val)
	}

	logger.Section("SCENARIO: Failover — Primary Crashes")
	log.Info("Primary crashes — operator initiates failover to replica-2")
	if store.Failover(2) {
		log.Info("Failover successful — replica-2 is now the new primary")
	} else {
		log.Error("Failover failed")
	}

	log.Info("Verifying data integrity on new primary after failover")
	if val, ok := store.Read("cache.ttl", false); ok {
		logger.KeyValue("cache_ttl_after_failover", val)
	}

	logger.Section("SCENARIO: Edge Cases")
	log.Info("Attempting failover with invalid index")
	if !store.Failover(99) {
		log.Info("Correctly rejected invalid failover index")
	}

	log.Info("Reading non-existent key from replica")
	if _, ok := store.Read("nonexistent_key", true); !ok {
		log.Info("Correctly returned false for missing key")
	}

	// Final stats summary
	logger.Section("FINAL STATS")
	logger.KeyValue("primary_keys", len(store.primary))
	logger.KeyValue("replica_count", len(store.replicas))
	for i, replica := range store.replicas {
		logger.KeyValue("replica_"+string(rune('0'+i))+"_keys", len(replica))
	}
	logger.KeyValue("total_writes", len(configs))
	log.Info("Primary-replica replication demo completed successfully")
}
