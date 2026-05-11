# Gossip Protocol

> **An epidemic/gossip protocol for eventually-consistent state propagation using versioned key-value pairs — O(nodes × state) convergence without a central coordinator.**

## The Problem It Solves

You're the platform engineer at a company running 500 microservice instances behind a global load balancer. Each instance needs to know the current feature flag configuration, rate limit thresholds, and circuit breaker states. You could put this data in a centralized Redis instance, but now Redis is a single point of failure and a bottleneck for 500 clients polling it every second. You could use etcd, but that requires every instance to maintain a persistent connection and introduces operational complexity. You need something dumber, simpler, and more resilient — something where any node can talk to any other node and the data just spreads like a rumor in a crowded room.

This is the gossip protocol. Modeled after the way infectious diseases spread through a population, gossip (also called *epidemic broadcast*) works by having each node periodically pick a random peer and exchange state. There is no leader, no election, no central authority. If one node learns a new configuration value, it eventually reaches every other node through a chain of peer-to-peer exchanges. The system degrades gracefully during network partitions — nodes within a partition converge, and when the partition heals, the next gossip round reconciles the two versions.

The critical design choice is *conflict resolution during merge*. Two nodes might have different values for the same key. Our implementation uses a simple version vector approach: each key carries a monotonically increasing version number. On merge, the higher version wins. This gives us *eventual consistency with last-writer-wins semantics* — good enough for configuration propagation, service discovery, and feature flag distribution.

## Architecture & Internals

```
┌───────────────────────────────────────────────────────────┐
│                    Gossip Protocol Topology                 │
│                                                           │
│    ┌──────┐     GossipTo()      ┌──────┐                  │
│    │Node-A│◄───────────────────►│Node-B│                  │
│    │      │  state = {          │      │                  │
│    │      │    flag: (en, v2)   │      │                  │
│    │      │    conn: (1000,v1)  │      │                  │
│    │      │  }                  │      │                  │
│    └──┬───┘                     └──┬───┘                  │
│       │                            │                      │
│       │    ┌──────┐    ┌──────┐    │                      │
│       └───►│Node-C│◄──►│Node-D│◄───┘                      │
│            │      │    │      │                           │
│            └──────┘    └──┬───┘                           │
│                           │                               │
│                        ┌──┴───┐                           │
│                        │Node-E│                           │
│                        └──────┘                           │
│                                                           │
│  GOSSIP FLOW (bidirectional):                             │
│  ┌────────┐    ┌──────────────┐    ┌────────┐            │
│  │ A sends│───►│ B merges A's │    │ B's    │            │
│  │ state  │    │ state into   │    │ state  │            │
│  │ to B   │    │ local (higher│    │ now    │            │
│  │        │    │ version wins)│    │ merged │            │
│  └────────┘    └──────────────┘    └────────┘            │
│                                                           │
│  ┌────────┐    ┌──────────────┐    ┌────────┐            │
│  │ B sends│───►│ A merges B's │    │ A's    │            │
│  │ state  │    │ state into   │    │ state  │            │
│  │ to A   │    │ local        │    │ merged │            │
│  └────────┘    └──────────────┘    └────────┘            │
│                                                           │
│  CONVERGENCE:                                             │
│  After O(log N) rounds of random pairings, all nodes      │
│  have the same state (with high probability).             │
└───────────────────────────────────────────────────────────┘
```

Each `GossipNode` (`main.go:22-25`) has an `ID` and a `state` map from string keys to `GossipValue` structs. A `GossipValue` (`main.go:14-17`) pairs a `Value` string with a `Version` integer — the version increments on every local `Set` call and resolves conflicts during gossip merges.

**On `Set`** (`main.go:36-41`): The node writes the key-value pair with a version one higher than the previous version for that key. This creates a total order of updates per node, which the merge logic uses to pick winners.

**On `GossipTo(peer)`** (`main.go:46-51`): This is the core of the protocol — a *bidirectional* state exchange. Node A calls `peer.merge(n.state)` to push its state into the peer, then `n.merge(peer.state)` to pull the peer's state. Both directions use the same merge logic: for each key, keep the entry with the highest version. After this single call, both nodes have the union of their knowledge with conflicts resolved.

**On `merge(remote)`** (`main.go:69-76`): For each key in the remote state, if the local node doesn't have that key, or the remote version is strictly greater than the local version, the local entry is overwritten. This is a commutative, associative, and idempotent operation — the mathematical properties that guarantee eventual convergence regardless of message order, duplicates, or timing.

## Production Use Cases

- **Apache Cassandra** — The internal gossip subsystem spreads node membership, token range assignments, and schema changes across the cluster. Every second, each node gossips with 1-3 randomly chosen peers, achieving cluster-wide convergence in seconds.
- **Consul's Serf** — HashiCorp's Serf library implements the SWIM (Scalable Weakly-consistent Infection-style Process Group Membership) protocol, a gossip variant optimized for membership failure detection. It's used by Consul, Nomad, and Vault.
- **Redis Cluster** — Redis Cluster nodes use a gossip protocol to exchange cluster topology information (which node owns which hash slots). There's no central metadata store — every node builds a complete picture of the cluster through peer gossip.
- **Amazon Dynamo** — Dynamo's failure detection uses a gossip-based protocol to disseminate node liveness information. Nodes gossip about which peers they've recently communicated with, building a probabilistic failure detector without a central monitor.
- **HashiCorp memberlist** — The `memberlist` Go library is a standalone implementation of SWIM gossip used by Consul, Nomad, and various open-source projects for cluster membership.

## When to Use It

| Scenario | Use Gossip? |
|----------|------------|
| You need to propagate configuration/state to 100+ nodes | **Yes** |
| You cannot afford a central point of failure for state distribution | **Yes** |
| Eventual consistency is acceptable (seconds of staleness is fine) | **Yes** |
| You need to tolerate network partitions and node churn | **Yes** |
| You need strong consistency (every read sees the latest write) | **No** — use Raft (40-Consensus) |
| Your data payload is large (> 1MB per key) | **No** — gossip bandwidth explodes with payload size |
| You need strict ordering guarantees | **No** — causality is not preserved across gossips |
| You have fewer than 5 nodes | **No** — just use a centralized store; gossip overhead isn't worth it |

**Alternatives**: **Centralized configuration stores** like etcd or Consul KV provide strong consistency and watches, but create a single point of failure in the availability sense (if quorum is lost). **Message queues** like Kafka provide durable ordered broadcast but require infrastructure. **Raft consensus** (40-Consensus) provides strong consistency with a leader, at the cost of lower write throughput under partitions. Gossip's superpower is that it works with zero infrastructure beyond the nodes themselves.

## Complexity / Behavior Characteristics

| Operation | Time Complexity | Space Complexity |
|-----------|----------------|------------------|
| Set (local write) | O(1) | O(1) additional per key |
| Get (local read) | O(1) | O(1) |
| GossipTo (one exchange) | O(s₁ + s₂) where sᵢ = state size of node i | O(s₁ + s₂) temporary during merge |
| Convergence (full cluster) | O(log N) rounds for high probability convergence | O(N × K) total where K = keys per node |

The convergence rate follows epidemic theory: in a population of N nodes where each node gossips with f random peers per round, the probability that all nodes know a piece of data reaches near-1 after approximately log(N)/log(f+1) rounds. For N=1000 and f=2, that's about 7 rounds.

## Implementation Deep Dive

**1. Bidirectional gossip ensures symmetric state convergence.** The `GossipTo` function at `main.go:46-51` does a two-way merge: `peer.merge(n.state)` then `n.merge(peer.state)`. This is critical. If gossip were unidirectional (only A pushes to B), data would flow in one direction per exchange, doubling convergence time. Bidirectional merges also ensure that after one gossip round between A and B, both nodes have identical state for the keys they share — they've reached *pairwise consistency*.

**2. Version-based conflict resolution (last-writer-wins).** The `merge` function at `main.go:69-76` uses a simple version comparison: `incoming.Version > local.Version`. This is the "last writer wins" (LWW) strategy used by Cassandra and Dynamo. It's not causally consistent — if two nodes concurrently set the same key, the one with the timing edge wins. For production use cases like feature flags and rate limits, this is perfectly acceptable. For use cases requiring causal ordering, you'd replace the version integer with a **vector clock** (34-Vector-Clocks), which can detect concurrent writes and surface conflicts for application-level resolution.

**3. Set always increments version — even if the value doesn't change.** At `main.go:38-39`, `Version: current.Version + 1` is computed even if the new value equals the old value. This is intentional: in a distributed system, the act of writing *at this point in time* has meaning. If node-A writes `flag=enabled` at version 1, then node-B writes `flag=enabled` at version 2 (even though the value is the same), node-B's version 2 wins during merge. This guarantees that the most recent write timestamp wins, not just the most recent value change.

## Running the Demo

```bash
go run ./33-Gossip-Protocol/
```

## Further Reading

- "Epidemic Algorithms for Replicated Database Maintenance" — Alan Demers et al. (1987). The foundational paper that introduced gossip protocols for distributed systems. *Proceedings of the 6th ACM Symposium on Principles of Distributed Computing.*
- "SWIM: Scalable Weakly-consistent Infection-style Process Group Membership Protocol" — Das, Gupta, and Motivala (2002). The paper behind HashiCorp's memberlist library, focusing on failure detection. *Proceedings of DSN 2002.*

---

*Part of the Design-With-TsGo system design curriculum*
