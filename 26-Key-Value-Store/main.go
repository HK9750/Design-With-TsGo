package main

import (
	"os"
	"sort"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// KVEntry represents a stored value with optional expiration and a monotonic
// version number for optimistic concurrency control.
type KVEntry struct {
	Value     string
	ExpiresAt time.Time
	Version   int64
}

// KeyValueStore is an in-memory key-value store with support for TTL-based
// expiration, versioning, range queries, and delete operations. Expired keys
// are lazily evicted on read.
// Time complexity: O(1) for Put/Get/Delete, O(n log n) for Range.
type KeyValueStore struct {
	data    map[string]KVEntry
	version int64
}

// NewKeyValueStore creates an empty key-value store with version counter at 0.
// Time complexity: O(1).
func NewKeyValueStore() *KeyValueStore {
	log.Debug("Created KeyValueStore")
	return &KeyValueStore{data: make(map[string]KVEntry)}
}

// Put stores a value under the given key with an optional time-to-live duration.
// If ttl > 0, the key automatically expires after the duration. Returns the
// new version number for optimistic concurrency control.
// Time complexity: O(1).
func (s *KeyValueStore) Put(key, value string, ttl time.Duration) int64 {
	defer log.Operation("KeyValueStore.Put", "key=%s value=%s ttl=%v", key, value, ttl)()
	s.version++
	entry := KVEntry{Value: value, Version: s.version}
	if ttl > 0 {
		entry.ExpiresAt = time.Now().Add(ttl)
		log.Debug("Key %s expires at %v", key, entry.ExpiresAt)
	}
	s.data[key] = entry
	log.Debug("Stored at version %d", entry.Version)
	return entry.Version
}

// Get retrieves the value for a key. Returns the value and true if the key
// exists and has not expired. Expired keys are lazily evicted and return false.
// Time complexity: O(1).
func (s *KeyValueStore) Get(key string) (string, bool) {
	defer log.Operation("KeyValueStore.Get", "key=%s", key)()
	entry, ok := s.data[key]
	if !ok {
		log.Warn("Key %q not found", key)
		return "", false
	}
	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		log.Debug("Key %q expired at %v, evicting", key, entry.ExpiresAt)
		delete(s.data, key)
		return "", false
	}
	log.Debug("Returning value version=%d", entry.Version)
	return entry.Value, true
}

// Delete removes a key from the store. Returns true if the key existed,
// false if it was not present.
// Time complexity: O(1).
func (s *KeyValueStore) Delete(key string) bool {
	defer log.Operation("KeyValueStore.Delete", "key=%s", key)()
	_, ok := s.data[key]
	delete(s.data, key)
	if !ok {
		log.Warn("Delete: key %q not found (no-op)", key)
	}
	return ok
}

// Range returns all key-value pairs in lexicographic order within [start, end].
// Expired keys are lazily evicted during the scan and excluded from results.
// Time complexity: O(n log n) where n is the number of keys.
func (s *KeyValueStore) Range(start, end string) [][2]string {
	defer log.Operation("KeyValueStore.Range", "start=%q end=%q", start, end)()
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
	log.Info("Range query returned %d pairs", len(result))
	return result
}

func main() {
	defer log.Operation("main", "Running Key-Value Store demo")()
	defer log.Info("Key-Value Store demo completed")

	logger.Section("Key-Value Store — Distributed Cache Backend")
	log.Info("Simulating a configuration service with TTL-based session management")
	log.Info("Keys support versioning for optimistic concurrency and range scans for listing")

	store := NewKeyValueStore()

	logger.Section("Populating Configuration Data")
	store.Put("app:config:db_host", "postgres.internal:5432", 0)
	store.Put("app:config:max_conn", "100", 0)
	store.Put("app:config:log_level", "debug", 0)
	store.Put("app:feature:dark_mode", "enabled", 0)
	store.Put("app:feature:beta_ui", "disabled", 0)
	store.Put("session:abc123", "user:42", 100*time.Millisecond)

	logger.Section("Point Reads — Retrieving Config Values")
	val, ok := store.Get("app:config:db_host")
	if ok {
		log.Info("db_host = %s", val)
	}
	val, ok = store.Get("app:config:max_conn")
	if ok {
		log.Info("max_conn = %s", val)
	}

	logger.Section("Range Scan — Listing All App Configuration")
	results := store.Range("app:config:", "app:config:~")
	log.Info("App configuration entries:")
	for _, pair := range results {
		log.Info("  %s = %s", pair[0], pair[1])
	}

	logger.Section("TTL Expiration — Session Timeout")
	log.Info("Waiting for session token to expire...")
	time.Sleep(150 * time.Millisecond)
	val, ok = store.Get("session:abc123")
	if ok {
		log.Warn("Session still valid unexpectedly: %s", val)
	} else {
		log.Info("Session expired and evicted (expected)")
	}

	logger.Section("Delete — Removing a Deprecated Feature Flag")
	deleted := store.Delete("app:feature:beta_ui")
	if deleted {
		log.Info("Feature flag 'beta_ui' removed")
	}
	_, ok = store.Get("app:feature:beta_ui")
	if !ok {
		log.Info("Confirmed: 'beta_ui' no longer exists")
	}

	logger.Section("Versioning — Checking Write Versions")
	dbVer := store.Put("app:config:db_host", "mysql.internal:3306", 0)
	log.Info("db_host updated to version %d", dbVer)

	logger.Section("Stats Summary")
	logger.KeyValue("total_keys", len(store.data))
	logger.KeyValue("current_version", store.version)
	logger.KeyValue("range_pairs", len(results))
	logger.KeyValue("session_expired", !ok)
}
