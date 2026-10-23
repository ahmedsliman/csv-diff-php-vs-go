# Transactional Outbox Order Sync

Two PHP order services, each with its own MySQL database, kept in sync by a Go
worker using the **Transactional Outbox** pattern.

Based on:
[Scott Logic - Solving Data Consistency in Distributed Systems with the
Transactional Outbox](https://blog.scottlogic.com/2025/09/08/solving-data-consistency-in-distributed-systems-with-the-transactional-outbox.html)

---

## What it does

You place an order on service-a. It is saved to service-a's database **and** a
message is written to an `outbox` table in the same transaction. A Go worker
polls that outbox and replays the order to service-b. The same happens in
reverse. Neither service shares a database or uses distributed transactions.

Ports:

| Service | URL |
|---------|-----|
| service-a | http://localhost:8000 |
| service-b | http://localhost:8001 |

---

## Quick start

```bash
docker compose up --build
```

Wait for MySQL to become healthy, then create an order:

```bash
curl -s -X POST http://localhost:8000/orders \
  -H 'Content-Type: application/json' \
  -d '{"order_number":"ORD-001","customer_name":"Alice"}'
```

Within ~3 seconds (the worker's default poll interval), check that it arrived on service-b:

```bash
curl -s http://localhost:8001/orders
```

You should see `ORD-001` with `"source":"synced"`.

Create an order on service-b too - it flows back to service-a:

```bash
curl -s -X POST http://localhost:8001/orders \
  -H 'Content-Type: application/json' \
  -d '{"order_number":"ORD-002","customer_name":"Bob"}'
```

Stop:

```bash
docker compose down
```

Remove data as well:

```bash
docker compose down -v
```

---

## How it works (short version)

1. PHP service receives `POST /orders`.
2. It inserts the order **and** an outbox message in one database transaction.
3. The Go worker polls `GET /outbox?processed=0` on both services.
4. It POSTs each pending message to the other service's `/orders` endpoint.
5. The receiving service stores the order but does **not** write a new outbox
   row (prevents an infinite sync loop).
6. The worker marks the outbox entry as processed.

If the worker crashes after delivering but before marking processed, the
message is redelivered - safely, because orders use `UNIQUE(order_number)` and
`ON DUPLICATE KEY UPDATE`, so duplicates are ignored.

See `DESIGN.md` for the full design.

---

## Endpoints (each PHP service)

| Method | Path | Body / notes |
|--------|------|--------------|
| GET | `/health` | Service is up |
| POST | `/orders` | `{"order_number","customer_name"}` - creates order + outbox event atomically |
| GET | `/orders` | Lists all orders |
| GET | `/outbox?processed=0` | Pending outbox messages |
| GET | `/outbox?processed=1` | Processed outbox messages |
| POST | `/outbox/{id}/process` | Marks an outbox entry processed (used by the worker) |

---

## Project structure

```
transactional-outbox-order-sync/
├── docker-compose.yaml   # mysql-a, mysql-b, service-a, service-b, sync-worker
├── README.md             # this file
├── DESIGN.md             # full design doc
├── php-service/          # shared PHP image for both services
│   ├── Dockerfile
│   ├── index.php         # front controller: /orders, /outbox, /health
│   ├── .htaccess         # routes requests to index.php
│   └── init.sql          # orders + outbox tables
└── sync-worker/          # Go background sync service
    ├── go.mod
    └── main.go           # polls both outboxes, relays to peers
```

---

## Learn more

Full design and trade-offs are in `DESIGN.md`. The pattern comes from the
Scott Logic article linked at the top of this file.
