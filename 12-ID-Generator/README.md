# ID Generator (Snowflake)

> **A decentralized, 64-bit unique ID scheme using a 41-bit millisecond timestamp, 10-bit machine identifier, and 12-bit monotonic sequence — generating ~4096 sortable IDs per second per machine without a coordinator.**

## The Problem It Solves

You're building a social platform with 50 million daily active users. Every post, comment, like, and share needs a unique ID. You start with auto-incrementing integers in your PostgreSQL primary key — this works until you split the database into 16 shards. Now two shards can both issue ID 42. You add a shard prefix, but now IDs lose sortability and your API clients need to know which shard holds which range. You switch to UUIDv4 — it's globally unique, but at 128 bits it bloats your indexes (UUIDs fragment B-tree pages due to randomness), and you lose the ability to sort by creation time without a separate `created_at` column.

Then you hit scale phase two: you need 100,000 IDs per second across 32 application servers. A centralized ID service (like a single Redis `INCR` counter) becomes a bottleneck and a single point of failure. You need a scheme where each machine can generate IDs independently, with a guarantee of global uniqueness, while preserving rough temporal ordering. No coordination protocol, no network calls, no lock contention.

This is exactly what Snowflake provides. Each machine generates IDs using a 64-bit integer that embeds three pieces of information: the millisecond since a custom epoch (41 bits, good for ~69 years), a machine/worker identifier (10 bits, supporting up to 1024 nodes), and a monotonically incrementing sequence number (12 bits, ~4096 IDs per millisecond). Because the timestamp occupies the high bits, Snowflake IDs are naturally k-sortable by creation time — newer IDs are numerically larger. Because the machine ID ensures no two nodes overlap, they require zero coordination. At Twitter's scale, this allowed hundreds of servers to generate IDs for tweets, DMs, and lists without a central bottleneck.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────┐
│                    64-bit Snowflake ID                        │
│                                                              │
│  ┌────────────────────┬───────────────────┬────────────────┐ │
│  │  41-bit timestamp  │  10-bit machine   │ 12-bit sequence│ │
│  │  (ms since epoch)  │   ID (0-1023)     │   (0-4095)     │ │
│  └────────────────────┴───────────────────┴────────────────┘ │
│   63                                         12          0   │
│                                                              │
│  Composition:                                                │
│    timestamp << 22 | machineID << 12 | sequence              │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                    SnowflakeGenerator                         │
│                                                              │
│  mu sync.Mutex ──────────── Guards concurrent invocations    │
│  machineID uint64 ───────── 10-bit node identifier           │
│  epochMs int64 ──────────── Custom epoch (e.g. 2024-01-01)  │
│  sequence uint64 ────────── Reset to 0 on new millisecond    │
│  lastTimestamp int64 ────── Previous call's millisecond      │
│                                                              │
│  NextID() flow:                                              │
│  1. Lock mutex                                               │
│  2. timestamp = now() - epoch                                │
│  3. If timestamp < lastTimestamp → CLOCK BACKWARD, abort     │
│  4. If timestamp == lastTimestamp:                           │
│       sequence = (sequence + 1) & 4095  (12-bit wrap)        │
│       If sequence == 0 → spin-wait next millisecond          │
│  5. Else: sequence = 0                                       │
│  6. id = timestamp<<22 | machineID<<12 | sequence            │
│  7. Unlock, return id                                        │
└──────────────────────────────────────────────────────────────┘
```

The `SnowflakeGenerator` struct (`main.go:16-22`) holds the minimal state needed: a mutex for thread safety, the assigned machine ID, a custom epoch (UNIX milliseconds), the monotonic `sequence` counter, and `lastTimestamp` to detect millisecond boundaries.

The core algorithm in `NextID` (`main.go:39-64`) runs under a mutex to make it goroutine-safe. It computes `timestamp = time.Now().UnixMilli() - g.epochMs` to get a 41-bit value. If the clock has moved backward (`timestamp < g.lastTimestamp`), it exits immediately — a clock regression means IDs could collide with previously issued values. If we're in the same millisecond as the last call, it increments `sequence` with 12-bit wrapping via `(g.sequence + 1) & 4095`; when sequence overflows to 0, it spin-waits by re-reading the timestamp until the next millisecond arrives. If we're in a new millisecond, `sequence` resets to 0.

The final assembly is a bitwise OR: `(uint64(timestamp) << 22) | (g.machineID << 12) | g.sequence`. This packs the three components into a 64-bit integer where the high 41 bits are the timestamp (ensuring sortability), the middle 10 are the machine ID, and the low 12 are the sequence.

## Production Use Cases

- **Twitter Snowflake IDs** — The original implementation, deployed in 2010 to generate tweet IDs. At Twitter's peak, hundreds of worker servers generated IDs for tweets, direct messages, lists, and user records. The worker ID was assigned via ZooKeeper.
- **Instagram Shard IDs** — Instagram's post ID system (described in their 2012 engineering blog) uses a variant: 41-bit timestamp + 13-bit shard ID + 10-bit auto-increment. The shard ID comes from the logical shard where the post was created, ensuring IDs are globally unique across thousands of PostgreSQL shards.
- **Discord Snowflakes** — Discord uses snowflakes for every entity: messages, users, guilds, channels. Their variant uses 42-bit timestamp + 5-bit worker ID + 5-bit process ID + 12-bit sequence. The extra 5 bits for process ID allow multiple generators per machine.
- **Sonyflake** — Sony's open-source Go variant extends the timestamp resolution to 10ms (instead of 1ms) and reduces the machine ID to 16 bits, giving up some ID density for a longer time range (~174 years from a 2014 epoch). Used internally at Sony for distributed service identifiers.
- **Baidu UID-Generator** — A Java implementation used across Baidu's microservices. Implements both the Snowflake scheme and a database-segment-based scheme (pre-allocating ID ranges from a central DB, then doling them out locally), allowing operators to pick the approach per workload.

## When to Use It

| Scenario | Use Snowflake? |
|----------|---------------|
| Multiple application servers need to generate unique IDs without a central coordinator | **Yes** |
| IDs need to be roughly time-sortable (newer = larger) | **Yes** |
| Throughput under ~4,096 IDs/ms per node is sufficient | **Yes** |
| Need globally unique, ordered IDs AND need strict sequential ordering (no gaps) | **No** — use a centralized sequencer or database auto-increment |
| Need human-readable, short IDs (URL shorteners, customer-facing codes) | **No** — use a hash-based scheme like hashids or nanoid |
| Operating in an environment with unreliable clocks that can jump backward | **No** — or implement a clock-backward tolerance mechanism (Sonyflake does this) |
| Only one application server (single node) | **No** — a simple atomic counter or UUID is simpler |

**Alternatives**: **UUIDv7** (RFC 9562) embeds a Unix timestamp in the first 48 bits, providing temporal sorting with globally unique randomness — preferred when you need a standard format interoperable with non-Go services. **KSUID** (Segment's library) is a 27-character base62-encoded string that embeds a 32-bit second-precision timestamp + 128-bit random payload, providing URL-safe, sortable IDs. **ULID** is a 26-character Crockford base32 string with 48-bit timestamp + 80-bit randomness. **UUIDv4** is the fallback when sortability and density don't matter.

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| NextID() | O(1) | O(1)* | O(1) |
| *Worst case: spin-waiting for the next millisecond when sequence overflows — bounded by 1ms wall-clock time | | | |

The amortized time is constant because the critical section contains fixed arithmetic and a single mutex lock. The worst case occurs when the current millisecond's 4096 sequence slots are exhausted, requiring `NextID` to spin-read the clock until the next millisecond boundary — an O(1) bounded wait of < 1ms. Space is constant: the generator holds exactly 5 fields regardless of how many IDs are produced.

Throughput ceiling per machine: 4096 IDs/ms × 1000 ms/s = 4,096,000 IDs/s. At Twitter's peak during the 2014 World Cup (~4,500 tweets/s), this was vastly more than needed for a single generator.

## Implementation Deep Dive

**1. Clock-backward detection is a hard exit, not a retry.** At `main.go:44-47`, clock regression triggers `os.Exit(1)` — a deliberate, brutal choice. The reasoning: if the system clock jumps backward, the generator cannot guarantee uniqueness because it might re-issue timestamps already assigned to earlier IDs. Production Snowflake systems typically handle this three ways: (a) abort and let the supervisor restart (as done here), (b) block until the clock catches up (Sonyflake's approach), or (c) use a backup sequence bit from an atomic counter. The abort approach is safest: a hard crash triggers a restart with a fresh machine ID from ZooKeeper/etcd.

**2. The sequence overflow spin-wait is a bounded busy-loop.** When all 4096 sequence numbers are consumed within a single millisecond (`main.go:50-55`), the generator enters a spin loop: `for g.sequence == 0 { timestamp = time.Now().UnixMilli() - g.epochMs; if timestamp > g.lastTimestamp { break } }`. This is a tight loop with no sleep — acceptable because (a) the condition is true for at most 1ms, (b) modern Go's scheduler preempts tight loops every ~10ms anyway, and (c) the mutex ensures only one goroutine spins. At >4M IDs/s per machine, this is rarely triggered in practice.

**3. The custom epoch is the primary space-saving trick.** The demo uses `epochMs := int64(1704067200000)` — January 1, 2024. Without a custom epoch, the 41-bit timestamp would start from January 1, 1970 (the Unix epoch), meaning the most significant bit of the 41-bit field wouldn't be used until 2039. By setting the epoch closer to the deployment date, you maximize the usable range: 2024 + 69 years = 2093, more than sufficient for any system's lifetime while keeping IDs compact.

## Running the Demo

```bash
go run ./12-ID-Generator/
```

## Further Reading

- "Snowflake" — Twitter Engineering Blog (2010). The original announcement post by Ryan King describing the motivation and design. A milestone in distributed systems practice.
- "Sharding & IDs at Instagram" — Instagram Engineering Blog (2012). Describes their Snowflake variant with PostgreSQL logical shards and the reasoning behind their 41+13+10 bit layout. Pairs well with the Snowflake paper for understanding how ID schemes evolve under real production constraints.

---

*Part of the Design-With-TsGo system design curriculum*
