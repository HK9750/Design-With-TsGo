# Distributed Lock

> **Lease-based distributed locking with monotonic tokens, TTL-based auto-expiry, and owner-guarded release — prevents concurrent access to shared resources across multiple processes.**

## The Problem It Solves

Picture yourself as the backend lead for a payments platform processing $2M in transactions per hour. You have 12 worker instances pulling from a shared Kafka queue, and one of the operations is "process refund for order #ORD-8842." If two workers pick up the same refund request simultaneously — perhaps due to an at-least-once delivery guarantee or a partition rebalance — you'll refund the customer twice. Your finance team will be very unhappy. You need a mechanism so that exactly one worker can claim exclusive rights to process a given order, and the lock must expire automatically if that worker crashes mid-operation.

The core challenge is that in a distributed system, there is no single-process mutex to rely on. Locks must be visible across machines, survive process crashes (via TTL expiry), and prevent accidental releases by stale lock holders. A naive lock might let worker-A acquire a lock, then worker-A pauses for GC for longer than the TTL, worker-B acquires the same lock, and then worker-A wakes up and releases the lock — except it just released worker-B's lock. You need ownership validation baked into the protocol.

This implementation models the core primitives behind Google's Chubby and the Redis Redlock algorithm: every lock acquisition returns a monotonically increasing *token*, and release requires presenting the matching token and owner. Combined with TTL-based auto-expiry, this creates a system where locks are self-healing (crashed holders lose the lock automatically) and safe (stale holders cannot release someone else's lock).

## Architecture & Internals

```
┌────────────────────────────────────────────────────────┐
│              DistributedLockManager                     │
│                                                        │
│  locks: {                                              │
│    "order:1": {                                        │
│      Owner:     "worker-a",                            │
│      Token:     42,         ◄── Monotonically increasing│
│      ExpiresAt: 2026-05-11T14:30:02                    │
│    },                                                  │
│    "inventory:99": {                                   │
│      Owner:     "worker-d",                            │
│      Token:     44,                                    │
│      ExpiresAt: 2026-05-11T14:30:04                    │
│    }                                                   │
│  }                                                     │
│  nextToken: 45                                         │
│                                                        │
│  ACQUIRE FLOW:                                         │
│  ┌────────┐    ┌──────────┐    ┌──────────┐           │
│  │ Check  │───►│ Lock     │───►│ Issue    │           │
│  │ expiry │    │ held?    │    │ lease    │           │
│  └────────┘    └────┬─────┘    └──────────┘           │
│                     │ Yes                              │
│                     ▼                                  │
│                ┌──────────┐                            │
│                │ Return   │                            │
│                │ false    │                            │
│                └──────────┘                            │
│                                                        │
│  RELEASE FLOW:                                         │
│  ┌────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐ │
│  │ Owner  │─►│ Token    │─►│ Delete   │  │ Deny     │ │
│  │ match? │  │ match?   │  │ lock     │  │ release  │ │
│  └────────┘  └──────────┘  └──────────┘  └──────────┘ │
│                                                        │
│  VALIDATE FLOW: O(1) check of token + expiry           │
└────────────────────────────────────────────────────────┘
```

The `DistributedLockManager` (`main.go:24-27`) holds a `locks` map keyed by resource name, where each value is a `Lease` struct containing the `Owner` (string identifier of the holder), `Token` (monotonic integer), and `ExpiresAt` (wall-clock deadline). A `nextToken` counter increments on every successful acquisition to guarantee uniqueness and monotonic ordering.

**On `Acquire`** (`main.go:39-54`): The manager checks if the resource is currently locked by examining the existing lease's expiry. If the lock is held and not expired, acquisition fails — this is the mutual exclusion guarantee. If the lock is expired (stale), the old lease is overwritten. A new `Lease` is created with the current token (which is then incremented), the owner's name, and an expiry set to `now + TTL`. The token serves as a *fencing token* — it's guaranteed to be larger than any previously issued token for this resource.

**On `Release`** (`main.go:58-77`): This is the safety-critical operation. The caller must present the owner name AND the token. If either doesn't match the current lease, the release is denied. This prevents the "GC pause" scenario described above: even if a stale holder wakes up, its token no longer matches the current lock, and its release attempt is rejected. Only after all three validations pass (lock exists, owner matches, token matches) is the lock deleted from the map.

**On `Validate`** (`main.go:80-92`): A lightweight check that the token matches and the lease hasn't expired. Used by workers before committing a critical operation (like processing a payment) to confirm they still hold the lock. In production, you'd call this before every write inside the critical section.

## Production Use Cases

- **Redlock (Redis)** — The Redlock algorithm, formalized by Redis's creator Salvatore Sanfilippo, uses multiple independent Redis instances to provide distributed locking with fencing tokens. It's one of the most widely used distributed lock implementations in production.
- **Chubby (Google)** — Google's internal lock service, which inspired many distributed coordination systems. Chubby provides coarse-grained distributed locks with a filesystem-like interface and is used for leader election, naming, and configuration storage across Google's infrastructure.
- **Apache ZooKeeper locks** — ZooKeeper provides ephemeral sequential znodes that implement distributed locks. If a client disconnects, its ephemeral node is automatically deleted (equivalent to TTL expiry), releasing the lock.
- **etcd concurrency primitives** — etcd's `concurrency` package provides `Mutex` and `Election` built on top of leases. The lease TTL ensures that locks are released if a client crashes. Used extensively in Kubernetes controllers.
- **Amazon DynamoDB lock client** — AWS provides the DynamoDB Lock Client library, which implements distributed locking using DynamoDB's TTL and conditional writes. Used for coordinating distributed workflows like AWS Step Functions.

## When to Use It

| Scenario | Use Distributed Lock? |
|----------|----------------------|
| Exactly-once processing of shared work items across workers | **Yes** |
| Coordinating access to a shared resource that doesn't support idempotency | **Yes** |
| Leader election for stateless services (acquire a lock = become leader) | **Yes** |
| You need sub-second lock granularity and high throughput | **Yes** (with a fast backend like Redis) |
| Your critical section is shorter than the lock TTL | **Yes** |
| Your critical section is long-running (minutes) | **No** — locks aren't the right primitive; use a workflow engine |
| You need strict serializability across multiple resources | **No** — use a distributed transaction (36-Distributed-Transaction) |
| You can make the resource itself idempotent | **No** — idempotency is simpler and more robust than locking |

**Alternatives**: **Optimistic concurrency control** (e.g., version vectors or CAS) avoids locks by checking a version token before committing — better for high-contention scenarios since it doesn't block. **Idempotency keys** (e.g., Stripe's `Idempotency-Key` header) ensure retries don't duplicate side effects without explicit locking. **Leader election** (31-Leader-Election) is a special case of distributed locking where the lock represents leadership.

## Complexity / Behavior Characteristics

| Operation | Time Complexity | Space Complexity |
|-----------|----------------|------------------|
| Acquire | O(1) | O(n) per lock manager state |
| Release | O(1) | O(1) per call |
| Validate | O(1) | O(1) per call |

All operations are constant time because the `locks` map provides direct key lookup. The TTL-based expiry is passive — there's no background goroutine sweeping expired locks. Instead, an expired lock is detected opportunistically on the next `Acquire` call (`main.go:42` checks `current.ExpiresAt.After(now)`). This "lazy expiration" is the same approach used by Redis and avoids the overhead of a timer per lock.

## Implementation Deep Dive

**1. Monotonic tokens prevent stale releases.** The `nextToken` counter at `main.go:26` increments on every successful acquisition (`main.go:50: m.nextToken++`). This guarantees that each lock acquisition produces a strictly larger token than any previous one. When `Release` is called (`main.go:69-72`), it checks `current.Token != token` — a stale holder with an old token is denied. This is the *fencing token* pattern described in Martin Kleppmann's "How to do distributed locking" blog post. In production, you'd also pass the fencing token to the resource itself (e.g., a storage system) so it can reject writes from stale lock holders.

**2. Lazy TTL expiry avoids background cleanup.** There is no janitor goroutine or timer wheel. Expired locks are detected on-demand in `Acquire` at `main.go:42` via `current.ExpiresAt.After(now)`. If the current lock has expired, the manager simply overwrites it (`main.go:47-48` logs a stale lease detection). This lazy approach is memory-efficient and avoids the complexity of a cleanup scheduler, but it means the `locks` map can accumulate dead entries. In production systems like Redis, you'd combine lazy expiry with a periodic sampling eviction.

**3. Triple validation on release.** The `Release` function (`main.go:58-77`) performs three sequential checks: lock existence (`main.go:61`), owner match (`main.go:65`), and token match (`main.go:69`). Only if all three pass is the lock deleted. This belt-and-suspenders approach means a malicious or buggy client cannot release a lock by guessing — it must present the exact (owner, token) pair it was issued. The order of checks matters: if the owner check failed first, a client could probe for valid tokens. By failing fast on owner mismatch, we minimize the information leaked.

## Running the Demo

```bash
go run ./32-Distributed-Lock/
```

## Further Reading

- "The Chubby Lock Service for Loosely-Coupled Distributed Systems" — Mike Burrows (2006). The paper that introduced lease-based distributed locking at Google scale. *Proceedings of OSDI 2006.*
- "How to do distributed locking" — Martin Kleppmann (2016). A critical analysis of the Redlock algorithm with a discussion of fencing tokens and why they're essential. *martin.kleppmann.com.*

---

*Part of the Design-With-TsGo system design curriculum*
