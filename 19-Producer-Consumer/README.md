# Producer-Consumer

> **A bounded, thread-safe queue that decouples data producers from data consumers — enabling asynchronous pipelines with built-in backpressure when production outpaces consumption.**

## The Problem It Solves

Your real-time analytics service receives clickstream events from a web frontend at a bursty rate: 500 events/s most of the time, but 5,000 events/s during traffic spikes. Each event needs to be parsed, enriched (look up user geo, device type), validated, and written to a columnar store. Writing to the store takes ~5ms per event (S3 PUT with batching), so a single consumer goroutine can handle at most ~200 events/s. You need multiple consumers, and more importantly, you need a buffer that absorbs the burst without dropping events.

If you connect producers directly to consumers via synchronous function calls, every producer blocks until a consumer is ready. During a burst, 5,000 producers (goroutines handling HTTP requests) pile up waiting for 200 consumers — the producer goroutines accumulate, memory grows, and the server OOMs. You need a queue between them: producers push events onto the queue without waiting for a consumer; consumers pull events at their own pace. When the queue is full, producers block — this is **backpressure**, and it's the mechanism that prevents unbounded memory growth.

This is the producer-consumer problem, first formalized by Dijkstra in 1965. Two (or more) concurrent processes share a bounded buffer: producers add items, consumers remove them. The synchronization constraint: producers must block when the buffer is full (no overwriting un-consumed data), consumers must block when the buffer is empty (no consuming phantom data). Go's buffered channels solve this natively — a buffered channel IS a bounded producer-consumer queue with blocking send (when full) and blocking receive (when empty).

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────┐
│                  Producer-Consumer System                      │
│                                                              │
│  ┌─────────────────┐                                         │
│  │   Producers      │                                         │
│  │                  │                                         │
│  │  ┌────────────┐ │     Enqueue()        Dequeue()          │
│  │  │ Producer 0 │─┼───▶ ┌──────────────────────────┐ ───▶  │
│  │  └────────────┘ │     │                          │       │
│  │  ┌────────────┐ │     │   BoundedQueue[T]        │       │
│  │  │ Producer 1 │─┼───▶ │                          │       │
│  │  └────────────┘ │     │  ch chan T               │       │
│  │  ┌────────────┐ │     │  (buffered, cap=N)       │       │
│  │  │ Producer 2 │─┼───▶ │                          │       │
│  │  └────────────┘ │     │  [_] [_] [_] [_] [_]    │       │
│  └─────────────────┘     └──────────────────────────┘       │
│                                                              │
│  Producer blocks on Enqueue when:             ──▶            │
│    queue is full (len == cap)                               │
│                                                              │
│  Consumer blocks on Dequeue when:             ──▶            │
│    queue is empty (len == 0)                 ┌──────────────┐│
│                                              │  Consumers   ││
│                                              │              ││
│                                              │  Consumer 0  ││
│                                         ───▶ │  Consumer 1  ││
│                                              │  Consumer 2  ││
│                                              └──────────────┘│
│                                                              │
│  Backpressure flow:                                          │
│                                                              │
│  producers > capacity > consumers:  producers block          │
│  producers < consumers:  consumers block                     │
│  producers ≈ consumers:  steady state (no blocking)          │
└──────────────────────────────────────────────────────────────┘
```

The `BoundedQueue[T]` struct (`main.go:16`) is a single-field wrapper: `ch chan T`. The buffered channel IS the queue — no slices, no linked lists, no mutexes. `NewBoundedQueue` (`main.go:20-27`) validates capacity and creates `make(chan T, capacity)`. The channel's buffer size is the queue's capacity.

`Enqueue` (`main.go:31-34`) is simply `q.ch <- value` — a channel send. If the channel has available buffer space (`len(q.ch) < cap(q.ch)`), the send succeeds immediately. If the channel is full, the send blocks until a consumer calls `Dequeue`. This is the backpressure mechanism: producers cannot outrun consumers indefinitely.

`Dequeue` (`main.go:38-42`) is `<-q.ch` — a channel receive. If the channel has buffered items, the receive succeeds immediately. If the channel is empty, the receive blocks until a producer calls `Enqueue`. This is the natural flow control: consumers idle when there's nothing to do.

The demo at `main.go:44-106` simulates 2 producers generating 12 log events total and 3 consumers processing them through a queue of capacity 3. With capacity=3, the queue fills after 3 unconsumed events — producers block, naturally limiting the rate of event generation to the rate of consumption.

## Production Use Cases

- **Apache Kafka internal queues** — Kafka brokers use bounded in-memory queues extensively. The network layer uses Java's `LinkedBlockingQueue` for request queuing; the replication layer uses bounded queues between the leader's log append path and follower fetch requests. Kafka's `queued.max.requests` controls the bound, and backpressure propagates from slow brokers to producers.
- **Go's channel-based pipelines** — Go's standard library and ecosystem use producer-consumer pipelines pervasively. `io.Pipe()` creates a synchronous in-memory pipe (Reader/Writer) that is a producer-consumer queue of byte slices. The `scanner.Scanner` reads from an `io.Reader` (producer) and yields tokens (consumer) via a line-buffered internal channel.
- **Redis pub/sub with bounded buffers** — While Redis pub/sub is fire-and-forget (no queuing), production systems often wrap it with a bounded Go channel as a client-side buffer. The Redis subscriber goroutine (producer) enqueues messages; application goroutines (consumers) dequeue and process. When the buffer fills, the subscriber applies backpressure by slowing its Redis `XREADGROUP` consumption.
- **Celery task queues** — Celery (Python distributed task queue) uses Redis or RabbitMQ as the bounded broker queue between task producers (web servers enqueuing tasks) and task consumers (worker processes). The broker enforces the bound; when the queue exceeds `CELERY_QUEUE_MAX_LENGTH`, producers are rejected or blocked.
- **LMAX Disruptor** — The Disruptor is a high-performance inter-thread messaging library that implements a lock-free ring buffer — essentially a specialized producer-consumer queue. Used in financial exchanges (LMAX itself) and Apache Log4j 2 for async logging. It achieves ~6 million operations per second by avoiding locks and exploiting CPU cache line padding.

## When to Use It

| Scenario | Use Producer-Consumer? |
|----------|----------------------|
| Decoupling data generation from data processing (different rates, different goroutines) | **Yes** |
| Need to smooth bursty traffic with a bounded buffer | **Yes** |
| Backpressure is desired — producers should slow down when consumers are overwhelmed | **Yes** |
| Need unbounded buffering (queue should grow without limit) | **No** — use an unbuffered queue or a linked list, but beware OOM |
| Single producer and single consumer with equal rates | **No** — a plain channel is simpler; the struct wrapper adds no value |
| Need persistent, durable queueing across process restarts | **No** — use Redis, Kafka, or RabbitMQ for durability |

**Alternatives**: **Unbuffered channel** (`make(chan T)`) provides synchronous handoff — producer blocks until consumer receives. Use when you need rendezvous semantics, not buffering. **`container/list` + mutex** provides unbounded queuing but requires manual synchronization. **Kafka/RabbitMQ** provide durable, distributed producer-consumer semantics with persistence, replication, and consumer groups. **Worker Pool** (topic 13) is a specialized producer-consumer where the "items" are jobs and results are collected in order.

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| Enqueue | O(1) — channel send | O(unbounded) when queue is full | O(capacity) |
| Dequeue | O(1) — channel receive | O(unbounded) when queue is empty | O(1) |
| Size/Len | O(1) — `len(q.ch)` | O(1) | — |

The O(1) enqueue/dequeue is guaranteed by Go's channel implementation — the runtime's `hchan` struct maintains a circular buffer with head and tail pointers, so sending and receiving are pointer-bumping operations. The worst-case blocking time depends on the relative speeds of producers and consumers — a slow consumer holding up a full queue delays all producers.

Space is O(capacity): the channel's buffer stores exactly `capacity` elements. Unlike unbounded queues (which can grow to `len(input)`), the memory footprint is fixed and predictable — this is the key safety property.

## Implementation Deep Dive

**1. The channel IS the queue — no extra synchronization needed.** The entire implementation (`main.go:16-42`) is 27 lines of code because Go's buffered channel provides all the necessary semantics: FIFO ordering, blocking send on full, blocking receive on empty, goroutine-safe access. There's no mutex, no condition variable, no linked list — just `make(chan T, capacity)`. This is the power of channels as a language primitive: they collapse what would be 100+ lines of synchronized data structure code into a single allocation.

**2. Backpressure is automatic and correct by construction.** When consumers are slower than producers (the normal case during bursts), `len(q.ch)` approaches `cap(q.ch)`. Once the channel is full, `q.ch <- value` blocks — the producer's goroutine suspends. This is correct behavior: the producer cannot flood the system because the bounded buffer enforces a hard limit. When consumers catch up, the channel drains, and producers unblock. No thresholds to tune, no metrics to monitor — the Go runtime scheduler handles it all.

**3. Generic `[T any]` makes the queue reusable for any type.** The `BoundedQueue[T]` signature at `main.go:16` means the same 27-line implementation works for `string` (the demo), `*Event`, `[]byte`, or any struct. The compiler generates type-specialized code — `BoundedQueue[string]` and `BoundedQueue[*Event]` are distinct types with no `interface{}` overhead. For a utility as fundamental as a queue, generics eliminate the need to copy-paste implementations for each type.

## Running the Demo

```bash
go run ./19-Producer-Consumer/
```

## Further Reading

- "The Producer-Consumer Problem" — Edsger Dijkstra (1965). Published as an appendix to the EWD 123 manuscript, this is the original formalization of the bounded buffer problem that launched the field of concurrent programming. Short and accessible.
- "Go Channels" — Go Language Specification. The formal semantics of buffered channels: capacity, blocking conditions, and ordering guarantees. Understanding the spec-level behavior of channels is essential for reasoning about producer-consumer correctness.

---

*Part of the Design-With-TsGo system design curriculum*
