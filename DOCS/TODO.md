# TODO — Build Phases

Status legend: `[ ]` not started · `[~]` in progress · `[x]` done

## Phase 0 — Project setup
- [x] Init Go module, repo structure (`cmd/api`, `cmd/worker`, `cmd/notifier`, `internal/`, `pkg/`)
- [x] Init `dashboard/` (React/Vite)
- [x] `.env.example`, `docker-compose.yml` (Postgres, Redis, NATS)
- [x] CI pipeline: lint (`golangci-lint`), unit test, build on push
- [x] Database migration tooling (`golang-migrate`)


## Phase 1 — Activity Schema & Ingest Path
- [ ] Postgres schema: `projects`, `api_keys`, `users`, `follows`, `event_log`
- [ ] Actor-Verb-Object-Target activity payload parser
- [ ] `POST /v1/events` — validate, write to Postgres `event_log`, return `202 Accepted`
- [ ] `DELETE /v1/events/:eventID` — activity retraction & un-fanout trigger
- [ ] Direct synchronous fanout-on-write (baseline end-to-end test flow)
- [ ] `GET /v1/feed/:userID` — read from Redis ZSET feed
- [ ] Integration test: post event $\rightarrow$ verify appearance in recipient feed

## Phase 2 — Decoupled Ingest & Queue Worker
- [ ] Stand up NATS JetStream container locally
- [ ] Ingest API publishes event to NATS JetStream subject (`events.{tenant_id}.*`)
- [ ] Worker service (`cmd/worker`) consumes from NATS JetStream queue
- [ ] Bounded-concurrency fanout worker pool (semaphore-limited goroutines)
- [ ] Idempotent feed writes using activity ID as ZSET member
- [ ] Feed size cap enforcement (`FEED_CACHE_SIZE` ZSET trimming)
- [ ] Idempotency deduplication lock (`dedup:{tenant_id}:{key}`)

## Phase 3 — Hybrid Fanout (Celebrity Split)
- [ ] `is_celebrity` flag in `users` table + configurable threshold (`CELEBRITY_THRESHOLD`)
- [ ] Fanout worker skips celebrity posts during write path
- [ ] Read path: merge cached ZSET feed + live celebrity post query
- [ ] Short-TTL cache for celebrity query results (`celeb_posts:{tenant_id}:{user_id}`)
- [ ] Periodic threshold reconciliation job (detect accounts crossing threshold in either direction)
- [ ] Load test: simulate 100k-follower account post, verify ingest API latency remains $O(1)$

## Phase 4 — Real-time Push, Unread Counters & Badging
- [ ] Real-time WebSocket hub (`cmd/notifier`)
- [ ] `WS /v1/ws/:userID` connection handler with heartbeats
- [ ] Redis Pub/Sub subscriber (`ws_pubsub:{tenant_id}:{user_id}`) across notifier instances
- [ ] Atomic unread counters in Redis (`unread:{tenant_id}:{user_id}`)
- [ ] `GET /v1/notifications/:userID/unread_count` endpoint
- [ ] `POST /v1/notifications/mark_read` endpoint
- [ ] Offline notification fallback retrieval endpoint

## Phase 5 — Multi-Channel Dispatch Engine & Webhook Security
- [ ] Channel dispatcher interface (`ChannelDispatcher` & `ProviderAdapter`)
- [ ] Webhook channel dispatcher with HMAC-SHA256 signature generation (`X-Ripple-Signature`)
- [ ] Native Push dispatcher stubs (FCM / APNs)
- [ ] Email dispatcher stubs (SendGrid / Postmark / AWS SES)
- [ ] SMS dispatcher stubs (Twilio)
- [ ] Channel dispatcher unit test suite

## Phase 6 — Notification Aggregation & Batching Engine
- [ ] Time-windowed sliding aggregation engine in Redis (`batch:{tenant_id}:{recipient_id}`)
- [ ] Notification coalescing logic ("Alice and 14 others liked your post")
- [ ] Liquid template parser integration (`osteele/liquid` or equivalent) for email/push formatting
- [ ] Template resolution pipeline (`templates` table in Postgres)

## Phase 7 — User Preferences & Dead Letter Queue (DLQ)
- [ ] User notification preference evaluation engine (`user_preferences` table)
- [ ] Quiet hours / Do-Not-Disturb (DND) window check against recipient timezone
- [ ] Channel dispatch exponential backoff retries (max retries = `MAX_DLQ_RETRIES`)
- [ ] DLQ table (`dlq_messages`) & failure router
- [ ] Admin API DLQ inspection (`GET /v1/admin/projects/:projectID/dlq`)
- [ ] Admin API DLQ replay (`POST /v1/admin/projects/:projectID/dlq/replay`)

## Phase 8 — Multi-Tenant Auth & API Keys
- [ ] API key generation, SHA-256 hashing, storage (`api_keys` table)
- [ ] Bearer auth middleware on `/v1/*` endpoints resolving `project_id`
- [ ] Strict Redis key and NATS subject namespacing by `project_id`
- [ ] Token Bucket rate limiter per API key (`ratelimit:{tenant_id}:{key}`)
- [ ] Multi-tenant security isolation test: verify tenant data leakage is impossible

## Phase 9 — Reliability, Observability & Metrics
- [ ] Prometheus metrics: fanout latency histogram, queue depth gauge, delivery errors counter, DLQ total counter, WS connections gauge
- [ ] Grafana starter dashboard (`deploy/grafana/`)
- [ ] Structured JSON logging with trace/request ID propagation through queue $\rightarrow$ worker
- [ ] Alerting rules for queue lag and delivery error rates
- [ ] Chaos test suite: kill worker mid-job, simulate Redis disconnect, verify zero dropped events

## Phase 10 — Customer Dashboard (UI Control Plane)
- [ ] Sign-up / Login flow (Session / OAuth)
- [ ] Project creation & API Key management interface
- [ ] Live delivery log table (filterable by event type, status, channel)
- [ ] Real-time metrics view (throughput, queue depth, p99 latency, connected clients)
- [ ] Visual Workflow Builder (map trigger events to channels & batch windows)
- [ ] Interactive Notification Template Editor with Liquid syntax live preview
- [ ] DLQ Inspector & 1-click batch replay UI
- [ ] User Preference inspector UI
- [ ] Usage view (event volume, delivery counts for billing tiers)

## Phase 11 — Developer Experience (DX), SDKs & Release
- [ ] JavaScript/TypeScript client SDK (`@ripple/js`) for WebSockets and badging
- [ ] React UI component library / hooks (`@ripple/react`)
- [ ] Local developer CLI simulator (`ripple-cli trigger --event ...`)
- [ ] Automated end-to-end `k6` load test suite with published performance benchmarks
- [ ] `SECURITY.md`, `CONTRIBUTING.md`, versioned `/v1/` API stability guarantees
- [ ] Tag `v1.0.0` release

---

**Current phase:** Phase 0 — not yet started.