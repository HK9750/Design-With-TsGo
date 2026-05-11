# Semaphore

> **A counting concurrency limiter built on a buffered channel of empty structs — the simplest and most idiomatic way to cap concurrent access to any resource in Go.**

## The Problem It Solves

Your analytics service receives 10,000 report-generation requests per second. Each report requires a database query that opens a TCP connection to PostgreSQL. Your connection pool is capped at 20 connections (PostgreSQL performs best with `cpu_cores × 2` active connections). If you launch a goroutine for each incoming request — 10,000 goroutines — all 10,000 will try to acquire a database connection from the pool. The first 20 succeed. The remaining 9,980 goroutines pile up waiting for connections, consuming memory (each goroutine stack is ~2-4KB, so ~40MB just for these waiters), and the database connection pool itself has a bounded wait queue. Requests time out, goroutines accumulate, and the server OOMs under memory pressure from waiters that will never be served.

You need a way to say: "At most N goroutines may proceed past this point; all others must block." This is the definition of a counting semaphore — a synchronization primitive that maintains a counter of available "permits" and blocks callers when no permits remain. Each goroutine acquires a permit before doing the protected work and releases it after. The semaphore enforces the concurrency cap at the application level, preventing goroutine buildup downstream.

The channel-based semaphore is particularly elegant in Go because it maps perfectly to the language's primitive: a buffered channel of `struct{}` (zero-size type) where `Acquire` is a send (`ch <- struct{}{}`) and `Release` is a receive (`<-ch`). The channel's buffer capacity is the permit count. When the buffer is full (all permits in use), sends block. When the buffer is empty (no permits in use), receives block. No mutex, no condition variable, no atomic counter — the Go runtime handles all the scheduling.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────┐
│                        Semaphore                              │
│                                                              │
│   permits chan struct{}  (buffered, capacity = N)            │
│                                                              │
│   ┌──────────────────────────────────────────────────────┐  │
│   │  Capacity = 3 (three permits)                         │  │
│   │                                                       │  │
│   │  Initial:  [_] [_] [_]  (3 empty slots, 3 available)  │  │
│   │                                                       │  │
│   │  Goroutine 1 Acquire():                               │  │
│   │    permits <- struct{}{}                              │  │
│   │    Channel:  [x] [_] [_]  (1 permit taken)            │  │
│   │                                                       │  │
│   │  Goroutine 2 Acquire():                               │  │
│   │    Channel:  [x] [x] [_]  (2 permits taken)           │  │
│   │                                                       │  │
│   │  Goroutine 3 Acquire():                               │  │
│   │    Channel:  [x] [x] [x]  (3 permits taken, FULL)     │  │
│   │                                                       │  │
│   │  Goroutine 4 Acquire():                               │  │
│   │    BLOCKS — channel is full                           │  │
│   │                                                       │  │
│   │  Goroutine 1 Release():                               │  │
│   │    <-permits                                          │  │
│   │    Channel:  [_] [x] [x]  (1 slot freed)              │  │
│   │    Goroutine 4 unblocks, acquires the freed permit    │  │
│   └──────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│  Semaphore vs Mutex                                          │
│                                                              │
│  Mutex:        1 goroutine at a time    (binary)             │
│  Semaphore:    N goroutines at a time   (counting)           │
│  Mutex with N=1 === Semaphore                               │
│                                                              │
│  Key difference: a Mutex is released by its owner;            │
│  a Semaphore can be released by any goroutine.               │
└──────────────────────────────────────────────────────────────┘
```

The `Semaphore` struct (`main.go:15`) contains a single field: `permits chan struct{}`. The empty struct `struct{}` occupies zero bytes — the channel stores only the scheduling metadata, not payload data. `NewSemaphore` (`main.go:19-26`) validates that `permits > 0` and creates a buffered channel of that capacity.

`Acquire` (`main.go:29-32`) sends an empty struct into the channel. If the channel has available capacity (fewer than N items), the send succeeds immediately. If the channel is full (N items), the send blocks until another goroutine calls `Release`. `Release` (`main.go:35-38`) receives from the channel, freeing one slot.

The demo at `main.go:40-75` creates a semaphore with 2 permits and launches 5 goroutines, each simulating an HTTP call to an external API. Only 2 goroutines execute simultaneously — the other 3 block on `Acquire` until a permit is released. The `defer semaphore.Release()` idiom ensures permits are returned even if the protected code panics.

## Production Use Cases

- **PostgreSQL connection limiting (PgBouncer)** — PgBouncer, the de facto PostgreSQL connection pooler, uses semaphore-style counting to limit concurrent server connections. When configured with `pool_mode = transaction`, PgBouncer allows at most `max_client_conn` concurrent transactions, queuing excess clients until a slot frees. This prevents the "too many clients" PostgreSQL error.
- **Kubernetes Pod concurrency** — Kubernetes `Jobs` with `spec.parallelism: N` create at most N Pods simultaneously out of the total `spec.completions`. This is a counting semaphore at the cluster level. The Job controller increments a counter when a Pod starts and decrements when it finishes.
- **Linux kernel semaphores** — The kernel's `struct semaphore` (defined in `include/linux/semaphore.h`) is a counting semaphore used internally for driver synchronization and filesystem locking. `down_interruptible()` acquires a permit; `up()` releases it. The kernel implementation uses wait queues rather than busy-waiting.
- **Go's `golang.org/x/sync/semaphore.Weighted`** — The official extended library provides a weighted semaphore where `Acquire(ctx, n)` blocks until `n` units of capacity are available. Used internally by `golang.org/x/tools/gopls` (the Go language server) to bound concurrent type-checking operations.
- **Rate limiting concurrent API calls** — The Stripe Go client doesn't use a pool, but many internal Stripe services use semaphores to limit concurrent outbound requests to third-party payment processors, preventing accidental DDoS and respecting processor rate limits.

## When to Use It

| Scenario | Use Semaphore? |
|----------|---------------|
| Capping concurrent access to a limited resource (DB connections, file descriptors, API rate windows) | **Yes** |
| Limiting goroutine count to prevent unbounded memory growth | **Yes** |
| Mutual exclusion where only one goroutine may access at a time | **No** — use `sync.Mutex`, it's faster and conveys clearer intent |
| Need a timeout on waiting for a permit | **No** — the channel-based semaphore blocks indefinitely; use `semaphore.Weighted.Acquire(ctx, n)` for context support |
| Need prioritized or fair queuing of waiters | **No** — channel send order is FIFO but not strictly guaranteed; use a custom implementation with a priority queue |
| Resource is CPU (want to match GOMAXPROCS) | **No** — let the Go scheduler manage goroutines; semaphores are for external resource constraints |

**Alternatives**: **`sync.Mutex`** is a binary semaphore — simpler, faster, but only allows one goroutine. **`semaphore.Weighted`** (golang.org/x/sync/semaphore) supports context cancellation and weighted acquisition (acquire N permits at once). **Worker Pool** (topic 13) is a semaphore applied specifically to job processing with a fixed worker count. **Rate Limiter** (topic 09) is a semaphore plus a time dimension — permits regenerate over time rather than being returned.

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| Acquire | O(1) when permit available | O(unbounded) when blocked | O(1) |
| Release | O(1) | O(1) | O(1) |

The channel send/receive operations are constant time when the channel is not full/empty. When `Acquire` blocks, the worst-case wait time depends on the workload — a goroutine holding a permit for 10 seconds blocks all waiters for 10 seconds. This is why semaphore permits should be held for the minimum possible duration.

The `struct{}` type ensures zero data overhead per permit — the channel buffer stores only scheduling metadata. The per-goroutine overhead is Go's stack size (~2-4KB), which exists regardless of the semaphore.

## Implementation Deep Dive

**1. `struct{}` is the zero-cost token type.** The channel `chan struct{}` at `main.go:15` stores tokens that occupy zero bytes. If you used `chan bool` (1 byte plus alignment) or `chan int` (8 bytes), 10,000 permits would waste 80KB-100KB. With `struct{}`, the channel buffer stores only the internal `hchan` ring buffer entries — pure scheduling overhead. This is an idiomatic Go pattern: `chan struct{}` is the standard "signal-only" channel.

**2. `defer semaphore.Release()` is the critical correctness idiom.** In the demo at `main.go:59`, `defer semaphore.Release()` ensures the permit is returned regardless of how the goroutine exits — normal return, error return, or panic. Without `defer`, a panic in the protected code would permanently consume a permit, eventually exhausting the semaphore and deadlocking the system. This is the same reasoning behind `defer mu.Unlock()` for mutexes.

**3. The channel-based semaphore provides FIFO fairness (by accident).** Go's channel send/receive operations are FIFO-ordered — goroutines blocked on `ch <-` are queued in arrival order. This means the semaphore is fair by default: the first goroutine to call `Acquire` while permits are exhausted will be the first to receive a permit when one is released. This is not a documented guarantee (the Go spec does not require FIFO ordering for channels), but in practice the runtime scheduler implements it this way. Production systems that require guaranteed fairness should use `semaphore.Weighted` or a custom implementation.

## Running the Demo

```bash
go run ./15-Semaphore/
```

## Further Reading

- "The Structure of the 'THE'-Multiprogramming System" — Edsger Dijkstra (1968). The paper that introduced semaphores as a synchronization primitive. Dijkstra describes the "P" (proberen/test) and "V" (verhogen/increment) operations that became Acquire and Release. A foundational text in computer science.
- `golang.org/x/sync/semaphore` — Go extended library documentation. The `Weighted` type adds context-awareness and multi-permit acquisition to the basic channel-based pattern, making it suitable for production use.

---

*Part of the Design-With-TsGo system design curriculum*
