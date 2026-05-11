# Connection Pool

> **A bounded, lazily-populated pool of reusable connections with health-checking on acquire and LIFO idle reuse — the standard pattern for managing expensive, stateful resources like database and network connections.**

## The Problem It Solves

You've launched your API server with PostgreSQL as the primary data store. Every request opens a new database connection, runs a query, and closes the connection. At 500 requests per second, you're opening and closing 500 TCP connections per second to PostgreSQL. Each connection requires a TCP handshake (1.5 RTTs), TLS negotiation (2 RTTs), and PostgreSQL authentication (1 RTT) — roughly 4-8ms of latency before a single query runs. Your p99 latency is 45ms for a query that takes 2ms to execute. Worse, PostgreSQL's `max_connections` is set to 100, and you're hitting it regularly, causing "too many clients" errors under load.

The solution is a connection pool: maintain a set of already-established connections that can be reused across requests. When a request needs a connection, it acquires one from the pool (already authenticated, already warmed), uses it, and releases it back. The pool caps the maximum number of concurrent connections to prevent overwhelming the database. Idle connections are validated before reuse — a TCP connection that was severed by an intermediate load balancer's idle timeout (RST packet) must not be handed to an unsuspecting query.

The connection pool pattern appears wherever creating a resource is expensive: database connections (PgBouncer, HikariCP, pgxpool), HTTP connections (Go's `http.Transport`), Redis connections (go-redis), and gRPC connections. It's arguably the single most impactful performance optimization for any networked service — replacing a 4-8ms connection setup with a <1μs pool acquire can reduce p99 latency by an order of magnitude.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────┐
│                   ConnectionPool[T]                            │
│                                                              │
│  idle []T          ── slice of reusable connections (LIFO)   │
│  active int        ── count of in-use connections            │
│  max int           ── hard cap on active connections         │
│  factory func() T  ── creates a new connection               │
│  validate func(T) bool ── health check before reuse          │
│  mu sync.Mutex     ── protects all fields                    │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  Acquire() flow:                                        │  │
│  │                                                         │  │
│  │  Lock mutex                                             │  │
│  │  while len(idle) > 0:                                   │  │
│  │      conn := idle[len(idle)-1]  (LIFO pop)              │  │
│  │      idle = idle[:len(idle)-1]                          │  │
│  │      if validate(conn):                                  │  │
│  │          active++; return conn  ✓                       │  │
│  │      else:                                               │  │
│  │          discard (connection is dead)                    │  │
│  │  if active >= max:                                       │  │
│  │      return error "pool exhausted"  ✗                   │  │
│  │  conn := factory()                                      │  │
│  │  active++; return conn  ✓                               │  │
│  │  Unlock mutex                                           │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  Release(conn) flow:                                    │  │
│  │                                                         │  │
│  │  Lock mutex                                             │  │
│  │  active--                                                │  │
│  │  if validate(conn):                                      │  │
│  │      idle = append(idle, conn)  (LIFO push)             │  │
│  │  else:                                                   │  │
│  │      discard (connection broken during use)              │  │
│  │  Unlock mutex                                           │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  Key properties:                                             │
│  - LIFO idle reuse:  most recently released = most likely   │
│    to have an active TCP connection (keepalive)              │
│  - Validation on both Acquire and Release: catch dead       │
│    connections before they cause query failures              │
│  - Lazy creation:  no connections allocated until first      │
│    Acquire (unlike Channel Pool's eager pre-warming)         │
└──────────────────────────────────────────────────────────────┘
```

The `ConnectionPool[T]` struct (`main.go:17-24`) holds five fields: `idle` (a slice of reusable connections, used as a LIFO stack), `active` (count of connections currently checked out), `max` (hard limit), `factory` (function to create new connections), and `validate` (health-check predicate). A `sync.Mutex` guards all state.

`Acquire` (`main.go:41-66`) tries to reuse an idle connection first. It pops from the end of `idle` (LIFO — most-recently-released first, because that connection is most likely to still have a valid TCP keepalive). If the popped connection passes `validate`, it's returned. If it fails (stale connection, severed by load balancer timeout), it's discarded and the pool tries the next idle connection. If no idle connection passes validation, and `active < max`, the pool calls `factory()` to create a new connection. If `active >= max`, `Acquire` returns an error — the pool is exhausted.

`Release` (`main.go:70-81`) decrements `active` and, if the connection passes validation, pushes it onto the idle stack via `append`. If validation fails, the connection is discarded permanently — no broken connection ever returns to the pool. This dual validation (on acquire AND release) is what separates a correct connection pool from a naive one.

The demo at `main.go:83-144` creates a pool of 3 database connections (simulated as strings), serving 5 concurrent requests with an 80ms simulated query time. Despite 5 concurrent requests and only 3 connections, the demo works because connections are released and reused.

## Production Use Cases

- **PostgreSQL connection pooling (PgBouncer)** — PgBouncer is the standard PostgreSQL connection pooler, deployed in front of virtually every production Postgres instance. It maintains a pool of server connections and multiplexes client connections onto them. In transaction pooling mode, a server connection is assigned to a client for the duration of a single transaction, then returned to the pool — the same Acquire/Release lifecycle.
- **HikariCP (Java/JDBC)** — The most widely deployed JDBC connection pool. HikariCP's design paper details its optimizations: LIFO idle reuse (for thermal cache locality), connection validation via `isValid()` timeout (a lightweight ping, not a full query), and a custom `ConcurrentBag` data structure that avoids lock contention on the fast path. Used by Spring Boot, Apache Spark, and virtually every Java web framework.
- **Redis connection pooling (go-redis)** — The `go-redis` Go client maintains a `ConnPool` with `PoolSize` (max connections), `MinIdleConns` (minimum warm connections), and `MaxConnAge` (force-recycle connections older than N). The pool validates connections with Redis `PING` before handing them to callers. This is the standard Redis client for Go, used in thousands of production services.
- **Go's `http.Transport` connection pooling** — The `net/http.Transport` maintains an idle connection pool (`idleConn map[connectMethodKey][]*persistConn`) keyed by destination host. `MaxIdleConns` and `MaxIdleConnsPerHost` cap the pool size. Connections are validated by checking if they've exceeded `IdleConnTimeout`. Every Go HTTP client uses this pool transparently.
- **MongoDB driver connection pooling** — The official MongoDB Go driver (`mongo-driver`) uses a connection pool with `MaxPoolSize`, `MinPoolSize`, and background goroutines that maintain the minimum idle count and prune expired connections. The pool checks connection liveness with a MongoDB `isMaster` command before use.

## When to Use It

| Scenario | Use Connection Pool? |
|----------|---------------------|
| Creating a resource is expensive (TCP handshake, TLS, authentication) | **Yes** |
| Resources are stateful and can be reused across requests | **Yes** |
| Need a hard cap on concurrent resource usage | **Yes** |
| Resources are stateless (a pool adds complexity with no benefit) | **No** — just create and discard |
| Resources leak state between uses (a connection leaves a transaction open) | **No** — or implement a `Reset` step before `Release` |
| Need ahead-of-time connection warming (all connections ready before first request) | **No** — this pool is lazy; add pre-warming if needed |

**Alternatives**: **Channel Pool** (topic 17) pools Go channels (stateless communication primitives) — simpler than a connection pool because channels don't need validation. **Semaphore** (topic 15) provides concurrency limiting without pooling — acquire a permit, create a resource, use it, discard it. **Lazy initialization** (create a connection on first use and cache it forever) works for single-connection scenarios but breaks under concurrency and doesn't handle connection death. **Pre-forked connection set** (create all connections at startup) provides predictable resource usage but wastes memory for connections that might never be used.

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| Acquire (idle hit) | O(1) — LIFO pop + 1 validation | O(idle) — all idle connections fail validation | O(max) |
| Acquire (new creation) | O(factory) — plus idle scan | O(idle + factory) | O(max) |
| Release | O(1) — LIFO push + 1 validation | O(1) | O(max) |

The O(idle) worst case for Acquire occurs when every idle connection fails validation (e.g., all connections were severed by a load balancer timeout). The pool iterates through all idle connections, discarding each, then creates a new one. In practice, idle pools are kept small, and mass connection death is rare.

Space is O(max): the pool can hold at most `max` connections (active + idle). The `idle` slice holds zero-copy references (pointers for reference types, values for value types) — no additional per-connection allocation beyond the connection itself.

## Implementation Deep Dive

**1. LIFO reuse (not FIFO) maximizes connection freshness.** At `main.go:46-47`, the pool pops from the *end* of the idle slice: `conn := p.idle[len(p.idle)-1]; p.idle = p.idle[:len(p.idle)-1]`. This is a LIFO (stack) strategy — the most recently released connection is reused first. This connection is most likely to still have a valid TCP connection (keepalive hasn't timed out yet) and may have warm server-side caches (PostgreSQL's query plan cache, MySQL's buffer pool). HikariCP's design paper specifically recommends LIFO for these reasons. The alternative, FIFO (oldest first), reuses connections that are closest to their idle timeout — maximizing the chance of a dead connection.

**2. Dual validation (Acquire + Release) prevents broken connections from ever being served.** The `validate` function is called in two places: during `Acquire` (checking an idle connection before handing it out, `main.go:48`) and during `Release` (checking a connection being returned, `main.go:75`). If a connection fails validation on Acquire, it's silently discarded and the next idle connection is tried. If it fails on Release, it's discarded rather than returned to the idle pool. This dual check means a connection that was broken during use (e.g., the database restarted mid-query, or the network dropped) never re-enters the pool. Without this, a single broken connection would be handed out, fail, returned, handed out again, fail again — an infinite error loop.

**3. The mutex is a single coarse-grained lock (acceptable for pool operations).** The `sync.Mutex` at `main.go:23` protects all pool state. For a connection pool, this is usually sufficient because Acquire and Release are infrequent relative to the actual work (a 2μs lock while acquiring a connection vs. 5ms of query execution). High-contention connection pools (10,000+ acquires/s) may benefit from sharding or lock-free structures, as HikariCP does with its `ConcurrentBag`. But for most Go services, the single mutex is the right tradeoff: simple, correct, and fast enough.

## Running the Demo

```bash
go run ./20-Connection-Pool/
```

## Further Reading

- "HikariCP: A High-Performance JDBC Connection Pool" — Brett Wooldridge (2012). The design paper for the fastest JDBC connection pool. Covers LIFO reuse semantics, validation strategies, `ConcurrentBag` data structure design, and the micro-optimizations that made HikariCP the default in Spring Boot.
- "PG Bouncer: A Lightweight Connection Pooler for PostgreSQL" — PgBouncer documentation. The architectural guide for the most widely deployed PostgreSQL connection pooler. Explains session pooling vs. transaction pooling vs. statement pooling and when to use each.

---

*Part of the Design-With-TsGo system design curriculum*
