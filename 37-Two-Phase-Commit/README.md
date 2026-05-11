# Two-Phase Commit (2PC)

> **The atomic commit protocol — Prepare phase (vote) then Commit/Abort phase across all participants — guarantees all-or-nothing transaction completion across distributed resources.**

## The Problem It Solves

You're the database reliability engineer at a fintech company. A wire transfer involves: debiting $10,000 from the sender's account in PostgreSQL and crediting $10,000 to the recipient's account in a different PostgreSQL instance (different bank, different data center). If you debit the sender but the credit to the recipient fails (network outage, constraint violation, disk full), you've lost $10,000. If you credit the recipient but the debit fails, you've created $10,000 out of thin air. Both outcomes are catastrophically wrong. You need atomicity across two independent database systems — both operations must succeed, or neither must.

This is the atomic commitment problem. Two operations on independent resource managers must either both commit or both abort. You cannot do them sequentially with a try-catch — if the second operation fails after the first committed, you can't un-commit the first. You need a protocol that coordinates the commit decision across all participants, ensuring that either all commit or all abort, even in the face of network failures and participant crashes (during the protocol itself).

Two-Phase Commit (2PC) was introduced by Jim Gray in 1978 and remains the standard solution. The protocol has exactly two phases: Phase 1 (Prepare/Vote) where each participant decides whether it can commit, and Phase 2 (Commit/Abort) where the coordinator tells everyone the final decision. The key insight is that once a participant votes "Yes" in Phase 1, it *must* be able to commit — it cannot change its mind or unilaterally abort. If the coordinator crashes after collecting all "Yes" votes but before sending "Commit," the participants are *blocked* — they must wait for the coordinator to recover. This is the famous "blocking problem" of 2PC, and it's the reason many distributed systems prefer Paxos/Raft for replication and sagas for business transactions.

## Architecture & Internals

```
┌─────────────────────────────────────────────────────────────┐
│              Two-Phase Commit Protocol                       │
│                                                             │
│  Coordinator           Participant A       Participant B    │
│      │                      │                   │          │
│      │── Prepare(txn) ─────►│                   │          │
│      │                      │─── Lock resources  │          │
│      │◄─── Vote: YES ───────│                   │          │
│      │                      │                   │          │
│      │── Prepare(txn) ────────────────────────►│          │
│      │                      │                   │── Lock   │
│      │◄─── Vote: YES ──────────────────────────│          │
│      │                      │                   │          │
│      │  ╔═══════════════════╗                               │
│      │  ║ PHASE 1 COMPLETE ║  All voted YES                │
│      │  ╚═══════════════════╝                               │
│      │                      │                   │          │
│      │── Commit(txn) ──────►│                   │          │
│      │                      │─── Apply changes  │          │
│      │── Commit(txn) ────────────────────────►│          │
│      │                      │                   │── Apply  │
│      │                      │                   │          │
│      │  ╔═══════════════════╗                               │
│      │  ║    COMMITTED      ║                              │
│      │  ╚═══════════════════╝                               │
│                                                             │
│  FAILURE SCENARIO (Participant B votes NO):                 │
│                                                             │
│      │── Prepare(txn) ──────►│                   │          │
│      │◄─── Vote: YES ────────│                   │          │
│      │── Prepare(txn) ────────────────────────►│          │
│      │◄─── Vote: NO ───────────────────────────│          │
│      │                      │                   │          │
│      │── Abort(txn) ───────►│  (rollback A)     │          │
│      │── Abort(txn) ────────────────────────►│  (no-op)  │
│      │                      │                   │          │
│      │  ╔═══════════════════╗                               │
│      │  ║     ABORTED       ║                              │
│      │  ╚═══════════════════╝                               │
└─────────────────────────────────────────────────────────────┘
```

The `TwoPhaseCommitCoordinator` (`main.go:23`) holds a slice of `Participant` interfaces. Each `Participant` (`main.go:14-18`) must implement three methods: `Prepare(transactionID string) bool` (vote Yes/No), `Commit(transactionID string)` (apply), and `Abort(transactionID string)` (rollback).

**Phase 1 — Prepare** (`main.go:39-56`): The coordinator calls `Prepare` on each participant sequentially. If any participant returns `false` (votes NO), the coordinator immediately aborts all participants that already voted YES (`main.go:48-50`). The participants that haven't been asked yet are never prepared, so they don't need aborting. If all participants vote YES, the transaction moves to Phase 2.

**Phase 2 — Commit** (`main.go:58-65`): The coordinator calls `Commit` on each participant (all of which voted YES). After this point, the transaction is committed and cannot be rolled back.

The protocol has a critical window: between a participant's YES vote and the coordinator's Commit/Abort decision. If the coordinator crashes during this window, the participant is *blocked* — it promised to commit if asked, and it can't commit until asked, so it must wait. This is the fundamental limitation of 2PC.

## Production Use Cases

- **PostgreSQL distributed transactions (PREPARE TRANSACTION)** — PostgreSQL supports 2PC via the `PREPARE TRANSACTION 'txn_id'` SQL command. After preparing, the transaction is durable on disk. A transaction manager can then issue `COMMIT PREPARED 'txn_id'` or `ROLLBACK PREPARED 'txn_id'` at any later time. Used with external transaction managers like MSDTC or JTA.
- **MySQL XA transactions** — MySQL's InnoDB engine supports XA (eXtended Architecture) transactions, the standard 2PC interface defined by the X/Open group. `XA START`, `XA END`, `XA PREPARE`, `XA COMMIT`, and `XA ROLLBACK` implement the full 2PC lifecycle. Used by Java application servers with JTA.
- **Apache Kafka transactions** — Kafka's exactly-once semantics (EOS) use an internal 2PC protocol. When a producer writes to multiple partitions, the Kafka transaction coordinator runs 2PC to atomically commit or abort all partition writes. The `transactional.id` identifies the producer, and `initTransactions`, `beginTransaction`, `commitTransaction`, and `abortTransaction` map to the 2PC phases.
- **Java Transaction API (JTA)** — JTA is the standard Java interface for distributed transactions. Application servers like WildFly, WebLogic, and WebSphere use JTA to coordinate 2PC across multiple XA-compliant resources (databases, JMS queues, JCA adapters). `UserTransaction.begin()`, `.commit()`, and `.rollback()` drive the 2PC lifecycle.
- **Microsoft Distributed Transaction Coordinator (MSDTC)** — MSDTC is Windows' built-in distributed transaction manager. SQL Server, MSMQ, and other Windows services register as XA resource managers with MSDTC, which coordinates 2PC across them. It's the backbone of distributed transactions in .NET enterprise applications.

## When to Use It

| Scenario | Use 2PC? |
|----------|---------|
| You need immediate, guaranteed atomicity across 2-5 resources | **Yes** |
| All resources support XA or a prepare/commit interface | **Yes** |
| The transaction duration is short (< 1 second typically) | **Yes** |
| You can tolerate the blocking problem (coordinator is highly available) | **Yes** |
| The transaction spans services owned by different teams or with different databases | **No** — use Sagas (36-Distributed-Transaction) |
| You need high throughput (2PC adds 2+ network round trips per transaction) | **No** — use a single database or eventual consistency |
| Participants might be unavailable for seconds or minutes | **No** — blocking during coordinator failure is unacceptable |
| You have more than 10 participants | **No** — the failure probability and latency multiply with each participant |

**Alternatives**: **Saga pattern** (36-Distributed-Transaction) trades immediate atomicity for availability — no blocking, no locks held across services, but temporary inconsistency is visible. **Paxos/Raft** (40-Consensus) provides replication consensus within a single resource (like a replicated log) but doesn't coordinate across independent resources. **XA with heuristic completion** allows participants to make a unilateral decision after a timeout (commit or abort) if the coordinator is unreachable, breaking strict 2PC guarantees but avoiding indefinite blocking — used in practice by many JTA implementations.

## Complexity / Behavior Characteristics

| Operation | Time Complexity | Network Round Trips |
|-----------|----------------|---------------------|
| Execute (all YES) | O(n × T_prepare + n × T_commit) | 2 (prepare broadcast + commit broadcast) |
| Execute (any NO) | O(k × T_prepare + k × T_abort) for k prepared | 1 + abort for prepared nodes |
| Blocking Window | Duration of coordinator crash + recovery | Participants hold locks during this window |

The protocol requires minimum 2 network round trips (prepare + commit) for the happy path, plus 1 for the abort path. The prepare phase is sequential in this implementation but could be parallelized — the coordinator at `main.go:43-55` iterates participants in a loop, but each `Prepare` call could be a concurrent goroutine for lower latency at scale.

## Implementation Deep Dive

**1. Interface-based participant model for extensibility.** The `Participant` interface at `main.go:14-18` decouples the coordinator from resource-specific logic. The demo provides two implementations: `MemoryParticipant` (`main.go:70-89`) that always votes YES (simulating a healthy database), and `FailingParticipant` (`main.go:93-112`) that always votes NO with a reason (simulating a constraint violation like insufficient balance). In production, you'd implement this interface for PostgreSQL (`db.Exec("PREPARE TRANSACTION ...")`), Kafka (`producer.commitTransaction()`), or any other XA-compliant resource. The interface is the extensibility point.

**2. Abort only prepared participants.** In the failure path at `main.go:47-50`, the coordinator iterates `prepared` (the slice of participants that already voted YES) and calls `Abort` on each. It does NOT call abort on participants that haven't been asked yet — since they haven't prepared, they have nothing to roll back. This is correct but subtle: if we called abort on a participant that never prepared, it might interpret the abort as pertaining to a different transaction, leading to subtle bugs. By tracking who voted YES, we abort exactly the right set.

**3. Sequential execution simplifies reasoning but limits throughput.** The coordinator executes Prepare calls sequentially (`main.go:43-55` uses a simple `for` loop), not concurrently. This is a deliberate simplification for clarity. In a production 2PC coordinator (like a JTA transaction manager), prepares are issued concurrently to all participants, and the coordinator waits for all responses. The sequential version is easier to reason about and debug, but it means the total latency is the sum of individual prepare latencies. For a 3-participant transaction with 10ms per prepare, sequential costs 30ms while concurrent costs 10ms.

## Running the Demo

```bash
go run ./37-Two-Phase-Commit/
```

## Further Reading

- "Notes on Data Base Operating Systems" — Jim Gray (1978). The original technical report that introduced two-phase commit. *IBM Research Report RJ2188.* Reprinted in "Operating Systems: An Advanced Course" (Springer, 1979).
- "Principles of Transaction-Oriented Database Recovery" — Theo Haerder and Andreas Reuter (1983). The foundational paper on database recovery, including the formal specification of 2PC. *ACM Computing Surveys, Vol. 15, No. 4.*

---

*Part of the Design-With-TsGo system design curriculum*
