# Design - Transactional Outbox demo

A short, practical design doc for this project. It explains what we are
building, why, and how the pieces fit together - without assuming you already
know the Transactional Outbox pattern.

---

## 1. What this does

Two PHP order services, each with its own private MySQL database. When you
place an order on service-a, it eventually appears on service-b too (and vice
versa), without the two services ever sharing a database or a distributed
transaction.

This is the demo described in:

- **Scott Logic - "Solving Data Consistency in Distributed Systems with the
  Transactional Outbox"** (September 2025)
  https://blog.scottlogic.com/2025/09/08/solving-data-consistency-in-distributed-systems-with-the-transactional-outbox.html

---

## 2. The problem: the dual-write trap

Imagine service-a receives an order. It needs to:

1. Save the order to its own MySQL.
2. Tell service-b about the order over HTTP.

These two steps **cannot** happen atomically:

- If you save first and the HTTP call fails, service-a has the order but
  service-b does not. Silent inconsistency.
- If you call service-b first and the save fails, service-b has an order that
  service-a does not know about. Also broken.

Because a MySQL commit and an HTTP request to another service can never join
one transaction, there is no correct non-outbox solution to this.

---

## 3. The solution: Transactional Outbox

Instead of writing to the database and calling the peer separately, service-a
does everything inside **one local database transaction**:

| Step | What happens |
|------|--------------|
| 1 | Insert the order into the `orders` table |
| 2 | Insert a message about that order into the `outbox` table |
| 3 | Commit both together |

If the commit succeeds, the order **and** the message are durable together.
The message is now waiting in the outbox for the sync-worker to pick it up.

### What each piece is

| Element | Implementation |
|---------|----------------|
| Data record | `orders` row in the service's own MySQL |
| Message payload | `outbox` row (JSON, type `OrderCreated`) |
| Atomic commit | Single PDO transaction in PHP (`POST /orders`) |
| Messaging service | Go `sync-worker`: polls outbox, POSTs to peer, marks processed |
| Downstream call | HTTP `POST /orders` on the peer service |
| Idempotency | `UNIQUE(order_number)` + `ON DUPLICATE KEY UPDATE` |

---

## 4. Data flow

```text
Client creates order on service-a
        │
        ▼
  PHP begins a DB transaction
  ├── INSERT into orders (source = "local")
  └── INSERT into outbox  (pending)
        │
        └── COMMIT  (order + message are now durable together)
                  │
                  ▼
      sync-worker polls "GET /outbox?processed=0"
                  │
                  ├── POST payload to service-b's "/orders"  (header: X-Synced: true)
                  │       → service-b stores order (source = "synced")
                  │       → service-b does NOT write its own outbox row (no loop)
                  │
                  └── POST "POST /outbox/{id}/process" on service-a
                          → marks the outbox row as processed
```

The same flow runs in reverse for orders created on service-b.

---

## 5. Idempotency (crash safety)

The worker delivers first, then marks processed. If it crashes between those
two steps, the same message is delivered again on the next poll. That is fine
because the receiving side is **idempotent**:

```sql
INSERT INTO orders (...) VALUES (...)
ON DUPLICATE KEY UPDATE customer_name = VALUES(customer_name), ...
```

`order_number` is unique, so redelivering `ORD-001` simply updates the same
row - the order appears exactly once on each service.

| Crash scenario | Result |
|----------------|--------|
| Delivered, crashed before marking processed | Redelivered; receiver ignores the duplicate |
| Peer is down (5xx) | Not marked processed; retried on next poll |
| Mark-processed request fails | Same as above - safe to retry |

This matches the article's warning: messages can be delivered more than once,
so design your system to handle duplicates.

---

## 6. Loop prevention

When service-b receives an order with the header `X-Synced: true`, it stores
the order but does **not** append a new outbox row. Without this, every synced
order would generate a new outbox event, which the worker would ship back to
service-a - an infinite loop (`a → b → a → b → …`).

The `source` column (`local` vs `synced`) records where each order originated.

---

## 7. Why Go for the sync-worker?

PHP owns the transactional boundary (orders + outbox in one MySQL transaction).
The sync-worker owns the **long-running background relay**, and Go fits that
job better than PHP:

| | Go (sync-worker) | PHP equivalent |
|---|---|---|
| Process model | Compiled binary that runs indefinitely as a daemon | Request/response runtime; fighting the lifecycle to run a `while(true)` loop |
| Concurrency | Goroutines make parallel polling cheap | Requires `pcntl`/`pthreads` or an external job queue |
| Deployment | One static binary (~10–15 MB), minimal image | PHP interpreter + web server + extensions + a process manager |
| Long-running stability | Predictable GC and memory over days/weeks | Not optimized for continuously-running processes |
| JSON payloads | `encoding/json` with structs - compile-time safety | `json_decode` returns arrays - typos surface at runtime |

PHP *can* do this (a CLI loop supervised by systemd), but it works against the
language's design. Each tool is used where it fits best.

---

## 8. Schema

Both MySQL databases use the same two tables:

**orders** - one row per order
```sql
CREATE TABLE orders (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    order_number VARCHAR(64) NOT NULL,
    customer_name VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'new',
    source VARCHAR(16) NOT NULL DEFAULT 'local',   -- 'local' | 'synced'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_order_number (order_number)      -- idempotency key
) ENGINE=InnoDB;
```

**outbox** - one pending message per event
```sql
CREATE TABLE outbox (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    aggregate_type VARCHAR(64) NOT NULL DEFAULT 'Order',
    aggregate_id VARCHAR(64) NOT NULL,             -- order_number
    type VARCHAR(64) NOT NULL,                     -- 'OrderCreated'
    payload JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP NULL DEFAULT NULL,      -- NULL = pending
    processing_attempts INT UNSIGNED NOT NULL DEFAULT 0,
    KEY idx_outbox_pending (processed_at, created_at)
) ENGINE=InnoDB;
```

---

## 9. Alternatives considered

The article also covers two alternatives to the outbox pattern:

- **CDC (Change Data Capture)** - reads the database binlog instead of an
  outbox table. Simpler for existing systems (no schema changes), but the
  messaging service becomes tightly coupled to the data schema, and it cannot
  capture *who* or *why* an action happened. Rejected here because the
  producing service should own the message payload explicitly.
- **Event Sourcing** - events are the source of truth. Avoids dual-write
  internally but needs an event store, aggregates, and read models (CQRS).
  Overkill for this demo.

---

## 10. Known limitations

- **Single worker instance.** One `sync-worker` replica polls both outboxes.
  Multiple replicas would need a claiming/locking mechanism (e.g. `SELECT ... FOR UPDATE` with a `locked_by` column) to avoid duplicate processing.
- **No ordering guarantees beyond outbox id.** Cross-service causal ordering
  is not guaranteed; acceptable for order-number sync.
- **No dead-letter queue.** Repeatedly failing events increment
  `processing_attempts` and are retried forever. Production would move them to
  a dead-letter table after N attempts.

---

## 11. Run it

```bash
docker compose up --build
```

See `README.md` for curl examples and teardown instructions.
