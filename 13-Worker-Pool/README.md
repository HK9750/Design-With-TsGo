# Worker Pool

> **A fixed-size pool of goroutines that parallelizes a batch of jobs across workers while preserving input order in the output array — the foundational pattern for CPU-bound batch processing in Go.**

## The Problem It Solves

Your image processing service receives a batch of 5,000 product photos that need thumbnailing. Each photo takes ~80ms to decode, resize, and re-encode. Processing them sequentially one-at-a-time takes 5,000 × 80ms = 400 seconds — nearly 7 minutes. The user who uploaded these photos is staring at a loading spinner. Your server has 16 cores, but you're using one. You need to parallelize the work.

The naive approach is to launch 5,000 goroutines — one per image. Go's runtime can handle millions of goroutines, so this seems fine. But image processing allocates large decode buffers (megabytes each), and 5,000 concurrent decoders would overwhelm memory, triggering GC pauses and OOM kills. Thread pools exist precisely for this reason: they bound concurrency to a fixed number of workers (matching CPU cores or some resource constraint), distributing work from a shared queue. Workers pull jobs, process, pull the next — no goroutine explosion, no memory explosion.

A subtler requirement is output ordering. The caller expects `results[i]` to correspond to `jobs[i]`. If you launched goroutines and collected results via a channel, you'd lose ordering — whichever goroutine finishes first writes first. You could tag each result with an index and sort, but that adds O(n log n) overhead. The worker pool pattern solves this by writing results directly to a pre-allocated array at the correct index, preserving order in O(n) time with no sorting step.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────┐
│                    RunWorkerPool[I, O]                        │
│                                                              │
│   jobs[0..n] ───────────────┐                                │
│                             ▼                                │
│                    ┌──────────────────┐                      │
│                    │   job{idx, val}   │  ── channel ──▶     │
│                    └──────────────────┘       queue           │
│                             │                                │
│          ┌──────────────────┼──────────────────┐            │
│          ▼                  ▼                  ▼            │
│   ┌────────────┐    ┌────────────┐    ┌────────────┐       │
│   │  Worker 0  │    │  Worker 1  │    │  Worker 2  │       │
│   │            │    │            │    │            │       │
│   │ for item   │    │ for item   │    │ for item   │       │
│   │  := range  │    │  := range  │    │  := range  │       │
│   │  queue {   │    │  queue {   │    │  queue {   │       │
│   │   results  │    │   results  │    │   results  │       │
│   │  [idx] =   │    │  [idx] =   │    │  [idx] =   │       │
│   │  handler   │    │  handler   │    │  handler   │       │
│   │  (val)     │    │  (val)     │    │  (val)     │       │
│   │ }          │    │ }          │    │ }          │       │
│   └──────┬─────┘    └──────┬─────┘    └──────┬─────┘       │
│          └──────────────────┼──────────────────┘            │
│                             │ sync.WaitGroup.Wait()          │
│                             ▼                                │
│                    ┌──────────────────┐                      │
│                    │  results[0..n]   │  (order preserved)   │
│                    └──────────────────┘                      │
└──────────────────────────────────────────────────────────────┘
```

The `RunWorkerPool` function (`main.go:17-54`) is generic over input type `I` and output type `O`. It pre-allocates `results := make([]O, len(jobs))` — an output array indexed identically to the input. A worker is a goroutine that loops over a shared `chan job`, where each `job` is a struct carrying `{index int, value I}`. The worker computes `results[item.index] = handler(item.value)` — direct assignment into the pre-allocated position, so order is preserved no matter which worker finishes when.

The main goroutine acts as the distributor: it iterates over `jobs`, sending each `{index, value}` pair into the queue channel. After all jobs are enqueued, it `close(queue)` — this is the termination signal. Workers that were blocked on `range queue` see the channel close and exit their loops. A `sync.WaitGroup` tracks all workers; `wg.Wait()` blocks the caller until every worker has processed every job.

This pattern is a form of fan-out (distributing work) with order-preserving aggregation. Unlike Fan-Out-Fan-In (topic 18), which sends only indices, this implementation sends the full value — the tradeoff is slightly more channel data in exchange for simpler worker logic (no need for shared access to the input slice).

## Production Use Cases

- **Go's `golang.org/x/sync/errgroup`** — The `errgroup` package provides `SetLimit(n)` to cap goroutine concurrency. Internally, it uses a semaphore channel to limit spawned goroutines, equivalent to a worker pool with a shared error accumulator. Used widely in Go servers to bound parallel upstream calls.
- **FastHTTP worker pool** — `fasthttp` maintains a pool of worker goroutines that accept `net.Conn` objects from a listener, process HTTP requests, and return to the pool. This avoids per-connection goroutine churn and gives FastHTTP its ~10x throughput advantage over `net/http` on connection-bound workloads.
- **Imgix image processing pipeline** — Imgix, the real-time image processing CDN, processes millions of image transformations per second. Their worker pool distributes incoming image URLs across a bounded number of processors to prevent resource exhaustion while maximizing cache throughput.
- **Cloudinary media pipeline** — Cloudinary's media transformation service uses worker pools to parallelize image/video encoding across compute nodes. Each transformation job (resize, crop, format-convert) is distributed to a pool worker and results are assembled in order before returning to the CDN edge.
- **Go's `net/http` server** — While `net/http` doesn't use an explicit worker pool, it achieves the same effect: the `Server.Serve` method spawns a goroutine per connection, but the connection count is bounded by `http.Server.MaxConnsPerIP` and the listener's accept backlog, preventing unbounded goroutine growth.

## When to Use It

| Scenario | Use Worker Pool? |
|----------|-----------------|
| CPU-bound batch processing with uniform per-job cost | **Yes** |
| IO-bound batch processing where concurrency must be bounded to respect downstream rate limits | **Yes** |
| Need results in the same order as inputs | **Yes** |
| Per-job processing time is highly variable (stragglers delay the entire batch) | **No** — use streaming with unordered results |
| Jobs are streaming in real-time (not a known batch) | **No** — use the Producer-Consumer pattern (topic 19) |
| Single job dominates total time (one job = 10 minutes, all others = 10ms) | **No** — worker pool parallelism offers no benefit for a single large job |

**Alternatives**: **Fan-Out-Fan-In** (topic 18) is the same concept but sends indices through the channel, letting workers read from the shared input slice — more memory efficient for large structs. **Semaphore** (topic 15) provides concurrency limiting without the job queue infrastructure — better when you need to bound goroutine count but don't need a pre-defined batch. **Producer-Consumer** (topic 19) is the right choice when jobs arrive continuously rather than as a known batch.

## Complexity Analysis

| Metric | Value |
|--------|-------|
| Work distribution time | O(n) — single pass over jobs to enqueue |
| Per-worker work | O(n/workers) — approximately equal distribution |
| Total work | O(n) — same as sequential, but wall-clock time reduced by up to workers× |
| Space | O(n) — results array + channel buffer (1 slot per enqueued job) |
| Ordering guarantee | Strict — `results[i]` always from `jobs[i]` |

The channel is unbuffered (`make(chan job)`), which means the distributor blocks whenever all workers are busy — this provides natural backpressure. A buffered channel would allow the distributor to enqueue ahead but risks memory blowup for large batches.

## Implementation Deep Dive

**1. The job struct carries its own index.** Each `job` at `main.go:26-29` bundles `{index int, value I}`. The index is the contract that preserves ordering — it tells the worker where to write the result, not when. Workers race to claim jobs from the queue, but each writes to its own pre-allocated slot. Even if worker 3 finishes before worker 0, `results[3]` gets written and `results[0]` remains empty until worker 0 finishes — the array correctly represents the mapping from input to output without any synchronization on the results array itself.

**2. `close(queue)` is the termination signal, not a poison pill.** At `main.go:49`, closing the channel causes all `range queue` loops to exit naturally. This is cleaner and faster than sending a sentinel value (like a nil job) that each worker must check — the Go runtime handles channel-close semantics natively. The `sync.WaitGroup` ensures all workers finish their final iteration before `RunWorkerPool` returns.

**3. Generic types [I any, O any] make the pool reusable.** The function signature `RunWorkerPool[I any, O any]` at `main.go:17` means the same code works for `int → int` (demo), `Image → Thumbnail`, `URL → HTTPResponse`, or any other pair of types. The compiler specializes the code at compile time — there's no `interface{}` boxing, no reflection, no type assertions. This is a key advantage of Go generics for infrastructure code.

## Running the Demo

```bash
go run ./13-Worker-Pool/
```

## Further Reading

- "Go Concurrency Patterns: Pipelines and Cancellation" — Go Blog. Rob Pike's canonical guide to channel-based pipeline design in Go, including fan-out/fan-in, worker cancellation, and error handling in concurrent pipelines.
- "Java Concurrency in Practice" — Brian Goetz, Chapter 8 (Applying Thread Pools). While Java-centric, the analysis of sizing thread pools (CPU-bound vs IO-bound), queue types (bounded vs unbounded), and saturation policies applies universally to any worker pool implementation.

---

*Part of the Design-With-TsGo system design curriculum*
