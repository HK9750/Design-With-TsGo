# B-Tree

> **A self-balancing tree with configurable fanout that keeps data sorted and minimizes disk I/O — the backbone of every database index you've ever used.**

## The Problem It Solves

Imagine you're building a key-value store that must persist billions of records on disk. Your first instinct might be a binary search tree — but here's the brutal reality: a balanced BST of 1 billion keys has a height of ~30 levels. Each level means a random disk seek (~10ms on HDDs). That's 300ms per lookup. Unacceptable.

The core issue is that traditional in-memory tree structures assume uniform access cost. On disk, reading a single integer costs the same as reading a 4KB page. What you actually want is a tree where each node fills an entire disk page, packing hundreds of keys per node so that a lookup touches far fewer nodes. This is exactly what a B-Tree does: it generalizes the binary search tree to an *n*-ary tree where each node holds many keys and has many children. A B-Tree of degree 500 with 1 billion keys has a height of just 3 or 4. That's the difference between 300ms and 30ms.

B-Trees also solve the write amplification problem. When you insert into a binary tree and it becomes unbalanced, you might need O(log n) rotations. A B-Tree uses a different strategy — *splitting* — where full nodes are divided at the median and the median moves up to the parent. This happens rarely (only when a node reaches 2t-1 keys), and each split is a local operation touching at most O(t) keys. Critically, the tree always grows upward from the root, never downward, which guarantees that all leaves remain at exactly the same depth. This uniform depth property is what makes range scans efficient: all leaf-level keys are at the same distance from the root, and a sequential scan can traverse them in order.

## Architecture & Internals

```
                    ┌──────────────────────┐
                    │       [17 | 30]       │  ← Root (internal node)
                    │      /    |     \     │     2 keys, 3 children
                    └──────────────────────┘
                    /            |           \
         ┌─────────────┐  ┌──────────┐  ┌─────────────┐
         │ [5 | 7 | 10] │  │ [20 | 25] │  │ [33 | 40 | 50] │  ← Internal nodes
         │  / |  |  \   │  │  /  |  \  │  │  /   |   |   \  │
         └─────────────┘  └──────────┘  └─────────────┘
           /  /   \   \      /   |   \      /    |    \    \
        ┌──┐┌──┐┌──┐┌──┐ ┌──┐┌──┐┌──┐ ┌──┐┌──┐┌──┐┌──┐
        │1,3││6││8,9││12│ │18││22││27│ │31││35││45││55│  ← Leaf nodes
        └──┘└──┘└──┘└──┘ └──┘└──┘└──┘ └──┘└──┘└──┘└──┘
                    ↑                                          (all at same depth)
              All leaves at identical depth
```

**Key properties:**
- **minDegree (t)**: Every node (except root) has between t-1 and 2t-1 keys, and between t and 2t children. With t=3, nodes hold 2-5 keys.
- **Root-upward splitting**: When a node overflows (hits 2t-1 keys), it splits. The median key is promoted to the parent. If the root splits, a new root is created — this is the *only* way the tree gains height.
- **All leaves at same depth**: The split property guarantees that every path from root to leaf has identical length. This is the B-Tree's superpower.

**How our Go implementation works:**

The `BTree` struct (`main.go:24`) holds a `root` pointer and a `minDegree` integer. The `BTreeNode` struct (`main.go:14`) stores a sorted `keys` slice, a `children` slice (nil for leaves), and a `leaf` boolean. Search (`main.go:56`) does a linear scan within each node to find the correct child pointer — since t is typically small (50–500), linear search within the node is faster than binary search due to cache locality. Insert (`main.go:74`) first checks if the root is full (has 2t-1 keys), and if so, creates a new root and splits the old root as its child. Then `insertNonFull` (`main.go:90`) recursively descends, splitting any full child encountered along the way before descending into it — this proactive splitting ensures we never need to split on the way back up.

## Production Use Cases

- **PostgreSQL B-tree indexes**: The default index type for every `CREATE INDEX` command. PostgreSQL's B-tree supports multi-column keys, unique constraints, and reverse scans.
- **MySQL InnoDB B+tree**: InnoDB's primary storage structure. Every table is a B+tree where leaf nodes hold actual row data (clustered index). Secondary indexes hold primary key references.
- **MongoDB WiredTiger**: The default storage engine since MongoDB 3.2. Uses a B-tree variant for document storage with copy-on-write pages and snapshot isolation.
- **SQLite**: The entire database is a single B-tree file. Each table and index is a separate B-tree in the same file, which is why SQLite databases are portable single files.
- **Filesystems (ext4, NTFS, Btrfs)**: Directory entries and file extent mappings are stored in B-tree variants. Btrfs uses B-trees for nearly all metadata — extents, directory entries, checksums, and free space tracking.

## When to Use It

| Use B-Tree when... | Don't use when... |
|---|---|
| Data must persist on disk/blocks and random reads are common | All data fits in memory with no persistence needed (use a hash map) |
| You need sorted-order iteration and range queries | You only need point lookups by exact key (hash table is simpler) |
| Read-heavy workloads with occasional writes | Write-heavy workloads with rare reads (use an LSM-Tree) |
| You need predictable worst-case O(log n) for all operations | You can tolerate amortized bounds and want simpler code |
| Keys are fixed-size or small variable-size | Keys are large unstructured blobs |

**Compared to LSM-Tree**: B-Trees pay the cost of random writes at insert time (updating pages in place), which makes reads fast. LSM-Trees batch writes sequentially and pay the read cost (scanning multiple SSTables). Choose B-Tree when reads dominate; choose LSM-Tree when writes dominate.

**Compared to Hash Index**: Hash indexes give O(1) point lookups but cannot do range scans, prefix lookups, or ordered iteration. If you ever need `WHERE key BETWEEN X AND Y`, you need a B-Tree.

## Complexity Analysis

| Operation | Time (worst-case) | Disk I/Os |
|---|---|---|
| Search | O(log n) | O(log_t n) |
| Insert | O(log n) | O(log_t n) |
| Delete | O(log n) | O(log_t n) |
| Range scan (k results) | O(log n + k) | O(log_t n + k/B) |
| Split node | O(t) | 2–3 pages written |
| Space | O(n) | n/(t-1) pages |

where t = minDegree and B = keys per page ≈ 2t-1.

## Implementation Deep Dive

### 1. Root-Upward Splitting (`main.go:121`)

The `splitChild` method is the heart of the B-Tree. When a child node reaches 2t-1 keys, we find the median key at index `t-1` (the middle), create a sibling node that takes everything to the right of the median, and promote the median into the parent. This is a *local* operation — only 2-3 nodes are modified. The parent key array is shifted right at the insertion point, and the child pointer array gets the new sibling. Because we always split children *on the way down* (proactively, before inserting), the parent is guaranteed to have room for the promoted key. This is the key insight that makes B-Tree insertion O(log n) without expensive rebalancing.

### 2. minDegree Parameterization (`main.go:33`)

The `minDegree` (t) is the fundamental tuning knob. Set it to 2 and you get a 2-3-4 tree (a B-tree of order 4). Set it to 500 and each node holds ~1000 keys — ideal for disk pages of 4KB with 4-byte integer keys. Our implementation validates `minDegree >= 2` because a degree-1 B-tree degenerates to a linked list. Every capacity check uses `2*t.minDegree-1` for the maximum keys per node, enforcing the invariant that nodes never exceed 2t-1 keys. This parameterization lets the same code model anything from an in-memory cache (t=2) to a petabyte-scale database index (t=500).

### 3. Leaf Unification (`main.go:14,17,92`)

All data is stored in leaf nodes — internal nodes only hold routing keys and child pointers. This is technically a B+tree convention. When `insertNonFull` reaches a leaf (`node.leaf == true`), it inserts the key in sorted position using `copy` to shift elements right. Non-leaf nodes only route by comparing keys and recursing. This separation means leaf nodes can be linked together for efficient range scans (though our implementation omits the linked-list for brevity). It also means keys in internal nodes are *separators*, not data — they can be deleted from leaves without cascading updates.

## Running the Demo

```bash
go run ./21-B-Tree/
```

The demo creates a B-Tree with `minDegree=3`, bulk-loads 20 integer keys (simulating primary key insertion into a database page manager), then runs point queries to demonstrate O(log n) lookups.

## Further Reading

- **"Organization and Maintenance of Large Ordered Indexes"** — Rudolf Bayer and Edward McCreight (1970). The original B-Tree paper from Boeing Scientific Research Labs. Introduced the concept of variable-fanout balanced trees for disk-based storage.
- **"Modern B-Tree Techniques"** — Goetz Graefe (2011). A comprehensive survey covering 40 years of B-Tree optimizations: prefix compression, latch-free concurrency, and write-optimized variants.

---

*Part of the Design-With-TsGo system design curriculum*
