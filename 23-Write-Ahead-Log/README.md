# Write-Ahead Log (WAL)

> **An append-only log of all mutations, checksummed for integrity, that guarantees no committed data is lost after a crash — the durability foundation of every database.**

## The Problem It Solves

It's 3 AM. Your pager goes off — the payment service is down. You SSH into the server and discover it rebooted after a kernel panic. The database process is back up, but customer transaction data from the last 30 seconds is gone. Why? Because databases buffer writes in memory for performance. Those in-memory buffers were lost when the process died. The $15,000 in-flight transactions? They're in limbo.

This is the durability problem. Without a persistence mechanism, a crash means losing all unflushed data. The naive fix — `fsync()` after every write — kills throughput (disk syncs are ~1ms each, limiting you to 1000 writes/second). The Write-Ahead Log (WAL) solves this: before any mutation touches the main data store, an entry describing the mutation is appended to a sequential log file. Because appending is fast (sequential I/O, no seeks), you can batch many writes into a single `fsync()`. After a crash, you replay the WAL from the last checkpoint to rebuild the exact pre-crash state.

The "write-ahead" name is literal: the log is written *before* the data. This is the WAL's single invariant — never modify the database without first logging the modification. If the database crashes between logging and applying, replay restores it. If it crashes during the log write itself, the partial entry is detected via checksums and discarded. At worst, you lose only the in-flight write that was being logged — and you know about it because the checksum fails.

## Architecture & Internals

```
                      NORMAL OPERATION
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  Client: Put("balance:42", "120.50")                │
    │     │                                               │
    │     ├──1. WAL.Append("put", "balance:42", "120.50") │
    │     │      │  Calculate checksum                    │
    │     │      │  Assign LSN (Log Sequence Number)      │
    │     │      ▼                                        │
    │     │   ┌─────────────────────────────────────┐     │
    │     │   │ WAL (append-only file)              │     │
    │     │   │ ┌──────┬─────┬──────────┬──────────┐│     │
    │     │   │ │ LSN  │ Op  │   Key    │  Value   ││     │
    │     │   │ ├──────┼─────┼──────────┼──────────┤│     │
    │     │   │ │  1   │ put │session:..│ active   ││     │
    │     │   │ │  2   │ put │user:42   │ alice    ││     │
    │     │   │ │  3   │ put │balance:42│ 120.50   ││ ... │
    │     │   │ └──────┴─────┴──────────┴──────────┘│     │
    │     │   │          + checksum per entry       │     │
    │     │   └─────────────────────────────────────┘     │
    │     │                                               │
    │     └──2. Apply to in-memory data store             │
    │            data["balance:42"] = "120.50"            │
    │                                                     │
    └─────────────────────────────────────────────────────┘

                      CRASH RECOVERY
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  *** SERVER CRASHES ***                             │
    │  In-memory state: GONE                              │
    │                                                     │
    │  Recovery process:                                  │
    │    1. Open WAL file                                 │
    │    2. For each entry (LSN=1, 2, 3, ...):           │
    │       a. Recompute checksum                        │
    │       b. Verify against stored checksum             │
    │       c. If valid: replay "put" or "delete"         │
    │       d. If corrupt: STOP, report error             │
    │    3. In-memory state is now identical to pre-crash │
    │                                                     │
    │  Recovered state:                                   │
    │    user:42    = "alice"                             │
    │    balance:42 = "120.50"                            │
    │    session:1001 = DELETED (tombstone applied)       │
    │                                                     │
    └─────────────────────────────────────────────────────┘
```

**Key concepts:**

- **LSN (Log Sequence Number)**: A monotonically increasing integer assigned to every log entry. It serves as the logical clock — you can ask "what was the state as of LSN 57?" and replay that far. In real databases (PostgreSQL, MySQL), the LSN maps to a byte offset in the WAL file, enabling random access to any point in the log.
- **Checksum**: Each entry carries a 32-bit FNV-1a hash computed over `LSN|Op|Key|Value`. This detects corruption from partial writes (power loss mid-write), bit flips on disk, or filesystem bugs. Production databases use CRC-32 or CRC-64.
- **Replay is idempotent**: Replaying a "put" for the same key twice produces the same state. Replaying a "delete" on a non-existent key is a no-op (our implementation handles this via map delete). This means you can safely replay from the beginning every time.

## Production Use Cases

- **PostgreSQL WAL**: The cornerstone of PostgreSQL's durability. Every `INSERT`, `UPDATE`, `DELETE` writes a WAL record. The WAL also powers replication — standby servers replay the primary's WAL stream, maintaining an exact copy. Point-in-time recovery (PITR) uses archived WAL segments to replay to any moment.
- **MySQL binlog**: MySQL's binary log is a WAL variant that records logical SQL statements (statement-based) or row changes (row-based). It powers replication between primary and replicas, and is the source for Change Data Capture (CDC) tools like Debezium.
- **etcd Raft log**: etcd uses the Raft consensus algorithm, which requires a persistent write-ahead log. Every state change proposed to the Raft cluster is logged before replication. On restart, etcd replays its log to restore the key-value state.
- **Kafka log segments**: Each Kafka partition is an append-only log. Brokers write messages sequentially to log segments, and consumers replay from a specific offset. Kafka's entire durability model is built on WAL principles.
- **Redis AOF (Append-Only File)**: Redis's alternative to RDB snapshots. Every write command is appended to an AOF file. On restart, Redis replays all commands to reconstruct the dataset. Supports `fsync` policies (always/everysec/no) to trade durability for performance.

## When to Use It

| Use WAL when... | Don't use when... |
|---|---|
| Durability matters — losing committed writes is unacceptable | Your data is ephemeral and can be regenerated (cache warming) |
| You need crash recovery without full-dataset snapshots | You use external durability (e.g., all data lives in an external replicated store) |
| You need point-in-time recovery or change data capture | You can tolerate losing the last N seconds of writes |
| You're building a consensus system (Raft/Paxos requires a log) | Your data is read-only and never mutates |
| You need replication based on change streams | Latency is so critical you can't afford any disk I/O |

**Checkpointing is essential**: A WAL that never truncates grows infinitely. Production systems periodically take a full snapshot of the data (a *checkpoint*), then truncate the WAL to entries after the checkpoint. Recovery replays from the last checkpoint + the remaining WAL. Without checkpoints, recovery time grows linearly with uptime.

**Alternatives**: Some systems (Redis RDB) use periodic snapshots without a WAL, losing all writes since the last snapshot on crash. Others (MongoDB's journal) use a WAL explicitly for crash recovery but not for replication.

## Complexity Analysis

| Operation | Time | Space |
|---|---|---|
| Append entry | O(1) | O(1) per entry |
| Checksum compute (FNV-1a) | O(m) where m = entry length | O(1) |
| Recover (replay all) | O(n) where n = entries | O(n) reads, O(k) output where k = surviving keys |
| Truncate (not in demo) | O(1) | Reclaims O(m) space |

The WAL's space grows O(n) with writes. Without periodic checkpointing and truncation, this is unbounded. Every production WAL implementation includes a checkpoint/truncation mechanism.

## Implementation Deep Dive

### 1. FNV-1a Checksums (`main.go:80-87`)

We use the Fowler-Noll-Vo hash (FNV-1a), a non-cryptographic 32-bit hash chosen for its simplicity and speed. The algorithm XORs each byte with the running hash, then multiplies by a prime (16,777,619). Production databases use CRC-32C (Intel SSE4.2 hardware-accelerated on x86) or CRC-64. Our checksum is computed over the string `"LSN|Op|Key|Value"`, creating a self-describing format that can be verified without a schema. During recovery (`main.go:61-65`), recomputing and comparing the checksum catches corruption immediately — we return an error rather than silently applying bad data.

### 2. Replay-Based Recovery (`main.go:57-76`)

`Recover()` iterates every entry in the log from beginning to end, applying each valid entry to a fresh state map. This is a full redo recovery (no undo needed since we don't have transactions with rollback). The order of replay matters: later entries for the same key overwrite earlier ones, exactly mirroring the chronological order of operations. For deletes, `delete(state, entry.Key)` removes the key, ensuring tombstones are respected. The result is a map containing exactly the last value for every non-deleted key — the same state that existed before the crash.

### 3. LSN as a Logical Clock (`main.go:30,42-50`)

`nextLSN` starts at 1 and increments atomically with each append. In our implementation, LSN is just a counter — but in production systems, it encodes the byte offset in the WAL file (e.g., PostgreSQL's LSN is a 64-bit value where the upper 32 bits are the WAL segment number and the lower 32 bits are the offset within the segment). This allows O(1) random access to any entry by LSN, which is essential for replication protocols that say "send me everything after LSN 0x17/8A000000."

## Running the Demo

```bash
go run ./23-Write-Ahead-Log/
```

The demo simulates three transactions writing session data, user data, and updating a balance, then simulates a crash. Recovery replays the WAL and verifies the reconstructed state matches expectations (including tombstone deletes).

## Further Reading

- **"ARIES: A Transaction Recovery Method Supporting Fine-Granularity Locking and Partial Rollbacks Using Write-Ahead Logging"** — C. Mohan, Don Haderle, Bruce Lindsay, Hamid Pirahesh, Peter Schwarz (1992). The canonical paper on WAL-based recovery. Introduced the ARIES algorithm (Analysis, Redo, Undo) used by IBM DB2 and later adopted by most relational databases.
- **PostgreSQL WAL Internals (postgresql.org/docs/current/wal-internals.html)** — Detailed explanation of PostgreSQL's WAL format, checkpointing, and how the WAL enables streaming replication and PITR.

---

*Part of the Design-With-TsGo system design curriculum*
