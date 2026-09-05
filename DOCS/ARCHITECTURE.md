# Architecture

## 0. Platform framing

Ripple is an enterprise-grade multi-tenant notification and feed service. Any number of customer projects (tenants) run on the same shared engine, isolated via API keys. Every key, database query, Redis cache entry, and queue subject is strictly namespaced by `tenant_id`.

## 1. Problem statement

Given an event from a customer's product (e.g. `post_created`, `comment_created`, `like_created`, `order_shipped`), deliver it into recipient feeds, in-app notification centers, native push alerts, email digests, and webhooks with low latency without:
- blocking the API write path on fanout/delivery,
- overwhelming datastores when high-follower accounts post,
- spamming recipients with duplicate or high-frequency alerts,
- dropping notifications on downstream provider outages,
- leaking traffic or state across tenants.

## 2. The core design decision: hybrid fanout

There are two naive fanout strategies for activity feeds, both of which fail at scale in opposite ways:

| Strategy | How it works | Fails when |
|---|---|---|
| Fanout-on-write | On post, immediately write into every follower's cached feed | A high-follower account posts → millions of writes for one event |
| Fanout-on-read | Do nothing on post; merge followees' posts at read time | A user follows thousands of accounts → every feed load is an expensive merge |

**Ripple uses both, split by follower count:**

- Accounts under the celebrity threshold (configurable, default 10,000 followers) $\rightarrow$ **fanout-on-write**. Background workers push the event into every follower's cached Redis ZSET feed.
- Accounts over the threshold $\rightarrow$ **fanout-on-read**. Posts are not fanned out to followers' Redis ZSETs. Instead, at read time, Ripple queries *"recent posts from followed celebrities"* (a cacheable, short-TTL query) and transparently merges it into the response.

## 3. Activity Data Model & Hydration

Events are structured using an **Actor-Verb-Object-Target** pattern (grounded in Activity Streams 2.0 specs):

```json
{
  "event_id": "evt_9f3a2c",
  "tenant_id": "proj_123",
  "verb": "comment_created",
  "actor_id": "usr_alice",
  "object_id": "cmt_789",
  "target_id": "post_abc",
  "recipients": ["usr_bob"],
  "payload": {
    "comment_text": "Great explanation!",
    "post_title": "Hybrid Fanout Architecture"
  },
  "dedup_key": "cmt_789:usr_bob",
  "created_at": "2026-09-05T16:00:00Z"
}
```

### Hydration Strategy
- **Raw ID Mode (Default):** Feeds return activity references (`object_id`, `actor_id`). Customer frontends hydrate from their own backend APIs.
- **Inline Hydrated Mode:** Optional cached payload embedded directly in the feed item so mobile/web clients render feeds in 1 RTT without secondary backend calls.

## 4. System Data Flow

```
                      ┌─────────────┐
     POST /events ──▶ │  Ingest API │──▶ Postgres (durable event store)
                      └──────┬──────┘
                             │ publish
                             ▼
                      ┌─────────────┐
                      │ Event Queue │  (NATS JetStream / Redis Streams)
                      └──────┬──────┘
                             │ consume
                             ▼
                   ┌───────────────────┐
                   │  Fanout Worker Pool│
                   └─────────┬─────────┘
                             │
            ┌────────────────┴────────────────┐
            ▼                                 ▼
   [Feed Path]                       [Notification Path]
            │                                 │
     is celebrity?                            ▼
     ├── yes ──▶ skip (handled at read)  ┌──────────────────┐
     └── no  ──▶ Redis ZSET write    │ Aggregator /     │
                 feed:{tenant}:{usr} │ Batching Engine  │
                                     └────────┬─────────┘
                                              │
                                              ▼
                                     ┌──────────────────┐
                                     │ Preference Check │
                                     └────────┬─────────┘
                                              │
                                              ▼
                                     ┌──────────────────┐
                                     │ Channel Dispatch │
                                     └────────┬─────────┘
                                              │
                    ┌──────────┬──────────────┼──────────────┬──────────┐
                    ▼          ▼              ▼              ▼          ▼
                 WebSocket  FCM / APNs    SendGrid / SES   Twilio   Webhooks
                 (In-app)   (Mobile Push) (Email Digest)   (SMS)    (HMAC SHA256)
                    │          │              │              │          │
                    └──────────┴──────────────┼──────────────┴──────────┘
                                              │
                                       failed 3x?
                                              │
                                              ▼
                                     ┌──────────────────┐
                                     │ Dead Letter Queue│ (Postgres DLQ Table)
                                     └──────────────────┘
```

## 5. Notification Aggregation & Batching Engine

To prevent user notification fatigue (e.g. 1,000 likes generating 1,000 emails/push alerts), Ripple includes a time-windowed **Aggregation Engine**:

1. **Sliding Window:** When a batchable notification arrives (e.g., `verb = like_created`), Ripple checks Redis for an active aggregation window `batch:{tenant_id}:{recipient_id}:{target_id}`.
2. **Coalescing:** If an open window exists, the actor is appended to the batch array (`actors: ["usr_1", "usr_2", ...]`) and timer is extended (up to `MAX_BATCH_TTL`).
3. **Flush & Render:** When the window timer expires, a single aggregated notification event is generated (*"usr_1, usr_2, and 12 others liked your post"*) and routed to the channel dispatcher.

## 6. Channel Dispatch & Provider Abstraction

Ripple decouples notification delivery from specific cloud providers using Go interface adapters:

```go
type ChannelDispatcher interface {
    Dispatch(ctx context.Context, n *Notification) error
}

type ProviderAdapter interface {
    Send(ctx context.Context, msg *Message) (*DeliveryReport, error)
}
```

Supported Provider Types:
- **WebSocket (In-app):** Pushed to `cmd/notifier` live sockets or stored in offline unread buffer.
- **Native Push:** FCM (Android/Web), APNs (iOS), WebPush (VAPID).
- **Email:** SendGrid, Postmark, AWS SES, Resend (rendered via Liquid template engine).
- **SMS:** Twilio, MessageBird.
- **Webhooks:** Outbound POST requests signed with HMAC-SHA256 signature header (`X-Ripple-Signature: t=1234,v1=9aef...`) to prevent spoofing and replay attacks.

## 7. Atomic Unread Counters & Badging

Ripple maintains exact, atomic unread counts per user per feed type in Redis:
- **Write Path:** On notification delivery, Ripple executes atomic Redis `HINCRBY unread:{tenant_id}:{user_id} {feed_type} 1`.
- **Read Path:** Client queries `GET /v1/notifications/:userID/unread_count` (latency < 2ms).
- **Mark Read Path:** Client calls `POST /v1/notifications/mark_read` with a timestamp cursor, atomically reducing or resetting the counter (`HSET`).

## 8. Dead Letter Queue (DLQ) & Failure Recovery

When channel dispatch fails (e.g., customer webhook endpoint times out or SendGrid API returns 5xx):
1. **Exponential Backoff:** Retried up to $N$ attempts (default 3 retries with jitter).
2. **DLQ Enqueue:** If all retries are exhausted, the delivery task is moved to `dlq_messages` table in Postgres.
3. **Operational Control:** Admin API & Dashboard UI provide search, trace log inspection, and 1-click batch replay (`POST /v1/admin/projects/:projectID/dlq/replay`).

## 9. User Preferences & Opt-out Rules

Before dispatching any notification, Ripple evaluates recipient preferences:
- **Tenant Rules:** Global tenant defaults per event type.
- **User Overrides:** End-user channel toggles (`email: false`, `push: true` for `comment_created`).
- **Quiet Hours:** Do-Not-Disturb (DND) time windows evaluated against recipient timezone. Delayed alerts are buffered until DND window closes.

## 10. Decoupled Ingest & Bounded Concurrency

The ingest API performs strictly: validate payload $\rightarrow$ append to Postgres `events` table $\rightarrow$ publish NATS event $\rightarrow$ return `202 Accepted`. Fanout workers process events asynchronously. Concurrency per worker job is capped via a semaphore pattern (default 100 concurrent Redis writes per fanout job) to prevent connection pool exhaustion.

## 11. Idempotency & Deduplication

- **Feed ZSET Deduplication:** Post/Activity IDs serve as Redis ZSET members. Retried writes update scores without introducing duplicates.
- **Event Deduplication Keys:** Incoming events accept optional `dedup_key`. Redis string locks (`SETNX dedup:{tenant_id}:{key}`) reject duplicate ingest requests within a configurable window.

## 12. Celebrity Threshold Reconciliation

When an account crosses the celebrity threshold ($N > 10,000$ followers):
1. A periodic background job updates `is_celebrity = true` in Postgres.
2. New posts from the account skip fanout-on-write.
3. Previously cached posts in followers' ZSET feeds remain until trimmed by feed size caps (`FEED_CACHE_SIZE`), avoiding expensive retroactive purges.

## 13. Real-Time Push Scaling

WebSocket connections are distributed across multiple `cmd/notifier` instances. Instances subscribe to Redis Pub/Sub channels (`ws_pubsub:{tenant_id}:{user_id}`). When a notification fires, any instance holding the target user's WebSocket connection receives the Pub/Sub message and delivers it immediately.

## 14. Failure Handling Matrix

| Failure | Handling Strategy |
|---|---|
| Ingest DB connection fails | API returns 503; client retries ingest with `dedup_key` |
| Fanout worker crashes mid-job | NATS JetStream redelivers message; idempotent ZSET writes prevent duplicate feed entries |
| Redis write fails for single follower | Retried locally with backoff; overall fanout job succeeds |
| Webhook endpoint times out | Retried 3x exponential backoff $\rightarrow$ routed to Postgres DLQ |
| Downstream email provider outage | Provider adapter returns retryable error $\rightarrow$ message enqueued to NATS retry stream |

## 15. Admin API & Dashboard Data Flow

```
Dashboard (React) ──▶ Admin API ──▶ Postgres (delivery logs, DLQ, workflows, templates)
                                 ──▶ Prometheus (queue depth, p99 latency, delivery rate)
                                 ──▶ Redis (live socket connection gauges)
```

The admin API operates separately from customer API keys, secured by dashboard session tokens (JWT / OAuth2).

## 16. What Ripple Deliberately Does Not Do

- No social graph hosting (follower relationships can be synced or queried via API).
- No video transcoding or image hosting.
- No client-side ML feed recommendation algorithms (chronological with pluggable scoring hook by default).