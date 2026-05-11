# Sharding

> **Horizontal data partitioning via hash-based key routing across independent shards — enables write scalability by distributing data across multiple machines, each responsible for a subset of keys.**

## The Problem It Solves

You're the data infrastructure engineer at a social media company. Your user profile database started as a single PostgreSQL instance. At 10,000 users, queries took 5ms. At 10 million users, the table is 500 GB — queries take 500ms and your `pg_dump` backup takes 14 hours. You can't vertically scale any further (AWS's largest instance is only so big), and read replicas help with reads but not with writes — every write still hits the single primary. You need to split your data across multiple independent databases, each responsible for a subset of users, so that writes and reads are distributed. This is **sharding**.

The core idea is deceptively simple: pick a key (like `user_id`), compute its hash, and use the hash to determine which shard owns that key. Every key lives on exactly one shard. When a request comes in for `user:42`, you hash `"user:42"`, determine it lives on shard-3, and route the query there. Shard-3 only stores ~1/N of the total data (where N is the number of shards), so its indexes are smaller, its queries are faster, and its write throughput is multiplied by N.

But sharding comes with tradeoffs. Cross-shard queries (JOINs, aggregations) become application-level problems — the database can't do them anymore because related data lives on different machines. Resharding (changing the number of shards) requires migrating data and is operationally painful. And if the hash distribution is skewed (some shards get more keys than others), you get hot shards that defeat the purpose. This implementation uses FNV-1a hashing with a fixed shard count — a basic but production-relevant approach similar to how Redis Cluster and early Vitess worked.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────┐
│                     ShardedStore (8 shards)                    │
│                                                               │
│  PUT("user:42", "Zara")                                       │
│       │                                                       │
│       ▼                                                       │
│  hash32("user:42") = 2166136261 ^ 'u'*16777619 ^ ...          │
│       │                                                       │
│       ▼                                                       │
│  index = hash % 8 = 3                                         │
│       │                                                       │
│       ▼                                                       │
│  ┌──────────────────────────────────────────────────┐        │
│  │ Shard 0 │ Shard 1 │ Shard 2 │ Shard 3 │ ... │ 7  │        │
│  │ (empty) │ user:1  │ user:5  │ user:42 │     │    │        │
│  │         │ user:4  │ user:256│         │     │    │        │
│  └──────────────────────────────────────────────────┘        │
│                                                               │
│  ROUTING FLOW:                                                │
│  ┌──────────┐    ┌───────────┐    ┌──────────┐              │
│  │ Key      │───►│ FNV-1a    │───►│ Shard    │              │
│  │ "user:42"│    │ Hash(key) │    │ Index    │              │
│  └──────────┘    └───────────┘    └────┬─────┘              │
│                                        │                     │
│                                        ▼                     │
│                                  ┌──────────┐              │
│                                  │ shards[3]│              │
│                                  │ [key]=val│              │
│                                  └──────────┘              │
│                                                               │
│  CROSS-SHARD OPERATIONS: NOT SUPPORTED                        │
│  ┌──────────────────────────────────────┐                    │
│  │ To find all users with name "Alice": │                    │
│  │ Query ALL 8 shards, merge results    │                    │
│  │ (scatter-gather — expensive!)        │                    │
│  └──────────────────────────────────────┘                    │
└──────────────────────────────────────────────────────────────┘
```

The `ShardedStore` (`main.go:14`) holds a slice of Go maps, each representing an independent shard. The `ShardIndex(key)` function (`main.go:57-59`) computes `hash32(key) % len(shards)` to determine which shard owns a key.

**On `Put(key, value)`** (`main.go:34-38`): The key is hashed to find the target shard index, and the value is inserted into that shard's map. This is O(len(key)) for hashing plus O(1) for the map insertion.

**On `Get(key)`** (`main.go:43-52`): The same hash computation finds the shard, and the map lookup on that shard returns the value. Since each key deterministically maps to one shard, there's no ambiguity — no need to check other shards.

**The hash function** (`hash32`, `main.go:64-71`) is a textbook FNV-1a (Fowler-Noll-Vo) implementation. It starts with a 32-bit offset basis (`2166136261`), XORs each byte of the key, and multiplies by the FNV prime (`16777619`). FNV-1a is not cryptographically secure — it doesn't need to be for sharding. It's fast, well-distributed, and deterministic. The key property for sharding is *uniform distribution*: given random keys, the hash should distribute them evenly across shards. FNV-1a achieves this with minimal computation.

## Production Use Cases

- **Vitess (YouTube)** — Vitess is YouTube's MySQL sharding layer, now a CNCF graduated project. It shards MySQL databases by a sharding key (like `user_id`) and provides a SQL proxy that routes queries to the correct shard. Vitess powers YouTube, Slack, Square, and GitHub's database infrastructure. It handles automatic resharding, connection pooling, and cross-shard query planning.
- **Citus (PostgreSQL)** — Citus is a PostgreSQL extension that transforms a single Postgres node into a distributed database. Data is sharded across worker nodes by a distribution column (like `tenant_id`). Queries that filter by the distribution column run on a single shard; analytical queries parallelize across all shards. Used by Cloudflare, Heap, and Mixmax.
- **Redis Cluster** — Redis Cluster shards data across 16,384 hash slots. Each key is hashed to a slot (`CRC16(key) % 16384`), and each Redis node owns a range of slots. The cluster supports automatic failover and online resharding (moving slots between nodes). Used by Twitter, Pinterest, and GitHub.
- **MongoDB sharding** — MongoDB shards collections by a shard key, distributing chunks of data (ranges of shard key values) across shards. The `mongos` router uses a config server to map key ranges to shards. Supports hash-based and range-based sharding strategies. Used by eBay, Adobe, and Verizon.
- **Amazon DynamoDB partitioning** — DynamoDB automatically shards data by the partition key. Each partition is a bounded-size storage unit (up to 10 GB), and DynamoDB transparently splits partitions as they grow. The partition key is hashed to determine which physical partition and which storage node owns the data.

## When to Use It

| Scenario | Use Sharding? |
|----------|-------------|
| Your dataset exceeds the capacity of a single machine (disk, memory, or IOPS) | **Yes** |
| Write throughput exceeds what a single machine can handle | **Yes** |
| Your access patterns are naturally partitioned by a key (tenant, user, region) | **Yes** |
| You can design your schema so most queries hit a single shard | **Yes** |
| You need cross-shard JOINs, transactions, or aggregations | **No** — or use a distributed SQL database |
| Your dataset fits comfortably on one machine | **No** — sharding adds operational complexity with no benefit |
| You frequently change your sharding scheme | **No** — resharding is expensive and risky |

**Alternatives**: **Vertical scaling** (buying a bigger machine) is always simpler and should be exhausted before sharding. **Read replicas** (38-Replication) solve read throughput but not write throughput or storage capacity. **Distributed SQL databases** like CockroachDB, YugabyteDB, or Google Spanner provide automatic sharding with full SQL semantics (JOINs, transactions) at the cost of higher latency per operation. **Consistent hashing** (08-Consistent-Hashing) adds and removes shards dynamically without rehashing all keys — use it when your shard topology changes frequently.

## Complexity / Behavior Characteristics

| Operation | Time Complexity | Space Complexity |
|-----------|----------------|------------------|
| Put(key, value) | O(L) for hashing + O(1) for insertion | O(1) per key-value pair (spread across shards) |
| Get(key) | O(L) for hashing + O(1) for lookup | O(1) |
| ShardIndex(key) | O(L) where L = key length | O(1) |
| Cross-shard scan | O(S × Dₛ) where S = shard count, Dₛ = data per shard | O(1) temporary |

The dominant cost is the FNV-1a hash, which is O(L) in the key length. For typical keys (50-100 bytes), this is sub-microsecond. The map operations are O(1) amortized. The key property is that single-key operations are independent of total data size — a Get on a 1 TB sharded dataset is just as fast as on a 1 MB one, assuming the target shard isn't overloaded.

## Implementation Deep Dive

**1. FNV-1a was chosen for speed and simplicity over cryptographic quality.** The `hash32` function at `main.go:64-71` implements FNV-1a with hardcoded constants: offset basis `2166136261` and prime `16777619`. These are the standard FNV constants. FNV-1a processes one byte at a time: XOR, then multiply. This is a non-cryptographic hash — it's fast (roughly 1-2 CPU cycles per byte on modern hardware) and produces well-distributed 32-bit values. Sharding doesn't need cryptographic properties because an attacker who can choose keys to create collisions can already degrade the shard, and the hash is only used for distribution, not security. Crypto libraries like SHA-256 would be orders of magnitude slower for no benefit.

**2. Fixed shard count simplifies routing but complicates resharding.** The shard count is set at construction time (`NewShardedStore(shardCount)`, `main.go:19-30`) and never changes. This means `ShardIndex(key)` simply computes `hash % count` — fast and deterministic. But it also means you can't add shards without rehashing every key (a shard count change changes every key's `hash % count` result). This is the "fixed shard" approach used by early versions of Vitess and many internal systems. Modern systems use consistent hashing (08-Consistent-Hashing) to minimize data movement when shards are added or removed.

**3. Each shard is an independent map with no cross-shard awareness.** The `shards` field at `main.go:14` is `[]map[string]string` — a slice of independent Go maps. There is no shared index, no distributed transaction coordinator, no cross-shard locking. This means operations on different shards are trivially parallelizable (no contention), but cross-shard operations (like "find all users with status=active") require a scatter-gather query to all shards. The demo's distribution analysis at `main.go:122-128` shows which keys landed on which shards — in a production system, you'd monitor this to detect hot shards and decide if resharding is needed.

## Running the Demo

```bash
go run ./39-Sharding/
```

## Further Reading

- "Sharding the Shards: Managing Datastore Locality at Scale with Akkio" — Facebook (2018). Describes Facebook's approach to shard management, including data locality optimization across geographic regions. *Proceedings of OSDI 2018.*
- "How Sharding Works" — MongoDB Documentation. A practical, production-oriented guide to choosing shard keys, understanding chunk migration, and avoiding the "hot shard" anti-pattern. *docs.mongodb.com/manual/sharding/*

---

*Part of the Design-With-TsGo system design curriculum*
