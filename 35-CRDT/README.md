# CRDT (Conflict-Free Replicated Data Types)

> **Conflict-free replicated data types — G-Counter (grow-only) and PN-Counter (positive-negative) for distributed counting with automatic merge — mathematical guarantees of eventual consistency without coordination.**

## The Problem It Solves

You're building the analytics pipeline at a global betting platform. During the World Cup final, 10 million users place bets simultaneously across 50 edge servers in 20 data centers. Each edge server tracks the number of bets placed per market. Every few seconds, edge servers exchange counts so the operations team can see live totals on a dashboard. But here's the problem: edge-1 reports 50,000 bets, edge-2 reports 45,000, and when you merge their data, you don't know if those counts overlap (bets counted twice) or are disjoint (some bets not counted). You need a data structure where concurrent increments from different nodes can be merged mathematically — with zero conflicts, zero coordination, and a correct total every time.

This is the CRDT problem. A Conflict-Free Replicated Data Type (CRDT) is a data structure designed so that concurrent updates from different replicas can always be merged into a consistent, correct state without any coordination protocol. The merge operation must be *commutative* (order doesn't matter), *associative* (grouping doesn't matter), and *idempotent* (merging the same data twice doesn't change the result). These algebraic properties guarantee that no matter what order gossip messages arrive, no matter how many times they're retransmitted, all replicas eventually converge to the same value.

The G-Counter (grow-only counter) is the simplest CRDT: each node gets its own slot in a per-node count map. To increment, node-A increments its own slot. To merge, take the element-wise maximum. To read the total, sum all slots. Since each node only ever increments its own slot and merges take the max, the total is always correct — no double-counting, no lost increments. The PN-Counter extends this to support decrements by splitting into two G-Counters: one for additions and one for subtractions. The net value is positive_sum - negative_sum.

## Architecture & Internals

```
┌─────────────────────────────────────────────────────────────┐
│                    GCounter (Grow-Only)                      │
│                                                             │
│  counts: { "edge-1": 100, "edge-2": 50, "edge-3": 200 }    │
│  Value() = 100 + 50 + 200 = 350                             │
│                                                             │
│  MERGE (element-wise max):                                  │
│  ┌───────────────┐     ┌───────────────┐                    │
│  │ A: {1:100,2:0}│     │ B: {1:0,2:75} │                    │
│  └───────┬───────┘     └───────┬───────┘                    │
│          │                     │                            │
│          └──────────┬──────────┘                            │
│                     ▼                                       │
│  A.Merge(B) → A: {1:100, 2:75}  (max of each node slot)    │
│  B.Merge(A) → B: {1:100, 2:75}                             │
│                                                             │
│  INCREMENT (node-local only):                               │
│  counts["edge-1"] += 25  →  { "edge-1": 125, ... }         │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                    PNCounter (Pos/Neg)                       │
│                                                             │
│  positive: GCounter { "edge-1": 100 }                       │
│  negative: GCounter { "edge-1": 10  }                       │
│  Value() = positive.Value() - negative.Value() = 90         │
│                                                             │
│  DECREMENT → actually increments the negative counter:      │
│  Decrement("edge-1", 5) → negative: { "edge-1": 15 }       │
│                                                             │
│  MERGE = Merge(positive) + Merge(negative)                  │
└─────────────────────────────────────────────────────────────┘
```

**GCounter** (`main.go:15-57`) is the foundational type. Its `counts` map tracks per-node integer values. Three operations:
- `Increment(node, amount)` adds to that node's slot. Crucially, the amount must be non-negative (`main.go:28-30` enforces this with a fatal error) — this maintains the grow-only property that makes merges safe.
- `Merge(other)` takes the element-wise maximum per node (`main.go:39-47`). Because each node only ever increments its own slot, the maximum across replicas is always the true count from that node.
- `Value()` sums all slots (`main.go:51-57`) to produce the total.

**PNCounter** (`main.go:63-101`) wraps two GCounters: `positive` for additions and `negative` for subtractions. An `Increment` adds to the positive counter; a `Decrement` adds to the negative counter (remember, GCounters can only grow, so "decrement" is implemented as an increment of the negative side). The net value is `positive.Value() - negative.Value()`. Merging a PNCounter means merging both internal GCounters, preserving the monotonic properties of each.

The mathematical properties that make CRDTs work:
- **Commutativity**: `A.Merge(B)` produces the same result as `B.Merge(A)` — the element-wise max is order-independent.
- **Associativity**: `A.Merge(B).Merge(C)` = `A.Merge(B.Merge(C))` — the element-wise max is grouping-independent.
- **Idempotence**: `A.Merge(A)` = A — merging a value with itself doesn't change it.

## Production Use Cases

- **Riak CRDTs** — Basho's Riak KV supports native CRDTs including counters, sets, maps, and flags. The PN-Counter in Riak is used for distributed vote counting in social applications and real-time analytics where strong consistency isn't required.
- **Redis CRDT (Active-Active)** — Redis Enterprise's Active-Active replication uses CRDTs for multi-master conflict resolution. The CRDB (Conflict-free Replicated Database) supports counters, sets, hashes, and lists that merge automatically across geographically distributed Redis instances.
- **SoundCloud's Roshi** — Roshi is SoundCloud's distributed, eventually-consistent timeline store built on CRDT sets. It powers the activity feeds that show you what tracks your friends are listening to, handling millions of concurrent writes.
- **Bet365** — The global sports betting platform uses CRDT-based counters for live bet tracking across dozens of edge locations. During peak events (World Cup, Super Bowl), the system handles hundreds of thousands of concurrent bet placements that eventually converge to correct totals.
- **League of Legends in-game chat** — Riot Games uses CRDTs for the in-game chat system, allowing players across different regional servers to communicate with eventually consistent message ordering and guaranteed delivery.

## When to Use It

| Scenario | Use CRDT? |
|----------|----------|
| You need to count things across geographically distributed edge nodes | **Yes** |
| Eventual consistency is acceptable (the count will be correct within seconds) | **Yes** |
| You want zero coordination overhead — no locks, no elections, no consensus | **Yes** |
| Your data structure is a counter, set, or map (CRDTs exist for many types) | **Yes** |
| You need strong consistency (immediate correctness after every write) | **No** — use Raft (40-Consensus) |
| You need to support arbitrary operations (CRDTs are limited to commutative ops) | **No** — use state machine replication |
| You need to track per-key state at massive scale (billions of keys) | **No** — the per-node slot overhead per key becomes prohibitive |

**Alternatives**: **State machine replication** via Raft (40-Consensus) provides strong consistency but requires a leader and quorum for every write — slower and unavailable during partitions. **Last-writer-wins registers** in gossip systems (33-Gossip-Protocol) are simpler but lose data on concurrent writes. **Delta state CRDTs** reduce network overhead by transmitting only changed entries rather than the full counter map — an optimization for production deployments. The key tradeoff is CRDTs give you availability + eventual consistency with zero coordination, at the cost of data model restrictions.

## Complexity / Behavior Characteristics

| Operation | Time Complexity | Space Complexity |
|-----------|----------------|------------------|
| GCounter.Increment | O(1) | O(N) per counter (N = distinct nodes) |
| GCounter.Merge | O(M) where M = nodes in other counter | O(N) per counter |
| GCounter.Value | O(N) where N = nodes in counter | O(1) |
| PNCounter.Increment | O(1) | O(N) per counter |
| PNCounter.Decrement | O(1) | O(N) per counter |
| PNCounter.Value | O(N + M) where N,M = nodes in pos/neg | O(1) |

The per-node slot overhead is the main scalability limit. In a system with 10,000 edge nodes all incrementing the same counter, each GCounter stores 10,000 integer entries — potentially 80 KB per counter instance. The PN-Counter doubles this. In practice, you'd use a bounded-size optimization like "dotted version vectors" or periodic garbage collection of slots that haven't changed recently.

## Implementation Deep Dive

**1. PN-Counter as two G-Counters — a compositional design.** The `PNCounter` at `main.go:63-66` embeds two `GCounter` values. This is elegant composition: rather than implementing a new merge algorithm for PNCounter, it delegates to GCounter's already-correct merge (`main.go:91-95`). The `Decrement` method at `main.go:84-87` is just `c.negative.Increment(node, amount)` — leveraging the existing grow-only semantics. This pattern — building complex CRDTs by composing simpler ones — is a recurring theme in the literature. A CRDT set, for example, can be built from two GCounters tracking additions and removals per element.

**2. G-Counter enforces monotonicity at the API level.** At `main.go:28-30`, the `Increment` method checks `if amount < 0` and calls `os.Exit(1)`. This is a hard assertion, not a silent return. The reasoning is that a negative increment violates the mathematical invariant that makes the merge correct — if a node could decrement its own slot, two concurrent merges might not converge to the same value. By failing fast, the implementation protects the CRDT guarantees. In a production system, you'd return an error rather than exiting, but the invariant enforcement is the same.

**3. Merge uses max, not sum, for per-node values.** The `Merge` function at `main.go:39-47` uses `if count > c.counts[node]` — element-wise maximum. This is the key insight. If edge-1 reports 100 increments and edge-2 reports 75 for edge-1 (perhaps from an older gossip message), taking the max (100) correctly preserves the true count. If merge used sum instead, replaying a gossip message would double-count. The max ensures idempotence: replaying the same merge message any number of times produces the same result.

## Running the Demo

```bash
go run ./35-CRDT/
```

## Further Reading

- "A Comprehensive Study of Convergent and Commutative Replicated Data Types" — Marc Shapiro, Nuno Preguiça, Carlos Baquero, and Marek Zawirski (2011). The definitive survey paper that formalized state-based and operation-based CRDTs. *INRIA Research Report RR-7506.*
- "Conflict-free Replicated Data Types" — Preguiça, Baquero, and Shapiro (2012). A shorter, more accessible introduction. *Proceedings of SSS 2012.*

---

*Part of the Design-With-TsGo system design curriculum*
