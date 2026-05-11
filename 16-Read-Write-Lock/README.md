# Read-Write Lock

> **A mutual exclusion primitive that permits unlimited concurrent readers but only one exclusive writer — the standard pattern for protecting data structures that are read-heavy with infrequent modifications.**

## The Problem It Solves

Your web service maintains an in-memory configuration cache — database hostnames, API keys, feature flags, and connection pool sizes. Every incoming HTTP request reads from this cache (the middleware checks a feature flag, the database layer resolves the hostname, the rate limiter checks thresholds). With 50,000 requests per second, you have 50,000 concurrent reads. A configuration update (a new feature flag, a rotated API key) arrives once every few minutes.

If you protect the cache with a plain `sync.Mutex`, every read blocks every other read. With 50,000 goroutines contending for the same mutex, throughput collapses — reads that should be instantaneous are serialized. The 99th percentile latency spikes from microseconds to milliseconds as readers queue up behind each other. The configuration update (which should take microseconds) waits behind thousands of queued readers.

A read-write lock solves this by distinguishing two lock types: a **read lock** (shared) that any number of goroutines can hold simultaneously, and a **write lock** (exclusive) that only one goroutine can hold, and only when no readers are active. Readers never block each other — 50,000 concurrent reads proceed in parallel. A writer waits for all active readers to finish, then acquires exclusive access, then releases. New readers arriving after a writer has requested the lock are queued to prevent writer starvation. The result: read throughput scales with goroutine count, and the infrequent writes still have predictable latency.

This pattern is so fundamental that Go's standard library provides `sync.RWMutex` — a read-write mutex — and the Linux kernel has `rwlock_t`. Both solve the same problem: data that is "read mostly, write rarely" should not pay the serialization cost of a full mutex on reads.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────┐
│                     ReadWriteLock                             │
│                                                              │
│   sync.RWMutex                                 state:        │
│                                                  ┌─────────┐ │
│  ┌───────────────────────────────────────┐       │  Locked  │ │
│  │          Reader Goroutines            │       │  by W    │ │
│  │                                       │       └─────────┘ │
│  │  [R] [R] [R] [R] [R] [R] [R] [R]    │       ┌─────────┐ │
│  │   │   │   │   │   │   │   │   │      │       │  Locked  │ │
│  │   ▼   ▼   ▼   ▼   ▼   ▼   ▼   ▼      │       │  by N    │ │
│  │  ┌──────────────────────────────┐     │       │  Readers │ │
│  │  │     Shared (Read) Access     │     │       └─────────┘ │
│  │  │     Allowed Concurrently     │     │                    │
│  │  └──────────────────────────────┘     │       ┌─────────┐ │
│  └───────────────────────────────────────┘       │ Waiting  │ │
│                                                  │ Writer   │ │
│  ┌───────────────────────────────────────┐       └─────────┘ │
│  │          Writer Goroutine             │                    │
│  │                                       │                    │
│  │  [W] ──── WAITS ────▶ until all      │                    │
│  │          readers finish, then:        │                    │
│  │                                       │                    │
│  │  ┌──────────────────────────────┐     │                    │
│  │  │   Exclusive (Write) Access   │     │                    │
│  │  │   One Writer, No Readers     │     │                    │
│  │  └──────────────────────────────┘     │                    │
│  └───────────────────────────────────────┘                    │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│  State Machine                                                │
│                                                              │
│                    ┌──────────┐                               │
│          ┌────────│ UNLOCKED │◀────────┐                      │
│          │        └────┬─────┘         │                      │
│          │  RLock      │     Lock       │ Unlock              │
│          │             │                │                      │
│          ▼             ▼                │                      │
│  ┌──────────────┐ ┌──────────┐         │                      │
│  │  READ-LOCKED │ │ WRITE-   │─────────┘                      │
│  │  (N readers) │ │ LOCKED   │  RUnlock x N                   │
│  │              │ │          │                                │
│  │  RLock ──────┤ │          │                                │
│  │  (increment) │ │          │                                │
│  │              │ │          │                                │
│  │  RUnlock ────┤ │          │                                │
│  │  (decrement, │ │          │                                │
│  │   last→free) │ │          │                                │
│  └──────────────┘ └──────────┘                                │
└──────────────────────────────────────────────────────────────┘
```

The `ReadWriteLock` struct (`main.go:16`) wraps `sync.RWMutex` — there is no custom logic. The implementation at `main.go:16-42` provides four methods that are direct delegation: `RLock()` calls `mu.RLock()`, `RUnlock()` calls `mu.RUnlock()`, `Lock()` calls `mu.Lock()`, and `Unlock()` calls `mu.Unlock()`. The value here is in the pattern, not the code — understanding when to use RWMutex and how to structure code around it.

The `sync.RWMutex` internally maintains a reader count and a writer-semaphore. `RLock` increments the reader count (atomically) and proceeds immediately if no writer is waiting. `Lock` sets a writer-pending flag (preventing new readers) and blocks until the reader count reaches zero. This "writer preference" design prevents writer starvation: once a writer signals intent, new readers queue behind it. However, it can reduce read throughput under mixed workloads — readers that hold locks for long periods delay the writer, which delays all subsequent readers.

## Production Use Cases

- **Go's `net/http` server maps** — The `http.ServeMux` uses `sync.RWMutex` to protect its internal routing table (`mux.m`). Reads (URL matching on every request) hold `RLock`; writes (registering a new handler) hold `Lock`. With millions of requests per second, read-locking the mux avoids creating a routing bottleneck.
- **Linux kernel `rwlock_t`** — The kernel uses read-write spinlocks for data structures like the dentry cache (directory entry cache), the inode cache, and the filesystem mount table. On a 64-core machine, hundreds of concurrent `stat()` calls can hold the read lock without contention. Used in every Linux kernel since 2.0.
- **PostgreSQL shared locks** — PostgreSQL's `AccessShareLock` (acquired by SELECT) allows any number of concurrent readers, while `AccessExclusiveLock` (acquired by ALTER TABLE) requires exclusive access. This is a read-write lock at the database level, implemented via the lock manager's shared memory hash table.
- **Configuration hot-reload systems** — The Viper configuration library (used in virtually every Go service) uses `sync.RWMutex` to protect its internal config map. Every config read (`viper.GetString`) acquires `RLock`; a config reload (SIGHUP or file watcher) acquires `Lock`. This allows thousands of in-flight requests to read config without contention.
- **In-memory caches** — `go-cache`, `bigcache`, and `ristretto` all use `sync.RWMutex` (or sharded RWMutexes) to protect their internal hash maps. Cache reads dominate cache writes by 100:1 or more, making the read-write distinction critical for performance.

## When to Use It

| Scenario | Use RWMutex? |
|----------|-------------|
| Read-to-write ratio is > 10:1 (reads dominate writes) | **Yes** |
| Individual reads hold the lock for a meaningful duration (microseconds+) | **Yes** |
| Need to protect a map/slice that is read on every request but modified rarely | **Yes** |
| Read-to-write ratio is roughly 1:1 or write-heavy | **No** — plain `sync.Mutex` is faster (no reader-count bookkeeping overhead) |
| Lock is held for nanoseconds (a few CPU instructions) | **No** — the RWMutex's internal atomics cost more than the protected code |
| High contention with writers arriving frequently | **No** — writer preference means readers block behind writers, negating the parallelism benefit |
| Need lock upgrades (read → write) or lock downgrades (write → read) | **No** — RWMutex doesn't support this; use a custom implementation or restructure the code |

**Alternatives**: **`sync.Mutex`** is simpler and faster when the read-to-write ratio is low (< 3:1) or when the lock is held for extremely short durations. **Sharded mutexes** (multiple RWMutexes, each protecting a subset of data, keyed by hash) reduce contention further for very high-throughput systems — `bigcache` uses 1024 shards. **Lock-free data structures** (atomic pointers, RCU-like patterns) eliminate blocking entirely for reads. **Copy-on-write** (COW) means writers replace the entire data structure with a clone, and readers access the old version until they release — reads are truly lock-free (Go's `sync.Map` uses a COW-like approach).

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) |
|-----------|---------------|-------------|
| RLock | O(1) — atomic increment | O(num_readers) if writer pending |
| RUnlock | O(1) — atomic decrement | O(1) |
| Lock | O(1) if no readers | O(duration of longest reader) |
| Unlock | O(1) | O(1) |

The worst-case write lock acquisition time is bounded by the duration of the longest-held read lock. A single reader holding `RLock` for 5 seconds delays every writer (and every new reader after the writer signals intent) by 5 seconds. This is why read-lock critical sections should be as short as possible — ideally just a map lookup or slice copy.

The space overhead is the `sync.RWMutex` struct itself (a handful of integers and a semaphore channel, approximately 40-60 bytes).

## Implementation Deep Dive

**1. Writer-preference prevents starvation but penalizes throughput under mixed loads.** `sync.RWMutex` uses a writer-preference policy: when `Lock()` is called, it sets a writer-pending flag. Subsequent `RLock()` calls see this flag and block, waiting for the writer to finish. This guarantees that a writer won't starve waiting for an endless stream of readers — but it also means that once a writer shows up, read throughput drops to zero until the writer completes. Under workloads with frequent writes, this can make RWMutex slower than a plain Mutex. The demo at `main.go:44-105` simulates a read-heavy workload: 10 readers and 2 writers, matching typical configuration-cache access patterns.

**2. `defer` pairing is non-negotiable for correctness.** The demo carefully pairs `lock.RLock()` with `defer lock.RUnlock()` and `lock.Lock()` with `defer lock.Unlock()`. Go's standard library documentation explicitly warns: "Unlock unlocks rw for writing. It is a run-time error if rw is not locked for writing on entry to Unlock." The same applies to `RUnlock()`. Unlike `sync.Mutex`, where a mismatched Unlock panics immediately, `RUnlock` on an unlocked RWMutex can cause silent corruption — it decrements the reader count below zero, allowing writers to proceed while readers are still active.

**3. The wrapper struct adds logging but no extra semantics.** The `ReadWriteLock` at `main.go:16-42` is a thin wrapper with debug logging. In production, you'd use `sync.RWMutex` directly. The wrapper exists here to make the acquire/release lifecycle visible during the demo — the debug logs at `main.go:21,27,33,39` show the sequence of read and write lock operations, which is valuable for understanding the interleaving of the 10 readers and 2 writers.

## Running the Demo

```bash
go run ./16-Read-Write-Lock/
```

## Further Reading

- `sync.RWMutex` documentation — Go standard library. Covers the exact semantics, the writer-preference design (Go 1.x), and the correct usage of `RLock`/`RUnlock` and `Lock`/`Unlock`. Essential reading before deploying any RWMutex-based code.
- "The Art of Multiprocessor Programming" — Herlihy and Shavit, Chapter 8. Provides the formal foundations of read-write locks including fairness guarantees, hand-over-hand locking, and the tradeoffs between reader-preference, writer-preference, and neutral algorithms.

---

*Part of the Design-With-TsGo system design curriculum*
