# Roadmap

The next step after the CSV benchmark: a small order system that grows into a distributed setup one phase at a time. Each phase must work end to end before the next one starts.

The domain is orders. A PHP API takes orders, and a Go worker moves them between services. The starting point is the outbox demo in [`lab/transactional-outbox-order-sync`](lab/transactional-outbox-order-sync).

```
Phase 1          Phase 2           Phase 3          Phase 4            Phase 5
Core services -> Messaging and  -> Kubernetes    -> Observability   -> Load testing
                 outbox
```

## Phase 1: Core services

A PHP order API (REST) and a Go worker, each with its own MySQL database, running with Docker Compose. gRPC is added for internal calls only if a real need for it shows up.

## Phase 2: Messaging and reliability

Replace HTTP polling with a message broker (RabbitMQ first, Kafka maybe later). Keep the transactional outbox, so an order and its event are saved in one transaction and delivery is at least once even when the broker is down.

## Phase 3: Kubernetes

Run everything on a local cluster (`kind` or `minikube`), with Deployments, Services, ConfigMaps and Secrets.

## Phase 4: Observability

Instrument both services with OpenTelemetry. Collect metrics in Prometheus and build a Grafana dashboard. The key metric is outbox lag: the time from when an order is saved until another service has processed it.

## Phase 5: Load testing

Use k6 to push high order volumes through the system, watch backpressure in Grafana and record the results in the README.

Task checklists are in [`TASKS.md`](TASKS.md).
