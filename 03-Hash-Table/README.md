# Hash Table

> **The fundamental O(1) average-time key-value store — two implementations (bucket chaining + open addressing with linear probing) covering the two major collision resolution strategies, both with dynamic resizing based on load factor thresholds.**

## The Problem It Solves

You're building a session store for a web application. Every HTTP request carries a session cookie with a 64-character random token, and you need to look up the corresponding user data — email, permissions, last activity. You could store sessions in a sorted array and binary search (O(log n)), but with 1 million active sessions, that's 20 comparisons per request. At 50,000 requests per second, those comparisons add up to measurable latency and CPU burn.

You could use a balanced binary tree — same O(log n). You could use a slice with the token as an index — O(1)! — but 64-character random tokens produce indices far larger than any practical array. The fundamental tension is: you want array-like O(1) access, but your keys map to an enormous (effectively infinite) integer space, and you only have a few million entries.

The hash table solves this by mapping the infinite key space onto a finite array through a **hash function**. The key "session_a1b2c3..." is converted to an integer (the hash), then reduced modulo the array size to get a bucket index. Different keys may map to the same index — a **collision**. The two families of hash table implementations differ in how they handle these collisions: **separate chaining** stores a linked list at each bucket; **open addressing** probes forward through the array to find the next empty slot. Both strategies appear in production systems: Java's HashMap uses chaining, Python's dict uses open addressing.

This module implements **both** strategies: `bucket-hash-table.go` (chaining, 338 lines) and `open-addressing-hash-table.go` (linear probing with tombstones, 282 lines). Both use FNV-1a hashing and dynamically resize when the load factor crosses 25% (shrink) or 75% (grow) thresholds.

## Architecture & Internals

### Bucket Chaining (Separate Chaining)

```
┌─────────────────────────────────────────────────────────┐
│              BucketHashTable                             │
│  capacity=16  size=5                                     │
│                                                          │
│  bucket[0]:  ●──→ [key="apple", value="red"] → nil      │
│  bucket[1]:  nil                                         │
│  bucket[2]:  ●──→ [key="carrot", value="orange"] → nil  │
│  bucket[3]:  ●──→ [key="dog", value="brown"]            │
│                       ↓                                  │
│                  [key="emu", value="grey"] → nil         │
│  bucket[4]:  ●──→ [key="fox", value="orange"] → nil     │
│  ...                                                     │
│  bucket[15]: nil                                         │
│                                                          │
│  Load factor: 5/16 = 31% (< 75% threshold — no resize)  │
└─────────────────────────────────────────────────────────┘
```

**Insert flow**: `FNVHash32(key) % capacity` → index. Walk the chain at that index. If key exists, update value. If not, prepend a new `KeyValuePair` to the chain (head insertion, O(1)). If `size > capacity * 0.75`, resize to `capacity * 2` by rehashing all entries into new buckets.

**Delete flow**: Walk the chain. Unlink the target node by updating `prev.next = entry.next` (or the bucket head if it's the first node). If `size < capacity * 0.25` and `capacity > minimumCapacity (16)`, resize to `capacity / 2`.

### Open Addressing (Linear Probing)

```
┌──────────────────────────────────────────────────────────┐
│           OpenAddressingHashTable                         │
│  capacity=16  size=4  used=5  (1 tombstone)              │
│                                                          │
│  buckets[0]:  [key="apple", value="red"]                 │
│  buckets[1]:  nil                                        │
│  buckets[2]:  [key="carrot", value="orange"]             │
│  buckets[3]:  [key="dog", deleted=true]  ← tombstone     │
│  buckets[4]:  [key="emu", value="grey"]                  │
│  buckets[5]:  nil                                        │
│  ...                                                     │
│  buckets[15]: nil                                        │
│                                                          │
│  Load factor: 4/16 = 25%  Used factor: 5/16 = 31%       │
└──────────────────────────────────────────────────────────┘
```

**Probe sequence**: `start = hash(key)`, then linear probe: `(start + offset) % capacity` for offset=0,1,2,... The probe stops at a nil slot (definitely absent) or after scanning the entire table (table is full). `findSlot` at line 117-145 returns the first available slot (nil or tombstone) if the key isn't found, preferring a tombstone slot over a nil slot so tombstones are recycled.

**Tombstone deletion**: Instead of physically removing an entry (which would break the probe chain for other entries that probed past it), `Delete` at line 231-262 marks the entry as `deleted=true`. The `used` counter (live entries + tombstones) tracks slots that cannot be recycled. When `used+1 > capacity * 0.75`, a resizing to the same capacity (`resize(capacity)`) cleans up all tombstones by rehashing only live entries.

## Production Use Cases

- **Python dict** — Python's dictionary uses open addressing with a custom probe sequence (not simple linear) and a max load factor of ~2/3. It's the foundation of Python's object model (all object attributes, module namespaces, and keyword arguments).
- **Java HashMap** — Uses separate chaining with treeification: when a bucket chain exceeds 8 nodes, it converts to a red-black tree to guard against worst-case O(n) degradation from crafted collision attacks.
- **Go maps** — Go's built-in `map` uses open addressing (specifically, a form of Robin Hood hashing) with incremental growth. The implementation is in Go's runtime (`runtime/map.go`) and compiles to highly optimized machine code.
- **Redis hash** — Redis uses a hash table for its main key-value store with incremental rehashing (two tables during resize) to avoid latency spikes. The `HSET`/`HGET` commands use a separate chaining hash internally.
- **PostgreSQL hash indexes** — PostgreSQL's hash index type uses a bucket-chaining hash table with write-ahead logging for crash safety. Unlike B-tree indexes, hash indexes provide O(1) equality lookups but don't support range queries.

## When to Use It

| Scenario | Chaining or Open Addressing? |
|----------|------------------------------|
| Keys are small, fixed-size (integers, short strings) | Open addressing — better cache locality |
| Keys are large or variable-length (long strings, blobs) | Chaining — avoids large bucket arrays |
| High deletion rate | Chaining — tombstones degrade open addressing |
| Memory-constrained (embedded systems) | Open addressing — no per-node overhead |
| Worst-case attack resistance needed | Chaining with treeification — guards against O(n) |
| Predictable latency (no resize pauses) is critical | Either + incremental rehashing |

**Don't use a hash table when** you need ordered iteration (use a B-Tree or Skip List), range queries (use a sorted structure), or prefix matching (use a Trie). Don't use open addressing if your deletion rate is high — tombstone accumulation forces periodic full rehashes.

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| Get | O(1) | O(n) | O(n) |
| Set | O(1) | O(n) | O(n) |
| Delete | O(1) | O(n) | O(n) |
| Resize | O(n) | O(n) | O(n) |

Average case is O(1) because the load factor is bounded (≤0.75), so probe chains are short (expected length ~1/(1-load) ≈ 1-4 entries). Worst case O(n) occurs when all keys hash to the same bucket (hash collision attack) — mitigated in chaining by treeification, in open addressing by using a strong hash function like SipHash or FNV-1a with a random seed.

## Implementation Deep Dive

**1. FNV-1a hashing for both implementations.** Both files use the Fowler–Noll–Vo hash (FNV-1a) with constants defined at `bucket-hash-table.go:17-26` and `open-addressing-hash-table.go:15-18`. FNV-1a is chosen because it's fast (one XOR and one multiply per byte), has good avalanche properties (a one-bit change in the key changes roughly half the output bits), and doesn't require a prime-sized table (unlike division hashing). The 32-bit variant is used for indexing; the 64-bit variant is available for larger tables.

**2. Dynamic resizing with 25%/75% load factor thresholds.** The `shouldGrow()` method at `bucket-hash-table.go:92-95` triggers when `size > capacity * 0.75`. The `shouldShrink()` method at line 99-102 triggers when `size < capacity * 0.25` AND `capacity > minimumCapacity (16)`. The hysteresis (shrink at 25%, grow at 75%) prevents "thrashing" — rapid grow-shrink cycles when size hovers around a threshold. The `minimumCapacity` of 16 prevents over-shrinking small tables.

**3. Open addressing tombstone cleanup via same-capacity resize.** In `open-addressing-hash-table.go:199-202`, when `used+1 > capacity * 0.75` (tombstones are consuming too many slots), the table is resized to the *same* capacity. This is not a typo — it's an intentional tombstone-compaction pass. Only live (non-deleted) entries are rehashed into the new array, effectively reclaiming all tombstone slots. This design avoids a separate "garbage collection" pass and ties cleanup to the insertion fast path where it's needed.

**4. Generic types in the open addressing variant.** `OpenAddressingHashTable[V any]` at line 37 is a Go generic, enabling type-safe storage of any value type without `interface{}` casting. The bucket chaining variant is string-only to keep the code focused on the collision resolution algorithm without generics noise. The `oaProbeResult` struct at line 29 captures the dual return (index + found) from `findSlot`, avoiding a sentinel index value like -1.

## Running the Demo

```bash
# Bucket chaining hash table
go run ./03-Hash-Table/bucket-hash-table.go
```

```bash
# Open addressing hash table (separate demo not provided; see tests)
go test ./03-Hash-Table/
```

## Further Reading

- "Fowler–Noll–Vo Hash Function" — Landon Curt Noll. The canonical specification of FNV-1 and FNV-1a hashing algorithms. http://www.isthe.com/chongo/tech/comp/fnv/
- "Open Addressing Hash Tables" — Chapter 6.4 of *The Art of Computer Programming, Vol. 3* by Donald Knuth. The definitive analysis of linear probing, quadratic probing, and double hashing.

---

*Part of the Design-With-TsGo system design curriculum*
