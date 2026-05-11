# Binary Heap (Min-Heap & Max-Heap)

> **A generic, array-backed complete binary tree parameterized by a comparator function — functions as a min-heap OR max-heap depending on the ordering provided. O(log n) insert/extract, O(n) heapify, O(1) peek.**

## The Problem It Solves

You're building a job scheduler for a worker system. Jobs arrive at different times with different priorities: a database backup (priority 8), a user-facing API call (priority 10), a nightly report (priority 2). You need to always dispatch the highest-priority job next. A sorted array gives O(1) extraction (pop from the end) but O(n) insertion (shift elements to maintain order). A sorted linked list is the same. A balanced tree gives O(log n) for both but carries implementation complexity and per-node memory overhead.

The binary heap solves this exact problem with brutalist simplicity. It's a complete binary tree stored in a flat array — no pointers, no nodes, no allocations beyond the slice itself. The highest-priority element lives at `data[0]`. Insertion appends to the end of the array and "sifts up" (bubbles the new element toward index 0 until the heap invariant is satisfied). Extraction swaps `data[0]` with the last element, truncates the array, and "sifts down" (bubbles the moved element until the invariant is restored). Both operations touch exactly O(log n) elements — the path from root to leaf in a tree of height log₂(n).

The beautiful twist in this implementation: the "higher priority" relationship is a **comparator function** (`func(a, b T) bool`), not hardcoded. Provide `func(a, b int) bool { return a < b }` and you get a min-heap (smallest = highest priority). Provide `func(a, b int) bool { return a > b }` and you get a max-heap (largest = highest priority). The exact same data structure, the exact same siftUp/siftDown logic, parameterized by a two-argument function. This is Go generics at their most elegant.

## Architecture & Internals

```
┌─────────────────────────────────────────────────────────────────┐
│  BinaryHeap[min-heap, comparator: a < b]                        │
│                                                                 │
│  Array representation of a complete binary tree:                │
│                                                                 │
│                         [1]          ← index 0 (root, min)      │
│                        /   \                                    │
│                     [3]     [2]      ← indices 1,2              │
│                    /   \    /  \                                │
│                  [5]  [8] [7] [4]   ← indices 3-6               │
│                                                                 │
│  Index relationships (zero-based):                              │
│    parent(i) = (i - 1) / 2                                     │
│    left(i)   = 2*i + 1                                         │
│    right(i)  = 2*i + 2                                         │
│                                                                 │
│  Insert(0):                                                     │
│    Append to end → [1,3,2,5,8,7,4,0]                           │
│    Sift-up from index 7:                                       │
│      parent=(7-1)/2=3 → compare(0,5) → 0 has priority → swap  │
│      parent=(3-1)/2=1 → compare(0,3) → 0 has priority → swap  │
│      parent=(1-1)/2=0 → compare(0,1) → 0 has priority → swap  │
│    Result: [0,1,2,3,8,7,4,5]                                    │
│                                                                 │
│  Extract():                                                     │
│    Take data[0] (minimum)                                       │
│    Move data[7] (last) to data[0]: [5,1,2,3,8,7,4]            │
│    Sift-down from index 0:                                     │
│      left=1→compare(1,5)→1 < 5→swap → [1,5,2,3,8,7,4]        │
│      left=3→compare(3,5)→3 < 5→swap → [1,3,2,5,8,7,4]        │
│      children ≥ parent → stop                                   │
└─────────────────────────────────────────────────────────────────┘
```

The binary heap stores elements in a slice `data []T` at `main.go:20`. The `higherPriority func(a, b T) bool` comparator at line 20 is the only thing that distinguishes min-heap from max-heap behavior.

**Sift-up** (`main.go:78-87`): Starting from a given index, repeatedly compare the element with its parent at `(i-1)/2`. If the element has higher priority than its parent, swap and continue upward. Stops when the element is at the root or doesn't outrank its parent.

**Sift-down** (`main.go:90-106`): Starting from a given index, compare the element with both children (indices `2*i+1` and `2*i+2`). Find the "best" among parent and children using the comparator. If the best is a child, swap with it and recurse downward into that child. Stops when the parent is the best among the three candidates.

**Heapify** (in `NewBinaryHeap`, line 25-31): If initial values are provided, the entire array is heapified using **Floyd's algorithm**: iterate from the last non-leaf node (`len(data)/2 - 1`) down to the root, calling `siftDown` at each position. This runs in O(n) time, not O(n log n) — a non-obvious result that makes batch initialization efficient.

## Production Use Cases

- **Linux CFS Scheduler** — The Completely Fair Scheduler uses a red-black tree (not a heap directly), but the virtual runtime (vruntime) ordering is effectively a min-heap: the task with the smallest vruntime gets the CPU next. Priority scheduling in older Linux kernels (O(1) scheduler) used explicit priority heaps.
- **Dijkstra's shortest-path algorithm** — Every graph library implements Dijkstra with a min-heap priority queue. Insert nodes with tentative distances; extract-min picks the next unvisited node closest to the source. Without a heap, Dijkstra degrades to O(V²) instead of O((V+E) log V).
- **Apache Kafka message prioritization** — While Kafka core is a FIFO log, prioritization layers built on top use heaps to reorder messages within windows, ensuring high-priority events (payment confirmations) are dispatched before lower-priority ones (analytics events).
- **Celery task queues** — Celery, the Python distributed task queue, uses Redis' sorted set (backed by a skip list, heap-like) for task prioritization. Workers poll for the highest-priority task first.
- **Event-driven simulators** — Discrete-event simulation engines (NS-3, OMNeT++) use a min-heap keyed by event timestamp. The simulation main loop extracts the next-earliest event, processes it, and inserts any newly generated future events.

## When to Use It

| Scenario | Use Binary Heap? |
|----------|------------------|
| You need the single highest/lowest-priority element efficiently | **Yes** |
| Elements are inserted dynamically and extracted one at a time | **Yes** |
| You need sorted iteration of all elements | **No** — heap order is partial; use a sorted array/tree |
| You need access to arbitrary elements (not just the root) | **No** — heaps only guarantee root accessibility |
| You need stable ordering (FIFO among equal priorities) | **No** — heap order is unstable; add a sequence counter |

**Alternatives**: **Fibonacci heaps** give O(1) insert and decrease-key (amortized), making them better for graph algorithms like Dijkstra with many decrease-key operations — but they're complex and have high constant factors. **Pairing heaps** are simpler than Fibonacci with similar practical performance. **B-heaps** (bounded heaps) trade some heap order for better cache locality, popular in game engine pathfinding.

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| Peek | O(1) | O(1) | O(1) |
| Insert | O(log n) | O(log n) | O(n) |
| Extract | O(log n) | O(log n) | O(n) |
| Heapify | O(n) | O(n) | O(n) |

Insert and extract touch exactly one path from root to leaf (or leaf to root), which is O(log n) swaps in a complete binary tree. Heapify is O(n) despite calling siftDown O(n) times because the majority of nodes are near the leaves where siftDown is O(1). The space is exactly the underlying slice — no per-element allocation overhead.

## Implementation Deep Dive

**1. Comparator parameterization via generics + higher-order function.** The `BinaryHeap[T any]` at `main.go:18-21` stores a `higherPriority func(a, b T) bool` function. The `NewBinaryHeap` constructor at line 25 takes this function as its first argument. In the demo (`main.go:116`), a min-heap is created with `func(a, b int) bool { return a < b }`; a max-heap at line 130 uses `return a > b`. The siftUp and siftDown functions never know or care whether they're operating on a min-heap or max-heap — they just call `h.higherPriority`. This is the Strategy pattern encoded as a type parameter.

**2. Floyd's O(n) heapify in the constructor.** Lines 28-30: `for i := len(heap.data)/2 - 1; i >= 0; i-- { heap.siftDown(i) }`. Starting from the last internal node (index `n/2 - 1`) and working backward to the root, each node is sifted down. Leaves (indices `n/2` through `n-1`) are already valid single-element heaps. The total work is `sum(k * n/2^(k+1))` which converges to O(n) — about 2n comparisons in practice. Building a heap by repeated insertion would be O(n log n).

**3. siftDown's three-way comparison for "best" selection.** At lines 92-105, `siftDown` tracks a `best` index initialized to the parent. It compares the left child against `best`, then the right child against `best`, using the comparator each time. This finds the single highest-priority element among the parent and its two children. If the parent is already the best, the invariant holds and the loop breaks. If a child is better, it swaps and continues downward. This "tournament" logic handles both min-heap and max-heap without any branching on heap type.

## Running the Demo

```bash
go run ./06-Min-Heap-Max-Heap/
```

## Further Reading

- "Algorithm 232: Heapsort" — J.W.J. Williams (1964). The original paper introducing the binary heap data structure and the heapsort algorithm. *Communications of the ACM, 7(6).*
- "The Art of Computer Programming, Vol. 3: Sorting and Searching" — Donald Knuth. Section 5.2.3 provides the definitive analysis of heap operations, including Floyd's O(n) heapify proof.

---

*Part of the Design-With-TsGo system design curriculum*
