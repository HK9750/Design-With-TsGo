# Key-Value Store

> **A versioned, TTL-aware key-value database supporting range scans and lazy expiry — the simplest distributed data model, powering caches, config stores, and session managers.**

## The Problem It Solves

You're building a feature flag service. Product managers want to toggle features on/off for specific user segments without deploying code. The data model is dead simple: each flag is a key-value pair (`dark_mode` → `enabled`). But you need more than a hash map — flags have TTLs (deactivate after the experiment ends), you need to list all flags for the dashboard (`SCAN` prefix `feature:`), and you need versioning to prevent two admins from overwriting each other's changes. Finally, the service handles 100,000 requests/second, so latency must be sub-millisecond.

A relational database is overkill (no JOINs, no schema, no transactions), and its overhead destroys latency at scale. A full-text search engine doesn't make sense. You need exactly what a key-value store provides: a simple interface of `Put`, `Get`, `Delete`, and `Range` — with the ability to set expiration, track versions for optimistic concurrency, and scan by key prefix.

The key-value store is the simplest useful database abstraction. It strips away everything that isn't essential: no query language, no schema, no relationships, no transactions (in the simplest form). This minimalism is its strength — it can be made blazingly fast (Memcached), highly available (DynamoDB), or strongly consistent (etcd), all by making different tradeoffs on the CAP theorem. Our implementation adds three production-essential features: TTL-based expiration with lazy eviction, monotonic versioning for optimistic concurrency, and sorted key range scans.

## Architecture & Internals

```
                   CORE DATA MODEL
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  KeyValueStore                                       │
    │  ┌───────────────────────────────────────────────┐  │
    │  │  data: map[string]KVEntry                      │  │
    │  │                                               │  │
    │  │  "app:config:db_host"  → {Value:"postgres..", │  │
    │  │                           Version:1,          │  │
    │  │                           ExpiresAt: zero}    │  │
    │  │                                               │  │
    │  │  "session:abc123"      → {Value:"user:42",    │  │
    │  │                           Version:6,          │  │
    │  │                           ExpiresAt: now+100ms}│  │
    │  │                                               │  │
    │  │  "app:feature:dark"    → {Value:"enabled",    │  │
    │  │                           Version:3,          │  │
    │  │                           ExpiresAt: zero}    │  │
    │  └───────────────────────────────────────────────┘  │
    │                                                     │
    └─────────────────────────────────────────────────────┘

                      TTL LIFECYCLE
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  Put("session:abc123", "user:42", 100ms)            │
    │    │  ExpiresAt = now + 100ms                       │
    │    │  Version = 6                                   │
    │    ▼                                                │
    │  ┌──────────────────────────────────────────────┐   │
    │  │  Key exists in map, ExpiresAt set             │   │
    │  └──────────────────────────────────────────────┘   │
    │                                                     │
    │  ... 150ms later ...                                │
    │                                                     │
    │  Get("session:abc123")                              │
    │    │  Entry found: ExpiresAt < now                  │
    │    │  → LAZY EVICTION: delete from map              │
    │    ▼                                                │
    │  Return ("", false)  ← Key no longer exists         │
    │                                                     │
    └─────────────────────────────────────────────────────┘

                    RANGE SCAN
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  Range("app:config:", "app:config:~")               │
    │                                                     │
    │  1. Collect all keys:                               │
    │     ["app:config:db_host", "app:config:log_level",  │
    │      "app:config:max_conn", "app:feature:dark_mode"]│
    │                                                     │
    │  2. Sort lexicographically:                         │
    │     ["app:config:db_host", "app:config:log_level",  │
    │      "app:config:max_conn", "app:feature:dark_mode"]│
    │                                                     │
    │  3. Filter by [start, end] and lazy-evict expired   │
    │                                                     │
    │  Result:                                            │
    │    [["app:config:db_host", "postgres.."],           │
    │     ["app:config:log_level", "debug"],              │
    │     ["app:config:max_conn", "100"]]                 │
    │                                                     │
    └─────────────────────────────────────────────────────┘
```

**Key structures in our Go implementation:**

- `KVEntry` (`main.go:15`): Bundles the value, an optional expiration time (`ExpiresAt`), and a monotonic version number. The `Version` field increments globally with every write — it's a simple form of optimistic concurrency control. Clients can check "has this key changed since I last read it?" by comparing versions.
- `KeyValueStore` (`main.go:25`): A simple wrapper around `map[string]KVEntry` with a global `version` counter. The design prioritizes simplicity: no sharding, no persistence, no replication — but the interface (`Put`, `Get`, `Delete`, `Range`) maps directly to what distributed key-value stores expose.

## Production Use Cases

- **Redis**: The most popular in-memory key-value store. Supports strings, hashes, lists, sets, sorted sets, streams, and modules. Used by Twitter (timeline caching), GitHub (job queues), and Stack Overflow (session storage). Redis's `EXPIRE` and `SCAN` commands directly correspond to our TTL and Range features.
- **etcd**: A strongly consistent, distributed key-value store used as Kubernetes' backing store. Every Kubernetes object (pods, services, configmaps) is stored as a key-value pair in etcd. Uses the Raft consensus algorithm for consistency and provides watch (subscription) APIs.
- **Amazon DynamoDB**: AWS's fully managed key-value and document database. Provides predictable single-digit millisecond latency at any scale, with auto-sharding, multi-region replication, and TTL-based item expiry. Powers Amazon.com's shopping cart, Alexa, and AWS services.
- **Riak**: A distributed key-value store inspired by Amazon's Dynamo paper. Used by Comcast, Best Buy, and the NHS for high-availability storage. Supports CRDTs (conflict-free replicated data types) for automatic conflict resolution.
- **FoundationDB**: Apple's distributed key-value store with serializable ACID transactions. Powers Apple's iCloud and CloudKit. The key insight: a key-value store with transactions can emulate any data model (relational, document, graph) as a layer on top.

## When to Use It

| Use Key-Value Store when... | Don't use when... |
|---|---|
| Data model is flat key-value with optional metadata | You need complex queries (JOINs, GROUP BY, subqueries) |
| Access pattern is point reads/writes by primary key | You need to search by value (full-text or secondary indexes) |
| Latency must be sub-millisecond at high throughput | You need strict schema enforcement and type safety |
| Data has natural TTL (sessions, cache entries, rate limits) | Your data has complex relationships between entities |
| You need simple, predictable performance characteristics | Transactions across multiple keys are required (use FoundationDB or a relational DB) |

**TTL design tradeoffs**: Our implementation uses *lazy eviction* — expired keys are only removed when accessed. This is memory-efficient for write-time checks but means stale keys can accumulate if never accessed. Redis uses a hybrid approach: lazy eviction on access *plus* a background task that randomly samples keys and evicts expired ones. For cache use cases, add a max-memory policy (LRU, LFU) to evict keys when memory is full.

**Versioning for concurrency**: Our monotonic version counter enables "compare-and-swap" patterns. A client reads `{value: "100", version: 5}`, makes a change, and writes `Put(key, "200")` only if version is still 5. If another client updated it to version 6, the first client's read is stale and it should retry. This is called optimistic concurrency control, and it avoids the overhead of distributed locking.

## Complexity Analysis

| Operation | Time | Space |
|---|---|---|
| Put | O(1) | O(1) per key |
| Get | O(1) | O(1) |
| Delete | O(1) | O(1) |
| Range (n keys) | O(n log n) | O(k) where k = matching keys |
| Lazy eviction | O(1) per access | Reclaims space |

The Range operation is O(n log n) due to the mandatory sort of all keys before filtering. Production key-value stores avoid this by using an ordered data structure (B-Tree, skip list) as the backing store, supporting O(k) range scans directly. Our implementation uses a hash map for simplicity, demonstrating the tradeoff: O(1) point operations at the cost of O(n log n) range scans.

## Implementation Deep Dive

### 1. TTL with Lazy Eviction (`main.go:57-71`)

`Get()` checks if `entry.ExpiresAt` is non-zero and in the past. If so, it deletes the key from the map and returns `("", false)` — the key effectively never existed. This is *lazy* because expired keys survive until accessed. The design choice is deliberate: checking expiration on every write would add overhead to the hot path; a background eviction thread would add complexity. For session stores where expired keys are frequently accessed (sessions are checked on every request), lazy eviction is efficient — keys are cleaned up promptly.

### 2. Monotonic Version Counter (`main.go:41-52`)

`Put()` increments `s.version` (the store's global counter) and stamps the entry. This is a *global* monotonic counter — every write across all keys gets a unique version. An alternative design would be per-key versions, where each key tracks its own version independently. Per-key versions are better for optimistic concurrency (you only care if *this key* changed), while global versions are better for total ordering (you can ask "what was the entire store's state at version 57?"). Our implementation uses global versions for simplicity, but real systems like DynamoDB use per-item version vectors for conflict resolution.

### 3. Range Scan via Sort (`main.go:89-106`)

`Range()` collects all keys, sorts them lexicographically, and filters those within `[start, end]`. The `~` character in the demo's end-bound (`"app:config:~"`) exploits ASCII ordering: `~` (ASCII 126) is greater than any printable character, ensuring all `app:config:` keys are included. This is a common pattern in lexicographic-range databases. The actual filtering calls `Get()` for each key to trigger lazy eviction — an expired key in range is silently excluded from results. Production systems use an ordered data structure (skip list in Redis, B-tree in etcd) to avoid the O(n log n) sort cost.

## Running the Demo

```bash
go run ./26-Key-Value-Store/
```

The demo populates a configuration store with app settings and feature flags, demonstrates range scans for listing all config entries, tests TTL expiration with a session token that expires after 100ms, and shows version tracking after updates.

## Further Reading

- **"Dynamo: Amazon's Highly Available Key-value Store"** — Giuseppe DeCandia et al. (2007). The seminal paper that launched the NoSQL movement. Describes Amazon's eventually-consistent, always-writable key-value store with vector clocks for conflict resolution. Directly influenced Riak, Cassandra, and Voldemort.
- **Redis Documentation (redis.io/documentation)** — Covers data types, eviction policies (LRU, LFU, TTL), persistence options (RDB, AOF), and clustering. Redis's `EXPIRE`, `SCAN`, and `WATCH` commands are the production implementation of our TTL, Range, and versioning features.

---

*Part of the Design-With-TsGo system design curriculum*
