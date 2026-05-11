# Retry Backoff

> **A configurable retry loop that multiplies wait time between attempts, adds random jitter to defuse thundering herds, and respects context cancellation — the standard pattern for resilient distributed systems.**

## The Problem It Solves

Your payment service calls Stripe's API to charge customers. At 2:47 PM on Black Friday, Stripe begins returning HTTP 503 errors on 4% of requests. Your first instinct is to catch the error and retry immediately — but now you've multiplied the load on an already-struggling service. Worse, every other merchant's payment service just did the same thing. The 503s spike to 503s × retries × merchants, and Stripe's recovery takes minutes longer than it should. You need a retry strategy that backs off, not piles on.

The naive fix is a fixed-delay retry: wait exactly 1 second between each of 5 attempts. But when a downstream outage affects thousands of clients simultaneously, they all retry on the same 1-second cadence, producing a "thundering herd" that pounds the recovering service with traffic spikes at predictable intervals. What you actually need is exponential backoff — each retry waits twice as long as the previous one — plus random jitter that smears those spikes across a distribution so no two clients retry at the same instant.

There's a second problem: context lifetime. A user refreshes their browser after 3 seconds of waiting. The HTTP request context is cancelled, but your retry loop doesn't know about it — it keeps hammering the payment gateway for 2 more attempts, burning goroutines and connection slots for a caller that no longer exists. A correct retry implementation must listen for context cancellation between attempts and stop immediately, freeing resources for actual work.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────┐
│                        Retry Loop                             │
│                                                              │
│  ┌──────────┐    ┌──────────────┐    ┌───────────────────┐  │
│  │ Caller   │───▶│  operation()  │───▶│ delayForAttempt() │  │
│  │ ctx      │    │  (attempt #)  │    │                   │  │
│  │ policy   │    └──────┬───────┘    │ delay = min(       │  │
│  └──────────┘           │            │   init × mult^att, │  │
│                         ▼            │   maxDelay         │  │
│              ┌──────────────────┐    │ )                  │  │
│              │   err != nil?    │    │ + jitter?          │  │
│              │                  │    │   rand(0, delay)   │  │
│              └───Yes──┬───No────┘    └────────┬──────────┘  │
│                      │     │                  │              │
│                      │     ▼                  ▼              │
│                      │   return nil    ┌─────────────┐      │
│                      │                 │ time.NewTimer│      │
│                      │                 │ (duration)  │      │
│                      │                 └──────┬──────┘      │
│                      │                        │              │
│                      │              ┌─────────▼────────┐    │
│                      │              │  select {         │    │
│                      │              │    <-ctx.Done():  │    │
│                      │              │      return error │    │
│                      │              │    <-timer.C:     │    │
│                      │              │      next attempt │    │
│                      │              │  }                │    │
│                      │              └──────────────────┘    │
│                      ▼                                       │
│               attempt++ (loop)                               │
└──────────────────────────────────────────────────────────────┘
```

The `Retry` function (`main.go:32-61`) is a single loop over `MaxAttempts` iterations. On each attempt it calls the user-supplied `operation` function. If the operation succeeds, `Retry` returns immediately — no delay, no waste. If it fails, the function computes the backoff duration via `delayForAttempt` and blocks on a `select` statement that races the timer against `ctx.Done()`.

The `delayForAttempt` function (`main.go:67-78`) implements the standard exponential backoff formula: `min(InitialDelay × Multiplier^(attempt-1), MaxDelay)`. The exponent is computed by a simple iterative `pow` helper (`main.go:81-87`). When `Jitter` is enabled, the final delay is a random value in [0, computedDelay], which turns synchronous retry waves into a flat, uncorrelated distribution. This is the "full jitter" strategy — the most effective of the three jitter variants described in the AWS Architecture blog.

The `RetryPolicy` struct (`main.go:21-27`) bundles all tuning knobs: `MaxAttempts` (total calls including the first), `InitialDelay` (the base wait), `MaxDelay` (a hard ceiling to prevent exponential runaway), `Multiplier` (the growth factor), and `Jitter` (a boolean toggle). The demo at `main.go:95-112` configures a policy with 4 attempts, 50ms initial delay, a 2s cap, 2× multiplier, and jitter enabled — then runs a simulated payment gateway that fails twice and succeeds on the third call.

## Production Use Cases

- **AWS SDK (Go v2)** — The retry package in aws-sdk-go-v2 implements exponential backoff with jitter as the default retryer for all AWS service clients. Every `DynamoDB.PutItem`, `S3.GetObject`, and `SQS.ReceiveMessage` call retries with this exact pattern, configurable via `retry.NewStandard()`.
- **gRPC retry interceptor** — The gRPC-Go library provides `grpc_retry` with exponential backoff and jitter, configurable per-method in the service config. Google's internal Stubby RPC framework (gRPC's predecessor) pioneered this approach.
- **Stripe API client (Go SDK)** — `stripe-go` retries idempotent API calls (GET, PUT, DELETE) with exponential backoff and jitter, defaulting to 2 retries. The `MaxNetworkRetries` param on `stripe.BackendConfig` controls this behavior.
- **RabbitMQ consumer retries** — RabbitMQ's dead-letter exchange pattern combined with per-message TTL and queue chaining produces a delayed retry mechanism. The exponential backoff is simulated by chaining queues with increasing TTLs: queue.1s → queue.5s → queue.25s.
- **Kubernetes controller reconciliation loops** — The `controller-runtime` library used by every Kubernetes operator implements a `workqueue` with `NewItemExponentialFailureRateLimiter(baseDelay, maxDelay)`. When a controller fails to reconcile a resource, it's re-queued with exponential backoff, preventing tight loops on persistent errors.

## When to Use It

| Scenario | Use Retry+Backoff? |
|----------|---------------------|
| Transient failures (network timeouts, DNS blips, 503s) with expected self-healing | **Yes** |
| Calls to external APIs over public internet where packet loss is normal | **Yes** |
| Idempotent operations (GET requests, idempotent-key POSTs) | **Yes** |
| Non-idempotent operations where duplicate execution is catastrophic (charging credit cards without idempotency keys) | **No** — use exactly-once delivery instead |
| Persistent errors (401 Unauthorized, 403 Forbidden, 404 Not Found) — retrying won't fix a bad API key | **No** — fail fast |
| Need guaranteed delivery with hours-long retry windows | **No** — use a message queue with persistent storage |
| Tight latency SLOs (must return in < 200ms) — backoff delays will violate the SLO | **No** — use hedging (send to multiple replicas, take first response) |

**Alternatives**: **Circuit Breaker** (10-Circuit-Breaker) stops calling a failing service entirely after a threshold of failures, preventing wasted retries and giving the downstream time to recover. **Hedging** sends the same request to multiple replicas simultaneously and uses whichever responds first — better for latency than for correctness. **Message queues** (Kafka, RabbitMQ) provide durable retry with hours-long windows and exactly-once semantics, but add infrastructure complexity.

## Complexity Analysis

| Metric | Value |
|--------|-------|
| Time per attempt | O(1) — single operation call + delay computation |
| Time worst case | O(MaxAttempts) — iterates up to the configured limit |
| Space | O(1) — no allocations beyond the timer |
| Jitter quality | Uniform random over [0, delay] — effective for thundering herd prevention |

The space complexity is constant because the retry loop is purely procedural — it holds no growing state. Each iteration reuses the same `lastErr` variable and the same timer. The `pow` helper uses iteration over `exp` steps, but `exp` equals `attempt-1` which is bounded by `MaxAttempts` (typically ≤ 10), so it's effectively O(1).

## Implementation Deep Dive

**1. The `select` on `ctx.Done()` is the cancellation contract.** At `main.go:47-54`, the retry loop creates a timer and blocks on a two-way `select`: either the timer fires (proceed to next attempt) or `ctx.Done()` yields a value (caller cancelled). This is the idiomatic Go pattern for making any blocking operation context-aware. The `timer.Stop()` call on the cancellation path prevents a goroutine leak — without it, the timer goroutine would linger until it fires, holding a reference to the retry loop's stack frame.

**2. Full jitter is the safest randomization strategy.** The implementation at `main.go:72-76` uses `rand.Int63n(int64(delay) + 1)`, which picks a uniform random value in [0, delay]. This is "full jitter" — the most aggressive and most effective variant. The AWS blog tested three strategies: (a) jitter applied to the sleep only, (b) jitter applied to the sleep but with a decorrelation factor, and (c) full jitter. Full jitter produces the flattest distribution of retry times and the lowest contention. The tradeoff is that full jitter can produce very short delays (even zero), which might be undesirable for DDoS-sensitive APIs — but for most distributed systems, it's the right default.

**3. The `MaxDelay` cap prevents unbounded growth.** Without a cap, `InitialDelay × Multiplier^attempt` would rapidly exceed reasonable bounds. For a 1-second initial delay with a 2× multiplier and 10 attempts, the last delay would be 512 seconds (8.5 minutes) — likely longer than any HTTP timeout or user tolerance. The cap at `main.go:68-70` clamps this with `min(computed, MaxDelay)`, preventing runaway waits.

## Running the Demo

```bash
go run ./11-Retry-Backoff/
```

## Further Reading

- "Exponential Backoff and Jitter" — AWS Architecture Blog. Marc Brooker's definitive analysis of three jitter variants with empirical data on contention patterns. Required reading before deploying any retry system.
- "Timeouts, Retries, and Backoff" — Google SRE Book, Chapter 22. Covers retry budgets, per-request retry limits, and how Google's internal services implement cascading retry without amplifying load.

---

*Part of the Design-With-TsGo system design curriculum*
