# Replication (Primary-Replica)

> **Primary-replica replication — synchronous writes to all replicas with read scale-out and automatic failover promotion — provides data durability and read scalability.**

## The Problem It Solves

You're the infrastructure lead for a growing SaaS product. Your single PostgreSQL instance has been a faithful workhorse, but last Tuesday it went down for 4 hours due to a disk failure. All 50,000 of your users saw "Service Unavailable." Your CEO wants 99.99% uptime (less than 53 minutes of downtime per year), and your CTO wants read queries that currently spike at 2,000 QPS to be spread across multiple machines so the primary doesn't buckle during traffic peaks. You need replication — multiple copies of your data on different machines, kept in sync.

The primary-replica (also called master-slave or leader-follower) replication model is the most battle-tested approach. One node is the **primary** — it accepts all writes. Every write is synchronously copied to one or more **replicas** (also called secondaries or standbys). Read queries can be served by either the primary or any replica, scaling read throughput horizontally. If the primary fails, one of the replicas is promoted to become the new primary — this is **failover**. The replicas serve dual purpose: they provide data durability (if the primary's disk dies, data lives on replicas) and read scalability (add more replicas to handle more read traffic).

The fundamental tradeoff is between consistency and latency. *Synchronous* replication (the primary waits for replicas to acknowledge the write before responding to the client) guarantees no data loss on failover, but every write incurs the network round-trip to all replicas. *Asynchronous* replication (the primary writes and immediately responds, replicas catch up later) has lower latency but can lose committed writes if the primary fails before replication completes. This implementation uses synchronous replication — every `Write` blocks until all replicas have the data — which is appropriate for configuration stores and metadata systems where consistency trumps write throughput.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────┐
│                Primary-Replica Replication                     │
│                                                               │
│                      ┌──────────┐                             │
│              ┌──────►│ Replica 0│  (read scale-out)           │
│              │       └──────────┘                             │
│              │                                                │
│  ┌────────┐  │       ┌──────────┐                             │
│  │Primary │──┼──────►│ Replica 1│  (hot standby)             │
│  │(writes)│  │       └──────────┘                             │
│  └────────┘  │                                                │
│              │       ┌──────────┐                             │
│              └──────►│ Replica 2│  (failover target)          │
│                      └──────────┘                             │
│                                                               │
│  WRITE FLOW (synchronous):                                    │
│  ┌──────────┐    ┌──────────┐    ┌──────────────┐            │
│  │ Client   │───►│ Primary  │───►│ Replicate to │            │
│  │ Write(k,v)│   │ writes   │    │ ALL replicas │            │
│  └──────────┘    │ locally  │    └──────┬───────┘            │
│                  └──────────┘           │                     │
│                                         ▼                     │
│                                  ┌──────────────┐            │
│                                  │ Ack to client│            │
│                                  │ (after all   │            │
│                                  │  replicas)   │            │
│                                  └──────────────┘            │
│                                                               │
│  READ FLOW:                                                   │
│  ┌──────────┐    ┌──────────────┐    ┌──────────┐            │
│  │ Client   │───►│ Choose source│───►│ Primary  │            │
│  │ Read(key)│    │ (primary or  │    │ OR       │            │
│  └──────────┘    │  replica-0)  │    │ Replica-0│            │
│                  └──────────────┘    └──────────┘            │
│                                                               │
│  FAILOVER FLOW:                                               │
│  ┌──────────┐    ┌──────────────┐    ┌──────────┐            │
│  │ Primary  │    │ Promote      │    │ New      │            │
│  │ crashes  │───►│ replica-2 to │───►│ primary  │            │
│  │          │    │ primary      │    │ elected  │            │
│  └──────────┘    └──────────────┘    └──────────┘            │
└──────────────────────────────────────────────────────────────┘
```

The `PrimaryReplicaStore` (`main.go:14-17`) maintains a `primary` map and a slice of `replicas` (each a map). The design is deliberately simple: each replica is an independent Go map, and writes are fanned out manually.

**On `Write(key, value)`** (`main.go:33-41`): The primary writes to its own map first (`s.primary[key] = value`), then iterates every replica map and writes the same key-value pair (`main.go:36-38`). All writes happen synchronously in a loop — the `Write` call doesn't return until every replica has the data. This is synchronous replication. If a replica were unavailable, `Write` would hang or fail depending on timeout behavior (this demo uses in-memory maps so it never fails, but a production system would need timeouts and error handling).

**On `Read(key, fromReplica bool)`** (`main.go:46-63`): If `fromReplica` is true and replicas exist, the read is served from `replicas[0]`. Otherwise, it's served from the primary. This gives clients the choice: reads that need the absolute latest data go to the primary; reads that can tolerate slight staleness (or just want to offload work from the primary) go to a replica.

**On `Failover(replicaIndex)`** (`main.go:69-82`): The specified replica's entire map is deep-copied to a new primary map, effectively promoting it. The old primary data is discarded. This simulates a clean failover where the promoted replica has been fully caught up (if a replica were behind, you'd lose the data that hadn't been replicated yet — the fundamental tradeoff of asynchronous replication).

## Production Use Cases

- **PostgreSQL streaming replication** — PostgreSQL's built-in replication streams WAL (Write-Ahead Log) records from the primary to replicas. Supports both synchronous (`synchronous_commit = on`) and asynchronous modes. Amazon RDS, Cloud SQL, and virtually every managed Postgres service use streaming replication for high availability.
- **MySQL replication** — MySQL supports three replication modes: asynchronous (default), semi-synchronous (primary waits for at least one replica to acknowledge), and Group Replication (a Paxos-based variant). Used by GitHub, Facebook, and YouTube for read scaling and disaster recovery.
- **Redis replication** — Redis replicas asynchronously replicate the primary's dataset. Combined with Redis Sentinel for automatic failover detection and promotion, this is the standard Redis high-availability architecture. The `REPLICAOF` command reconfigures a node's replication topology.
- **MongoDB replica sets** — MongoDB's replica set is a group of mongod processes that maintain the same data set. The primary receives all writes, and secondaries replicate the primary's oplog (operation log). Automatic failover occurs via an election among the secondaries. Used by eBay, MetLife, and countless other enterprises.
- **Kafka partition replication** — Each Kafka topic partition has one leader replica and N follower replicas (controlled by `replication.factor`). All produces and consumes go through the leader, and followers fetch from the leader. If the leader fails, an in-sync replica is elected as the new leader. This is the foundation of Kafka's durability guarantees.

## When to Use It

| Scenario | Use Primary-Replica? |
|----------|---------------------|
| Read throughput exceeds what a single machine can handle | **Yes** — scale reads horizontally |
| You need data durability against single-machine failures | **Yes** |
| Your write throughput is moderate (thousands of writes/second, not millions) | **Yes** |
| You need disaster recovery (replica in a different data center) | **Yes** |
| You need write scalability (more machines = more write throughput) | **No** — use sharding (39-Sharding) or multi-master |
| You need strong consistency for reads from replicas | **No** — replicas are inherently slightly behind |
| You can't tolerate any data loss on failover (use synchronous replication) | **Yes** — but accept the latency penalty |

**Alternatives**: **Multi-master replication** (also called multi-primary or active-active) allows writes on any node with conflict resolution (CRDTs or last-writer-wins), providing write scalability at the cost of consistency complexity. **Sharding** (39-Sharding) partitions data across nodes so each shard handles a subset — combined with replication, this gives both read and write scalability (each shard is a primary-replica pair). **Quorum replication** (used by DynamoDB and Cassandra) requires only a majority of replicas to acknowledge writes, trading some read consistency for lower write latency and better partition tolerance.

## Complexity / Behavior Characteristics

| Operation | Time Complexity | Network Cost |
|-----------|----------------|-------------|
| Write | O(R) where R = number of replicas | R fan-out writes |
| Read (primary) | O(1) | 1 round trip |
| Read (replica) | O(1) | 1 round trip to a less-loaded node |
| Failover | O(D × R) where D = keys in replica, R = replicas | Local operation (copy) |

The Write cost is O(R) because it fans out to every replica in a loop (`main.go:36-38`). For 3 replicas, that's 3 additional map writes — negligible. For 100 replicas, that's O(100) per write, which is why production systems typically use a replication tree (primary → a few replicas, which themselves replicate to more replicas). The failover cost is O(D) where D is the size of the promoted replica's dataset — a full copy of the map.

## Implementation Deep Dive

**1. Synchronous fan-out writes guarantee consistency at the cost of latency.** The `Write` function at `main.go:33-41` does `s.primary[key] = value` followed by a loop over every replica. This means the writer blocks until all replicas have the data. This is synchronous replication — also called "all-replicas" or "quorum N/N" consistency. The advantage: if the primary crashes after `Write` returns, every replica has the data, so no writes are lost during failover. The disadvantage: the slowest replica determines write latency. In production, you'd typically set a timeout and report the replica as lagging, degrading to asynchronous for that replica.

**2. Read-from-replica option enables read scale-out.** The `Read` function at `main.go:46-63` accepts a `fromReplica bool` parameter. When true, reads are served from `replicas[0]`. This simple API enables application-level read routing: performance-sensitive reads (like serving a web page) go to a replica to offload the primary, while correctness-sensitive reads (like checking a balance before allowing a transfer) go to the primary. In production, you'd use a load balancer or connection pool (like PgBouncer or ProxySQL) to distribute reads across all replicas, not just the first one, but the principle is identical.

**3. Failover deep-copies data to a clean primary.** The `Failover` function at `main.go:69-82` creates a fresh primary map and copies every key from the promoted replica. It doesn't just swap pointers — it deep-copies. This is important for correctness: the old primary map might still be referenced elsewhere in the program, and sharing the replica's map directly would create aliasing bugs. By deep-copying, the new primary is independent. In production, the promoted replica would continue serving reads using its existing data, and the promotion is a metadata change (update the consensus store, redirect clients), not a data copy.

## Running the Demo

```bash
go run ./38-Replication/
```

## Further Reading

- "Dynamo: Amazon's Highly Available Key-value Store" — DeCandia et al. (2007). Describes Amazon's quorum-based replication strategy that trades consistency for availability. The contrast with primary-replica is instructive. *Proceedings of SOSP 2007.*
- PostgreSQL streaming replication documentation — https://www.postgresql.org/docs/current/warm-standby.html — The authoritative guide to production PostgreSQL replication, including synchronous commit, cascading replication, and failover procedures.

---

*Part of the Design-With-TsGo system design curriculum*
