# LRU Cache

> **An O(1) key-value store that automatically evicts the least-recently-used entry when capacity is reached — ideal for response caches, buffer pools, and any bounded memory scenario where recency predicts future access.**

## The Problem It Solves

Imagine you're building a web API with 10,000 concurrent users. Each request requires a database query to fetch the user's profile — that's 10,000 queries per second hammering your PostgreSQL instance. You deploy a Redis cache, but Redis is backed by memory, and memory is finite. You allocated 4 GB for your cache. What happens when the cache fills up? You need an eviction policy — a rule for deciding which entry to discard when a new one arrives.

The naive approach is a FIFO queue: evict whatever was inserted first. But your API has power users who log in every 5 minutes and infrequent users who log in once a month. FIFO would evict the power user's cached profile (inserted at server startup) and keep the once-a-month user. The cache hit rate plummets because the entries people actually need have been flushed out. You need a policy that tracks *recency of access*, not recency of insertion.

This is the problem LRU solves. It's a bounded-size key-value store that tracks the order of access (both reads and writes) and always evicts the entry that hasn't been touched for the longest time. The insight from decades of systems research is that recency is a strong heuristic for popularity — if someone accessed a record recently, they're likely to access it again soon. LRU is the most widely deployed cache eviction policy in production systems because it captures this temporal locality with a simple, predictable implementation.

## Architecture & Internals

```
┌─────────────────────────────────────────────────────────────┐
│                         LRUCache                            │
│                                                             │
│  ┌─────────────┐                    ┌────────────────────┐  │
│  │   HashMap    │                    │  Doubly-Linked List│  │
│  │  key→*Node   │                    │                    │  │
│  │             │                    │  [Head] ← sentinel │  │
│  │  key:42 ────┼───┐                │    ↕               │  │
│  │  key:17 ────┼───┼──┐             │  [Node]  key=42   │  │
│  │  key:99 ────┼───┼──┼──┐          │    ↕               │  │
│  │             │   │  │  │          │  [Node]  key=17   │  │
│  └─────────────┘   │  │  │          │    ↕               │  │
│                    │  │  │          │  [Node]  key=99   │  │
│                    ▼  ▼  ▼          │    ↕               │  │
│                  (pointers into list)│  [Tail] ← sentinel │  │
│                                     └────────────────────┘  │
│                       MRU → Tail    LRU → Head.Next          │
└─────────────────────────────────────────────────────────────┘
```

The LRU cache is a composite data structure: a **hashmap** (`Map map[int]*ListNode`) stores key-to-node mappings for O(1) lookup, and a **doubly-linked list** tracks access order. The list is anchored by two **sentinel nodes** — `Head` and `Tail` — which are dummy nodes that never hold data. Sentinels eliminate nil checks: every real node always has a `Prev` and `Next`, so `addNode` and `deleteNode` are trivial 4-pointer reassignments with no conditional branches.

**On `get(key)`**: The hashmap provides O(1) lookup. If found, the node is detached from its current position via `deleteNode` (two pointer swaps) and re-attached to the tail via `addNode` (four pointer swaps). This marks it as the most-recently-used entry. The list is ordered from LRU (head side) to MRU (tail side), so every access promotes the entry to the MRU end.

**On `put(key, value)`**: If the key already exists, update its value and move it to MRU. If it's new and the cache is at capacity (`Size == Capacity`), evict `Head.Next` — the least-recently-used entry — by removing it from both the list and the map. Then insert the new node at the tail. All pointer manipulations remain O(1).

The Go implementation at `main.go:26-51` creates sentinel nodes in `NewLRUCache` and wires them together: `head.Next = tail` and `tail.Prev = head`. The `addNode` function at `main.go:98-103` splices a node just before the tail sentinel by manipulating four pointers, and `deleteNode` at `main.go:107-110` splices it out with two pointer updates. There are no loops, no scans — every operation is bounded by constant work.

## Production Use Cases

- **Redis** — The `allkeys-lru` eviction policy powers one of the most widely deployed caching strategies in the world. When Redis maxmemory is reached, it samples keys and evicts the least recently used among the sample.
- **memcached** — The original LRU cache server. memcached uses per-slab-class LRU queues to evict entries, making it the backbone of Facebook's, YouTube's, and Twitter's caching layers.
- **PostgreSQL shared_buffers** — PostgreSQL's buffer manager uses a clock-sweep variant of LRU (sometimes called Clock or Second-Chance) to manage which data pages stay in `shared_buffers` and which get evicted back to disk.
- **Varnish HTTP Cache** — Varnish uses an LRU eviction with an object lifecycle that keeps frequently requested web resources in memory while expiring stale ones.
- **CPU Caches (L1/L2/L3)** — While hardware caches use pseudo-LRU approximations due to transistor budgets, the fundamental idea of evicting the least-recently-used cache line is identical. Intel's Ivy Bridge and later use an adaptive replacement policy that approximates LRU.

## When to Use It

| Scenario | Use LRU? |
|----------|----------|
| Access patterns show strong temporal locality (recently accessed = likely accessed again) | **Yes** |
| Bounded memory with a fixed capacity ceiling | **Yes** |
| Need strict O(1) worst-case Get/Put | **Yes** |
| Items have dramatically different access frequencies (a few hot keys, many cold) | **No** — use LFU instead |
| Need to evict based on TTL/expiration, not recency | **No** — use a TTL map |
| Need scan-resistant eviction (one large scan shouldn't flush the working set) | **No** — consider ARC or 2Q |

**Alternatives**: LFU (02-LFU-Cache) tracks access frequency rather than recency — better for CDNs where popular videos should never be evicted even if untouched for hours. **TTL caches** evict by wall-clock age regardless of access. **ARC** (Adaptive Replacement Cache) balances recency and frequency dynamically, used in ZFS and IBM storage systems, but has roughly 2x the space overhead of LRU.

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| Get | O(1) | O(1) | O(n) |
| Put | O(1) | O(1) | O(n) |
| Evict | O(1) | O(1) | O(n) |

All operations are constant time in both average and worst case because the hashmap provides direct node access and the doubly-linked list operations (add, delete) are fixed-pointer manipulations. Space is O(n) where n is the capacity — one map entry and one list node per cached item.

## Implementation Deep Dive

**1. Sentinel head/tail nodes eliminate nil checks.** In `NewLRUCache` (`main.go:37-51`), two dummy `ListNode` values are allocated and wired: `head.Next = tail; tail.Prev = head`. Every insert splices between `Tail.Prev` and `Tail` (`addNode` at line 98-103). Every deletion splices out by re-linking neighbors (`deleteNode` at line 107-110). Without sentinels, every operation would need `if node.Prev != nil`, `if node.Next != nil` checks — the sentinel trick makes the code branchless and trivially correct.

**2. Map stores pointers into the list, not copies.** The `Map map[int]*ListNode` field at line 29 maps keys directly to list node pointers. This is the critical insight that enables O(1) operations: when we need to move a node on access, the map gives us the exact list position with no search. The same pointer is shared between both structures — when a node is evicted, it's deleted from the map (`delete(l.Map, lru.Key)`) and removed from the list in constant time.

**3. Eviction happens inline during Put, not on a timer.** At line 81-87, when `l.Size == l.Capacity`, the LRU entry is evicted synchronously *before* inserting the new one. This "evict-on-insert" strategy keeps the cache size bounded without a background janitor goroutine. It means Put is always O(1) even when the cache is full — the eviction target is always `Head.Next`, the oldest node in the list.

## Running the Demo

```bash
go run ./01-LRU-Cache/
```

## Further Reading

- "The LRU-K Page Replacement Algorithm for Database Disk Buffering" — O'Neil, O'Neil, and Weikum (1993). Extends LRU to track the last K references, making it scan-resistant. *Proceedings of the 1993 ACM SIGMOD International Conference on Management of Data.*
- Redis documentation: "Using Redis as an LRU cache" — https://redis.io/docs/latest/develop/reference/eviction/

---

*Part of the Design-With-TsGo system design curriculum*
