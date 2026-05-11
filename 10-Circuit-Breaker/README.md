# Circuit Breaker

> **A fault-tolerance pattern that prevents cascading failures by "tripping open" after a threshold of consecutive failures, fast-failing requests for a timeout period, then probing with a half-open state to detect recovery.**

## The Problem It Solves

You run a payment processing service that depends on a third-party payment gateway. Under normal conditions, the gateway responds in under 50ms. One afternoon, the gateway starts experiencing intermittent timeouts — about 30% of requests take 5 seconds before failing. Your payment service has a connection pool of 50 connections to the gateway, and each request holds a connection for the duration of the call. Within seconds, all 50 connections are tied up waiting for the slow gateway. New payment requests pile up in your service's request queue. Your health check endpoint (which also calls the gateway) starts timing out. The load balancer marks your instances as unhealthy and routes traffic away. A single slow dependency just took down your entire payment pipeline.

This is a **cascading failure**: a failure in one component propagates through dependent services, amplifying damage. The pattern is: dependent service degrades → your service's resources (threads, connections, memory) saturate waiting for it → your service becomes unresponsive → downstream services that call you also fail → the blast radius expands.

The **circuit breaker** pattern, introduced by Michael Nygard in *Release It!*, stops this chain reaction. It wraps calls to the dependency in a state machine with three states:

1. **Closed** (normal): Requests flow through. Consecutive failures are counted.
2. **Open** (tripped): After the failure threshold is reached, all requests fail immediately without calling the dependency. This protects the dependency from overload and frees your resources.
3. **Half-Open** (probing): After a timeout, a limited number of requests are allowed through to test if the dependency has recovered. Success resets to Closed; failure re-trips to Open.

The circuit breaker trades a small number of rejected requests during an outage for preventing total system collapse. It's a bulkhead pattern: isolate failure so it doesn't sink the ship.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────────────┐
│                         CircuitBreaker                                │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │                    CLOSED (normal)                            │    │
│  │  ┌─────────────────┐    ┌─────────────────┐                  │    │
│  │  │ Request allowed  │───→│ Operation runs   │                  │    │
│  │  │ through         │    │                  │                  │    │
│  │  └─────────────────┘    └───────┬─────────┘                  │    │
│  │                                 │                             │    │
│  │                    ┌────────────┴────────────┐               │    │
│  │                    │                         │               │    │
│  │               Success                    Failure             │    │
│  │           (failures=0)               (failures++)           │    │
│  │              stay CLOSED             if failures ≥ threshold │    │
│  │                                               │              │    │
│  │                                               ▼              │    │
│  │  ──────────────────── TRANSITION ────────────────────────── │    │
│  │                                               │              │    │
│  └───────────────────────────────────────────────┼──────────────┘    │
│                                                  │                   │
│  ┌───────────────────────────────────────────────┼──────────────┐    │
│  │                    OPEN (tripped)             │              │    │
│  │                                               ▼              │    │
│  │  ┌─────────────────┐                                        │    │
│  │  │ Request rejected │  (fast-fail, no call to dependency)   │    │
│  │  │ immediately      │                                        │    │
│  │  └─────────────────┘                                        │    │
│  │                                                              │    │
│  │  Timer: openedAt + openTimeout                              │    │
│  │  When timeout elapses:                                       │    │
│  │                                               │              │    │
│  │  ──────────────────── TRANSITION ────────────────────────── │    │
│  │                                               │              │    │
│  └───────────────────────────────────────────────┼──────────────┘    │
│                                                  │                   │
│  ┌───────────────────────────────────────────────┼──────────────┐    │
│  │                  HALF-OPEN (probing)          ▼              │    │
│  │                                                              │    │
│  │  ┌─────────────────┐    ┌─────────────────┐                  │    │
│  │  │ Request allowed  │───→│ Operation runs   │                  │    │
│  │  │ through (probe)  │    │                  │                  │    │
│  │  └─────────────────┘    └───────┬─────────┘                  │    │
│  │                                 │                             │    │
│  │                    ┌────────────┴────────────┐               │    │
│  │                    │                         │               │    │
│  │               Success                    Failure             │    │
│  │           → transition to            → transition to         │    │
│  │             CLOSED                     OPEN                  │    │
│  └──────────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────────┘
```

The circuit breaker at `main.go:32-39` stores: `state` (one of `CircuitClosed`, `CircuitOpen`, `CircuitHalfOpen`), `failures` (consecutive failure count), `failureThreshold` (how many failures trip the breaker), `openTimeout` (how long to stay open before probing), `openedAt` (when the circuit transitioned to Open), and `mu sync.Mutex`.

**State transition on `State()` call** (`main.go:54-63`): The `State()` method is the only place where Open → Half-Open transition can occur. When the state is `CircuitOpen` and `time.Since(cb.openedAt) >= cb.openTimeout`, the state is atomically changed to `CircuitHalfOpen` under the mutex. This is checked at the start of every `Execute()` call.

**Execute operation** (`main.go:69-82`): Calls `cb.State()` to check/transition state. If `CircuitOpen`, returns an error immediately without calling the operation. Otherwise, runs the operation. On success: calls `recordSuccess()` which resets failures to 0 and transitions to `CircuitClosed`. On failure: calls `recordFailure()` which increments failures and transitions to `CircuitOpen` if the threshold is reached OR if the current state is `CircuitHalfOpen` (a single probe failure re-trips).

**recordFailure in Half-Open** (`main.go:97-106`): A single failure in Half-Open immediately opens the circuit again — this is crucial. The half-open state is a probe, not a gradual recovery. One failure proves the dependency is still unhealthy.

## Production Use Cases

- **Netflix Hystrix** — Netflix's latency and fault-tolerance library used circuit breakers as its core pattern. Every inter-service call was wrapped in a Hystrix command with configurable failure thresholds, timeouts, and fallback behaviors. Hystrix is now in maintenance mode, but its design patterns live on in Resilience4j.
- **Resilience4j** — The successor to Hystrix in the Java ecosystem. Resilience4j provides a lightweight circuit breaker decorator for functional interfaces, with sliding window-based failure counting and configurable ring buffer sizes. Used in Spring Cloud Circuit Breaker.
- **Istio / Envoy circuit breaking** — Istio service mesh uses Envoy's circuit breaker for outbound traffic. Configurable per-upstream-cluster: max connections, max pending requests, max retries. When thresholds are exceeded, Envoy opens the circuit and returns 503 immediately.
- **AWS Lambda with DLQ** — Lambda's asynchronous invocation model with Dead Letter Queues implements circuit-breaking semantics: when a Lambda function repeatedly fails, the event source mapping throttles invocations (effectively opening a circuit) until the error rate drops.
- **Polly (.NET)** — The Polly resilience library for .NET provides a circuit breaker policy. Combined with retry and timeout policies, it forms the standard resilience stack for .NET microservices.

## When to Use It

| Scenario | Use Circuit Breaker? |
|----------|----------------------|
| Your service calls an external dependency that can fail or become slow | **Yes** |
| Failures in the dependency could consume your resources (threads, connections) | **Yes** |
| You want to fail fast rather than wait for timeouts when the dependency is known-down | **Yes** |
| The dependency occasionally has transient errors that self-resolve quickly | **No** — retries alone may suffice; circuit breaker adds complexity |
| The failure is expected and handled (e.g., a 404 "not found" is normal) | **No** — circuit breaker should trip on unexpected failures, not 4xx responses |
| You can't define what "failure" means for the dependency (too many different error types) | **No** — circuit breaker needs a clear success/failure binary |

**Alternatives**: **Retry with exponential backoff** (see 11-Retry-Backoff) handles transient failures without breaking the circuit. **Bulkhead pattern** isolates resources (separate thread pools per dependency) so one slow dependency doesn't exhaust all threads. **Timeout** alone prevents indefinite hangs but doesn't stop making calls. The combination of all four (circuit breaker + retry + bulkhead + timeout) is the standard resilience stack.

## Complexity Analysis

| Operation | Time (Average) | Time (Worst) | Space |
|-----------|---------------|-------------|-------|
| State | O(1) | O(1) | O(1) |
| Execute | O(1) + operation time | O(1) + operation time | O(1) |
| recordSuccess | O(1) | O(1) | O(1) |
| recordFailure | O(1) | O(1) | O(1) |

All circuit breaker operations are O(1): they involve at most a couple of mutex lock/unlocks, integer increments, and time comparisons. The actual operation passed to `Execute` runs in its own time. Space is O(1) — the circuit breaker is a fixed-size struct with no dynamic allocations during operation.

## Implementation Deep Dive

**1. Three-state machine with `CircuitState` type.** The states are defined as string constants at `main.go:20-27`: `CircuitClosed`, `CircuitOpen`, `CircuitHalfOpen`. Using a string type (not an int enum) makes log output self-documenting — `state=closed` instead of `state=0`. The `State()` method at line 54-63 is the **only** function that changes state (besides `recordSuccess` and `recordFailure`). This centralized state management prevents inconsistent transitions.

**2. Open timeout is checked at the start of every `Execute` call.** At line 70, `cb.State()` is called before every operation. This is where the Open → Half-Open transition happens automatically when the timeout has elapsed. The design intentionally couples state-checking with timeout evaluation — there's no background goroutine or timer callback. This means the transition happens on the first request after the timeout, not at the exact moment the timer fires. The difference is a few microseconds at most, and it saves a goroutine.

**3. Half-Open is a single-probe state.** In `recordFailure` at line 101, the condition is `cb.state == CircuitHalfOpen || cb.failures >= cb.failureThreshold`. A single failure in Half-Open immediately re-opens the circuit. This is conservative: it prevents thundering-herd problems where many half-open requests all slam a still-degraded dependency. In a production circuit breaker, you might allow N concurrent half-open probes — but this implementation's single-probe design is the safer default, inspired by Hystrix's default behavior.

**4. Thread-safe via mutex, not channels or atomics.** The `sync.Mutex` at line 33 guards state transitions. Every method that reads or writes state (`State`, `recordSuccess`, `recordFailure`) acquires the lock. This is appropriate for a circuit breaker where operations are I/O-bound (the operation function dominates time) — the mutex hold time is negligible. Using atomic operations or channels would add complexity for no benefit in this use case.

## Running the Demo

```bash
go run ./10-Circuit-Breaker/
```

## Further Reading

- *Release It! Design and Deploy Production-Ready Software* — Michael T. Nygard (2007). Chapter 5 ("Stability Patterns") introduces the Circuit Breaker pattern along with other resilience patterns: Timeouts, Bulkheads, and Handshaking. *Pragmatic Bookshelf.*
- "Fault Tolerance in a High Volume, Distributed System" — Ben Christensen (Netflix). The Hystrix tech talk that popularized circuit breakers in the microservices era. https://www.youtube.com/watch?v=RfnxwSomU3M

---

*Part of the Design-With-TsGo system design curriculum*
