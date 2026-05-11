# Leader Election

> **The Bully Algorithm for distributed leader election — the highest-ID alive node automatically becomes the leader, with immediate re-election on failure or recovery.**

## The Problem It Solves

Imagine you're the infrastructure engineer at a fast-growing SaaS company. You have a 5-node PostgreSQL cluster behind Patroni for high availability. One of those nodes needs to be the *primary* — the one that accepts writes. If two nodes think they're the primary simultaneously, you get split-brain: two divergent copies of the database, unreconcilable data loss, and an on-call page at 3 AM. You need a deterministic, automatic way for the cluster to pick exactly one leader and for everyone else to agree on who it is.

The core challenge is that distributed nodes cannot trust a single external arbiter without introducing a new single point of failure. The system must be self-organizing: when nodes come online, when the leader crashes, or when a failed node recovers, the cluster must converge to a new single leader without human intervention. This is the leader election problem — one of the oldest and most fundamental challenges in distributed systems.

The Bully Algorithm (Garcia-Molina, 1982) solves this with a disarmingly simple rule: the node with the highest numeric ID among all alive nodes wins. A "bully" node with a higher ID simply asserts dominance. When a node detects the leader has failed, it triggers an election. When a node recovers, it triggers an election — because it might have a higher ID than the current leader and should rightfully reclaim the throne. This deterministic property makes the system trivially predictable: given a set of alive node IDs, you can compute the expected leader without running the algorithm.

## Architecture & Internals

```
┌──────────────────────────────────────────────────┐
│            BullyElectionCluster                   │
│                                                   │
│  alive: { 1: true, 2: true, 3: true }            │
│  roles: { 1: follower, 2: follower, 3: leader }  │
│  leader: 3                                        │
│                                                   │
│     Node 1 (follower)                             │
│       │                                           │
│     Node 2 (follower)                             │
│       │                                           │
│     Node 3 (leader)    ◄── Highest alive ID       │
│       │                                           │
│                                                   │
│  ELECT FLOW:                                      │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    │
│  │ Collect  │───►│  Sort    │───►│ Highest  │    │
│  │ alive IDs│    │  IDs     │    │ ID wins  │    │
│  └──────────┘    └──────────┘    └──────────┘    │
│                                                   │
│  FAIL FLOW:                                       │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    │
│  │ Mark node│───►│ Was it   │───►│ Run new  │    │
│  │ as dead  │    │ leader?  │    │ election │    │
│  └──────────┘    └──────────┘    └──────────┘    │
│                       │                           │
│                       ▼ No                        │
│                  ┌──────────┐                     │
│                  │ No-op    │                     │
│                  └──────────┘                     │
│                                                   │
│  RECOVER FLOW:                                    │
│  ┌──────────┐    ┌──────────┐                    │
│  │ Mark node│───►│ Always   │                    │
│  │ as alive │    │ elect    │                    │
│  └──────────┘    └──────────┘                    │
└──────────────────────────────────────────────────┘
```

The `BullyElectionCluster` (`main.go:15-19`) maintains two maps: `alive` tracks which nodes are operational (boolean map keyed by node ID), and `roles` maps each node to either `"leader"` or `"follower"`. The `leader` field caches the current leader's ID for O(1) access.

**On initialization** (`NewBullyElectionCluster`, `main.go:24-35`): All provided node IDs are marked alive and set as followers. An initial election is run immediately — this is crucial because a cluster must have a leader from the moment it boots.

**On failure** (`Fail`, `main.go:39-53`): The failed node is removed from the `alive` map. If the failed node was the current leader, a new election is triggered. If a follower failed, no election is needed — the leader remains unchanged. This conditional election avoids unnecessary churn.

**On recovery** (`Recover`, `main.go:58-63`): The node is re-inserted into `alive` and an election is *always* triggered. This is the "bully" behavior: a recovering node might have a higher ID than the current leader and must reclaim leadership. Without this, a node with ID 100 that recovers would remain a follower under node ID 3 permanently.

**The election itself** (`Elect`, `main.go:68-91`): All alive node IDs are collected into a slice, sorted in ascending order, and the maximum (last element) is chosen as the new leader. Every node's role is updated accordingly. The sort dominates the time complexity at O(n log n).

## Production Use Cases

- **MongoDB replica sets** — MongoDB uses a variant of the Bully algorithm with priority values and optime comparisons for primary election in replica sets. Nodes with higher priority and more up-to-date oplogs win elections.
- **Elasticsearch master election** — Elasticsearch uses the Zen Discovery module (and in 7.x+, the coordinated election system) based on a Bully-like algorithm where the master-eligible node with the lowest ID that joins first wins the election.
- **Apache ZooKeeper** — ZooKeeper uses Zab (ZooKeeper Atomic Broadcast) for leader election, which is a protocol influenced by the Bully concept — the node with the highest zxid (transaction ID) wins the leader election.
- **PostgreSQL Patroni** — Patroni uses etcd or Consul for leader election via distributed locks, but the conceptual model is the same: exactly one primary at all times, with automatic failover when the primary becomes unhealthy.
- **Kubernetes leader election (client-go)** — The `client-go` library provides a leader election mechanism using Kubernetes leases or configmaps. The pattern is used by kube-controller-manager and kube-scheduler to ensure exactly one active instance.

## When to Use It

| Scenario | Use Bully? |
|----------|-----------|
| Node IDs are naturally ordered (e.g., increasing instance numbers) | **Yes** |
| You need deterministic leadership — same inputs always yield same leader | **Yes** |
| Nodes can crash and recover unpredictably | **Yes** |
| You need sub-second failover in a small cluster (< 10 nodes) | **Yes** |
| You need fairness — lower-ID nodes should get a chance to lead | **No** — Bully favors high IDs |
| You have hundreds or thousands of nodes | **No** — use Raft or Paxos |
| You need to tolerate network partitions (not just crash failures) | **No** — Bully assumes a reliable network for the election itself |

**Alternatives**: **Raft consensus** (40-Consensus) provides leader election with stronger guarantees including log consistency and term-based ordering. **Paxos** is the theoretical foundation but notoriously hard to implement correctly. **Gossip-based election** (33-Gossip-Protocol) can work for very large clusters at the cost of slower convergence. The Bully algorithm's strength is its simplicity: you can implement it correctly in 30 lines of code and reason about its behavior exhaustively.

## Complexity / Behavior Characteristics

| Operation | Time Complexity | Trigger |
|-----------|----------------|---------|
| Elect | O(n log n) | Leader failure, node recovery, initial bootstrap |
| Fail (non-leader) | O(1) | Node crash, maintenance |
| Fail (leader) | O(n log n) | Leader crash |
| Recover | O(n log n) | Node rejoining cluster |
| GetLeader | O(1) | Any time |

The O(n log n) cost is from sorting the alive node IDs. For small clusters (3-7 nodes, which is the typical production deployment), this is negligible. For large clusters, switch to a consensus protocol that avoids full sorting.

## Implementation Deep Dive

**1. Sorted IDs determine the winner deterministically.** The `Elect` function at `main.go:68-91` collects all alive IDs, sorts them with `sort.Ints(ids)`, and picks `ids[len(ids)-1]` as the winner. The sort is essential — it makes the outcome deterministic and predictable. Without sorting, if you iterated a map directly, Go's map iteration order is randomized, and you'd get a different "highest" ID each time. The sort guarantees the same result every time for the same set of alive nodes.

**2. Conditional re-election avoids unnecessary churn.** In `Fail` at `main.go:47-49`, a new election is triggered *only* if the failed node was the leader (`if c.leader == nodeID`). A follower failure doesn't change who should be the leader, so no election is needed. This is a practical optimization that prevents the cluster from re-electimating every time any node goes down — which would cause brief periods of no leadership during the election window.

**3. Recovery always triggers re-election (the "bully" behavior).** In `Recover` at `main.go:58-63`, the node is marked alive and an election runs unconditionally. This is the defining characteristic of the Bully algorithm: a newly-joined or recovered node might have a higher ID than the current leader, and it must be given the chance to take over. If node 5 recovers while node 2 is leading, the re-election will correctly promote node 5. This is why the algorithm is called "Bully" — the higher-ID node bullies its way into leadership.

## Running the Demo

```bash
go run ./31-Leader-Election/
```

## Further Reading

- "Elections in a Distributed Computing System" — Hector Garcia-Molina (1982). The original Bully Algorithm paper. *IEEE Transactions on Computers, Vol. C-31, No. 1.*
- "In Search of an Understandable Consensus Algorithm" — Ongaro & Ousterhout (2014). The Raft paper, which provides the modern alternative to Bully with much stronger guarantees. *Proceedings of USENIX ATC 2014.*

---

*Part of the Design-With-TsGo system design curriculum*
