# Consistent Hashing

> **A distributed hashing scheme that maps keys to nodes on a sorted ring — when nodes join or leave, only a fraction (~1/N) of keys need remapping rather than nearly all of them.**

## The Problem It Solves

You run a distributed cache cluster with 5 Redis nodes. Keys are distributed using `hash(key) % 5` — the modulus of the hash determines which node owns the key. One night, node 3's CPU spikes to 100% and you need to take it offline. You remove it from the rotation and change the modulo to `hash(key) % 4`. Suddenly, nearly **every key remaps to a different node**. User `user:42:session` that was on node 3 now maps to node 2; `user:17:profile` that was on node 2 now maps to node 1. Your cache hit rate drops from 95% to near 0% in an instant. Every request now misses the cache and hammers the database. This is the **rehashing problem**: `hash(key) % N` is brittle to changes in N.

The problem gets worse as you scale. If you add a 6th node during a traffic spike, the modulo changes from 5 to 6, and 5/6 = 83% of keys remap. In a cluster with 1,000 nodes, removing one node would invalidate 99.9% of the cache. This makes standard modulo-based sharding fundamentally incompatible with dynamic scaling.

**Consistent hashing** solves this by mapping both keys and nodes onto the same ring — a circular space of hash values (0 to 2³²-1). A key is assigned to the nearest node **clockwise** on the ring. When a node is added, it takes over keys from only its clockwise neighbor — a fraction ~1/N of the total key space. When a node is removed, its keys shift to the next node clockwise — again only ~1/N. The rest of the keys stay exactly where they were. This turns horizontal scaling from a "rebuild the entire cache" event into a "migrate 1% of data" operation.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────────────┐
│  Consistent Hash Ring (uint32 space: 0 to 2^32-1)                    │
│                                                                      │
│                          hash=0                                       │
│                            │                                         │
│                    ┌───────┴───────┐                                 │
│                   ╱                 ╲                                │
│              A'  ╱                   ╲  B'                           │
│            A'   ╱                     ╲   B'                         │
│          A'    │      Virtual Nodes     │    B'                       │
│         A'     │    (3 per physical)     │     B'                     │
│       ─────────A─────────────────────────B─────────                  │
│         C'     │                         │     B'                    │
│          C'    │                         │    B'                     │
│            C'   ╲                     ╱   B'                         │
│              C'  ╲                   ╱  C'                           │
│                   ╲                 ╱                                │
│                    └───────┬───────┘                                 │
│                            │                                         │
│                     hash=2^31                                        │
│                                                                      │
│  Physical nodes: A, B, C                                             │
│  Virtual replicas: A_0..A_{R-1}, B_0..B_{R-1}, C_0..C_{R-1}        │
│                                                                      │
│  Key lookup:                                                         │
│    hash(key) → binary search on sorted ring → clockwise node        │
│    "user:42:data" hashes to position X                               │
│    X falls between A_2 and B_0 → assigned to node B                 │
│                                                                      │
│  Add node D:                                                         │
│    Insert virtual nodes D_0..D_{R-1} into ring                      │
│    D takes over keys between each D_i and its clockwise neighbor    │
│    Only keys previously owned by those clockwise neighbors move     │
│    ~1/(N+1) of total keys remap                                     │
└──────────────────────────────────────────────────────────────────────┘
```

The ring is represented as a sorted slice of `RingPoint` structs (`main.go:20-23`), each containing a `hash uint32` and `node string`. Adding a node (`AddNode`, line 46-56) inserts R virtual replicas (default 100), each computed by hashing `fmt.Sprintf("%s#%d", node, i)` with FNV-1a, then re-sorts the ring. Removing a node (`RemoveNode`, line 60-74) filters out all points belonging to that physical node.

**Key lookup** (`GetNode`, line 79-89): Hash the key with FNV-1a, then use `sort.Search` (Go's binary search) to find the first ring point whose hash is >= the key's hash. That point's node is the owner. If the key's hash is larger than any ring point (wrapping around the uint32 space), `idx % len(ring)` wraps to index 0.

**Virtual replicas** are the critical design element that prevents hot-spots. Without them, physical nodes appear at a single point on the ring, leading to uneven key distribution (some nodes get large arcs, others get tiny arcs). With R=100 replicas per node, the law of large numbers smooths the distribution: each node owns approximately 1/N of the keyspace, with variance decreasing as R increases. The tradeoff is ring size (R × N points) vs. uniformity.

## Production Use Cases

- **Amazon Dynamo** — The seminal Dynamo paper (2007) introduced consistent hashing as the partitioner for Amazon's highly-available key-value store. Dynamo uses virtual nodes to distribute data across the ring, with each physical node responsible for multiple virtual node positions.
- **Discord** — Discord famously scaled from 0 to millions of concurrent users using a consistent hashing ring to distribute guild (server) data across Cassandra and later ScyllaDB nodes. When they added capacity, only a small fraction of guilds needed to be rebalanced.
- **Apache Cassandra** — Cassandra's `Murmur3Partitioner` and `RandomPartitioner` use consistent hashing by default. Each node is assigned a token range on the ring, and virtual nodes (vnodes, default 256) are used for fine-grained load distribution without manual token assignment.
- **Akamai CDN** — Akamai's edge server assignment uses consistent hashing to map content URLs to cache servers. When a cache server fails or new capacity is deployed, only the URL space adjacent to the changed server needs redistribution.
- **NGINX upstream with `hash` directive** — NGINX's `upstream` block with `hash $request_uri consistent` uses consistent hashing to map requests to backend servers. It's used for session stickiness without a centralized session store.

## When to Use It

| Scenario | Use Consistent Hashing? |
|----------|-------------------------|
| Nodes join/leave the cluster dynamically (autoscaling, failures) | **Yes** |
| A stable mapping between keys and nodes must survive topology changes | **Yes** |
| Cache hit rate during scaling events is critical | **Yes** |
| The number of nodes is small and stable (e.g., 3-node primary/replica) | **No** — simple modulo is fine |
| Data must be perfectly evenly distributed (strict load balancing) | **No** — consistent hashing with virtual nodes is *probabilistically* even, not perfectly exact |
| You need to control *which* data moves (e.g., move a specific key range) | **No** — consistent hashing gives you no control over what moves |

**Alternatives**: **Rendezvous hashing** (Highest Random Weight hashing) assigns each key to the node with the highest `hash(key, node)` value. It gives the same minimal-remapping property but doesn't require a sorted ring — used in Apache Ignite and Microsoft's object store. **Jump consistent hashing** uses a constant-space algorithm for bucket selection without storing the ring, but requires sequential bucket numbering and works best when nodes are homogeneous.

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| AddNode | O(R log(R×N)) | O(R log(R×N)) | O(R×N) |
| RemoveNode | O(R×N) | O(R×N) | O(R×N) |
| GetNode | O(log(R×N)) | O(log(R×N)) | O(1) |

Where R = virtual replicas per node and N = number of physical nodes. GetNode uses binary search over the sorted ring, hence O(log(R×N)). AddNode re-sorts the ring (O(R×N log(R×N))). RemoveNode scans the entire ring to filter out R points (O(R×N)). Space is the ring slice: R × N RingPoint structs.

## Implementation Deep Dive

**1. Virtual replicas (default 100) for uniform distribution.** In `AddNode` at line 46-56, each physical node is represented by R virtual points. The number is configurable via `NewConsistentHashRing(replicas)` at line 35. 100 replicas is the default based on Dynamo's findings: with 100 replicas and 10 nodes, the standard deviation of load is ~10%. Increasing to 1,000 replicas reduces it to ~3%, but increases ring size and lookup time proportionally.

**2. Binary search for O(log R) key lookup.** `GetNode` at line 79-89 uses `sort.Search` with a predicate `r.ring[i].hash >= hash`. This is the standard "find first element ≥ target" binary search pattern. The modulus wrap `idx % len(ring)` handles keys that hash beyond the largest ring point, wrapping around to the start of the sorted slice.

**3. Remove-before-add for idempotent AddNode.** At line 47, `AddNode` calls `r.RemoveNode(node)` before adding replicas. This makes `AddNode` idempotent — calling it twice for the same node removes old replicas and inserts fresh ones, preventing duplicate ring points. A production system might instead check for existence and skip/update, but the remove-then-add pattern is simple and correct.

**4. FNV-1a for both key hashing and virtual node placement.** The same `hash32` function at line 92-99 is used for hashing both user keys and virtual node identifiers (`fmt.Sprintf("%s#%d", node, i)`). This consistency means the ring is well-distributed regardless of the node naming convention — node names don't need to be chosen to distribute evenly on the ring because the replica suffix does that work.

## Running the Demo

```bash
go run ./08-Consistent-Hashing/
```

## Further Reading

- "Consistent Hashing and Random Trees: Distributed Caching Protocols for Relieving Hot Spots on the World Wide Web" — Karger, Lehman, Leighton, Levine, Lewin, Panigrahy (1997). The foundational paper introducing consistent hashing. *Proceedings of the 29th ACM Symposium on Theory of Computing (STOC).*
- "Dynamo: Amazon's Highly Available Key-value Store" — DeCandia et al. (2007). Describes Dynamo's use of consistent hashing with virtual nodes for partitioning. *Proceedings of the 21st ACM Symposium on Operating Systems Principles (SOSP).*

---

*Part of the Design-With-TsGo system design curriculum*
