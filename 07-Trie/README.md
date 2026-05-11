# Trie (Prefix Tree)

> **A tree structure where each node is keyed by a character (rune) and paths from root to leaf spell out stored words — O(m) insert/search/delete where m is key length, with native prefix-based retrieval for autocomplete engines.**

## The Problem It Solves

You're building the autocomplete search bar for a developer CLI tool. When a user types "sys", you need to instantly suggest all commands that start with "sys": `system`, `syscall`, `syslog`, `sync`, `syntax`, `sysctl`, `systemd`, `sysstat`. With a hash table, you can check if "systemd" is a valid command in O(1), but you can't easily ask "give me all keys starting with 'sys'." You'd have to iterate every key in the hash table and check `strings.HasPrefix` — O(n) where n is the total number of stored commands. With 5,000 commands, that's 5,000 prefix checks per keystroke. On a 60 Hz display, that's 300,000 checks per second just for one user's typing.

A sorted array with binary search could find the first key ≥ "sys" in O(log n), but then you still need to scan forward to collect all matches — O(log n + k) for k matches, plus O(n) insertion when adding new commands. A hash table gives O(1) exact match but zero prefix support. You need a data structure where the key *is* the navigation path.

The **trie** (pronounced "try," from re**trie**val) stores strings character-by-character as paths in a tree. Each node is a map from characters to child nodes, and a boolean `end` flag marks complete words. To find all words starting with "sys," you walk down the path `s → y → s` (O(m) where m=3), then perform a depth-first collection of all complete words reachable from that node. The trie inherently *shares* common prefixes — "system" and "systemd" share the "system" path, saving memory for overlapping keys. This makes tries the de facto standard for autocomplete systems, spell checkers, and IP routing table lookups (longest prefix match).

## Architecture & Internals

```
┌───────────────────────────────────────────────────────────────────┐
│  Trie: root                                                       │
│                                                                    │
│                   children: {'s','c'}                              │
│                   ┌─────────┴─────────┐                           │
│                   │                   │                            │
│              's': *Node           'c': *Node                       │
│                   │                   │                            │
│              children: {'y'}     children: {'o'}                   │
│                   │                   │                            │
│              'y': *Node           'o': *Node                       │
│                   │                   ├────────────┐              │
│              children: {'s','n'}  children:      children:        │
│                   │           │    {'n','m'}      {'n'}           │
│              ┌────┘      ┌────┘       │              │            │
│         's': *Node  'n': *Node   'n': *Node      'n': *Node       │
│              │           │            │              │            │
│         children:    end=true    end=true       children:         │
│         {'t','c'}    value:      value:         {'t','f'}         │
│              │      "sync"      "syscall"          │              │
│         ┌────┴────┐                      ┌─────────┴──────┐      │
│    't': *Node 'c': *Node           't': *Node        'f': *Node   │
│         │         │                     │                │        │
│    children:  children:           end=true           end=true     │
│     {'e'}     {'a'}              value:            value:         │
│         │         │            "container"        "config"        │
│    'e': *Node 'a': *Node                                          │
│         │         │                                               │
│    children:  children:                                           │
│     {'m'}      {'l'}                                              │
│         │         │                                               │
│    'm': *Node 'l': *Node                                          │
│         │         │                                               │
│    end=true  end=true                                             │
│    value:    value:                                               │
│   "system"  "syscall"  (if "sysca" led here)                     │
│                                                                   │
│  WordsWithPrefix("sys"):                                          │
│    → Walk root→'s'→'y'→'s'                                       │
│    → DFS collect: "system", "syscall", "syslog", ...              │
│    → Returns all complete words under that node                   │
└───────────────────────────────────────────────────────────────────┘
```

The trie is defined by two types in `main.go:18-29`. `TrieNode` (line 18) holds `children map[rune]*TrieNode` (the branching structure), `end bool` (marks complete words), and `value string` (the associated value for terminal nodes). `Trie` (line 27) wraps a single `root *TrieNode`.

**Insert** (`main.go:40-51`): Walks the word rune-by-rune, creating missing child nodes. At the final character, sets `end=true` and stores the value. O(m) where m = word length.

**Search** (`main.go:55-63`): Walks to the terminal node via `findNode` (line 101). Returns the value only if `end` is true — this prevents partial prefixes from matching as complete words.

**Delete** (`main.go:75-83`): Recursively descends to the terminal node via `deleteFrom` (line 114). Marks `end=false`. On the way back up, if a child node has `end=false` AND an empty `children` map, it's deleted from its parent's map — **lazy cleanup** that prunes the trie of orphaned nodes.

**WordsWithPrefix** (`main.go:87-97`): Walks to the prefix node, then performs a DFS `collect` (line 134-141) that appends all terminal words in that subtree. Used directly for autocomplete.

## Production Use Cases

- **Elasticsearch Completion Suggester** — Elasticsearch's completion suggester uses a finite-state transducer (a compressed trie variant) to provide autocomplete suggestions with millisecond latency. It stores n-grams in a trie-like structure for search-as-you-type queries.
- **Google Search Autocomplete** — Google's search box uses a trie (or a probabilistic variant) to suggest completions based on popular queries. The "Google Instant" feature that updates results as you type relies on prefix tree structures to match partial queries against billions of indexed phrases.
- **IP routing tables (Longest Prefix Match)** — Internet routers use trie variants (Patricia tries, LC-tries) to store routing prefixes (e.g., 192.168.0.0/16). When a packet arrives, the router walks the trie to find the longest matching prefix, determining the next-hop gateway. Linux kernel's FIB (Forwarding Information Base) uses LC-tries.
- **Spell checkers** — Hunspell (used in LibreOffice, Chrome, Firefox) and Aspell use tries to store dictionaries. Finding all valid words within edit distance 1 of a misspelling involves walking the trie and skipping/inserting/changing one character — naturally expressed as a recursive trie traversal.
- **Redis Sorted Sets with Lexicographic Ordering** — When all scores in a Redis sorted set are equal (e.g., 0), Redis falls back to lexicographic ordering, which uses a trie-like data structure internally for ZRANGEBYLEX operations.

## When to Use It

| Scenario | Use Trie? |
|----------|-----------|
| You need prefix-based retrieval (autocomplete, "starts with") | **Yes** |
| Keys share common prefixes (e.g., URLs, file paths, commands) | **Yes** — memory savings from prefix sharing |
| You need to count/aggregate by key prefixes | **Yes** |
| Keys are short (tens of characters) and alphabet is small (ASCII, alphanumeric) | **Yes** |
| Keys are very long (thousands of characters) and sparse | **No** — trie depth = key length, excessive nodes |
| You only need exact-match lookups | **No** — a hash table is simpler and faster |
| Memory is extremely constrained and keys don't share prefixes | **No** — each character creates a node; hash tables have lower overhead for random keys |

**Alternatives**: **Ternary search trees** (TST) use less memory per node (3 child pointers vs. a full map at each node) at the cost of slightly higher search time. **Patricia tries (radix trees)** compress single-child paths into edges with string labels, dramatically reducing node count for sparse keys. **Directed Acyclic Word Graphs (DAWGs)** minimize tries to their minimal DFA form — maximum compression but static (no insertion after construction).

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| Insert | O(m) | O(m) | O(m * alphabet_size) |
| Search | O(m) | O(m) | O(1) additional |
| Delete | O(m) | O(m) | O(1) additional |
| StartsWith | O(p) | O(p) | O(1) additional |
| WordsWithPrefix | O(p + k) | O(p + k) | O(k) where k = number of matches |

Where *m* is the key length and *p* is the prefix length. Time depends only on key length, not on the number of stored entries — the trie's defining advantage over hash table + prefix scan. Space is O(total_characters_stored) in the worst case (no prefix sharing), but typically much better due to shared prefixes.

## Implementation Deep Dive

**1. Rune-keyed children map for Unicode support.** The `children map[rune]*TrieNode` at line 19 uses Go's `rune` type (Unicode code point) rather than `byte`. This means the trie correctly handles non-ASCII characters: é, ñ, 中文, emoji. For strings that are known to be ASCII-only, a `[26]*TrieNode` array keyed by `char-'a'` would be faster and more memory-efficient — but the rune map is correct for general text.

**2. Recursive delete with lazy child cleanup.** The `deleteFrom` function at line 114-131 implements a recursive descent. The base case (line 115-121) clears the `end` flag when the full word is reached. The recursive case (line 123-131) descends into the child, and on return, checks `if !child.end && len(child.children) == 0` — if the child is no longer a terminal and has no children of its own, it's safe to delete from the parent's map. This prevents orphaned node chains from accumulating after deletions.

**3. Collect for autocomplete via DFS.** The `collect` function at line 134-141 performs a depth-first traversal of the subtree. When it encounters a node with `end=true`, it appends the accumulated word to the result slice. For each child, it recurses with the child's character appended to the word. Since map iteration order is random in Go, the results are **not** in lexicographic order — a production autocomplete would use a sorted data structure (like a `[]struct{rune, *TrieNode}` slice) within each node, or sort the results after collection.

**4. `findNode` as the common traversal primitive.** The unexported `findNode` at line 101-110 factors out the character-by-character walk shared by Search, StartsWith, and WordsWithPrefix. This DRY design means prefix matching correctness lives in exactly one place. If a character isn't found in the children map, it returns nil immediately — the prefix doesn't exist.

## Running the Demo

```bash
go run ./07-Trie/
```

## Further Reading

- "Trie Memory" — Edward Fredkin (1960). The original paper coining the term "trie" and describing the data structure. *Communications of the ACM, 3(9).*
- "Fast IP Routing with LC-Tries" — Nilsson and Karlsson (1999). Describes a level-compressed trie variant used in the Linux kernel for FIB lookups. *IEEE/ACM Transactions on Networking, 7(4).*

---

*Part of the Design-With-TsGo system design curriculum*
