# LSM-Tree

> **Write-optimized storage engine that turns random writes into sequential I/O by buffering in memory and merging sorted tables — the engine behind RocksDB, Cassandra, and LevelDB.**

## The Problem It Solves

You're running a metrics ingestion pipeline. Every second, 50,000 application servers push CPU, memory, and request-latency samples to your time-series database. That's 4.3 billion writes per day — all random key space. If you use a B-Tree, each write triggers a random disk page update. At 100 random writes per second per disk, you'd need 500 spinning disks just to keep up. Your cloud bill would be astronomical.

The Log-Structured Merge-Tree (LSM-Tree) solves this by inverting the write/read tradeoff. Instead of writing to random disk locations, it buffers all writes in a sorted in-memory structure (the *memtable*). When the memtable fills up, it's flushed to disk as a sorted, immutable file called an SSTable (Sorted String Table). Because the memtable is already sorted, the flush writes sequentially — the fastest thing a disk can do. Reads scan from newest to oldest: check the memtable first, then each SSTable in reverse chronological order. Writes become O(1) amortized; reads become O(k) where k is the number of SSTables.

The catch is that over time, you accumulate hundreds of SSTables, and reads must check each one. That's where *compaction* comes in: a background process merges multiple SSTables into one, keeping only the latest version of each key and discarding tombstones (deletion markers). This is why LSM-Trees are sometimes called "write-optimized but read-penalized" — you trade read amplification (scanning multiple files) for near-zero write amplification (sequential appends).

## Architecture & Internals

```
                          WRITE PATH
    ┌─────────────────────────────────────────────────────┐
    │  Put("cpu.host1", "0.55")                           │
    │      │                                               │
    │      ▼                                               │
    │  ┌───────────────────┐                               │
    │  │   MEMTABLE (RAM)   │  ← Sorted in-memory map      │
    │  │  {cpu.host1: 0.55,│    Sequence numbers track    │
    │  │   mem.host1: 0.83} │    write order              │
    │  └───────┬───────────┘                               │
    │          │ threshold reached (size >= 4)              │
    │          ▼                                           │
    │  ┌───────────────────┐                               │
    │  │   SSTable[0]       │  ← Immutable on-disk segment │
    │  │  (flushed, sorted) │    Written sequentially      │
    │  └───────────────────┘                               │
    └─────────────────────────────────────────────────────┘

                          READ PATH
    ┌─────────────────────────────────────────────────────┐
    │  Get("cpu.host1")                                    │
    │      │                                               │
    │      ├──1. Check memtable (newest)                   │
    │      │     Found? Return value                       │
    │      │                                               │
    │      ├──2. Check SSTable[N-1] (newest on disk)       │
    │      │     Found? Return value                       │
    │      │                                               │
    │      ├──3. Check SSTable[N-2]                        │
    │      │     ...                                       │
    │      │                                               │
    │      └──4. Check SSTable[0] (oldest)                 │
    │            Found? Return value : Not found           │
    └─────────────────────────────────────────────────────┘

                       COMPACTION
    ┌─────────────────────────────────────────────────────┐
    │  SSTable[0] + SSTable[1] + ... + SSTable[N-1]       │
    │         │  merge (keep highest seq per key)          │
    │         ▼                                           │
    │  ┌───────────────────┐                               │
    │  │   SSTable[0]       │  ← Single merged segment     │
    │  │  (compacted,       │    Tombstones removed        │
    │  │   deduplicated)     │    Space reclaimed          │
    │  └───────────────────┘                               │
    └─────────────────────────────────────────────────────┘
```

**Key structures in our Go implementation:**

- `LSMValue` (`main.go:14`): Each value carries a `Sequence` (monotonic counter) and a `Deleted` flag. The sequence number is critical — during compaction, when the same key appears in multiple SSTables, we keep the entry with the highest sequence. This is multi-version concurrency without locks.
- `LSMTree` (`main.go:25`): Holds an in-memory `memtable` (map[string]LSMValue), a slice of `sstables` (each a map simulating an on-disk file), a global `sequence` counter, and a `flushThreshold` that triggers automatic flushing.
- The `flushThreshold` (set to 4 by default) controls write buffering. Higher values batch more writes per flush, reducing write amplification at the cost of more memory and potentially larger data loss on crash (unflushed memtable entries are lost).

## Production Use Cases

- **RocksDB (Meta, LinkedIn, Netflix)**: Facebook's fork of Google's LevelDB, powering MyRocks (MySQL storage engine), Kafka Streams state stores, and Flink checkpointing. LinkedIn uses it for Samza; Netflix for EVCache.
- **Apache Cassandra**: SSTables are the on-disk format for Cassandra's LSM-backed storage. Each table is a collection of SSTables, with leveled or size-tiered compaction strategies.
- **HBase**: The Hadoop database uses HDFS-based SSTables (HFiles) with an in-memory MemStore, exactly mirroring the LSM architecture.
- **ScyllaDB**: A Cassandra-compatible database written in C++ that uses a sharded LSM design, achieving millions of writes/second per node.
- **MongoDB WiredTiger**: Alongside its B-tree option, WiredTiger also supports LSM trees for write-heavy collections.

## When to Use It

| Use LSM-Tree when... | Don't use when... |
|---|---|
| Write throughput is your bottleneck (logs, metrics, IoT) | Read latency is critical and must be sub-millisecond |
| Data arrives in bursts and you need to absorb spikes | You need strong transactional semantics with rollbacks |
| You can tolerate slightly higher read latency | Data is mostly static with rare writes |
| Storage space is constrained (compaction reclaims space) | You need exact point-in-time snapshots across keys |
| Keys have temporal locality (recent data is queried more) | All keys are queried uniformly including cold data |

**Compared to B-Tree**: LSM-Trees are ~10x faster for writes, ~2-5x slower for reads (depending on SSTable count). If your workload is write-heavy (80%+ writes), LSM-Trees win. If read-heavy (80%+ reads), B-Trees win. Many modern databases support both (MySQL: InnoDB=B-tree, MyRocks=LSM).

**Bloom filters mitigate read penalty**: Production LSM systems add Bloom filters per SSTable to skip tables that definitely don't contain a key, reducing read amplification from O(k) to O(1) for most lookups. Our implementation omits this for clarity.

## Complexity Analysis

| Operation | Time | Notes |
|---|---|---|
| Put | O(1) amortized | Memtable insert + occasional O(n) flush |
| Delete | O(1) amortized | Tombstone write into memtable |
| Get | O(k) | k = number of SSTables; bloom filters make it O(1) |
| Flush | O(n) | n = memtable size; sequential I/O |
| Compact | O(N) | N = total entries across all SSTables |
| Space | O(unique keys × avg versions) | Tombstones inflate until compaction |

## Implementation Deep Dive

### 1. Sequence Numbers as MVCC (`main.go:50-51,65-66`)

Every write increments the global `sequence` counter and stamps the value. When Get scans SSTables newest-to-oldest, the first match wins — since newer SSTables have higher sequence numbers, this naturally returns the latest write. During compaction, when the same key appears across multiple SSTables, `merged[key]` keeps the entry with `entry.Sequence > current.Sequence` (`main.go:122`). This is lock-free multi-version concurrency: writes never block reads, and reads see a consistent snapshot of the latest committed value.

### 2. Tombstone Deletion (`main.go:63-71,129-133`)

Deletes don't erase data — they write a tombstone (an entry with `Deleted: true`). This is necessary because older SSTables may still contain the key. During reads, if the first match is a tombstone, `Get` returns `!entry.Deleted` = false — the key is considered deleted. During compaction, tombstones are physically removed (`main.go:128-133`), but only after ensuring no older SSTable still references the key. This is why compaction is compacting *all* SSTables together, not individually.

### 3. Automatic Flush on Threshold (`main.go:53-56,68-70`)

After every Put or Delete, we check `len(memtable) >= flushThreshold`. If so, `Flush()` is called automatically. This is a simplification — production systems use a separate background thread that flushes based on memtable size in bytes (typically 64–256 MB). The flush creates an immutable copy of the memtable, appends it to the SSTables slice, and resets the memtable. The copy is necessary because the memtable continues accepting writes during flush (though our synchronous version blocks briefly).

## Running the Demo

```bash
go run ./22-LSM-Tree/
```

The demo creates an LSM-Tree with `flushThreshold=3`, ingests 6 metrics (triggering automatic flushes), runs point queries, demonstrates tombstone deletes, and runs compaction to show space reclamation.

## Further Reading

- **"The Log-Structured Merge-Tree"** — Patrick O'Neil, Edward Cheng, Dieter Gawlick, Elizabeth O'Neil (1996). The original paper proposing LSM-Trees as a write-optimized alternative to B-Trees for disks where random I/O is expensive.
- **RocksDB Wiki (github.com/facebook/rocksdb)** — Covers leveled compaction, universal compaction, Bloom filter policies, and tuning for different SSD/HDD configurations.

---

*Part of the Design-With-TsGo system design curriculum*
