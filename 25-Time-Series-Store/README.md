# Time-Series Store

> **A database optimized for timestamped, append-heavy metric data with tag-based multi-dimensional queries and time-range aggregations — the engine behind every monitoring dashboard.**

## The Problem It Solves

You're an SRE managing 5,000 microservices across 3 regions. Each service emits CPU, memory, request latency, and error count every 10 seconds. That's 5,000 × 4 × 6 = 120,000 data points per minute, or 172 million per day. You need to answer questions like: "What was the average CPU usage of web-01 in us-east between 2:00 and 2:30 AM?" and "Show me all hosts where memory exceeded 80% in the last hour."

A relational database can store this — a table with columns `(timestamp, metric, host, region, value)`. But query performance degrades as billions of rows accumulate. The `WHERE metric='cpu' AND host='web-01' AND timestamp BETWEEN x AND y` query requires a composite index on `(metric, host, timestamp)`, which grows to terabytes. Worse, time-series data is write-heavy and append-only — nobody updates yesterday's CPU reading. Traditional databases pay the overhead of read-write page management for data that's never modified.

A time-series store is purpose-built for this pattern. Data is organized by *series* — a unique combination of metric name and tags (key-value labels). All points for a given series are stored together, sorted by timestamp. This makes range queries natural (read a contiguous slice) and aggregation efficient (iterate one series, not the whole table). The write path is optimized for high-throughput ingestion: append + sort. The read path supports filtering by time range and tags, and computing aggregates (AVG, SUM, MAX, P99) over arbitrarily large windows.

## Architecture & Internals

```
                    DATA ORGANIZATION
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  Ingested Point:                                    │
    │    {ts:3, metric:"cpu", tags:{host:"web-01",        │
    │              region:"us-east"}, value:0.38}          │
    │        │                                            │
    │        ▼  seriesKey() → "cpu|host=web-01,region=us-east"
    │                                                     │
    │  ┌─────────────────────────────────────────────┐    │
    │  │  series map (map[string][]TimePoint)         │    │
    │  │                                             │    │
    │  │  "cpu|host=web-01,region=us-east"           │    │
    │  │    → [{ts:1, 0.45}, {ts:2, 0.72}, {ts:3, 0.38}] │
    │  │                                             │    │
    │  │  "cpu|host=web-02,region=us-west"           │    │
    │  │    → [{ts:1, 0.61}, {ts:2, 0.55}]           │    │
    │  │                                             │    │
    │  │  "mem|host=web-01,region=us-east"           │    │
    │  │    → [{ts:1, 0.82}, {ts:2, 0.79}, {ts:3, 0.86}] │
    │  │                                             │    │
    │  │  "disk|host=web-01,region=us-east"          │    │
    │  │    → [{ts:1, 0.33}]                         │    │
    │  └─────────────────────────────────────────────┘    │
    │                                                     │
    └─────────────────────────────────────────────────────┘

              QUERY: Average("cpu", {host:"web-01"}, 1, 3)
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  1. Look up series key "cpu|host=web-01,region=..." │
    │  2. Filter points: ts >= 1 and ts <= 3              │
    │     → [0.45, 0.72, 0.38]                            │
    │  3. Compute: (0.45 + 0.72 + 0.38) / 3 = 0.5167     │
    │                                                     │
    │  Result: avg = 0.5167, alert if > 0.5              │
    │                                                     │
    └─────────────────────────────────────────────────────┘

                  TAG-BASED KEYING
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  Tags: {host: "web-01", region: "us-east"}          │
    │     │                                               │
    │     ▼  sorted alphabetically by tag key             │
    │  "host=web-01,region=us-east"                       │
    │     │                                               │
    │     ▼  prepend metric name                          │
    │  "mem|host=web-01,region=us-east"                   │
    │                                                     │
    │  This deterministic key ensures:                    │
    │    1. Same tags always produce the same key         │
    │    2. Tag order doesn't matter (sorted internally)  │
    │    3. Series are uniquely identifiable              │
    │                                                     │
    └─────────────────────────────────────────────────────┘
```

**Key design choices:**

- **Composite key = metric + sorted tags**: The `seriesKey()` function (`main.go:87-98`) concatenates the metric name with alphabetically-sorted tag pairs (e.g., `"cpu|host=web-01,region=us-east"`). This ensures that `{host: "web-01", region: "us-east"}` and `{region: "us-east", host: "web-01"}` produce the identical series key, even if provided in different order. This is the same approach used by Prometheus.
- **Sorted insertion via sort.Slice**: After every `Ingest`, the series slice is re-sorted by timestamp (`main.go:45`). This is O(n log n) per insert — appropriate for a demo but not production. Real systems maintain sorted order during insertion (binary search + shift) or use an append-only format with periodic compaction.
- **Range queries scan linearly**: `Query()` iterates all points in a series and filters by timestamp range. This is O(series length). Production systems use time-partitioned blocks (e.g., 2-hour chunks) to skip irrelevant time ranges entirely.

## Production Use Cases

- **Prometheus TSDB**: The most widely deployed open-source time-series database. Powers monitoring at SoundCloud, DigitalOcean, and thousands of Kubernetes clusters. Uses a custom TSDB with 2-hour block compaction, inverted index for label matching, and a WAL for durability.
- **InfluxDB**: Purpose-built time-series database in Go. Supports nanosecond timestamps, continuous queries (precomputed aggregations), and downsampling policies. Used by Tesla, IBM, and Comcast for IoT and infrastructure monitoring.
- **TimescaleDB**: PostgreSQL extension that auto-partitions tables by time and compresses old chunks. Provides full SQL with time-series optimizations (hyperfunctions for percentiles, histograms). Used by CERN, Comcast, and thousands of others.
- **Graphite**: Pioneer of time-series monitoring at scale. Used by GitHub, Etsy, and Booking.com. Stores metrics as whisper files (fixed-size circular buffers) for predictable disk usage.
- **Datadog / VictoriaMetrics**: Commercial monitoring platforms processing trillions of data points daily. VictoriaMetrics is open-source, written in Go, and achieves 20× compression vs. Prometheus for the same data.

## When to Use It

| Use Time-Series Store when... | Don't use when... |
|---|---|
| Data is append-only with a timestamp primary key | You need frequent updates to historical data |
| Queries filter by time range and metric tags | Your primary query pattern is full-text search |
| Aggregations (AVG, SUM, P99) are the core use case | You need multi-table JOINs (use relational DB) |
| High write throughput (millions of points/second) | Writes are infrequent and reads are ad-hoc |
| Data retention is time-based (delete after 30 days) | You need random-access by arbitrary keys |

**Cardinality is the enemy**: Time-series stores struggle with high *cardinality* — the number of unique series. If you use user IDs as tags, 1 million users × 10 metrics = 10 million series. Each series requires its own sorted slice, memory overhead, and index entries. Production systems recommend keeping total series count under ~10 million per node. This is why Prometheus warns against using email addresses or request IDs as labels.

**Downsampling is essential for long retention**: Keeping per-second data for a year is cost-prohibitive. Production systems automatically downsample: after 24 hours, data is rolled up to 5-minute averages; after 30 days, to 1-hour averages. This is called a *retention policy* and *rollup*.

## Complexity Analysis

| Operation | Time (our impl) | Time (production) | Space |
|---|---|---|---|
| Ingest | O(n log n) per series | O(log n) with sorted insert | O(points) |
| Query (range) | O(k) where k = series points | O(log n + k/B) with blocks | O(1) |
| Average | O(k) | O(1) with pre-aggregated blocks | O(1) |
| seriesKey | O(t log t) where t = tags | O(t log t) | O(key length) |
| Total space | O(total points) | O(compressed points) ~1-3 bytes/point | |

## Implementation Deep Dive

### 1. Deterministic Series Key (`main.go:87-98`)

The `seriesKey` function creates a canonical string representation by sorting tag keys alphabetically before concatenating. This is critical for correctness: `Query("cpu", {host:"web-01", region:"us-east"}, ...)` must find the same series regardless of how the tags were provided. The sort uses `sort.Strings` on the tag keys, then builds a deterministic string. The pipe `|` separates metric from tags; commas separate tag pairs; equals signs separate tag keys from values. This format is human-readable and URL-safe.

### 2. Sorted Insertion (`main.go:41-47`)

Each `Ingest` appends the point to the series slice and then calls `sort.Slice` to re-sort by timestamp. This is the simplest possible approach — and inefficient for production. A better approach (used by Prometheus) is to find the insertion point via binary search and shift elements, maintaining O(n) per insert (still expensive) or to buffer incoming points and sort in batches (amortized O(n log n) per batch). The absolute best approach is to use an append-only format where data naturally arrives in timestamp order (most monitoring systems push data in near-real-time), making sorting unnecessary.

### 3. Average Aggregation (`main.go:68-82`)

`Average` reuses `Query` to get matching points, then computes the arithmetic mean. This is a simple single-series aggregation. Production systems support: multi-series aggregation (average CPU across all hosts in a region), windowed aggregation (rolling 5-minute average), and percentile approximations (P99 latency using t-digest). The separation of `Query` (data retrieval) and `Average` (computation) is deliberate — it allows plugging in different aggregation functions (SUM, MAX, MIN, COUNT) over the same query results.

## Running the Demo

```bash
go run ./25-Time-Series-Store/
```

The demo creates a time-series store, ingests 9 data points across CPU, memory, and disk metrics tagged by host and region, then runs range queries with average aggregations to simulate alerting on threshold violations.

## Further Reading

- **"Gorilla: A Fast, Scalable, In-Memory Time Series Database"** — Tuomas Pelkonen et al., Facebook (2015). Describes Facebook's in-memory TSDB that powered their monitoring system with 700 million data points/minute, using delta-of-delta timestamp compression and XOR value compression achieving 10× space reduction.
- **Prometheus TSDB Documentation (prometheus.io/docs/prometheus/latest/storage)** — Detailed explanation of Prometheus's on-disk format: 2-hour blocks, index file structure, tombstones, and compaction. The most accessible production-grade TSDB internals documentation available.

---

*Part of the Design-With-TsGo system design curriculum*
