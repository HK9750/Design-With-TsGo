# Column Store

> **A database that stores data by column instead of by row, enabling analytical queries to read only the 3 columns they need out of 200 — the foundational architecture behind Parquet, ClickHouse, and BigQuery.**

## The Problem It Solves

You're the data engineer for a SaaS billing platform. Your analytics team runs a weekly query: "What's the total revenue per country for paid invoices?" The invoices table has 200 columns (customer metadata, tax identifiers, line items, payment gateway logs) and 500 million rows. The query needs only 3 columns: `country`, `amount`, and `status`. In a row-oriented database (PostgreSQL, MySQL), reading those 3 columns means reading the entire row — all 200 columns, about 2KB per row. For 500M rows, that's 1TB of I/O. Even with SSDs at 500MB/s, the query takes 33 minutes.

The core problem is that row-oriented storage is optimized for OLTP (Online Transaction Processing) — inserting a new invoice means writing one contiguous row. But analytics (OLAP — Online Analytical Processing) only touches a few columns across many rows. Reading 200 columns to use 3 is 98.5% wasted I/O. Column stores flip the layout: each column is stored as a separate, contiguous array. The `country` column is one array of 500M values; `amount` is another; `status` is a third. The query reads exactly those 3 arrays — about 30MB total — achieving the same result in under a second.

Columnar storage also unlocks extreme compression. A column of 500M country codes probably has only 200 distinct values — ideal for dictionary encoding (store a lookup table of 200 values + 500M small integers). Integer columns can use run-length encoding (1,000,000 consecutive zeros). Timestamp columns can use delta-of-delta encoding (store the difference between consecutive timestamps, which is usually small). Row-oriented storage can't compress as effectively because values in a row are heterogeneous — a country code next to a floating-point amount next to a timestamp don't form patterns.

## Architecture & Internals

```
            ROW-ORIENTED vs COLUMN-ORIENTED
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  ROW LAYOUT (OLTP):                                │
    │                                                     │
    │  Row 0: [INV-001, C42, PK, 150.00, USD, paid]     │
    │  Row 1: [INV-002, C17, US, 320.00, USD, pending]  │
    │  Row 2: [INV-003, C42, PK,  89.50, USD, paid]     │
    │  Row 3: [INV-004, C99, UK, 210.00, GBP, paid]     │
    │  Row 4: [INV-005, C17, US,  75.00, USD, overdue]  │
    │                                                     │
    │  Query "SELECT country, amount WHERE status=paid": │
    │    → Reads ALL rows, ALL columns → wasted I/O      │
    │                                                     │
    └─────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  COLUMN LAYOUT (OLAP):                             │
    │                                                     │
    │  invoice_id: [INV-001, INV-002, INV-003, INV-004, INV-005]
    │  customer_id:[C42,     C17,     C42,     C99,     C17]
    │  country:    [PK,      US,      PK,      UK,      US]
    │  amount:     [150.00,  320.00,  89.50,   210.00,  75.00]
    │  currency:   [USD,     USD,     USD,     GBP,     USD]
    │  status:     [paid,    pending, paid,    paid,    overdue]
    │                                                     │
    │  Query "SELECT country, amount":                   │
    │    → Reads ONLY country + amount columns → minimal I/O
    │                                                     │
    └─────────────────────────────────────────────────────┘

               COLUMN PROJECTION PIPELINE
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  Project(["country", "amount"])                     │
    │                                                     │
    │  Step 1: For each row index (0 to rows-1):          │
    │    Read columns["country"][i]                       │
    │    Read columns["amount"][i]                        │
    │    Assemble into row map                            │
    │                                                     │
    │  Result:                                           │
    │    [{country:PK, amount:150.00},                    │
    │     {country:US, amount:320.00},                    │
    │     {country:PK, amount:89.50},                     │
    │     {country:UK, amount:210.00},                    │
    │     {country:US, amount:75.00}]                     │
    │                                                     │
    └─────────────────────────────────────────────────────┘

          SPARSE COLUMN HANDLING (BACKFILL)
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  Row 0: {event: "login", user: "alice"}            │
    │  Row 1: {event: "purchase", user: "bob", item: "book"}
    │  Row 2: {event: "logout", user: "alice", duration_ms: "1200"}
    │                                                     │
    │  ❌ Row 0 has no "item" column                      │
    │  Solution: pad with empty string                    │
    │                                                     │
    │  event:       ["login",    "purchase", "logout"]    │
    │  user:        ["alice",    "bob",      "alice"]     │
    │  item:        ["",         "book",     ""]          │
    │  duration_ms: ["",         "",         "1200"]      │
    │                                                     │
    └─────────────────────────────────────────────────────┘
```

**Key structures:**

- `ColumnStore.columns` (`main.go:18`): A `map[string][]string` where each column name maps to a slice of string values. All slices have the same length (`rows`), and the value at index `i` in each slice belongs to row `i`. This is a *column family* layout — simpler than pure columnar but with the same projection benefits.
- `ColumnStore.rows` (`main.go:19`): A counter tracking the total number of rows, used for backfilling and projection bounds.

## Production Use Cases

- **Apache Parquet**: The standard columnar file format for Hadoop/Spark ecosystems. Used by Netflix (data warehousing), Uber (trip analytics), and every company running Spark SQL. Parquet files contain row groups (~128MB), each with column chunks, statistics (min/max per column chunk for predicate pushdown), and dictionary pages.
- **Apache ORC**: Optimized Row Columnar — a competing columnar format used by Hortonworks/Hive. ORC adds lightweight indexes (min, max, sum, count) per stripe for even faster filtering.
- **ClickHouse**: An open-source OLAP database that can process billions of rows per second on a single server. Uses columnar storage with vectorized query execution (SIMD), adaptive compression (LZ4, ZSTD, Delta), and materialized views. Used by Cloudflare (DNS analytics), Yandex (Metrika), and Contentsquare.
- **Amazon Redshift**: AWS's data warehouse, built on a columnar version of PostgreSQL 8. With columnar compression and zone maps (min/max per block), achieving 3-4x compression ratios.
- **Google BigQuery / Snowflake**: Cloud-native data warehouses with columnar storage at petabyte scale. BigQuery's Capacitor format uses columnar storage with nested record shredding. Snowflake's micro-partitions store columnar data with automatic clustering.

## When to Use It

| Use Column Store when... | Don't use when... |
|---|---|
| Queries scan many rows but few columns (analytics, reporting) | Workload is OLTP — point reads/writes of entire rows |
| You need high compression ratios (10× or more) | Data is updated frequently and columnar write overhead is high |
| Aggregations (SUM, COUNT, AVG) over a subset of columns | You frequently `SELECT *` and need the entire row |
| Data is append-only or batch-loaded | Transactions modify individual columns in place |
| Queries run on cold data (infrequently accessed historical data) | Sub-millisecond latency for individual row lookups is required |

**Columnar writes are slow (but it doesn't matter)**: Inserting a single row into a column store means writing to every column array — O(c) where c is the number of columns. In a row store, it's one write. This is why OLTP systems use row storage and OLAP systems use column storage. The tradeoff is explicit: column stores optimize reads at the expense of writes. For analytics, this is acceptable because data is typically batch-loaded (nightly ETL) and queried heavily, not inserted one row at a time.

**Hybrid systems are emerging**: Some databases (TiDB's TiFlash, SingleStore) use row storage for transactional data and automatically replicate to columnar storage for analytics — called HTAP (Hybrid Transactional/Analytical Processing). MySQL's HeatWave is an in-memory columnar accelerator for MySQL. The industry is converging on "use the right layout for the right workload."

## Complexity Analysis

| Operation | Time | Space |
|---|---|---|
| Insert (one row, c columns) | O(c) | O(c) values |
| Insert (n rows with k new columns) | O(n + k × rows) | O(k × rows) for backfill |
| Project (p columns) | O(rows × p) | O(rows × p) output |
| Single column scan | O(rows) | O(1) read, O(rows) if returning |
| Space (total) | N/A | O(rows × columns) raw, O(rows × columns × 0.1) compressed |

## Implementation Deep Dive

### 1. Per-Column Slices (`main.go:18,33-52`)

The core data structure is `columns map[string][]string` — each column is an independent slice. When inserting a row, we iterate over existing columns and append the value from the row (or an empty string if the column is absent from this row — sparse column handling). Then we iterate over the row's keys to detect new columns: if a column doesn't exist yet, we create it by making a slice of `rows` empty strings (backfill) and appending the new value. This ensures all columns always have the same length. The `rows` counter tracks how many rows have been inserted, serving as the length reference.

### 2. Column Projection (`main.go:58-72`)

`Project()` is the primary read path and the column store's superpower. Given a list of column names (e.g., `["country", "amount"]`), it reads only those columns and assembles row maps. For each row index `i` from 0 to `rows-1`, it creates a `map[string]string` with the projected columns' values at that index. The result is a slice of row maps — the same shape as the original input, but reconstructed from columnar storage. This is the inverse of Insert. For production systems, projection skips row assembly entirely: the query engine operates directly on column vectors, performing vectorized `country == "US"` filtering and `SUM(amount)` aggregation without ever reconstructing rows.

### 3. Sparse Column Handling (`main.go:37-42,154-157`)

Our store handles *sparse* data — where different rows have different columns. When a row lacks a column that already exists, we insert an empty string (the null placeholder). When a new column is introduced, all prior rows get backfilled with empty strings. This is a simplified null handling strategy. Production columnar formats have explicit null bitmaps (one bit per value indicating null) to distinguish "value is zero-length string" from "value is null." Parquet uses definition levels; ORC uses null bitmaps. Our empty-string approach works for string-only data but would need revision for numeric columns (where 0 and null must be distinguishable).

## Running the Demo

```bash
go run ./30-Column-Store/
```

The demo simulates a billing analytics platform, inserting 5 invoices with 6 columns each. It then demonstrates column projection: querying only `country` and `amount` for revenue analysis, projecting a single column (`invoice_id`), and reconstructing full rows (SELECT *). An edge case demonstrates sparse column handling with variable-width events.

## Further Reading

- **"C-Store: A Column-oriented DBMS"** — Mike Stonebraker et al. (VLDB, 2005). The seminal paper that launched the column-store revolution. Introduced the concepts of projections (column subsets sorted by different keys for different query patterns), columnar compression, and the write-optimized store (WOS) + read-optimized store (ROS) architecture. C-Store directly influenced Vertica and the entire OLAP industry.
- **Dremel / Parquet Paper — "Dremel: Interactive Analysis of Web-Scale Datasets"** — Sergey Melnik et al., Google (VLDB, 2010). Describes Google's columnar format for nested data (Protocol Buffers), including the record shredding and assembly algorithm that became the basis for Apache Parquet's definition/repetition level encoding.

---

*Part of the Design-With-TsGo system design curriculum*
