# LFU Cache

> **A bounded cache that evicts by access frequency rather than recency — the least-used entries drop first, making it ideal for CDNs and content platforms where popularity (not timing) determines what stays cached.**

## The Problem It Solves

You're an infrastructure engineer at a video streaming platform. Your CDN edge nodes cache the first 30 seconds of every video so that playbacks start instantly. Most users watch the latest Marvel trailer — it gets hit 50,000 times per minute. A handful of users browse a 2016 documentary that gets 3 hits per day. You have 256 GB of SSD cache per edge node, and it fills up. An LRU cache would keep the documentary (accessed 10 seconds ago) and evict the Marvel trailer (not accessed in the last 2 seconds of the refill window). That's catastrophic — 50,000 users per minute would suddenly hit origin, saturating your backbone.

The problem is that LRU measures *recency*, but your workload cares about *frequency*. The Marvel trailer is hot because it's popular, not because it was touched recently. In fact, any content popular enough to get hit 50,000 times per minute will inevitably have moments where *this specific second* no one happens to request it — and an LRU would evict it. You need a cache that accumulates a "reputation score" per item and evicts the items with the lowest scores.

The **Least Frequently Used (LFU)** cache solves exactly this. Every `get` increments a counter on the entry. When eviction is necessary, the entry with the lowest access count is removed. Ties (multiple entries with the same minimum frequency) are broken by LRU within that frequency bucket — so among equally-unpopular items, the stalest one goes first. This creates a natural stratification: viral content rises to higher frequency tiers and becomes effectively immune to eviction, while one-hit-wonder content sinks and gets recycled.

## Architecture & Internals

```
┌────────────────────────────────────────────────────────────────┐
│                         LFUCache                               │
│                                                                │
│  KeyToNodes (map)                FreqToList (map)              │
│  ┌──────────────┐               ┌──────────────────┐          │
│  │ key:42 → ●───┼──┐            │ freq=1 → DLL ●───┼──────────│──┐
│  │ key:17 → ●───┼──┼──┐         │ freq=2 → DLL     │          │  │
│  │ key:99 → ●───┼──┼──┼──┐      │ freq=3 → DLL ●───┼──────────│──│──┐
│  │ key:55 → ●───┼──┼──┼──┼──┐   │ freq=6 → DLL     │          │  │  │
│  └──────────────┘  │  │  │  │   └──────────────────┘          │  │  │
│                    ▼  ▼  ▼  ▼                                 │  │  │
│                  Freq=1  Freq=2  Freq=3  Freq=6               │  │  │
│                  DLL     DLL     DLL     DLL                  │  │  │
│                                                                │  │  │
│   MinFreq = 1                                                  │  │  │
│   (eviction target: removeLast from FreqToList[MinFreq])      │  │  │
└────────────────────────────────────────────────────────────────┘  │  │  │
   key=42: ListNode{Freq:1,Value:...} ◄────────────────────────────┘  │  │
   key=17: ListNode{Freq:2,Value:...} ◄───────────────────────────────┘  │
   key=99: ListNode{Freq:3,Value:...} ◄──────────────────────────────────┘
```

The LFU cache uses **two maps**: `KeyToNodes` (key → `*ListNode`) for O(1) lookup and `FreqToList` (frequency → `*DoublyLinkedList`) for O(1) access to per-frequency groupings. Each doubly-linked list is itself a standard DLL with sentinel head/tail nodes, providing O(1) add-to-head, remove-node, and remove-last operations.

**On `get(key)`**: The hashmap finds the node in O(1), then `update(node)` is called. This function (`main.go:149-165`) removes the node from its current frequency bucket's list, checks if that bucket is now empty (and was the `minFreq` — if so, `minFreq++`), increments `node.Freq`, and adds it to the next frequency's list (creating the list if needed). All O(1).

**On `put(key, value)`**: If the key exists, update value and increment frequency. If it's new and the cache is full, eviction targets `FreqToList[MinFreq].removeLast()` — the least-recently-used entry within the least-frequently-used bucket. Then the new node is inserted into `FreqToList[1]` and `MinFreq` is reset to 1.

The critical insight is `MinFreq` tracking: instead of scanning all frequency buckets to find the lowest non-empty one, we maintain this as a running variable. It only increments when a node from the `MinFreq` bucket gets promoted and empties that bucket. A new insertion always resets it to 1. This avoids O(k) scans over frequency buckets and keeps every operation O(1).

## Production Use Cases

- **Redis `allkeys-lfu`** — Redis 4.0 introduced the LFU eviction policy as an alternative to LRU. It uses an approximate LFU with a logarithmic counter and time-decay to avoid overflow, but the core idea is the same: evict the least frequently accessed keys.
- **Cloudflare CDN** — Cloudflare's edge caches use frequency-based eviction to ensure that assets for popular websites stay cached while one-off requests don't displace them. Their Argo Smart Routing also uses frequency-weighted path selection.
- **Akamai CDN** — Akamai's edge servers maintain popularity scores for cached objects. Objects with low request counts are purged first under memory pressure, keeping the working set of the top-N most popular objects.
- **Netflix Open Connect** — Netflix's CDN appliances cache video chunks at ISP peering locations. Chunks for popular shows (Stranger Things) maintain high hit counts and are pinned; chunks for niche content that rarely get accessed are LFU-evicted.

## When to Use It

| Scenario | Use LFU? |
|----------|----------|
| Access distribution is skewed — a few items dominate traffic | **Yes** |
| Popular items should never be evicted even if untouched for a while | **Yes** |
| Cache pollution from one-hit-wonders is a problem (scan workloads) | **Yes** |
| Items naturally expire or have TTLs | **No** — LFU doesn't handle time-based expiry well without decay |
| Access patterns change frequently (trending topics, breaking news) | **No** — old popular items accumulate high counts and won't evict for new hot items; consider LFU with decay or LRU |
| Memory overhead of frequency buckets is a concern | **No** — LFU uses more per-entry metadata than LRU |

**Alternatives**: **LRU** (01-LRU-Cache) is simpler, uses less memory, and handles shifting workloads better because old entries naturally age out. **Window-TinyLFU** (used in Caffeine, the Java caching library) combines a window LRU for recent entries with a TinyLFU sketch for frequency estimation — it solves the "stale popularity" problem of pure LFU.

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| Get | O(1) | O(1) | O(n) |
| Put | O(1) | O(1) | O(n) |
| Evict | O(1) | O(1) | O(n) |
| Update frequency | O(1) | O(1) | O(n) |

Every operation is O(1) because `KeyToNodes` provides direct node access, each `DoublyLinkedList` supports O(1) add/remove/removeLast, and `minFreq` tracking avoids scanning frequency buckets. Space is O(n) for the cached items, plus O(f) for frequency bucket lists where f is the number of distinct frequency levels (bounded by the highest access count).

## Implementation Deep Dive

**1. Separate doubly-linked list per frequency bucket (not one global list).** The Go implementation at `main.go:76-82` stores `FreqToList map[int]*DoublyLinkedList` — a mapping from each frequency integer to its own list. This is what makes frequency-increment O(1): when a node goes from freq=3 to freq=4, we just remove it from the freq=3 list and add to the freq=4 list. If we had a single global list sorted by frequency, increment would require repositioning the node (O(n) scan). The per-bucket design is the key architectural decision.

**2. `MinFreq` tracked as a monotonically-increasing variable.** At line 77, `MinFreq int` is maintained not by scanning but by incrementing only when the min-freq bucket empties (line 154: `if freq == lfu.MinFreq && list.Size == 0 { lfu.MinFreq++ }`). This works because frequencies only increase — a node never drops back to a lower freq. When a new item is inserted at freq=1, `MinFreq` is reset to 1 (line 142). This is a lazy, O(1) maintenance strategy.

**3. LRU tie-breaking within frequency buckets.** Each `DoublyLinkedList` at lines 24-71 maintains insertion order. New nodes are added to the head (line 47-53: `addNode`). Eviction calls `removeLast()` (line 64-71) which removes from the tail. So within the same frequency bucket, the least-recently-added entry is evicted first. This gives a graceful degradation: even among equally-unpopular items, the stalest one goes.

## Running the Demo

```bash
go run ./02-LFU-Cache/
```

## Further Reading

- "Top 10 Algorithms in Data Mining" — Wu et al. (2008). Surveys the most influential algorithms in data mining, including frequency-based approaches like LFU. *Knowledge and Information Systems, 14(1).*
- Redis LFU implementation notes — https://redis.io/docs/latest/develop/reference/eviction/#the-new-lfu-mode

---

*Part of the Design-With-TsGo system design curriculum*
