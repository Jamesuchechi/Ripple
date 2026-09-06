# TODO — Build Phases

Status legend: `[ ]` not started · `[~]` in progress · `[x]` done

## Phase 0 — Project setup
- [x] Init Go module, repo structure (`cmd/api`, `cmd/worker`, `cmd/notifier`, `internal/`, `pkg/`)
- [x] Init `dashboard/` (React/Vite)
- [x] `.env.example`, `docker-compose.yml` (Postgres, Redis, NATS)
- [x] CI pipeline: lint (`golangci-lint`), unit test, build on push
- [x] Database migration tooling (`golang-migrate`)


## Phase 1 — Activity Schema & Ingest Path
- [x] Postgres schema: `projects`, `api_keys`, `users`, `follows`, `event_log`
- [x] Actor-Verb-Object-Target activity payload parser
- [x] `POST /v1/events` — validate, write to Postgres `event_log`, return `202 Accepted`
- [x] `DELETE /v1/events/:eventID` — activity retraction & un-fanout trigger
- [x] Direct synchronous fanout-on-write (baseline end-to-end test flow)
- [x] `GET /v1/feed/:userID` — read from Redis ZSET feed
- [x] Integration test: post event $\rightarrow$ verify appearance in recipient feed

## Phase 2 — Decoupled Ingest & Queue Worker
- [x] Stand up NATS JetStream container locally
- [x] Ingest API publishes event to NATS JetStream subject (`events.{tenant_id}.*`)
- [x] Worker service (`cmd/worker`) consumes from NATS JetStream queue
- [x] Bounded-concurrency fanout worker pool (semaphore-limited goroutines)
- [x] Idempotent feed writes using activity ID as ZSET member
- [x] Feed size cap enforcement (`FEED_CACHE_SIZE` ZSET trimming)
- [x] Idempotency deduplication lock (`dedup:{tenant_id}:{key}`)

## Phase 3 — Hybrid Fanout (Celebrity Split)
- [x] `is_celebrity` flag in `users` table + configurable threshold (`CELEBRITY_THRESHOLD`)
- [x] Fanout worker skips celebrity posts during write path
- [x] Read path: merge cached ZSET feed + live celebrity post query
- [x] Short-TTL cache for celebrity query results (`celeb_posts:{tenant_id}:{user_id}`)
- [x] Periodic threshold reconciliation job (detect accounts crossing threshold in either direction)
- [x] Load test: simulate 100k-follower account post, verify ingest API latency remains $O(1)$

## Phase 4 — Real-time Push, Unread Counters & Badging
- [x] Real-time WebSocket hub (`cmd/notifier`)
- [x] `WS /v1/ws/:userID` connection handler with heartbeats
- [x] Redis Pub/Sub subscriber (`ws_pubsub:{tenant_id}:{user_id}`) across notifier instances
- [x] Atomic unread counters in Redis (`unread:{tenant_id}:{user_id}`)
- [x] `GET /v1/notifications/:userID/unread_count` endpoint
- [x] `POST /v1/notifications/mark_read` endpoint
- [x] Offline notification fallback retrieval endpoint

## Phase 5 — Multi-Channel Dispatch Engine & Webhook Security
- [x] Channel dispatcher interface (`ChannelDispatcher` & `ProviderAdapter`)
- [x] Webhook channel dispatcher with HMAC-SHA256 signature generation (`X-Ripple-Signature`)
- [x] Native Push dispatcher stubs (FCM / APNs)
- [x] Email dispatcher stubs (SendGrid / Postmark / AWS SES)
- [x] SMS dispatcher stubs (Twilio)
- [x] Channel dispatcher unit test suite

## Phase 6 — Notification Aggregation & Batching Engine
- [x] Time-windowed sliding aggregation engine in Redis (`batch:{tenant_id}:{recipient_id}`)
- [x] Notification coalescing logic ("Alice and 14 others liked your post")
- [x] Liquid template parser integration (`osteele/liquid` or equivalent) for email/push formatting
- [x] Template resolution pipeline (`templates` table in Postgres)

## Phase 7 — User Preferences & Dead Letter Queue (DLQ)
- [x] User notification preference evaluation engine (`user_preferences` table)
- [x] Quiet hours / Do-Not-Disturb (DND) window check against recipient timezone
- [x] Channel dispatch exponential backoff retries (max retries = `MAX_DLQ_RETRIES`)
- [x] DLQ table (`dlq_messages`) & failure router
- [x] Admin API DLQ inspection (`GET /v1/admin/projects/:projectID/dlq`)
- [x] Admin API DLQ replay (`POST /v1/admin/projects/:projectID/dlq/replay`)

## Phase 8 — Multi-Tenant Auth & API Keys
- [x] API key generation, SHA-256 hashing, storage (`api_keys` table)
- [x] Bearer auth middleware on `/v1/*` endpoints resolving `project_id`
- [x] Strict Redis key and NATS subject namespacing by `project_id`
- [x] Token Bucket rate limiter per API key (`ratelimit:{tenant_id}:{key}`)
- [x] Multi-tenant security isolation test: verify tenant data leakage is impossible

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

**Current phase:** Phase 3 — Hybrid Fanout (Celebrity Split) (Completed).