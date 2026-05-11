# Consensus (Raft)

> **The Raft consensus algorithm — leader election, log replication, and a committed state machine — provides strong consistency across a distributed cluster with an understandable, production-tested design.**

## The Problem It Solves

You're the site reliability engineer for a Kubernetes cluster running 500 production services. At the heart of your cluster sits etcd — a 5-node distributed key-value store that holds all cluster state: which pods are running, what services exist, network policies, secrets. Every deployment, every scheduler decision, every service discovery lookup reads from or writes to etcd. If the etcd nodes disagree about state — if node A thinks pod-nginx-7f8c9 is on worker-3 but node B thinks it's on worker-7 — Kubernetes will try to schedule conflicting operations, pods will get orphaned, and your cluster will degrade into chaos. You need **consensus**: every non-faulty node in the cluster must agree on the exact same sequence of operations, even in the face of network partitions, crashes, and slow nodes.

Consensus is the hardest problem in distributed systems, formalized by the FLP impossibility result (Fischer, Lynch, and Paterson, 1985): in an asynchronous system, it's impossible to guarantee consensus with even one faulty node. Raft (Ongaro & Ousterhout, 2014) sidesteps FLP by assuming bounded message delays in practice and using randomized election timeouts to break symmetry. It was designed to be *understandable* — its authors observed that Paxos, the classic consensus algorithm, was notoriously difficult to implement correctly, leading to subtle bugs in production systems. Raft decomposes consensus into three cleanly separated subproblems: **leader election** (picking exactly one leader per term), **log replication** (the leader propagates entries to followers), and **safety** (committing entries only when a majority has them).

The key insight: Raft guarantees that once a log entry is committed (replicated to a majority), it will *never* be overwritten or lost, even if the leader changes. This gives you a replicated state machine — a sequence of commands that every node applies in the same order, producing the same result.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────┐
│                  Raft Consensus — 5-Node Cluster               │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐    │
│  │ NODE ROLES: Leader(1), Followers(4), Candidates(0)    │    │
│  └──────────────────────────────────────────────────────┘    │
│                                                               │
│  LEADER ELECTION:                                             │
│  ┌──────────┐  inc term,      ┌──────────┐                   │
│  │ Follower │  vote for self   │Candidate │                   │
│  │          │────────────────►│          │                   │
│  │ (timeout)│                 │ Requests │                   │
│  └──────────┘                 │  votes   │                   │
│                               └────┬─────┘                   │
│                                    │                         │
│                    ┌───────────────┼───────────────┐         │
│                    ▼               ▼               ▼         │
│              ┌──────────┐   ┌──────────┐   ┌──────────┐     │
│              │Follower A│   │Follower B│   │Follower C│     │
│              │ Vote:YES │   │ Vote:YES │   │ Vote:NO  │     │
│              └──────────┘   └──────────┘   └──────────┘     │
│                    │               │                         │
│                    └───────────────┘                         │
│                            │                                 │
│                    ┌───────▼────────┐                        │
│                    │ VOTES > N/2 ? │                        │
│                    │   3 > 2.5 ✓   │                        │
│                    └───────┬────────┘                        │
│                            ▼                                 │
│                      ┌──────────┐                            │
│                      │  LEADER  │                            │
│                      └──────────┘                            │
│                                                               │
│  LOG REPLICATION:                                             │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐               │
│  │ Leader   │───►│Follower 1│───►│Follower 2│               │
│  │ Appends  │    │ Replicate│    │ Replicate│               │
│  │ entry    │    │ to log   │    │ to log   │               │
│  └──────────┘    └──────────┘    └──────────┘               │
│       │                                                      │
│       │ Quorum reached (3/5 nodes have entry)                │
│       ▼                                                      │
│  ┌──────────┐                                                │
│  │ COMMIT   │  Entry is now durable and visible              │
│  │ entry    │                                                │
│  └──────────┘                                                │
│                                                               │
│  STATE MACHINE:                                               │
│  Committed entries applied in log order → all nodes agree    │
│  CommittedCommands() returns the ordered list of committed   │
│  commands — this is the "state" the cluster has consensus on.│
└──────────────────────────────────────────────────────────────┘
```

The implementation models Raft's core concepts with three Go types:

**`RaftNode`** (`main.go:35-42`) represents a single node with fields for `ID`, `Role` (Follower/Candidate/Leader), `Term` (monotonic integer, increases on each election attempt), `VotedFor` (which node this node voted for in the current term), `Log` (the replicated log entries), and `CommitIndex` (the index of the last committed entry).

**`RaftCluster`** (`main.go:46-49`) manages a collection of nodes and a `leaderID`. The cluster provides the two fundamental Raft operations: `Elect` (leader election) and `Append` (log replication).

**`LogEntry`** (`main.go:28-31`) is a tuple of `Term` (when the entry was created) and `Command` (the actual data to apply to the state machine).

**On `Elect(candidateID)`** (`main.go:69-116`): The candidate increments its term, votes for itself, and requests votes from every peer. A peer grants its vote if the candidate's term ≥ the peer's term *and* the peer hasn't already voted for someone else this term (`main.go:87`). If the candidate receives votes from a majority of nodes (`votes > len(c.nodes)/2`), it becomes leader. This is Raft's election restriction: a candidate must have an up-to-date log (implied by term comparison in this simplified model) and must win a majority.

**On `Append(command)`** (`main.go:122-155`): The leader appends the entry to its own log, then replicates it to all followers by appending directly to their logs. If the entry is replicated to a majority (`replicated > len(c.nodes)/2`), the leader advances the commit index on all nodes. This is Raft's log commitment rule: an entry is committed once a majority of nodes have it in their logs.

**On `CommittedCommands()`** (`main.go:160-172`): Returns all commands up to and including the commit index, in log order. This represents the agreed-upon state of the replicated state machine.

## Production Use Cases

- **etcd (Kubernetes)** — etcd is the definitive Raft implementation in production. It stores all Kubernetes cluster state (pods, services, configmaps, secrets) and uses the Raft protocol for consensus across 3-5 node clusters. Every `kubectl apply` you run eventually lands as a Raft log entry in etcd. etcd's Raft library is also used by CockroachDB and TiKV.
- **HashiCorp Consul** — Consul uses Raft for its consistent key-value store, service catalog, and health check state. The Raft protocol ensures that all Consul servers agree on which services are healthy, which KV entries exist, and which nodes are in the cluster.
- **TiKV** — TiKV is a distributed transactional key-value store used by TiDB. It uses the etcd Raft library to replicate data across multiple nodes. Each region (range of keys) forms its own Raft group, allowing independent consensus for different data ranges.
- **CockroachDB** — CockroachDB uses Raft for replication at the range level. Each range (roughly 64 MB of data) is replicated across 3+ nodes using Raft, with the Raft leader handling all reads and writes for that range. This provides serializable isolation across a geographically distributed database.
- **NATS JetStream** — NATS JetStream, the persistence layer for NATS messaging, uses Raft for its clustering layer. Raft ensures that stream metadata and consumer offsets are consistently replicated across JetStream cluster nodes, providing fault-tolerant messaging.

## When to Use It

| Scenario | Use Raft? |
|----------|----------|
| You need strong consistency (all nodes see the same data at the same logical point) | **Yes** |
| You need a replicated state machine (log → apply → identical result) | **Yes** |
| You need automatic leader election with safe failover | **Yes** |
| Your cluster is 3-7 nodes (Raft's sweet spot) | **Yes** |
| You need high write throughput (100K+ writes/second) | **No** — Raft is bottlenecked by the leader's ability to replicate |
| You need to survive > N/2 node failures (you can't, that's the FLP result) | **No** — Raft tolerates up to floor((N-1)/2) failures |
| You have 50+ nodes | **No** — Raft's heartbeat overhead and log replication scale poorly |

**Alternatives**: **Paxos** (Multi-Paxos, EPaxos) predates Raft and offers similar guarantees but is substantially harder to implement correctly. **Viewstamped Replication** is a consensus protocol similar to Raft but with a different leader election mechanism — used in some storage systems. **Gossip protocols** (33-Gossip-Protocol) provide eventual consistency with much higher throughput and no leader bottleneck, at the cost of no strong consistency guarantees. **Byzantine Fault Tolerant (BFT)** consensus (like PBFT or HotStuff) tolerates malicious nodes, not just crash failures — used in blockchain systems but with higher overhead (3f+1 nodes vs Raft's 2f+1).

## Complexity / Behavior Characteristics

| Operation | Time Complexity | Network Rounds |
|-----------|----------------|---------------|
| Leader Election | O(N) | 1 (request + response votes) |
| Log Append (committed) | O(N) to replicate | 1 (append to followers) |
| Log Commit | O(N) to advance commit index | 0 (purely local after majority replicated) |
| CommittedCommands | O(C) where C = commit index | 0 (local read) |

Raft's performance is dominated by the leader's ability to replicate log entries to followers. In the minimum configuration (3 nodes), each committed write requires the leader to send the entry to at least 1 other node (2 nodes = majority of 3). In a 5-node cluster, each write requires replication to at least 2 followers (3 nodes = majority of 5).

## Implementation Deep Dive

**1. Term-based voting with the "voted for this term" constraint.** In `Elect` at `main.go:86-91`, a peer grants its vote only if `candidate.Term >= peer.Term` and `peer.VotedFor == -1 || peer.VotedFor == candidate.ID`. The two conditions together ensure: (a) the peer hasn't seen a higher term (which would mean another leader is active or another election is in progress), and (b) the peer hasn't already voted for someone else *in this term*. Once a node votes in a term, it cannot change its vote — this is Raft's "one vote per term" rule that prevents two leaders from being elected in the same term.

**2. Quorum commit requires strict majority (> N/2, not >=).** At `main.go:102`, the commit condition is `votes > len(c.nodes)/2`. For a 5-node cluster, that means 3 votes (5/2 = 2, 3 > 2). For a 3-node cluster, 2 votes. The strict majority (>) rather than half-or-more (>=) is the mathematically correct formulation: in a cluster of 2f+1 nodes, you need f+1 votes to survive f failures. If you used >=, a 4-node cluster would commit with 2 votes — but 2 votes out of 4 means the other 2 could also form a majority and commit conflicting entries (split-brain). With strict >, any two majorities must intersect in at least one node, which prevents conflicting commits.

**3. The commit index advances monotonically across all nodes.** In `Append` at `main.go:144-148`, when a quorum is reached, `node.CommitIndex = index` is set on every node in the cluster. This represents Raft's commitment rule: once an entry is replicated to a majority, the leader declares it committed and broadcasts the commit index. The commit index never decreases — it's a monotonic property that ensures once a command is visible to the state machine, it stays visible. The `CommittedCommands()` function at `main.go:160-172` reads up to the commit index, guaranteeing all nodes that have applied to that index see the same commands in the same order.

## Running the Demo

```bash
go run ./40-Consensus/
```

## Further Reading

- "In Search of an Understandable Consensus Algorithm" — Diego Ongaro and John Ousterhout (2014). The Raft paper that changed the distributed systems landscape by making consensus accessible. Required reading for any backend engineer. *Proceedings of USENIX ATC 2014.*
- The Raft Consensus Algorithm website — https://raft.github.io/ — Interactive visualizations of leader election, log replication, and membership changes. Essential for building intuition about how Raft behaves under network partitions.

---

*Part of the Design-With-TsGo system design curriculum*
