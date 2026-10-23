# Tasks

See [`ROADMAP.md`](ROADMAP.md) for what each phase is about.

## Phase 1: Core services

- [ ] Move `lab/transactional-outbox-order-sync` into its own folder and clean it up
- [ ] PHP order API: `POST /orders`, `GET /orders`
- [ ] Go worker: read pending outbox rows and deliver them
- [ ] `make up` starts everything with Docker Compose
- [ ] Tests for the order API and the worker
- [ ] README: one curl command that shows an order being synced

## Phase 2: Messaging and reliability

- [ ] Add RabbitMQ to Docker Compose
- [ ] Worker publishes outbox events to RabbitMQ instead of calling HTTP
- [ ] Consumer stores orders idempotently (same event twice = one order)
- [ ] Test: stop the broker, create orders, start it again, nothing is lost

## Phase 3: Kubernetes

- [ ] Multi-stage Dockerfiles for the PHP API and the Go worker
- [ ] Local cluster with `kind`
- [ ] Deployment and Service manifests
- [ ] ConfigMap for settings, Secret for database passwords
- [ ] `make k8s` deploys everything from scratch

## Phase 4: Observability

- [ ] OpenTelemetry in the PHP API and the Go worker
- [ ] Prometheus in the cluster
- [ ] Grafana dashboard: request rate, error rate, outbox lag
- [ ] Dashboard screenshot in the README

## Phase 5: Load testing

- [ ] k6 script that creates orders at a fixed rate
- [ ] Run it against the cluster and raise the rate until something breaks
- [ ] Write down what broke first and why (database, broker, worker)
- [ ] Results table in the README
