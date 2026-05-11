# Document Store

> **A schema-flexible database storing semi-structured documents with secondary indexes for field-level queries — the sweet spot between rigid SQL tables and chaotic blob storage.**

## The Problem It Solves

You're building a user profile management system. A profile has a name, email, and role — straightforward. But then Marketing wants to add "preferred_language" and "last_campaign_clicked." Engineering wants "deployment_region" for internal users. The Sales team wants "company_size" and "annual_revenue." Every team has different fields, and they change weekly. If you're using a relational database, every new field means an `ALTER TABLE ADD COLUMN` migration — which locks the table, requires downtime, and creates a sprawling schema with 200 nullable columns, 95% of which are NULL for any given row.

A document store solves this by embracing schema flexibility. Each document is a self-describing map of key-value pairs — `map[string]any` in Go, a JSON object in MongoDB, a nested structure in Firestore. Different documents in the same collection can have completely different fields. There's no migration to run; you just start writing the new field. The database doesn't care that `{"name": "Alice", "role": "admin"}` and `{"name": "Bob", "role": "dev", "dept": "eng", "skills": ["Go", "Rust"]}` have different shapes.

But flexibility creates a query problem. Without a schema, how do you efficiently find "all documents where `dept` = 'engineering'"? Scanning every document (O(n) full collection scan) doesn't scale. This is where *secondary indexes* come in: you can create an index on any field, on demand. The index maps field values to document IDs, enabling O(1) lookups on that field. Indexes can be created lazily (as in our implementation) or declared ahead of time (as in MongoDB). Either way, they transform the document store from a dumb blob store into a queryable database.

## Architecture & Internals

```
                   DOCUMENT STORAGE
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  DocumentStore                                       │
    │  ┌──────────────────────────────────────────────┐   │
    │  │  docs: map[string]Document                    │   │
    │  │                                              │   │
    │  │  "u1" → {"type":"user","role":"admin",       │   │
    │  │           "name":"Ada","dept":"engineering"}  │   │
    │  │                                              │   │
    │  │  "u2" → {"type":"user","role":"developer",   │   │
    │  │           "name":"Grace","dept":"engineering"}│   │
    │  │                                              │   │
    │  │  "u3" → {"type":"user","role":"manager",     │   │
    │  │           "name":"Alan","dept":"research"}    │   │
    │  │                                              │   │
    │  │  "u4" → {"type":"service_account",           │   │
    │  │           "role":"bot","name":"ci-builder",   │   │
    │  │           "dept":"infra"}                     │   │
    │  └──────────────────────────────────────────────┘   │
    │                                                     │
    └─────────────────────────────────────────────────────┘

                 SECONDARY INDEXES
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  indexes: map[string]map[string]map[string]bool     │
    │                                                     │
    │  "type" → ┌──────────────────────────────────┐     │
    │           │ "user"            → {"u1","u2","u3"}    │
    │           │ "service_account" → {"u4"}              │
    │           └──────────────────────────────────┘     │
    │                                                     │
    │  "dept" → ┌──────────────────────────────────┐     │
    │           │ "engineering" → {"u1","u2"}             │
    │           │ "research"    → {"u3"}                  │
    │           │ "infra"       → {"u4"}                  │
    │           └──────────────────────────────────┘     │
    │                                                     │
    │  "role" → ┌──────────────────────────────────┐     │
    │           │ "admin"     → {"u1"}                   │
    │           │ "developer" → {"u2"}                   │
    │           │ "manager"   → {"u3"}                   │
    │           │ "bot"       → {"u4"}                   │
    │           └──────────────────────────────────┘     │
    │                                                     │
    └─────────────────────────────────────────────────────┘

             QUERY: FindByField("dept", "engineering")
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  Index lookup path (fast):                          │
    │    1. indexes["dept"]["engineering"]                │
    │       → {"u1": true, "u2": true}                    │
    │    2. docs["u1"] + docs["u2"]                       │
    │    Result: [Ada (admin), Grace (developer)]         │
    │                                                     │
    │  Full-scan fallback (no index on field):             │
    │    1. For each doc in docs:                         │
    │       if fmt.Sprint(doc["color"]) == "red":         │
    │         append to result                            │
    │    Result: O(n) scan                                │
    │                                                     │
    └─────────────────────────────────────────────────────┘
```

**Key structures:**

- `Document` (`main.go:15`): A type alias for `map[string]any` — Go's closest equivalent to a schema-less document. The `any` value type means a field can hold a string, number, boolean, nested map, or slice. This is a simplified model: production document stores support richer types (dates, ObjectIDs, binary data) with type-specific query operators (`$gt`, `$in`, `$regex`).
- `DocumentStore.indexes` (`main.go:24`): A triple-nested map: `field → value → set of document IDs`. The innermost `map[string]bool` acts as a set — using `bool` values avoids storing duplicate IDs.

## Production Use Cases

- **MongoDB**: The most popular document store. Documents are BSON (Binary JSON), stored in collections. Supports ad-hoc queries, secondary indexes (single-field, compound, text, geospatial, TTL), aggregation pipelines, and multi-document transactions since 4.0. Used by Forbes, Cisco, and the UK government's GOV.UK.
- **Couchbase**: A distributed document database with a memory-first architecture. Supports N1QL (SQL for JSON), full-text search via Bleve, and cross-datacenter replication (XDCR). Used by LinkedIn (caching layer), PayPal, and many telecom companies.
- **Firebase Firestore**: Google's serverless document store. Documents are organized in collections, with real-time listeners for live updates. Powers the backend for millions of mobile and web apps. Automatically scales and handles offline support.
- **Amazon DocumentDB**: AWS's MongoDB-compatible managed service. Built on top of Aurora's distributed storage engine, providing MongoDB 3.6/4.0 API compatibility with automatic scaling up to 64TB.
- **Elasticsearch (document mode)**: While primarily a search engine, Elasticsearch stores JSON documents with dynamic mapping and supports secondary indexes on every field by default (inverted index on all fields).

## When to Use It

| Use Document Store when... | Don't use when... |
|---|---|
| Your data model is naturally expressed as JSON-like documents | Your data has strict schemas with enforced relationships |
| Different entities in the same collection have different fields | You need complex JOINs across collections (denormalize instead) |
| You iterate fast and the schema changes frequently | You need ACID transactions across multiple documents |
| You need to evolve data format without downtime | Every document must have exactly the same fields and types |
| Your access pattern is mostly "get document by ID" or "find by indexed field" | Your queries involve complex aggregations with GROUP BY and HAVING |

**Index everything or nothing?** MongoDB and Elasticsearch can automatically index all fields (dynamic mapping). This is convenient but expensive: each index consumes memory and slows writes (every field update must update the corresponding index). The sweet spot is to index only the fields you actually query — typically 3-5 per collection. Our implementation supports this with explicit `CreateIndex(field)`.

**Denormalization is normal**: In a relational database, you'd normalize a user's profile into separate tables (users, addresses, preferences). In a document store, you'd embed everything in one document. A query for a user profile returns one document, not a 3-table JOIN. The tradeoff is data duplication (an address embedded in 5 orders is stored 5 times) — but storage is cheap, and query simplicity is valuable.

## Complexity Analysis

| Operation | Time (indexed) | Time (no index) | Space |
|---|---|---|---|
| Upsert | O(i) where i = index count | O(i) | O(doc size + index entries) |
| FindByField | O(1) + O(matching docs) | O(n) full scan | O(results) |
| CreateIndex | O(n) | N/A | O(unique values × docs per value) |
| Full collection scan | N/A | O(n) | O(results) |

## Implementation Deep Dive

### 1. Lazy Index Creation (`main.go:40-48`)

`CreateIndex` scans every existing document in the collection and builds the index from scratch. This is *lazy/on-demand* — indexes aren't maintained automatically; you must create them explicitly. For each document, `addToIndex` inserts the document ID into a set keyed by the string representation of the field value (`fmt.Sprint(doc[field])`). Using `fmt.Sprint` means all field values are converted to strings for indexing — a simplification that works for demo purposes but in production you'd want type-specific indexing (numeric fields should be compared numerically, not lexicographically).

### 2. Index Maintenance on Upsert (`main.go:53-60`)

When a document is upserted, every existing index is updated: `addToIndex(index, fmt.Sprint(doc[field]), id)`. This is O(i) where i is the number of indexes. Note that we don't remove old index entries for this document ID — if a document's `dept` changes from "engineering" to "research", the old `indexes["dept"]["engineering"]` entry still contains the document ID. This is a known limitation in our implementation. Production systems track the old document and remove stale index entries, or use a write-then-clean pattern.

### 3. Full-Scan Fallback (`main.go:77-83`)

When no index exists for a queried field, `FindByField` falls back to iterating every document and comparing `fmt.Sprint(doc[field]) == fmt.Sprint(value)`. This is O(n) and linear in collection size. The log message (`main.go:77`) explicitly warns about this. Production document databases either (a) reject unindexed queries requiring a full scan, or (b) require an explicit `$natural` hint to acknowledge you're paying the scan cost. MongoDB's query planner will use an index if available and fall back to `COLLSCAN` only if no index matches — and it logs a warning when it does.

## Running the Demo

```bash
go run ./27-Document-Store/
```

The demo creates a document store with 4 user profiles, creates secondary indexes on `type`, `dept`, and `role`, then demonstrates indexed lookups (find all users by type, find engineers by department, find admins by role), and shows the full-scan fallback when querying an unindexed field.

## Further Reading

- **"MongoDB: The Definitive Guide"** — Kristina Chodorow and Michael Dirolf (O'Reilly). The authoritative book on MongoDB internals, covering the WiredTiger storage engine, index structures (B-tree indexes, compound indexes, geospatial indexes), and aggregation pipelines.
- **"Designing Data-Intensive Applications" Chapter 2** — Martin Kleppmann (O'Reilly, 2017). Compares document, relational, and graph data models. Explains the impedance mismatch between relational schemas and application objects, and when document stores excel (one-to-many relationships with tree-structured data).

---

*Part of the Design-With-TsGo system design curriculum*
