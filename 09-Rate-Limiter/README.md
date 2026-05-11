# Token Bucket Rate Limiter

> **A thread-safe rate limiter that regulates request throughput using a "bucket" of tokens — tokens refill at a constant rate, and each request consumes tokens. Supports burst capacity (bucket size) and steady-state throughput (refill rate).**

## The Problem It Solves

You've deployed a new API endpoint that queries your legacy billing database. The database can handle 100 queries per second comfortably, but starts timing out at 150 qps. Your API is public-facing, and one customer has a buggy script that accidentally calls your endpoint 500 times per second in a tight loop. Without rate limiting, this one customer can degrade performance for everyone else — the shared database becomes a victim of the tragedy of the commons.

You need to enforce a per-customer limit: 10 requests per second, with a burst allowance of 20. This means a customer calling at exactly 10 req/s steady-state always gets served. A customer who briefly spikes to 20 req/s should also be served (burst protection). But sustained traffic above 10 req/s should be rejected. And a single 500-req/s burst should consume the burst capacity immediately and then be rate-limited.

The **token bucket algorithm** models this elegantly. Imagine a bucket that holds tokens. The bucket has a fixed capacity (the burst limit). Tokens are added to the bucket at a constant rate (e.g., 5 tokens per second). Each request costs one or more tokens. If enough tokens are available, the request proceeds and the tokens are consumed. If not, the request is rejected. This naturally provides both a burst ceiling (the bucket can accumulate up to `capacity` tokens during idle periods) and a sustained rate cap (tokens are consumed at the same average rate they're replenished).

This is the algorithm behind virtually every production rate limiter: AWS API Gateway throttling, Kong rate-limiting plugin, Nginx `limit_req`, Stripe API, Google Cloud Endpoints. The implementation here uses **lazy refill** — tokens are only computed when `Allow()` is called, based on the elapsed time since the last refill. This avoids a background goroutine and timer overhead.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────────────┐
│                    TokenBucketRateLimiter                             │
│                                                                      │
│  capacity=10    refillPerSecond=5    tokens=7.3                      │
│                                                                      │
│  ┌──────────────────────────────────────┐                            │
│  │  ████████░░░░░░░░░░░░░░░░░░░░░░░░░░  │  Bucket (10 slots)        │
│  │  tokens: 7.3 / 10                    │                            │
│  └──────────────────────────────────────┘                            │
│                                                                      │
│  Token dynamics:                                                     │
│    t=0.0s:  tokens=10 (full at startup)                             │
│    t=0.1s:  Allow(4)  → tokens=6.0   (allowed)                     │
│    t=0.3s:  Allow(5)  → tokens=1.0   (allowed, 5 consumed)         │
│    t=0.5s:  Allow(3)  → tokens=0.0   (rejected, not enough)        │
│    t=1.0s:  (refill: 0.5s elapsed × 5/s = 2.5 tokens)              │
│             Allow(3)  → tokens=0.5   (allowed)                     │
│    t=2.0s:  (refill: 1.0s × 5/s = 5.0)                             │
│             tokens = min(10, 0.5+5) = 5.5                          │
│                                                                     │
│  Lazy refill: tokens are computed on each Allow() call:             │
│    elapsed = now - lastRefill                                       │
│    tokens = min(capacity, tokens + elapsed * refillPerSecond)       │
│    lastRefill = now                                                 │
└──────────────────────────────────────────────────────────────────────┘
```

The rate limiter at `main.go:19-25` stores five fields: `capacity` (max tokens, the burst size), `tokens` (current token count, floating-point), `refillPerSecond` (rate of token accumulation), `lastRefill` (timestamp of last token computation), and `mu sync.Mutex` (thread safety).

**Allow(cost float64) bool** (`main.go:47-61`): Locks the mutex, calls `refill(time.Now())` to lazily update the token count, then checks if `tokens >= cost`. If yes, deduct and return true. If no, return false without deducting. A cost of ≤ 0 always returns true (free request, no consumption).

**refill(now time.Time)** (`main.go:64-75`): Computes `elapsed = now - lastRefill`, then `newTokens = min(capacity, tokens + elapsed * refillPerSecond)`. The `mathMin` cap at line 69 prevents the bucket from exceeding `capacity` — idle periods don't accumulate infinite tokens. The timestamp is updated to `now`, making the next refill computation start from this point.

The key insight is **floating-point token accounting**. Tokens are tracked as `float64`, not integers. This means sub-token refill works correctly — after 0.1 seconds with a refill rate of 5/s, the bucket gains 0.5 tokens. Integer-based token buckets either lose precision (rounding down) or require periodic bulk refills. Floating-point enables smooth, continuous refill.

## Production Use Cases

- **AWS API Gateway throttling** — AWS API Gateway uses the token bucket algorithm for usage plans. Each API key gets a rate (tokens/second) and a burst limit. When a key exceeds its rate, API Gateway returns 429 Too Many Requests.
- **Google Cloud Endpoints** — Cloud Endpoints rate limiting uses a token bucket per consumer (API key or service account). The quotas are configured in the OpenAPI spec and enforced at the proxy layer.
- **Kong API Gateway** — Kong's `rate-limiting` plugin implements token bucket, leaky bucket, and sliding window strategies. The token bucket is the default for distributed rate limiting backed by Redis or Cassandra.
- **Nginx `limit_req`** — Nginx's request rate limiting module uses a leaky bucket variant internally, but the `burst` parameter (`limit_req zone=one burst=5`) adds token-bucket-like burst capacity on top of the steady rate.
- **Stripe API** — Stripe enforces rate limits on its API using token buckets per API key. The `X-Stripe-Rate-Limit-Remaining` and `Retry-After` headers provide transparency into the bucket state.

## When to Use It

| Scenario | Use Token Bucket? |
|----------|-------------------|
| You need to enforce average throughput with burst tolerance | **Yes** |
| Requests have different costs (light read = 1 token, heavy write = 5 tokens) | **Yes** — cost parameter |
| You're rate-limiting an API endpoint or message queue consumer | **Yes** |
| You need strict "at most X per second, no bursts ever" | **No** — use a leaky bucket (smoothed, no burst) |
| You need per-time-window semantics ("100 per hour") | **No** — use a fixed-window or sliding-window counter |
| You're rate-limiting in a distributed system without centralized state | **No** — token bucket is per-process; needs Redis for distributed enforcement |

**Alternatives**: **Leaky bucket** processes requests at a fixed rate with a queue — bursts are queued rather than immediately served or rejected, providing traffic shaping rather than rate limiting. **Fixed window counter** is simpler (increment counter, reset on window boundary) but allows double-burst at window edges. **Sliding window log** is exact but memory-intensive for high-throughput systems.

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| Allow | O(1) | O(1) | O(1) |
| refill | O(1) | O(1) | O(1) |

Every operation is O(1) — the refill computation is a few multiplications and a min, and the token check is a single comparison. The mutex is held for ~microseconds. Space is O(1) per rate limiter instance (five fields, all fixed-size). No background goroutines, no timers, no allocations after construction.

## Implementation Deep Dive

**1. Lazy refill eliminates background goroutines.** The `refill` function at line 64-75 is called at the start of every `Allow()` call (line 50). It updates the token count based on real elapsed time. This is a deliberate design choice: no background ticker, no goroutine, no timer management. The downside is that tokens aren't refilled during idle periods until the next `Allow()` call — but since the lag is nanoseconds, it doesn't matter. The upside is a simpler, more testable implementation with no concurrency bugs around goroutine lifecycle.

**2. Floating-point token accounting for smooth refill.** Tokens are `float64` (`main.go:22`), and the refill formula at line 69 multiplies `elapsed * refillPerSecond`. This naturally handles fractional tokens — after 100ms with a 5/s rate, 0.5 tokens are added. Alternative approaches using integers would need to either accept precision loss or use a "token accumulator" pattern with a separate ticker, adding complexity.

**3. Mutex-guarded thread safety.** The `sync.Mutex` at line 20 guards the entire `Allow` method. The lock is acquired, refill and check happen, and the lock is released via `defer`. This is correct for Go's concurrency model where `Allow` may be called from many goroutines servicing HTTP requests simultaneously. The critical section is tiny (a few float64 ops), so lock contention is negligible even at 100K+ requests/second.

**4. Cost-based consumption for heterogeneous requests.** The `Allow(cost float64)` signature at line 47 supports variable request costs. A lightweight health-check endpoint could `Allow(0.5)`, a heavy batch operation could `Allow(5)`. This maps naturally to API gateway tiered-pricing models where different endpoints have different weightings against the rate limit budget.

## Running the Demo

```bash
go run ./09-Rate-Limiter/
```

## Further Reading

- "Specification of Guaranteed Quality of Service" — RFC 2212 (1997). Defines the token bucket traffic specification (TSPEC) for the Integrated Services (IntServ) QoS model. https://datatracker.ietf.org/doc/html/rfc2212
- "Rate Limiting" — Nginx documentation. Practical guide to the leaky bucket and token bucket implementations in the `ngx_http_limit_req_module`. https://nginx.org/en/docs/http/ngx_http_limit_req_module.html

---

*Part of the Design-With-TsGo system design curriculum*
