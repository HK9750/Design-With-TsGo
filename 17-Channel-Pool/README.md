# Channel Pool

> **A pre-warmed, bounded pool of reusable channels that eliminates allocation churn in high-throughput systems where channels are frequently created and discarded.**

## The Problem It Solves

You're building a real-time message router that processes 500,000 events per second. Each incoming event spawns a short-lived pipeline: create a channel, launch a goroutine to process the event, send the result on the channel, receive the result, and discard the channel. The channels are used for exactly one message each — they're single-use conduits. With 500,000 events per second, you're allocating and garbage-collecting 500,000 channels per second.

Go channels are not free. Each `make(chan T)` call allocates a `hchan` struct (roughly 96 bytes on 64-bit systems) plus a ring buffer (if buffered). At 500,000 allocations per second, that's ~48 MB/s of allocation pressure — enough to trigger GC cycles multiple times per second. Each GC cycle pauses all goroutines (even if briefly with Go's concurrent GC), adding latency jitter to your real-time pipeline. The GC's mark phase also scans every allocated channel, consuming CPU that should be processing messages.

The solution is object pooling: instead of creating and discarding channels, you pre-allocate a fixed pool of channels at startup and reuse them from an `available` queue. An `Acquire` takes a channel from the pool; `Release` returns it. The pool has a fixed capacity, which also bounds concurrency — if all channels are in use, `Acquire` blocks, providing natural backpressure. When the system reaches steady state, there are zero channel allocations and zero channel GC scans. The only allocations are the initial pool creation at startup.

This is exactly how `sync.Pool` works for arbitrary objects, how `fasthttp` pools request/response structs, and how database connection pools work. The channel pool applies the same idea specifically to Go channels, leveraging the language's primitive as the pooled resource.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────┐
│                     ChannelPool[T]                             │
│                                                              │
│   available chan chan T     ── pool of channels              │
│   factory func() chan T     ── creates a new channel         │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  Initialization (NewChannelPool, size=3)                │  │
│  │                                                         │  │
│  │  factory() ──▶ chan T ──▶ available                    │  │
│  │  factory() ──▶ chan T ──▶ available                    │  │
│  │  factory() ──▶ chan T ──▶ available                    │  │
│  │                                                         │  │
│  │  available = [ch0, ch1, ch2]                            │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  Acquire/Release Cycle                                  │  │
│  │                                                         │  │
│  │  Goroutine 1:  ch := pool.Acquire()  →  get ch0        │  │
│  │                available = [ch1, ch2]                    │  │
│  │                ch <- "order.created"                    │  │
│  │                msg := <-ch                              │  │
│  │                pool.Release(ch)  →  put ch0 back        │  │
│  │                available = [ch1, ch2, ch0]              │  │
│  │                                                         │  │
│  │  Goroutine 2:  ch := pool.Acquire()  →  get ch1        │  │
│  │  Goroutine 3:  ch := pool.Acquire()  →  get ch2        │  │
│  │  Goroutine 4:  ch := pool.Acquire()  →  BLOCKS         │  │
│  │                (available is empty)                      │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  Key property: pool.Acquire() blocks when pool is exhausted  │
│  → Natural backpressure = bounded concurrency                │
└──────────────────────────────────────────────────────────────┘
```

The `ChannelPool[T]` struct (`main.go:15-18`) has two fields: `available` (a channel of channels — `chan chan T`) and a `factory` function. The nested channel structure is the key insight: the outer channel is the pool's availability queue, and each element in that queue is an inner channel (the actual communication pipe) provided by the factory.

`NewChannelPool` (`main.go:22-35`) validates `size > 0`, creates the `available` channel with the pool's capacity, then pre-warms the pool by calling `factory()` `size` times and sending each created channel into `available`. This is an O(size) initialization step, done once at startup.

`Acquire` (`main.go:39-43`) receives a channel from `available`. If the pool is empty (all channels in use), this blocks. `Release` (`main.go:47-50`) sends a channel back into `available`, making it available for the next caller. The demo at `main.go:52-91` creates a pool of 3 string channels (each buffered with capacity 1) and routes 5 messages through them, reusing channels as they become available.

## Production Use Cases

- **Go's `sync.Pool` for object reuse** — The principle is the same as `ChannelPool`, but `sync.Pool` is for arbitrary objects. `fmt.Printf` internally uses `sync.Pool` to reuse `fmt.pp` structs (buffers for formatting). The JSON encoder (`encoding/json`) pools encoder state structs. Any Go program with high allocation pressure benefits from `sync.Pool`.
- **FastHTTP connection pooling** — `fasthttp` pools `RequestCtx` and `Response` objects, avoiding per-request allocations. Their `HostClient` also pools TCP connections via a `conns []*clientConn` slice, using LIFO reuse (same as this `ChannelPool`'s Acquire behavior on a buffered channel).
- **gRPC channel pooling** — The gRPC-Go library pools HTTP/2 transport connections. The `ClientConn` maintains a pool of sub-connections (each a TCP+TLS connection to a backend), and RPC calls acquire and release them. This is connection pooling at the RPC layer, but the Acquire/Release semantics are identical.
- **Database connection pools** — HikariCP (Java) and `pgxpool` (Go) both implement bounded connection pools with Acquire/Release semantics. `pgxpool.Pool.Acquire(ctx)` returns a `*Conn` from an idle pool or creates a new one; `Release()` returns it. The pool acts as the rate-limiter for concurrent database operations.
- **Custom allocators in high-frequency trading** — HFT firms pre-allocate pools of fixed-size buffers (for market data packets, order book updates) and use Acquire/Release-style interfaces to eliminate GC pressure. The LMAX Disruptor's `RingBuffer` is essentially a pre-allocated, lock-free pool of event slots.

## When to Use It

| Scenario | Use Channel Pool? |
|----------|------------------|
| Creating and discarding channels at high frequency (>10K/s) | **Yes** |
| Channels are used for single or few messages and then discarded | **Yes** |
| Need bounded concurrency with backpressure (pool is exhausted → block) | **Yes** |
| Channels hold complex state that persists across uses (need reset/cleanup) | **No** — or add a `Reset` method before `Release` |
| Low-throughput system where channel allocation is negligible | **No** — premature optimization; plain `make(chan T)` is fine |
| Need unbounded availability (more channels than pool capacity) | **No** — the pool blocks when exhausted; you'd need an overflow creation path |
| Channels need different configurations (different buffer sizes) | **No** — all pooled channels are homogeneous from the same factory |

**Alternatives**: **Plain `make(chan T)`** is simpler and correct for low-to-moderate throughput. **`sync.Pool`** is the general-purpose object pool in Go — better for structs and buffers, but doesn't provide blocking backpressure (it creates new objects if the pool is empty). **Connection Pool** (topic 20) is the same pattern applied to stateful connections with health checking.

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| NewChannelPool | O(size) — pre-warms all channels | O(size) | O(size) |
| Acquire | O(1) — channel receive | O(unbounded) when blocked | O(1) |
| Release | O(1) — channel send | O(1) | O(1) |

The O(size) initialization cost is paid once at startup. After initialization, both Acquire and Release are single channel operations — constant time from the caller's perspective. Blocking time on Acquire depends on the workload: how long other goroutines hold channels before releasing them.

The space footprint is `size` × (channel overhead + buffer overhead). A buffered `chan T` with capacity 1 allocates ~96 bytes (hchan struct) + sizeof(T) × 1. For 10,000 `chan string` (each string channel ~112 bytes), the pool consumes ~1.1 MB — small relative to most Go application memory budgets.

## Implementation Deep Dive

**1. The `chan chan T` type is a double-channel for maximum clarity.** The pool field `available chan chan T` at `main.go:16` reads as "a channel of channels." This is a common Go pattern for pools where the pooled resource is itself a channel. The outer channel (the pool) is buffered to the pool size; the inner channels (the resources) are whatever the factory creates — usually small buffers for single-message use. The double-channel pattern makes it explicit that we're pooling communication primitives, not data values.

**2. The factory function enables per-use customization without code changes.** The `factory func() chan T` parameter at `main.go:17` is a function that creates channels. The demo at `main.go:59-61` passes `func() chan string { return make(chan string, 1) }` — a buffered channel with capacity 1. But the same pool type could use unbuffered channels (`make(chan string)`), or channels with larger buffers, or channels initialized with a specific capacity based on runtime configuration. The factory indirection keeps the pool generic while enabling per-instance tuning.

**3. Pre-warming in the constructor avoids cold-start latency.** The loop at `main.go:28-31` calls `factory()` `size` times and sends every channel into `available` before `NewChannelPool` returns. This means the first `Acquire` after server startup is O(1) — no allocation, no factory call. In a high-throughput system, cold-start latency on the first few requests can trigger cascading timeouts if upstream services retry. Pre-warming eliminates this risk for the channel allocation layer.

## Running the Demo

```bash
go run ./17-Channel-Pool/
```

## Further Reading

- "Go's sync.Pool" — Go Blog. Details the design of `sync.Pool` including its interaction with the garbage collector, the per-P local caches that make it scalable, and when to use `New` vs. just calling `Get`. The same principles apply to custom pools.
- "Object Pool Pattern" — Mark Grand, Patterns in Java, Volume 1. While Java-specific, the analysis of when pooling improves performance vs. when it wastes memory and adds complexity is universal and directly applicable to Go pool decisions.

---

*Part of the Design-With-TsGo system design curriculum*
