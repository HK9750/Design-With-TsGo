# Distributed Transaction (Saga)

> **The Saga pattern — a sequence of local transactions with compensating actions for rollback — provides eventual consistency across microservices without distributed locks or two-phase commit.**

## The Problem It Solves

You're the architect at an e-commerce company, and a customer clicks "Place Order." This single action must: reserve inventory in the Warehouse Service, charge the customer's credit card via the Payment Service, create a shipment in the Logistics Service, and send a confirmation email via the Notification Service. Each of these is a separate microservice with its own database. If the first three steps succeed but the fourth fails, you've charged the customer's card for an order that doesn't have inventory reserved, a shipment that shouldn't exist, and no confirmation. Your reconciliation team will spend the next week manually refunding charges and canceling phantom shipments.

The naive solution is a distributed transaction via two-phase commit (2PC) — lock all resources, prepare all participants, then commit or abort atomically. But 2PC requires all services to hold locks for the duration of the transaction, which can be seconds or minutes across unreliable networks. If the coordinator crashes during the "prepare" phase, all participants are blocked indefinitely — the infamous "2PC blocking problem." In a microservices world where services are owned by different teams and databases are purpose-built (Postgres for orders, MongoDB for inventory, Stripe for payments), 2PC is operationally impractical.

The Saga pattern solves this with a fundamentally different approach: break the business transaction into a sequence of local transactions, each paired with a *compensating action* (a semantic undo). Execute them in order. If any step fails, run the compensations for all previously completed steps *in reverse order*. There are no distributed locks, no two-phase coordination, and no blocking. Each service manages its own local transaction autonomously. The tradeoff is that between the failure and the completion of compensations, the system is in a temporarily inconsistent state — but that's acceptable for most business workflows where a few seconds of inconsistency is far better than indefinite lock holding.

## Architecture & Internals

```
┌──────────────────────────────────────────────────────────────┐
│                   Saga Coordinator                            │
│                                                              │
│  steps: [                                                    │
│    { Name: "reserve-inventory", Action, Compensate },        │
│    { Name: "charge-payment",    Action, Compensate },        │
│    { Name: "create-shipment",   Action, Compensate },        │
│    { Name: "send-confirmation", Action, Compensate }         │
│  ]                                                           │
│                                                              │
│  SUCCESS FLOW:                                               │
│                                                              │
│  reserve-inventory ──► charge-payment ──► ... ──► DONE ✓    │
│  (each step returns nil)                                     │
│                                                              │
│  FAILURE FLOW (step 3 fails):                                │
│                                                              │
│  reserve-inventory ──► charge-payment ──► create-shipment   │
│       ✓                      ✓                  ✗ FAIL!     │
│                                 │                            │
│                                 ▼                            │
│              ┌─────────────────────────────┐                 │
│              │    COMPENSATION (reverse)    │                 │
│              │  ◄── payment-refunded       │                 │
│              │  ◄── inventory-released     │                 │
│              └─────────────────────────────┘                 │
│                                                              │
│  KEY PROPERTY: compensations run in reverse order of         │
│  execution. This preserves causal dependencies — you         │
│  can't release inventory before refunding the payment        │
│  that reserved it.                                           │
└──────────────────────────────────────────────────────────────┘
```

A `SagaStep` (`main.go:15-19`) is a struct with three fields: `Name` (a label), `Action` (the forward function returning `error`), and `Compensate` (the rollback function returning `error`). The `SagaCoordinator` (`main.go:24`) simply holds a slice of steps.

**On `Execute()`** (`main.go:37-59`): The coordinator iterates through steps in order. For each step, it calls `step.Action()`. On success, the step is appended to a `completed` list. On failure, the coordinator iterates the `completed` list *in reverse order* (`main.go:45: for j := len(completed) - 1; j >= 0; j--`) and calls each step's `Compensate()` function. The reverse order is essential — it respects the dependency graph of the forward operations. You must refund the payment before releasing the inventory, because the payment charge logically depends on the inventory reservation.

The entire execution is synchronous and blocking — the coordinator waits for each step and each compensation to complete. In production sagas, you'd typically make this asynchronous with a message queue, but the logical flow is identical.

## Production Use Cases

- **Uber payments** — Uber's payment system uses a saga-based approach for trip transactions. A single trip involves: charging the rider, paying the driver, calculating Uber's commission, and handling promotions. If any step fails (e.g., the rider's card is declined after the trip), compensating actions reverse the partial charges. Uber open-sourced Cadence, a workflow engine that implements sagas at massive scale.
- **AWS Step Functions** — Step Functions provides a serverless saga implementation through its state machine DSL. You define a sequence of Lambda invocations with automatic retry and compensation logic. Used by thousands of AWS customers for order processing, data pipelines, and microservice orchestration.
- **Apache Camel Saga** — Camel's Saga EIP (Enterprise Integration Pattern) provides a saga implementation with a `Saga` DSL that automatically triggers compensating routes on failure. Used in Java-based microservice architectures for financial transactions.
- **Eventuate Tram** — Chris Richardson's Eventuate Tram framework implements sagas for microservices using event-driven choreography. Each service publishes events that the next service consumes, with compensating events for rollback. The canonical example is the "Customers and Orders" saga from Microservices.io.
- **Microsoft Azure Durable Functions** — Durable Functions support the saga pattern through orchestrator functions that sequence activity functions and automatically handle compensation. Used by enterprises for long-running business workflows like insurance claim processing.

## When to Use It

| Scenario | Use Saga? |
|----------|----------|
| Your business transaction spans 3+ microservices with different databases | **Yes** |
| Steps are naturally independent and can be compensated semantically | **Yes** |
| You can tolerate seconds of temporary inconsistency (eventual consistency) | **Yes** |
| You need the transaction to survive coordinator crashes | **Yes** (with persistent saga log) |
| All steps must complete or none must appear to have happened (atomicity) | **Maybe** — sagas provide eventual, not immediate, atomicity |
| You need strict isolation (no other transaction can see partial saga state) | **No** — use 2PC (37-Two-Phase-Commit) or a monolithic database |
| Compensation is impossible (e.g., sending an email can't be undone) | **No** — at least send a correction email as compensation |
| Your transaction has only 1-2 steps | **No** — just use a local database transaction |

**Alternatives**: **Two-phase commit** (37-Two-Phase-Commit) provides immediate atomicity but blocks during coordinator failures and requires all participants to support the XA protocol. **Eventual consistency with retry** means no compensation — just retry failed steps indefinitely; works when steps are idempotent and order doesn't matter. **Routing slip pattern** passes a "routing slip" through services, each performing their step and forwarding; compensations still apply but without a central coordinator. The Saga's core advantage is that it doesn't require any special infrastructure — just ordinary local transactions in each service.

## Complexity / Behavior Characteristics

| Operation | Time Complexity | Failure Behavior |
|-----------|---------------|-------------------|
| Execute (all success) | O(n × T_action) | N/A |
| Execute (step k fails) | O(k × T_action + k × T_compensate) | All k completed steps compensated |
| Compensation failure | Unbounded | Compensation failure is logged, saga continues compensating remaining steps |

The worst case for a failure is O(n × (T_action + T_compensate)) — every step succeeds, the last step fails, and every compensation must run. The coordinator at `main.go:48` logs compensation failures but does NOT retry them. In production, you'd implement a compensation retry with exponential backoff, and if compensation still fails, escalate to a human operator or a dead-letter queue.

## Implementation Deep Dive

**1. Reverse-order compensation preserves causal dependencies.** The loop at `main.go:45-50` iterates `completed` from right to left: `for j := len(completed) - 1; j >= 0; j--`. This is not arbitrary — it reflects the dependency chain of the forward operations. If step 2 (charge payment) depends on step 1 (reserve inventory), then the compensation for step 1 (release inventory) must logically follow the compensation for step 2 (refund payment). If you compensated in the same order (forward), you'd release inventory while the payment was still charged — a broken state. Reverse compensation guarantees that each step's compensation runs in an environment where all "later" effects have already been undone.

**2. Fail-on-error semantics for action vs. best-effort for compensation.** The action functions return errors that halt the saga (`main.go:42` checks `err != nil` and triggers compensation). But compensation errors are logged and the saga continues (`main.go:48-49`). The rationale: if an action fails, forward progress is impossible. But if a compensation fails, *not* compensating the remaining steps would leave the system even more inconsistent. For example, if the payment refund fails but you stop, the inventory is still reserved for an order that can't be paid. It's better to release the inventory too and flag both for manual reconciliation.

**3. Completed steps are tracked in an ordered slice, not a set.** The `completed` slice at `main.go:39` stores steps in execution order. Using a slice preserves the order that `Reverse` iteration depends on. A map or set would lose ordering. The slice is pre-allocated to capacity `len(s.steps)` to avoid reallocations during the happy path — most sagas succeed and don't need compensation, so this is a pragmatic optimization.

## Running the Demo

```bash
go run ./36-Distributed-Transaction/
```

## Further Reading

- "Sagas" — Hector Garcia-Molina and Kenneth Salem (1987). The original paper that defined the saga abstraction for long-lived transactions. *Proceedings of ACM SIGMOD 1987.*
- "Pattern: Saga" — Chris Richardson, Microservices.io. Comprehensive guide to implementing sagas in microservices, including choreography vs. orchestration tradeoffs. *microservices.io/patterns/data/saga.html*

---

*Part of the Design-With-TsGo system design curriculum*
