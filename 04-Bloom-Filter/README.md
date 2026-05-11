# Bloom Filter

> **A space-efficient probabilistic data structure for set membership: "definitely not present" or "possibly present." False positives are possible but false negatives are impossible — ideal for eliminating unnecessary disk/network lookups.**

## The Problem It Solves

You're building a username registration system for a social media platform with 500 million users. When a new user tries to sign up with "coolguy99," you need to check if that username is already taken. You could query the users database — but that's a disk seek (or at minimum a network round-trip to a replica). At peak sign-up rates of 10,000 per second, that's 10,000 database queries per second just for availability checks. The database team is already overprovisioned, and 99.9% of these queries return "not found" — most username choices are unique.

You could cache all 500 million usernames in memory. That's roughly 500 million × 20 bytes = 10 GB — possible, but wasteful when you're just doing negative lookups. What if you could answer "is this username taken?" with a tiny in-memory structure that's 100x smaller, at the cost of occasionally saying "maybe taken" when it's actually free? For a username check, a 1% false positive rate means 1% of users get a false "username taken" response and have to try another — a minor UX inconvenience versus 10 GB of RAM and 10,000 db queries/second.

This is exactly what a **Bloom filter** provides. It's a bit array of size *m* with *k* independent hash functions. To add an item, hash it with all *k* functions and set those *k* bits to 1. To check membership, hash the query item and check if all *k* bits are 1. If any bit is 0, the item was **definitely never added**. If all bits are 1, the item **might have been added** (or those bits might have been set by other items — a false positive). The genius is the space-membership tradeoff: with *m* bits and *k* hash functions, you can store *n* items with a false positive rate of approximately (1 - e^(-kn/m))^k.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────────┐
│                         BloomFilter                               │
│                                                                   │
│  bitSize = m      hashCount = k                                  │
│                                                                   │
│  bits[] (byte array):                                             │
│  ┌───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┐  │
│  │ 0 │ 1 │ 0 │ 0 │ 1 │ 1 │ 0 │ 1 │ 0 │ 1 │ 0 │ 0 │ 1 │ 0 │ 0 │  │
│  └───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┘  │
│                                                                   │
│  Add("alice"):                                                    │
│    h1("alice", seed=2166136261) = ...  → sets bit 3              │
│    h2("alice", seed=16777619)  = ...  → sets bit 7              │
│    h3 = h1 + 1*h2               = ...  → sets bit 11            │
│                                                                   │
│  Has("bob"):                                                      │
│    Check bits at index(h1), index(h2), index(h3)                 │
│    If all are 1 → "possibly present"        (true)               │
│    If any is 0 → "definitely not present"   (false)              │
│                                                                   │
│  Double Hashing: generate k hashes from just 2 base hashes:       │
│    hash_i = (h1 + i * h2) mod m    for i = 0..k-1               │
└──────────────────────────────────────────────────────────────────┘
```

The Bloom filter has three core components:

**Bit array** (`bits []byte`): Stored as a byte slice for efficient memory usage. Bit-level manipulation uses bitwise operations: `bits[index>>3]` selects the byte (index/8), `(index & 7)` selects the bit position within the byte (index % 8). Setting a bit: `bits[index>>3] |= 1 << (index & 7)`. Testing a bit: `bits[index>>3] & (1 << (index & 7))`.

**Double hashing for *k* hash functions**: Generating *k* truly independent hash functions is expensive. Instead, the implementation at `main.go:79-90` uses **Kirsch-Mitzenmacher optimization**: generate two base hashes (h1, h2) using FNV-1a with different seeds, then compute hash_i = (h1 + i * h2) mod m. This produces *k* hash values that are statistically indistinguishable from *k* independent hashes for Bloom filter purposes.

**Optimal parameter computation**: The constructor `NewBloomFilter` at `main.go:29-46` computes the optimal *m* (bit size) and *k* (hash count) from the expected number of items *n* and target false positive rate *p*:

```go
bitSize = ceil(-n * ln(p) / (ln(2)^2))
hashCount = max(1, round((bitSize/n) * ln(2)))
```

## Production Use Cases

- **Apache Cassandra** — Cassandra uses Bloom filters on SSTable files to avoid disk seeks. Before reading an SSTable for a key lookup, the Bloom filter is checked. If the filter says "no," the SSTable is skipped entirely. With typical false positive rates of 1%, Cassandra avoids 99% of unnecessary disk reads.
- **Google Bigtable** — Bigtable uses Bloom filters per tablet (SSTable) to reduce disk seeks for non-existent rows/columns. The filter is stored in memory alongside the tablet metadata so lookups never touch disk for missing keys.
- **Chrome Safe Browsing** — Google Chrome downloads a Bloom filter containing hashes of known malicious URLs. When a user navigates to a page, the browser checks the local filter first. A negative result avoids a network round-trip to Google's Safe Browsing servers.
- **Akamai CDN** — Akamai's edge servers use Bloom filters in their cache eviction pipeline. When a purge request arrives ("invalidate all URLs matching /images/\*"), a Bloom filter is checked to determine which cache servers might hold matching entries, avoiding broadcasting the purge to all 250,000+ servers.
- **PostgreSQL bloom indexes** — PostgreSQL 9.6+ includes a `bloom` index access method that creates a Bloom filter over multiple columns. It's designed for equality searches on wide tables where a B-tree on each column would be too expensive.

## When to Use It

| Scenario | Use Bloom Filter? |
|----------|-------------------|
| You need to eliminate expensive lookups for non-existent keys | **Yes** |
| A small false positive rate is acceptable (1% or less) | **Yes** |
| You need an in-memory set representation that's much smaller than the data | **Yes** |
| You need to iterate over elements or get the actual values | **No** — Bloom filters don't store data, only membership bits |
| You need exact answers with zero false positives | **No** — use a hash set |
| You need to delete items | **No** — standard Bloom filters don't support deletion (use Counting Bloom filter instead) |

**Alternatives**: **Cuckoo filters** support deletion and have better space efficiency at low false positive rates, but require more complex implementation. **Counting Bloom filters** replace each bit with a small counter (usually 4 bits), enabling deletion at the cost of 4x space. **Hash sets** (like Go's `map[string]struct{}`) give exact answers but store full keys — 100-1000x larger for the same element count.

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| Add | O(k) | O(k) | O(m) bits |
| Has | O(k) | O(k) | O(m) bits |
| Construction | O(1) | O(1) | O(m) bits |

Where *k* is the number of hash functions (typically 3-20) and *m* is the bit array size. Both Add and Has are independent of the number of items stored — that's the key benefit: lookup time doesn't grow as the set fills up. Space is fixed at construction time based on the expected items and target false positive rate; the filter doesn't grow automatically.

## Implementation Deep Dive

**1. Double hashing with FNV-1a and different seeds.** Rather than implementing *k* separate hash functions, `indexes()` at `main.go:79-90` generates all *k* positions from two base hashes using `h1 + i*h2 mod bitSize`. The two base hashes come from `fnv1aWithSeed()` at line 93-100, called with the standard FNV offset basis (2166136261) and the FNV prime (16777619) as seeds. A zero-guard at line 82-84 ensures h2 is non-zero so the sequence doesn't degenerate to all-identical values.

**2. Byte-level bit manipulation.** The bit array is `[]byte`, not `[]bool` or `[]uint64`. This is a deliberate engineering choice: Go's `bool` is 1 byte per element (massive overhead), while a byte slice stores 8 bits per byte. `bits[index>>3]` performs integer division by 8 via right-shift (faster than `/ 8`). `1 << (index & 7)` creates a bitmask via left-shift by `index % 8` (bitwise AND with 7 is faster than `% 8`). These micro-optimizations make bit operations cycle-count cheap.

**3. Optimal parameter computation at construction time.** `NewBloomFilter` at line 29-46 uses the well-known formulas from the original Bloom paper:
- `m = -n * ln(p) / (ln(2))^2` — optimal number of bits for *n* items at false positive rate *p*
- `k = (m/n) * ln(2)` — optimal number of hash functions

This means the user specifies business requirements ("I expect 1M usernames, I can tolerate 0.1% false positives") and the code computes the engineering parameters. The resulting filter is minimal in size for the given requirements.

## Running the Demo

```bash
go run ./04-Bloom-Filter/
```

## Further Reading

- "Space/Time Trade-offs in Hash Coding with Allowable Errors" — Burton H. Bloom (1970). The original paper that introduced the data structure. *Communications of the ACM, 13(7).*
- "Less Hashing, Same Performance: Building a Better Bloom Filter" — Kirsch and Mitzenmacher (2006). Shows that two hash functions suffice for Bloom filter performance, enabling the double-hashing technique used in this implementation. *Random Structures & Algorithms, 33(2).*

---

*Part of the Design-With-TsGo system design curriculum*
