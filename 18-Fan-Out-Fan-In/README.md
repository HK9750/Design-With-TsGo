# Fan-Out Fan-In

> **A parallel processing pattern that distributes work items across N worker goroutines (fan-out) and aggregates results in order (fan-in) — the Go-native equivalent of MapReduce's map phase.**

## The Problem It Solves

You're building a bulk image processing pipeline. A user uploads a ZIP file with 200 product photos. Each photo needs: decode from JPEG, resize to 800×800, apply a watermark, and re-encode to WebP. On a single core, one image takes ~200ms. Processing 200 images sequentially takes 40 seconds. Your server has 8 cores — you should be able to process 8 images simultaneously, bringing the total time down to ~5 seconds. But you need results in the original order (image[0] → result[0], image[1] → result[1]) so the frontend can display thumbnails in the correct sequence.

The naive approach is to launch 200 goroutines and collect results from a channel. Two problems: (1) order is lost — whichever goroutine finishes first sends first, so the frontend gets a shuffled array, and (2) 200 concurrent image decoders compete for memory and CPU cache, causing thrashing. The right approach is a fixed number of workers (matching CPU cores) that pull jobs from a shared queue, process them, and write results to the correct index in a pre-allocated array.

Fan-Out Fan-In is one of the most fundamental concurrent patterns in Go. "Fan-out" refers to the distribution phase: a single source (the main goroutine) sends work to multiple workers. "Fan-in" refers to the aggregation phase: results from all workers are collected into a single output structure. The pattern was formalized in Rob Pike's 2012 talk "Go Concurrency Patterns" and has been the building block for parallel processing in Go ever since.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────┐
│                    FanOutFanIn[I, O]                          │
│                                                              │
│                     FAN-OUT                                   │
│   ┌──────────────────────────────────────────────────────┐  │
│   │  Main goroutine                                       │  │
│   │                                                       │  │
│   │  for index := range items {                           │  │
│   │      jobs <- index   ──── sends index to channel      │  │
│   │  }                                                    │  │
│   │  close(jobs)          ──── signals "no more work"     │  │
│   └─────────────────────┬────────────────────────────────┘  │
│                         │                                    │
│         ┌───────────────┼───────────────┐                   │
│         ▼               ▼               ▼                   │
│   ┌──────────┐   ┌──────────┐   ┌──────────┐                │
│   │ Worker 0 │   │ Worker 1 │   │ Worker 2 │                │
│   │          │   │          │   │          │                │
│   │ for idx  │   │ for idx  │   │ for idx  │                │
│   │  := range│   │  := range│   │  := range│                │
│   │  jobs {  │   │  jobs {  │   │  jobs {  │                │
│   │  results │   │  results │   │  results │                │
│   │ [idx] =  │   │ [idx] =  │   │ [idx] =  │                │
│   │ handler  │   │ handler  │   │ handler  │                │
│   │ (items   │   │ (items   │   │ (items   │                │
│   │ [idx])   │   │ [idx])   │   │ [idx])   │                │
│   │ }        │   │ }        │   │ }        │                │
│   └────┬─────┘   └────┬─────┘   └────┬─────┘                │
│        └───────────────┼───────────────┘                     │
│                        │                                     │
│                     FAN-IN                                    │
│                        ▼                                     │
│   ┌──────────────────────────────────────────────────────┐  │
│   │  sync.WaitGroup.Wait()  ──── blocks until all done    │  │
│   │  return results[0..n]   ──── ordered output           │  │
│   └──────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

The `FanOutFanIn` function (`main.go:17-43`) is generic over input type `I` and output type `O`. It pre-allocates `results := make([]O, len(items))` and creates `jobs := make(chan int)` — an unbuffered channel of indices. Workers are goroutines that `range` over `jobs`, reading each index and computing `results[index] = handler(items[index])`. Since each worker writes to a unique index (determined by the job it pulled), there is no data race and no need for synchronization on the results array.

The fan-out phase sends every index (0, 1, 2, ..., n-1) into the `jobs` channel. The fan-in phase is implicit: `close(jobs)` signals completion, `wg.Wait()` ensures all workers finish, and the results array is returned. Unlike the Worker Pool (topic 13), this implementation sends only indices through the channel — workers read `items[index]` directly from the shared input slice. This is more memory-efficient when items are large structs (you're not copying them into the channel), but requires that items be read-only during processing.

The demo at `main.go:45-75` simulates resizing 8 images across 4 workers, with each "resize" taking 15ms. The total wall-clock time is approximately `ceil(8/4) × 15ms = 30ms` instead of `8 × 15ms = 120ms` for sequential processing.

## Production Use Cases

- **Google MapReduce** — The canonical fan-out fan-in. In the Map phase, input data is split into M shards, each processed by a mapper worker (fan-out). The shuffle phase collects intermediate key-value pairs and groups them by key. In the Reduce phase, R reducer workers process each group (fan-in). The original 2004 paper inspired Hadoop, Spark, and the entire big data ecosystem.
- **Parallel HTTP scraping** — Web crawlers fan out: a coordinator sends URLs to N worker goroutines, each fetches and parses an HTML page. Results fan in: parsed links, metadata, and extracted text are collected into a shared index or message queue. Colly (Go's web scraping framework) uses this exact pattern with a configurable `Async` mode and `Limit(&colly.LimitRule{Parallelism: N})`.
- **Bulk image resizing (Imgix)** — Imgix's rendering pipeline fans out image transform requests across worker nodes. Each node processes a subset of images, applying resize, crop, watermark, and format conversion. The results are fan-in'd via a CDN edge layer that assembles the final response.
- **Parallel database queries** — When a dashboard needs data from 12 different aggregate queries, a service fans out to 12 goroutines (or 12 database replicas via a load balancer). Results fan in via a shared aggregation struct or a `sync.WaitGroup`. Amazon Redshift's query execution engine uses fan-out fan-in to distribute query fragments across compute nodes.
- **Go's `errgroup` with parallel execution** — `errgroup.Group` provides `Go(func() error)` to fan out and `Wait() error` to fan in, with automatic error collection. Combined with `SetLimit(N)`, it provides bounded worker-pool semantics. Used internally in Kubernetes controllers, Terraform providers, and thousands of Go microservices.

## When to Use It

| Scenario | Use Fan-Out Fan-In? |
|----------|--------------------|
| CPU-bound batch processing where per-item cost is uniform | **Yes** |
| Input is a known slice with a fixed size | **Yes** |
| Output must preserve input ordering | **Yes** |
| Processing order doesn't matter (unordered results are acceptable) | **No** — a simple result channel is simpler and slightly faster |
| Items are streaming in real-time (not a known batch) | **No** — use Producer-Consumer (topic 19) |
| Single item processing time dominates (one item takes 10s, all others 10ms) | **No** — parallelism gains are negligible; a straggler blocks the final Wait |
| Handler function can error | **No** — this implementation doesn't support error propagation; extend it with a separate error channel or use `errgroup` |

**Alternatives**: **Worker Pool** (topic 13) is the same pattern but sends the full value through the channel — better when items are small structs, worse for memory when items are large. **Streaming fan-in** (multiple goroutines sending to a single channel) is better for real-time workloads where results should be processed as they arrive. **MapReduce** is a distributed fan-out fan-in across machines — use when the dataset exceeds single-machine memory.

## Complexity Analysis

| Metric | Value |
|--------|-------|
| Job distribution time | O(n) — single pass over indices |
| Per-worker work | O(n/workers) on average |
| Wall-clock time (uniform cost) | O(n/workers) — up to workers× speedup |
| Wall-clock time (skewed cost) | O(max(per-item cost)) — bottlenecked by the slowest item |
| Space | O(n) — results array + O(1) channel buffer |
| Ordering guarantee | Strict — results[i] always from items[i] |

The channel is unbuffered (`make(chan int)`), providing synchronous handoff: the distributor blocks until a worker picks up the job. For large batches, a buffered channel (e.g., `make(chan int, 2*workers)`) can improve throughput by decoupling distribution from processing, at the cost of slightly higher memory.

## Implementation Deep Dive

**1. Sending indices (not values) through the channel preserves memory for large structs.** The `jobs chan int` at `main.go:20` carries only integers — 8 bytes each. Workers access `items[index]` directly. Compare this to the Worker Pool (topic 13), which sends `job{index, value}` through the channel — copying the entire value struct. For an `Image` struct that's 8MB (raw pixel data), the index-only approach avoids a 8MB channel copy. The tradeoff is that all workers share read access to `items` — this is safe because they only read, never write, and the slice is not modified during processing.

**2. `close(jobs)` signals termination without a poison pill.** At `main.go:38`, closing the `jobs` channel causes all `range jobs` loops to exit after draining the last index. This is the canonical Go termination pattern — cleaner than sending a sentinel value (-1) that each worker must check, and faster because the runtime handles it at the channel level. The `sync.WaitGroup` at `main.go:21,24,39` ensures that `FanOutFanIn` doesn't return until every worker has finished processing every index.

**3. Generic `[I any, O any]` enables type-safe reuse.** The signature at `main.go:17` uses Go generics to make the function work with any input/output types. The compiler monomorphizes the code — there's no `interface{}` boxing. The demo uses `int → int`, but the same function can process `Image → Thumbnail`, `URL → *http.Response`, or `Record → Report`. This is the key advantage of Go 1.18+ for infrastructure code: write once, use everywhere, with compile-time type safety.

## Running the Demo

```bash
go run ./18-Fan-Out-Fan-In/
```

## Further Reading

- "Go Concurrency Patterns" — Rob Pike (Google I/O, 2012). The talk that introduced fan-out/fan-in to the Go community. Covers the pattern alongside pipelines, cancellation, and timeouts. The slides and video are available on the Go blog.
- "MapReduce: Simplified Data Processing on Large Clusters" — Dean and Ghemawat (OSDI, 2004). The paper that defined the distributed fan-out fan-in pattern. While MapReduce operates across machines and Fan-Out Fan-In operates within a single process, the architectural parallels are instructive.

---

*Part of the Design-With-TsGo system design curriculum*
