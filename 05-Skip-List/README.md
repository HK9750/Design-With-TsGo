# Skip List

> **A probabilistic alternative to balanced trees — layered sorted linked lists with "express lanes" that achieve O(log n) average-case search, insert, and delete without complex rebalancing logic.**

## The Problem It Solves

You're building an in-memory database index. Your data arrives as key-value pairs with integer keys, and you need to support point lookups (`SELECT value WHERE key = 42`), range scans (`SELECT * WHERE key BETWEEN 100 AND 200`), and ordered iteration. A hash table gives O(1) point lookups but can't do range scans. A sorted array enables binary search (O(log n)) and range scans but insertions are O(n) because you have to shift elements. A balanced binary search tree (AVL, Red-Black) gives O(log n) for everything, but the rebalancing logic after insert/delete requires multi-step rotations and color flips — complex, bug-prone, and a nightmare to tune under concurrency.

You want something simpler. Something where insert doesn't require global rebalancing, where the structure is naturally concurrent (different threads can modify different parts without contention), and where the code fits on a single screen. The **skip list** is the answer: a hierarchy of sorted linked lists where each higher level skips over more elements. Search starts at the top "express lane," skips as far as possible, then drops down a level and skips further — like binary search on a linked structure.

Conceived by William Pugh in 1989, the skip list replaces tree rebalancing with randomized level assignment. A newly inserted node "flips a coin" to determine how many levels it participates in. The probability distribution ensures that, on average, the top level has 1 node, the next level has 2, the next has 4, and so on — exactly the structure needed for O(log n) search. There is no worst-case guarantee (a run of bad coin flips could produce a degenerate structure), but the probability of pathological behavior is vanishingly small.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────────────┐
│  SkipList (maxLevel=4, probability=0.5)                              │
│                                                                      │
│  Level 3:  [HEAD] ──────────────────────────→ [100] → nil            │
│  Level 2:  [HEAD] ─────→ [7] ──────────────→ [100] → nil            │
│  Level 1:  [HEAD] ─────→ [7] ─────→ [20] ──→ [100] → nil            │
│  Level 0:  [HEAD] → [3] → [7] → [10] → [20] → [55] → [100] → nil  │
│                                                                      │
│  forward[N] arrays: each node has an array of pointers,              │
│  forward[0] = next node at level 0 (ground-level)                   │
│  forward[3] = next node at level 3 (top express lane)               │
│                                                                      │
│  Search(20):                                                         │
│    Level 3: HEAD→100 (too far, drop)                                │
│    Level 2: HEAD→7→100 (20 between 7 and 100)                      │
│    Level 1: 7→20 (found at level 1, continue to level 0)           │
│    Level 0: confirmed 20 exists                                     │
└──────────────────────────────────────────────────────────────────────┘
```

A skip list is built from nodes (`SkipNode[V any]` at `main.go:19-23`) that each contain a key, a value, and a `forward[]` slice of pointers — one per level. The list itself (`SkipList[V any]` at line 28-33) holds a sentinel `head` node (with key = `-1 << 62`, effectively negative infinity) and tracks the current `level` (0-indexed, where level 0 is the base linked list with all elements).

**Search** (`main.go:55-70`): Starts at `head` on the highest level. At each level, walks forward while the next key is less than the target. When the next key would overshoot, drops down one level. The `update[]` array (used in insert/delete) records the "rightmost node before the target" at each level — these are the splice points.

**Insert** (`main.go:74-102`): First, finds the splice points via the same traversal and records them in `update[]`. Then `randomLevel()` (line 136-142) generates the node's level via a geometric distribution: starting at 0, repeatedly flip a coin (rand.Float64() < 0.5); each head promotes the node one level higher. The node is spliced into all levels 0 through its assigned level by updating the `forward` pointers in `update[]`.

**Delete** (`main.go:106-132`): Finds splice points, verifies the target exists, then unlinks it from each level where `update[i].forward[i]` points to the target. Finally, trims empty top levels (lines 127-129).

The probabilistic level assignment is the heart of the algorithm. With a promotion probability *p* = 0.5, each node has a 50% chance of being at level ≥ 1, 25% at level ≥ 2, 12.5% at level ≥ 3, etc. This creates the exponential level distribution that guarantees O(log n) expected operations.

## Production Use Cases

- **Redis sorted sets (ZSET)** — Redis uses skip lists as the underlying data structure for sorted sets when stored in memory. Each element has a score (double) and a string member. The skip list enables O(log n) insert and O(log n + m) range queries (ZRANGE, ZRANGEBYSCORE).
- **LevelDB memtable** — Google's LevelDB uses a skip list for its in-memory table (memtable). The skiplist maintains sorted key-value pairs that are periodically flushed to SSTable files on disk.
- **Apache HBase** — HBase uses a concurrent skip list (Java's `ConcurrentSkipListMap`) for its in-memory store (MemStore). The lock-free properties of skip lists make them suitable for HBase's high-concurrency region server workloads.
- **Java ConcurrentSkipListMap** — Part of `java.util.concurrent`, this is a thread-safe sorted map built on a skip list. It's more scalable than `Collections.synchronizedSortedMap(new TreeMap())` because skip list modifications are local (no global rebalance).
- **MemSQL (now SingleStore)** — Used skip lists as an in-memory index structure for its lock-free ordered indexes, enabling concurrent reads and writes without blocking.

## When to Use It

| Scenario | Use Skip List? |
|----------|----------------|
| You need sorted iteration and range queries | **Yes** |
| Implementation simplicity matters more than worst-case guarantees | **Yes** |
| Concurrent access patterns (multiple readers/writers on different parts) | **Yes** — skip lists localize modifications |
| You need strict worst-case O(log n) guarantees | **No** — use a Red-Black or AVL tree |
| Memory overhead of multiple pointers per node is a concern (e.g., billions of small entries) | **No** — skip lists use more pointers than trees |
| Keys are very small (e.g., 32-bit integers) | **No** — B-Trees have better cache locality |

**Alternatives**: **Red-Black trees** provide deterministic O(log n) worst-case but require complex rebalancing. **B-Trees** are optimized for disk-backed storage with large fan-out nodes. **Treaps** (tree + heap) are another randomized balanced tree with similar probabilistic guarantees but a different structure (heap priority on a binary search tree).

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| Search | O(log n) | O(n) | O(n log n) expected |
| Insert | O(log n) | O(n) | O(n log n) expected |
| Delete | O(log n) | O(n) | O(n log n) expected |

Average case O(log n) with probability approaching 1 as n grows. Worst-case O(n) occurs when all nodes end up at level 0 (a flat linked list), but with p=0.5, the probability of this is 2^(-n) — astronomical for any real n. Space is O(n) on average because the expected number of forward pointers per node is 1/(1-p) = 2 (for p=0.5), so total space is ~2n pointers.

## Implementation Deep Dive

**1. Generic [V any] with integer keys.** The skip list at `main.go:28-33` is parameterized on `V any`, making it a reusable sorted-map implementation. The key type is fixed as `int` — a deliberate simplification since the focus is on the skip list algorithm, not key generics. The sentinel head has key `-1 << 62` (a very negative integer) ensuring all real keys are greater.

**2. Geometric distribution for randomLevel.** The `randomLevel()` function at line 136-142 implements `while level < maxLevel && rand.Float64() < probability { level++ }`. This generates levels according to a geometric distribution with success probability *p*. With p=0.5, ~50% of nodes stay at level 0, ~25% reach level 1, ~12.5% reach level 2, etc. The `maxLevel` parameter (default 16) caps the number of levels to prevent unbounded growth, which would theoretically support up to 2^16 ≈ 65,000 elements before the top level becomes crowded.

**3. update[] array for splice-point bookkeeping.** During insert (line 75) and delete (line 107), an `update[maxLevel+1]` array records, for each level, the rightmost node whose key is less than the target. This is populated during the same top-down traversal used for search. For insert, the node is then spliced into each level by setting `created.forward[i] = update[i].forward[i]` and `update[i].forward[i] = created`. This is the same pointer-splicing pattern as a linked list insert, replicated across levels.

**4. Configurable probability and maxLevel.** Unlike many skip list implementations that hardcode p=0.5, this implementation at `NewSkipList` (line 38) allows both parameters to be configured. Higher *p* (e.g., 0.75) produces more levels per node, trading memory for faster search (fewer steps per level). Lower *p* (e.g., 0.25) produces fewer levels, saving memory at the cost of longer level traversals.

## Running the Demo

```bash
go run ./05-Skip-List/
```

## Further Reading

- "Skip Lists: A Probabilistic Alternative to Balanced Trees" — William Pugh (1990). The original paper introducing the skip list data structure. *Communications of the ACM, 33(6).*
- "A Skip List Cookbook" — William Pugh. A practical guide covering implementation tricks, concurrency, and performance tuning. University of Maryland Technical Report CS-TR-2286.

---

*Part of the Design-With-TsGo system design curriculum*
