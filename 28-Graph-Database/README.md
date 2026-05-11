# Graph Database

> **A database where relationships are first-class citizens — nodes connected by edges, supporting path-finding, pattern matching, and graph algorithms that would be prohibitively expensive in SQL.**

## The Problem It Solves

You're building LinkedIn's "People You May Know" feature. The data model is inherently a graph: users are nodes, and connections (friends, colleagues, classmates) are edges. The core query is: "Given Alice, find all users who are friends-of-friends but not already connected, ranked by mutual connections." In SQL, this is a recursive CTE with multiple JOINs — and for a user with 500 friends, each of whom has 500 friends, that's a 250,000-row intermediate result set, just for one user. Multiply by 800 million users, and you're computing trillions of JOINs in real time.

The problem is that relational databases model relationships implicitly through foreign keys. Finding a friend-of-a-friend requires joining the `friendships` table with itself, once per hop. A 3-hop query ("friends of friends of friends") requires 3 self-JOINs. A 6-hop query (like "six degrees of Kevin Bacon") is practically impossible. This is the *join bomb*: performance degrades exponentially with path length.

A graph database makes relationships explicit, first-class entities. Each node stores an adjacency list of edges to neighboring nodes — a direct pointer rather than a lookup through a foreign key. BFS (Breadth-First Search) from a starting node traverses edges in O(V + E) time, exactly the number of nodes and edges visited, regardless of how many nodes exist in the graph. Shortest-path queries that would require 6 self-JOINs in SQL are a single BFS traversal in a graph database. This is why social networks, fraud detection systems, and knowledge graphs all use graph databases: relationships *are* the data.

## Architecture & Internals

```
                  GRAPH STRUCTURE
    ┌─────────────────────────────────────────────────┐
    │                                                 │
    │  Nodes (with properties):                       │
    │                                                 │
    │  alice ────── bob ────── charlie ────── diana   │
    │  {London}     {Paris}   {Berlin}      {Tokyo}   │
    │    │                                  │         │
    │    │                                  │         │
    │    └──────── eve ──────────────────────┘         │
    │            {New York}                            │
    │                                                 │
    │  frank (isolated)                               │
    │  {Sydney}                                        │
    │                                                 │
    └─────────────────────────────────────────────────┘

           ADJACENCY LIST REPRESENTATION
    ┌─────────────────────────────────────────────────┐
    │                                                 │
    │  edges: map[string]map[string]bool              │
    │                                                 │
    │  "alice"   → {"bob": true, "eve": true}         │
    │  "bob"     → {"alice": true, "charlie": true}   │
    │  "charlie" → {"bob": true, "diana": true,       │
    │               "eve": true}                       │
    │  "diana"   → {"charlie": true}                   │
    │  "eve"     → {"alice": true, "charlie": true}   │
    │  "frank"   → {}  (isolated)                      │
    │                                                 │
    └─────────────────────────────────────────────────┘

                  BFS SHORTEST PATH
    ShortestPath("alice", "diana")
    ┌─────────────────────────────────────────────────┐
    │                                                 │
    │  Queue: [[alice]]                               │
    │  Step 1: dequeue [alice], check neighbors       │
    │    bob (unseen), eve (unseen)                   │
    │    Queue: [[alice,bob], [alice,eve]]            │
    │  Step 2: dequeue [alice,bob], check neighbors   │
    │    charlie (unseen)                             │
    │    Queue: [[alice,eve], [alice,bob,charlie]]    │
    │  Step 3: dequeue [alice,eve], check neighbors   │
    │    charlie (already seen, skip)                 │
    │    Queue: [[alice,bob,charlie]]                 │
    │  Step 4: dequeue [alice,bob,charlie]            │
    │    Check diana → MATCH!                         │
    │    Append diana → [alice,bob,charlie,diana]     │
    │    Return: [alice, bob, charlie, diana]          │
    │      (3 hops, path length 4)                    │
    │                                                 │
    └─────────────────────────────────────────────────┘
```

**Key structures:**

- `GraphDatabase.nodes` (`main.go:17`): A map from node ID (string) to property map (`map[string]string`). Properties are simple key-value pairs, but production graph databases support typed properties (integers, booleans, dates, lists) and property indexes for filtering (e.g., "find all nodes where `city = 'London'`").
- `GraphDatabase.edges` (`main.go:18`): An adjacency list where each node maps to a set of neighbor IDs (`map[string]bool`). The `bool` value is unused — `map[string]struct{}` would be more memory-efficient but less idiomatic in Go for read/write patterns. For bidirectional edges, both `edges[from][to]` and `edges[to][from]` are set.

## Production Use Cases

- **Neo4j**: The most popular graph database. Uses the Cypher query language (`MATCH (a)-[:FRIEND]->(b)-[:FRIEND]->(c) RETURN c`). Powers fraud detection at banks (UBS, Citibank), recommendation engines at eBay, and the International Consortium of Investigative Journalists' Panama Papers investigation.
- **Amazon Neptune**: AWS's fully managed graph database supporting both Property Graph (Gremlin, openCypher) and RDF (SPARQL). Used by Siemens for industrial IoT knowledge graphs and Intuit for identity resolution.
- **JanusGraph**: An open-source distributed graph database backed by Cassandra, HBase, or Bigtable. Used by Uber for their entity graph and by large enterprises for master data management.
- **Facebook TAO (The Associations and Objects)**: Facebook's social graph store. Handles billions of nodes and trillions of edges, serving the Friends, Likes, and Comments you see. Read-dominant workload with cache-follower architecture.
- **LinkedIn Knowledge Graph**: Powers LinkedIn's entity understanding — mapping people, companies, skills, and jobs into a unified graph. Enables features like "People also viewed" and skill endorsements.

## When to Use It

| Use Graph Database when... | Don't use when... |
|---|---|
| Relationships are as important as the data itself | Your data has no relationships (simple key-value or documents) |
| Queries involve variable-length paths (friend-of-friend, supply chain) | Queries are mostly point lookups with fixed depth |
| Your domain model naturally looks like a whiteboard drawing with boxes and arrows | All queries can be expressed as simple foreign-key JOINs |
| You need to find patterns in connections (fraud rings, influencer networks) | You need full-text search or analytical aggregation |
| Path-finding is a core feature (shortest route, dependency chain) | Your workload is write-heavy with simple data shapes |

**Graph vs. SQL for relationships**: A SQL query for "friends of friends" is a self-JOIN. For 3 hops, it's 3 self-JOINs. Each additional hop multiplies the query complexity. A graph database's BFS is O(V + E) regardless of depth — the cost is proportional to how many nodes/edges you actually traverse, not the total graph size. For deep traversals (5+ hops), graph databases are orders of magnitude faster.

**Property graphs vs. RDF**: Property graphs (Neo4j, our implementation) attach key-value properties to nodes and edges. RDF (Resource Description Framework) uses subject-predicate-object triples and is better for semantic reasoning and linked data (DBpedia, Wikidata). If you care about inference ("If X works_at Y, and Y located_in Z, then X works_in Z"), use RDF. For application-facing graphs, use property graphs.

## Complexity Analysis

| Operation | Time | Space |
|---|---|---|
| AddNode | O(1) | O(properties) |
| AddEdge | O(1) | O(1) per edge |
| ShortestPath (BFS) | O(V + E) | O(V) for visited set + queue |
| Space (total) | O(V + E) | O(V × avg properties + E) |

BFS is optimal for unweighted shortest-path finding. For weighted edges (e.g., road distances), you'd need Dijkstra's algorithm (O((V + E) log V) with a priority queue) — our implementation doesn't support edge weights.

## Implementation Deep Dive

### 1. BFS Shortest Path (`main.go:68-98`)

`ShortestPath` uses classic Breadth-First Search with a twist: it stores entire paths in the queue rather than just nodes. Each queue entry is a `[]string` representing the path from start to the current node. When a neighbor is discovered, a new path is created by copying the current path and appending the neighbor (`append(append([]string{}, path...), next)`). The deep copy is necessary because slices are reference types — without it, all paths in the queue would point to the same backing array and get corrupted. When the target node is dequeued, the full path is returned immediately — since BFS explores in order of increasing path length, the first time we reach the target is guaranteed to be the shortest path.

### 2. Bidirectional Edge Support (`main.go:47-61`)

`AddEdge` accepts a `directed` boolean. When `directed=false`, it creates edges in both directions: `g.edges[from][to] = true` and `g.edges[to][from] = true`. This models undirected relationships like friendships (if Alice is friends with Bob, Bob is friends with Alice). When `directed=true`, only one edge is created — modeling relationships like "follows" (Alice follows Bob doesn't mean Bob follows Alice) or "parent_of." The auto-creation of missing nodes (`main.go:49-56`) ensures the graph is always consistent: an edge can reference a node before it's explicitly created.

### 3. Property Storage (`main.go:16,34-41`)

Each node stores a `map[string]string` of properties. This is deliberately simple — production graph databases support typed properties and property indexes. For example, Neo4j can index `city` so that `MATCH (u:User {city: 'London'})` is O(1) rather than a full node scan. Our implementation stores properties on nodes but doesn't index them; filtering by property would require scanning all nodes. This is a reasonable tradeoff for a demonstration that focuses on graph traversal rather than property-based search.

## Running the Demo

```bash
go run ./28-Graph-Database/
```

The demo builds a social graph with 6 users (including one isolated user), creates 5 bidirectional friendship edges, then runs BFS shortest-path queries: direct friends (1 hop), friend-of-friend paths (2-3 hops), isolated node (no path), and non-existent node (no path).

## Further Reading

- **"TAO: Facebook's Distributed Data Store for the Social Graph"** — Nathan Bronson et al. (USENIX ATC, 2013). Describes Facebook's graph store serving billions of reads/second. Key insights: read-dominant workload, cache-follower architecture, and async replication for writes.
- **Neo4j Graph Algorithms Documentation (neo4j.com/docs/graph-data-science)** — Comprehensive guide to graph algorithms (PageRank, community detection, centrality) implemented on top of the property graph model. Explains how graph-native storage (adjacency lists) achieves constant-time neighbor traversal.

---

*Part of the Design-With-TsGo system design curriculum*
