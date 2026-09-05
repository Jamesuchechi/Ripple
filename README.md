# Ripple

**Notifications and feed fanout as a service — an enterprise-grade platform, powered by a Go engine, for developers who don't want to build complex fanout and notification pipelines themselves.**

Ripple solves one problem extremely well: when something happens (a post, a like, a comment, a signup, an order shipped), get it delivered to the right users across the right channels — in-app feeds, real-time WebSockets, native mobile push (FCM/APNs), email, SMS, and secure webhooks — fast, reliably, and at scale. Instead of every team building their own fanout and notification infrastructure, they integrate Ripple's API and manage everything through a modern customer dashboard: visual workflows, delivery logs, live metrics, notification templates, user preferences, and dead-letter queues.

It is not a social network. It is developer infrastructure with a rich control plane — built for modern engineering teams, similar in vision to Novu or Knock.

## Why this exists

Every feed- or notification-driven product eventually hits the same engineering wall:
1. Naive **"query everyone's posts on load"** fails to scale as user databases grow.
2. Naive **"write to every follower's feed synchronously"** collapses the moment a celebrity account posts or a viral notification event fires.
3. Naive **"send 500 emails/push alerts for 500 likes"** spams users, causing app uninstalls and email domain reputation damage.

Solving this — hybrid fanout, decoupled ingest, bounded-concurrency worker pools, notification batching/digesting, native channel adapters, atomic unread counters, and DLQ replay — is a difficult, distributed-systems challenge. Ripple packages a clean, high-performance Go implementation into a single multi-tenant service.

## Core capabilities

### Engine (Go)
- **Hybrid Fanout** — Fanout-on-write for standard accounts, fanout-on-read for high-follower ("celebrity") accounts, merged transparently at read time.
- **Decoupled Ingest** — Ingest API accepts events (`POST /v1/events`) in constant time ($O(1)$) and enqueues them onto a durable queue (NATS JetStream / Redis Streams).
- **Bounded-Concurrency Delivery** — Semaphore-bounded worker pools with backpressure so high-follower posts cannot exhaust system resources.
- **Multi-Channel Provider Architecture** — Plug-and-play dispatchers for In-App (WebSocket/ZSET), Native Mobile Push (FCM, APNs, WebPush), Email (SendGrid, Postmark, AWS SES), SMS (Twilio), and Webhooks with HMAC-SHA256 signatures (`X-Ripple-Signature`).
- **Notification Aggregation & Digesting** — Time-windowed batching (e.g. sliding 5-minute window: *"Alice and 14 others liked your post"*) to prevent user notification fatigue.
- **Atomic Unread Counters & Badging** — High-performance Redis atomic counters (`HINCRBY`/`BITFIELD`) for live unread notification counts per recipient.
- **Recipient Preferences & Opt-outs** — Built-in end-user channel preference evaluation before dispatching notifications.
- **Dead Letter Queue (DLQ) & Replay Engine** — Automatic routing of exhausted retries to DLQ storage with manual/automated batch replay APIs.
- **Idempotent by Design** — Deduplication keys (`dedup_key`) and ZSET scoring ensure safe retries with zero duplicate deliveries.
- **Multi-Tenant Scoping** — Tenant/project isolation across all database queries, cache namespaces, queue subjects, and API keys.

### Platform (Dashboard)
- Multi-project management & secure hashed API key generation (`pk_live_...`, `pk_test_...`).
- Visual Workflow Builder — Map trigger events to multi-step channel pipelines with conditional rules.
- Interactive Notification Template Editor with Liquid syntax variable rendering.
- Real-Time Live Metrics — Queue depth, p99 fanout latency, delivery success rate, and active WebSocket connection count.
- Live Delivery Log — Searchable, filterable audit log of every event, recipient, channel status, and failure payload.
- DLQ Inspector — View failed deliveries, inspect failure trace logs, and trigger batch replays.
- End-User Preference Inspector — Manage tenant recipient notification channel rules.

## Tech stack

- **Engine:** Go 1.22+ — `chi` (HTTP API), `gorilla/websocket` (real-time push)
- **Queue:** NATS JetStream (or Redis Streams)
- **Cache & Feed Store:** Redis 7+ (namespaced sorted sets, atomic unread counters, pub/sub for WS)
- **Durable Store:** PostgreSQL 16+ (events, tenants, API keys, workflows, templates, DLQ, follow graph)
- **Dashboard:** React, Vite, Vanilla CSS design system
- **Metrics:** Prometheus client, Grafana dashboard for operational monitoring
- **Templating:** Liquid syntax parser (`osteele/liquid` or equivalent in Go)
- **Local Dev:** Docker Compose (Postgres, Redis, NATS, engine, worker, notifier, dashboard)

## Project status

Early build phase — see [`DOCS/TODO.md`](./DOCS/TODO.md) for the phase-by-phase build plan and progress tracking.

## Documentation

- [`ARCHITECTURE.md`](./DOCS/ARCHITECTURE.md) — Detailed system design, data flow, activity schema, batching logic, and DLQ handling
- [`DOCUMENTATION.md`](./DOCS/DOCUMENTATION.md) — API reference, environment configuration, database schemas, and integration guide
- [`TODO.md`](./DOCS/TODO.md) — Comprehensive 12-phase build plan

## Quick start

```bash
git clone <repo-url> ripple && cd ripple
cp .env.example .env
docker compose up -d       # Postgres, Redis, NATS
go run ./cmd/migrate up    # Run schema migrations
go run ./cmd/api            # Start Ingest + Read + Admin API (:8080)
go run ./cmd/worker         # Start Fanout & Delivery Worker pool
go run ./cmd/notifier       # Start WebSocket Real-time Hub (:8081)

# Dashboard
cd dashboard && npm install && npm run dev   # Start React Dashboard (:5173)
```

See [`DOCS/DOCUMENTATION.md`](./DOCS/DOCUMENTATION.md) for full configuration details and API reference.

## License

TBD.