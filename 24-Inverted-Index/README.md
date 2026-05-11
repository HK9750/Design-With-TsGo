# Inverted Index

> **A data structure that maps terms to the documents containing them — the core engine powering every search bar you've ever typed into, from Google to Elasticsearch.**

## The Problem It Solves

You've built a documentation platform with 50,000 articles. Users want to type "distributed systems design" and instantly find relevant articles. The naive approach — a SQL `WHERE content LIKE '%distributed%' AND content LIKE '%systems%'` — is catastrophically slow. The database scans every row, character by character, O(n × m) where n is documents and m is average document length. At 50,000 documents × 5KB each, that's 250MB of text scanned per query. Concurrently serving 100 queries/second means reading 25GB/second. Your database melts.

The inverted index solves this by preprocessing: at index time, each document is tokenized into terms, and for each unique term, we store a list of document IDs where that term appears (the *posting list*). At query time, we look up each query term's posting list and intersect them. Now "distributed systems design" becomes three hash-map lookups followed by set intersection — O(t × p) where t is the number of query terms and p is the average posting list size. Instead of scanning 250MB per query, we read maybe 10KB of posting lists. That's a 25,000× speedup.

The inverted index also stores *term frequencies* — how many times a term appears in each document. This powers relevance ranking (TF-IDF, BM25): a document where "distributed" appears 10 times is more relevant than one where it appears once. Our implementation stores these frequencies (`map[string]int` per term in `postings`), ready for ranking extensions.

## Architecture & Internals

```
                       INDEXING PIPELINE
    ┌─────────────────────────────────────────────────────────┐
    │                                                         │
    │  Document: "distributed systems design patterns"        │
    │     │                                                   │
    │     ▼                                                   │
    │  ┌──────────────┐                                       │
    │  │  Tokenizer    │  regex: [a-z0-9]+                    │
    │  │  (lowercase + │  Input → ["distributed","systems",   │
    │  │   regex split)│           "design","patterns"]       │
    │  └──────┬───────┘                                       │
    │         │                                               │
    │         ▼                                               │
    │  ┌──────────────┐  Count term frequencies:              │
    │  │  Term Freq   │  {distributed:1, systems:1,           │
    │  │  Counter     │   design:1, patterns:1}               │
    │  └──────┬───────┘                                       │
    │         │                                               │
    │         ▼                                               │
    │  ┌──────────────────────────────────────────────┐       │
    │  │           INVERTED INDEX (postings)           │       │
    │  │                                              │       │
    │  │  "distributed" → {doc1: 1, doc3: 1}          │       │
    │  │  "systems"     → {doc1: 1, doc2: 1}           │       │
    │  │  "design"      → {doc1: 1, doc3: 1, doc4: 1} │       │
    │  │  "patterns"    → {doc1: 1, doc3: 1}           │       │
    │  │  "programming" → {doc2: 1, doc3: 1}           │       │
    │  │  "rust"        → {doc2: 1, doc3: 1}           │       │
    │  │  "go"          → {doc2: 1, doc4: 1}           │       │
    │  │  "scalable"    → {doc1: 1, doc4: 1}           │       │
    │  │  ...                                          │       │
    │  └──────────────────────────────────────────────┘       │
    │                                                         │
    └─────────────────────────────────────────────────────────┘

                      QUERY EXECUTION (AND)
    ┌─────────────────────────────────────────────────────────┐
    │                                                         │
    │  Query: "systems design"                                │
    │     │                                                   │
    │     ▼                                                   │
    │  Terms: ["systems", "design"]                           │
    │     │                                                   │
    │     ├── systems → {doc1, doc2}         (posting list 1) │
    │     │                                                   │
    │     └── design  → {doc1, doc3, doc4}  (posting list 2) │
    │                │                                        │
    │                ▼                                        │
    │         INTERSECTION: {doc1, doc2} ∩ {doc1, doc3, doc4} │
    │                      = {doc1}                           │
    │                                                         │
    │  Result: ["doc1"]  (sorted)                             │
    │                                                         │
    └─────────────────────────────────────────────────────────┘
```

**Why it's called "inverted":** A normal index maps document → words. An inverted index maps word → documents. It's a transpose of the natural organization.

**Key design in our Go code:**
- `postings map[string]map[string]int` (`main.go:23`): The outer map is term → inner map, the inner map is documentID → term frequency. This double-map design is memory-efficient for small-scale indexes. Production systems (Lucene) use skip lists and delta-encoded integers for posting list compression.
- `tokenPattern` regex (`main.go:17`): `[a-z0-9]+` extracts only lowercase alphanumeric sequences. This is deliberately simple — production tokenizers handle Unicode, compound words, stemming ("running" → "run"), stopword removal, and n-gram generation.
- Conjunctive AND (`main.go:56`): Only returns documents containing ALL query terms. This is the strictest search mode. Lucene also supports OR (union) and NOT (subtraction).

## Production Use Cases

- **Elasticsearch (Apache Lucene)**: The de facto standard for full-text search. Lucene's inverted index powers Elasticsearch, handling everything from log analytics (ELK stack) to e-commerce product search. Each Lucene segment is an independent inverted index.
- **Apache Solr**: The older sibling of Elasticsearch, also built on Lucene. Powers search for Netflix (content discovery), Instagram (user search), and the White House website.
- **Algolia**: A hosted search API used by Stripe, Slack, and Medium. Uses a custom inverted index optimized for instant-search latency (typo tolerance, prefix search).
- **PostgreSQL GIN indexes**: Generalized Inverted Index — used for full-text search (`tsvector`/`tsquery`), JSONB key existence queries, and array containment (`@>`). The GIN index maps indexed values to heap tuple IDs.
- **Meilisearch**: An open-source search engine in Rust, used by Louis Vuitton and other enterprise customers. Uses an inverted index with typo-tolerant prefix matching.

## When to Use It

| Use Inverted Index when... | Don't use when... |
|---|---|
| Users search unstructured text (articles, logs, emails) | Data is structured with exact-match lookups only (use a hash map) |
| Queries involve multiple terms with AND/OR semantics | You only need prefix search in an ordered key set (use a B-Tree) |
| Results need relevance ranking by term frequency | Documents never change after indexing (append-only is simpler) |
| Query latency is critical for user-facing search | Documents are tiny and brute-force scan is fast enough |
| You need faceted search (filter by category + search text) | You only need full-scan pattern matching (grep-style) |

**Relevance ranking is essential**: Without it, a search for "database design" might return a document with each word appearing once on page 50 as equally relevant as the cover page. Production systems use TF-IDF (term frequency × inverse document frequency) or BM25. Common words like "the" get near-zero weight; rare terms like "B-Tree" get high weight. Our implementation omits ranking and returns sorted IDs — but the term frequencies stored in postings are ready for TF computation.

**Updates are the weakness**: Inverted indexes are optimized for append-only indexing. Deleting or updating a document requires re-tokenizing it and adjusting every term's posting list — potentially touching thousands of posting lists for a single document. Real systems handle this with segment merging (Lucene) or write-optimized inverted indexes.

## Complexity Analysis

| Operation | Time | Space |
|---|---|---|
| Add document | O(n + d) where n = tokens, d = distinct terms | O(d × avg posting list overhead) |
| Search (AND) | O(t × p) where t = query terms, p = avg posting list size | O(results) |
| Tokenize | O(m) where m = text length | O(tokens) |
| Space (total) | O(unique terms × avg documents per term) | ~10-50% of original text size |

## Implementation Deep Dive

### 1. Tokenization via Regex (`main.go:94`)

The `tokenize` function lowercases the input and applies `tokenPattern.FindAllString`, extracting all `[a-z0-9]+` sequences. This is a *word tokenizer* — it splits on whitespace and punctuation. The key design choice is simplicity: no stemming, no stopword filtering, no n-gram generation. This makes the index transparent (what you index is exactly what you search), but less effective for natural-language queries ("running" won't match "run"). Production tokenizers apply linguistic analysis (Porter stemmer, Snowball) to collapse morphological variants.

### 2. AND-Conjunctive Query (`main.go:56-89`)

`SearchAll` implements Boolean AND semantics. It starts with the posting list of the first query term as the candidate set, then iterates over remaining terms, removing document IDs from the candidate set that don't appear in each term's posting list. This is efficient when posting list sizes vary — by starting with the smallest posting list (the rarest term), we minimize the number of deletions. Our implementation always uses the first term, but a simple optimization would be to iterate terms in ascending order of posting list size.

### 3. Term Frequency Storage (`main.go:37-47`)

For each document, `AddDocument` first counts term frequencies locally (a map from term to count within the document), then stores these counts in the global postings. Storing frequencies (not just binary presence) enables future relevance scoring: a document with 5 occurrences of "distributed" should rank higher than one with 1 occurrence. Lucene extends this with *positional information* (byte offset of each occurrence) for phrase queries — "distributed systems" must appear as adjacent words, not just both present.

## Running the Demo

```bash
go run ./24-Inverted-Index/
```

The demo indexes 4 technical documents about distributed systems programming, then runs 5 conjunctive queries ("systems design", "distributed programming", "Rust Go", etc.) and reports matching document IDs.

## Further Reading

- **"The Anatomy of a Large-Scale Hypertextual Web Search Engine"** — Sergey Brin and Lawrence Page (1998). The original Google paper. Describes their inverted index architecture, PageRank relevance ranking, and how they handled 24 million web pages in 1998.
- **Lucene Index File Formats (lucene.apache.org)** — Detailed specification of Lucene's on-disk inverted index format: posting list compression (FOR, PFOR), skip lists, and segment merging strategies.

---

*Part of the Design-With-TsGo system design curriculum*
