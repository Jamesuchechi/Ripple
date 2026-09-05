# Documentation

## Table of contents

- [Running locally](#running-locally)
- [Configuration](#configuration)
- [Authentication & Webhook Security](#authentication--webhook-security)
- [API Reference](#api-reference)
- [Admin API (Dashboard)](#admin-api-dashboard)
- [Data Model](#data-model)
- [Metrics](#metrics)
- [Integrating Ripple into another product](#integrating-ripple-into-another-product)

## Running locally

Requirements: Go 1.22+, Docker, Docker Compose.

```bash
git clone <repo-url> ripple && cd ripple
cp .env.example .env
docker compose up -d       # Postgres, Redis, NATS
go run ./cmd/migrate up    # Run migrations
go run ./cmd/api            # Ingest + Read + Admin API, default :8080
go run ./cmd/worker         # Fanout & Delivery worker pool
go run ./cmd/notifier       # WebSocket notification hub, default :8081
```

Start the customer dashboard:

```bash
cd dashboard
npm install
npm run dev                 # default :5173, proxies API calls to :8080
```

## Configuration

All configuration is managed via environment variables (see `.env.example`):

| Variable | Description | Default |
|---|---|---|
| `DATABASE_URL` | Postgres connection string | `postgres://ripple:secret@localhost:5432/ripple?sslmode=disable` |
| `REDIS_URL` | Redis connection string | `redis://localhost:6379/0` |
| `QUEUE_URL` | NATS JetStream connection URL | `nats://localhost:4222` |
| `CELEBRITY_THRESHOLD` | Follower count above which fanout-on-read is used | `10000` |
| `FANOUT_CONCURRENCY` | Max concurrent Redis writes per fanout job | `100` |
| `FEED_CACHE_SIZE` | Max cached posts per user feed in Redis | `1000` |
| `CELEBRITY_CACHE_TTL` | TTL for cached celebrity-post queries (seconds) | `10` |
| `API_PORT` | Ingest, read, and admin API port | `8080` |
| `NOTIFIER_PORT` | WebSocket hub port | `8081` |
| `WEBHOOK_HMAC_SECRET` | Secret key used for signing outbound webhooks | `change-me-in-prod` |
| `MAX_DLQ_RETRIES` | Dispatch attempt count before sending to DLQ | `3` |

## Authentication & Webhook Security

### API Authentication
Every customer project (tenant) has API keys created in the dashboard (`pk_live_...` or `pk_test_...`).
Requests to public `/v1/*` endpoints require header:
```http
Authorization: Bearer pk_live_xxxxxxxxxxxx
```

Admin API endpoints (`/v1/admin/*`) require dashboard session authentication (JWT or OAuth2 session cookie).

### Outbound Webhook Security
Outbound webhooks dispatched by Ripple include an HMAC-SHA256 signature header:
```http
X-Ripple-Signature: t=1757088000,v1=a3f892b11e7492c0...
```
Recipient servers verify signatures by computing `HMAC-SHA256(timestamp + "." + raw_payload, WEBHOOK_HMAC_SECRET)` to prevent spoofing and replay attacks.

---

## API Reference

### `POST /v1/events`
Ingest an activity or notification event.

```json
{
  "verb": "comment_created",
  "actor_id": "usr_alice",
  "object_id": "cmt_789",
  "target_id": "post_abc",
  "recipients": ["usr_bob"],
  "payload": {
    "comment_text": "Great explanation!",
    "post_title": "Hybrid Fanout Architecture"
  },
  "dedup_key": "cmt_789:usr_bob"
}
```

Response: `202 Accepted`

```json
{ "event_id": "evt_9f3a2c", "status": "accepted" }
```

### `DELETE /v1/events/:eventID`
Retract an event (e.g. post deletion, un-like). Triggers background un-fanout and unread counter adjustments.

Response: `202 Accepted`

### `GET /v1/feed/:userID`
Returns the merged, ranked feed for a user.

Query parameters:
- `limit` (default 50)
- `before` (cursor, post ID)
- `hydrate` (boolean, default `false`) — embed full payload objects inline.

```json
{
  "activities": [
    {
      "event_id": "evt_9f3a2c",
      "verb": "post_created",
      "actor_id": "usr_alice",
      "object_id": "post_abc",
      "created_at": "2026-09-05T16:00:00Z",
      "payload": { "title": "Building Ripple" }
    }
  ],
  "next_cursor": "post_xyz"
}
```

### `GET /v1/notifications/:userID/unread_count`
Returns unread notification badge counts for a user.

```json
{
  "user_id": "usr_bob",
  "total_unread": 5,
  "by_channel": {
    "in_app": 5
  }
}
```

### `POST /v1/notifications/mark_read`
Mark specific notifications or all notifications up to a timestamp cursor as read.

```json
{
  "user_id": "usr_bob",
  "read_before": "2026-09-05T16:05:00Z"
}
```

### `GET /v1/users/:userID/preferences`
Fetch notification channel preference settings for a user.

```json
{
  "user_id": "usr_bob",
  "channels": {
    "email": true,
    "push": false,
    "in_app": true
  },
  "quiet_hours": {
    "enabled": true,
    "start_utc": "22:00",
    "end_utc": "07:00"
  }
}
```

### `PUT /v1/users/:userID/preferences`
Update notification channel preferences for a user.

### `WS /v1/ws/:userID`
Upgrade connection to WebSocket. Notifications are pushed in real time as JSON frames:

```json
{
  "event_id": "evt_9f3a2c",
  "verb": "like_created",
  "actor_id": "usr_alice",
  "target_id": "post_abc",
  "created_at": "2026-09-05T16:01:00Z"
}
```

---

## Admin API (Dashboard)

Endpoints consumed by the customer dashboard (require dashboard session auth):

- `GET /v1/admin/projects/:projectID/events` — Paginated, filterable event delivery log.
- `GET /v1/admin/projects/:projectID/metrics` — Scoped live metrics (queue depth, p99 latency, success rate, connected clients).
- `POST /v1/admin/projects/:projectID/workflows` — Create or update event routing workflows.
- `POST /v1/admin/projects/:projectID/templates` — Create or update Liquid templates for Email/Push notifications.
- `GET /v1/admin/projects/:projectID/dlq` — Inspect messages in the Dead Letter Queue.
- `POST /v1/admin/projects/:projectID/dlq/replay` — Trigger batch re-execution of DLQ items.
- `POST /v1/admin/projects/:projectID/keys` — Issue new API key.
- `DELETE /v1/admin/projects/:projectID/keys/:keyID` — Revoke API key.

---

## Data Model

### PostgreSQL Schema

```sql
CREATE TABLE projects (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE api_keys (
  id UUID PRIMARY KEY,
  project_id UUID NOT NULL REFERENCES projects(id),
  key_hash TEXT NOT NULL,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ
);

CREATE TABLE workflows (
  id UUID PRIMARY KEY,
  project_id UUID NOT NULL REFERENCES projects(id),
  event_type TEXT NOT NULL,
  channels JSONB NOT NULL, -- ["in_app", "email", "push", "webhook"]
  batch_window_seconds INT DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE templates (
  id UUID PRIMARY KEY,
  project_id UUID NOT NULL REFERENCES projects(id),
  event_type TEXT NOT NULL,
  channel TEXT NOT NULL, -- email | push | sms
  subject_template TEXT,
  body_template TEXT NOT NULL, -- Liquid template string
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE users (
  id UUID PRIMARY KEY,
  project_id UUID NOT NULL REFERENCES projects(id),
  external_user_id TEXT NOT NULL,
  follower_count INT NOT NULL DEFAULT 0,
  is_celebrity BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL,
  UNIQUE (project_id, external_user_id)
);

CREATE TABLE user_preferences (
  project_id UUID NOT NULL REFERENCES projects(id),
  user_id UUID NOT NULL REFERENCES users(id),
  preferences JSONB NOT NULL DEFAULT '{}',
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (project_id, user_id)
);

CREATE TABLE follows (
  project_id UUID NOT NULL REFERENCES projects(id),
  follower_id UUID NOT NULL REFERENCES users(id),
  followee_id UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (project_id, follower_id, followee_id)
);

CREATE TABLE event_log (
  id UUID PRIMARY KEY,
  project_id UUID NOT NULL REFERENCES projects(id),
  verb TEXT NOT NULL,
  actor_id UUID REFERENCES users(id),
  object_id TEXT NOT NULL,
  target_id TEXT,
  payload JSONB NOT NULL,
  status TEXT NOT NULL, -- accepted | fanned_out | failed | dlq
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE dlq_messages (
  id UUID PRIMARY KEY,
  project_id UUID NOT NULL REFERENCES projects(id),
  event_id UUID REFERENCES event_log(id),
  channel TEXT NOT NULL,
  recipient_id TEXT NOT NULL,
  error_message TEXT NOT NULL,
  retry_count INT NOT NULL,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  replayed_at TIMESTAMPTZ
);
```

### Redis Key Patterns (Namespaced by `project_id`)

| Key Pattern | Type | Purpose |
|---|---|---|
| `feed:{project_id}:{user_id}` | ZSET | Cached fanned-out feed, score = timestamp Unix epoch |
| `unread:{project_id}:{user_id}` | HSET | Atomic unread count hash per feed/channel |
| `batch:{project_id}:{user_id}:{target_id}` | HSET + TTL | Active aggregation window for batching notifications |
| `celeb_posts:{project_id}:{user_id}` | STRING (JSON, TTL) | Short-TTL cache of merged celebrity queries |
| `dedup:{project_id}:{key}` | STRING (TTL) | Deduplication lock for idempotency |
| `ratelimit:{project_id}:{key}` | STRING (Counter, TTL) | Token bucket API key rate limiter |

---

## Metrics

Prometheus metrics exposed at `/metrics`:

- `ripple_fanout_duration_seconds` (Histogram) — Per-job fanout latency.
- `ripple_queue_depth` (Gauge) — Current unprocessed event count in queue.
- `ripple_delivery_errors_total` (Counter, labeled by `channel`, `project_id`) — Failed channel dispatches.
- `ripple_dlq_messages_total` (Counter) — Total messages sent to DLQ.
- `ripple_ws_connections` (Gauge) — Active WebSocket connections.
- `ripple_feed_read_latency_seconds` (Histogram) — Read path merge and query latency.

---

## Integrating Ripple into another product

1. **Ingest Events:** Trigger `POST /v1/events` from your backend service whenever an activity occurs (post, like, comment, order).
2. **Client Feeds & Badges:** Use `GET /v1/feed/:userID` to render activity feeds and `GET /v1/notifications/:userID/unread_count` for badging.
3. **Live Push:** Connect frontend clients to `WS /v1/ws/:userID` for instant in-app alerts.
4. **Webhook Handler Verification (Example in Node.js):**

```javascript
const crypto = require('crypto');

function verifyRippleWebhook(rawBody, signatureHeader, secret) {
  const parts = Object.fromEntries(signatureHeader.split(',').map(p => p.split('=')));
  const timestamp = parts.t;
  const signature = parts.v1;
  
  const expectedSig = crypto
    .createHmac('sha256', secret)
    .update(`${timestamp}.${rawBody}`)
    .digest('hex');
    
  return crypto.timingSafeEqual(Buffer.from(signature), Buffer.from(expectedSig));
}
```