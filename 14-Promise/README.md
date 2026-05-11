# Promise (Future)

> **A generic Future/Promise primitive that executes a computation asynchronously and provides chaining via `Then` — enabling sequential async composition without nested callbacks.**

## The Problem It Solves

You're building a user profile page that needs data from three microservices: the Auth service (validate the user), the Profile service (name, email, avatar URL), and the Activity service (recent posts). Each call takes ~30ms over the network. Done sequentially, that's 90ms of wall-clock time. Done in parallel, it's 30ms. But the Profile call depends on the Auth call's result (you need the validated user ID), and the Activity call depends on the Profile result (you need the user's handle). You have a dependency chain, not independent parallelism.

Without a Promise abstraction, you'd write nested callbacks: the Auth callback launches the Profile request, whose callback launches the Activity request, whose callback renders the page. Three levels of nesting, each with its own error handling (did Auth fail? did Profile fail?), and the code reads inside-out. If you later need to add a fourth step, you increase the nesting level. This is callback hell — the problem that JavaScript Promises, Java CompletableFuture, and C++ std::future were all designed to solve.

A Promise (or Future) represents a value that will be available at some point in the future. You can chain operations on it with `Then` — a method that takes the resolved value of one promise and produces a new promise. Error propagation is built in: if any step in the chain fails, subsequent steps are skipped and the error surfaces at the final `Await`. The code reads linearly (or at least as a flat chain) rather than as nested blocks.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────┐
│                       Future[T]                               │
│                                                              │
│   Async(fn) ───▶ goroutine ───▶ fn() ───▶ result{T, error}  │
│                      │                          │            │
│                      │                          ▼            │
│                      │               ┌──────────────────┐   │
│                      │               │ done chan result  │   │
│                      │               │   (buffered 1)    │   │
│                      │               └────────┬─────────┘   │
│                      │                        │              │
│                      ▼                        ▼              │
│                return *Future           Await() blocks       │
│                                              │               │
│                                              ▼               │
│              Then(future, next):        res := <-done        │
│   ┌──────────────────────────────┐     return res.v, res.err │
│   │ return Async(func() {        │                            │
│   │   value, err := future.Await()│   ┌─────────────────────┐ │
│   │   if err != nil {            │   │  Chaining Example:   │ │
│   │     return zero, err         │   │                     │ │
│   │   }                          │   │  Auth()              │ │
│   │   return next(value)         │   │    .Then(Profile)    │ │
│   │ })                           │   │    .Then(Activity)   │ │
│   └──────────────────────────────┘   │    .Await()          │ │
│                                      └─────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

The `Future[T]` struct (`main.go:14`) is a minimal container: just a buffered channel `done chan result[T]` of capacity 1. The buffering is critical — it allows the goroutine executing `fn()` to send the result and exit immediately, even if no one is listening yet. Without the buffer, the goroutine would block forever if `Await` is never called.

`Async[T]` (`main.go:24-32`) wraps a function `fn() (T, error)` and launches it in a goroutine. The goroutine calls `fn()`, then sends the result into `done`. The caller gets back a `*Future[T]` and can continue doing other work. When the caller is ready, `Await()` (`main.go:36-44`) blocks on `<-f.done`, returning the value and error.

The power is in `Then` (`main.go:49-59`). It takes an existing `*Future[T]` and a transformation function `func(T) (U, error)`, and returns a new `*Future[U]`. Internally, it calls `Async` with a closure that first `Await`s the previous future. If the previous future errored, `Then` returns a zero value for `U` and propagates the error — short-circuiting the chain. If the previous future succeeded, it calls `next(value)` and returns the result. This creates a dependency chain: each step starts only after the previous step completes.

## Production Use Cases

- **JavaScript Promises** — The most widely deployed Promise implementation. ECMAScript 2015 standardized `Promise.then()`, `Promise.catch()`, and `Promise.all()`. Every modern web application uses them, and Node.js 16+ includes `fs/promises` for all filesystem operations. The `await` keyword in async functions is syntactic sugar over `.then()` chains.
- **Java CompletableFuture** — Java 8's `CompletableFuture<T>` implements both `Future<T>` and `CompletionStage<T>`, providing `thenApply`, `thenCompose`, `thenCombine`, and 50+ other methods for async composition. Used throughout Spring WebFlux, Apache Kafka Streams, and any Java service making concurrent async calls.
- **C++ std::future** — The C++11 standard library includes `std::future<T>` and `std::promise<T>`, with `std::async` for launch. C++20 added coroutines that interoperate with futures. Used in high-frequency trading systems where async composition with zero-allocation paths is critical.
- **Scala Future** — Scala's `scala.concurrent.Future` powers the Akka actor framework and the Play web framework. Its `map`, `flatMap`, and `recover` methods make it a monadic type, composable in for-comprehensions for clean async code.
- **Finagle (Twitter)** — Twitter's RPC framework uses `com.twitter.util.Future` extensively. Finagle futures support cancellation, interruption, and `Future.join` for parallel composition. Every RPC call in Twitter's microservices fabric returns a Finagle Future.

## When to Use It

| Scenario | Use Future/Promise? |
|----------|--------------------|
| Composing sequential async operations where each depends on the previous | **Yes** |
| Need to launch work now and collect the result later | **Yes** |
| Building a pipeline of async transforms (fetch → parse → validate → store) | **Yes** |
| Independent parallel operations that should run concurrently | **No** — use `sync.WaitGroup` or just launch goroutines with a results channel |
| Single async operation with no chaining | **No** — a plain goroutine + channel is more idiomatic Go |
| High-throughput systems where allocation overhead of Futures matters | **No** — Go's goroutine-per-op model with raw channels avoids the Future allocation |

**Alternatives**: **Raw goroutines + channels** are the idiomatic Go pattern — launch a goroutine, have it send results on a channel. More verbose for chaining but zero abstraction overhead. **`sync.WaitGroup` + shared slice** is better for parallel batch operations where you need all results before proceeding. **`errgroup`** (golang.org/x/sync/errgroup) provides parallel execution with error propagation — ideal for independent parallel calls.

## Complexity Analysis

| Operation | Time | Space |
|-----------|------|-------|
| Async | O(1) launch | O(1) — one channel, one goroutine |
| Await | O(fn) — depends on computation | O(1) |
| Then | O(1) launch + O(prev + next) resolution | O(1) per chain link |

Each `Async` call allocates one `Future` struct (a single channel pointer) and one `result` struct. Each `Then` creates a new closure that captures the previous future — Go closures capture by reference, so there's no deep copy. The goroutine overhead is Go's standard ~2-4KB per goroutine stack.

## Implementation Deep Dive

**1. The buffered channel (capacity 1) decouples producer and consumer lifetimes.** At `main.go:25`, `make(chan result[T], 1)` creates a buffered channel. This is a deliberate choice: the goroutine executing `fn()` sends the result and exits, regardless of whether anyone has called `Await`. An unbuffered channel would require the sender to block until a receiver is ready — if the caller never calls `Await`, the sender goroutine leaks forever. The single-slot buffer is the minimal memory cost for this decoupling.

**2. Error propagation in `Then` is a short-circuit.** The closure inside `Then` (`main.go:50-58`) checks `if err != nil` after awaiting the previous future and returns immediately with a zero value. This means errors cascade through the chain: if `Auth()` fails, neither `Profile()` nor `Activity()` is ever invoked. The final `Await` sees the original error. This is the same semantics as JavaScript's `.then(onFulfilled, onRejected)` when `onRejected` is absent — the rejection passes through.

**3. The generic [T, U] signature on `Then` enables type-changing chains.** The signature `func Then[T any, U any](future *Future[T], next func(T) (U, error)) *Future[U]` means the chain can start with `Future[int]` (user ID), become `Future[string]` (profile JSON), and then become `Future[*Profile]` (parsed struct). Each transformation changes the type, and the compiler enforces correctness — you can't accidentally chain a `func(string)` onto a `Future[int]`.

## Running the Demo

```bash
go run ./14-Promise/
```

## Further Reading

- "Promises: Linguistic Support for Efficient Asynchronous Procedure Calls" — Barbara Liskov and Liuba Shrira (1988). The seminal paper that introduced the Promise construct. The term "Promise" in computer science traces directly to this paper, published well before JavaScript or Go existed.
- "Futures and Promises" — Scala Documentation. Provides the clearest treatment of the Future/Promise distinction: a Future is a read-only handle to a computation result; a Promise is a write-once container that completes a Future. The paper covers composition semantics, execution contexts, and error models.

---

*Part of the Design-With-TsGo system design curriculum*
