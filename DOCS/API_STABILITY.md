# Versioning & API Stability Guarantees

Ripple follows [Semantic Versioning 2.0.0](https://semver.org/).

## `/v1/` REST & WebSocket Stability

All endpoints under `/v1/` are guaranteed stable throughout the `v1.x` major release lifecycle.

### Guaranteed Stable Endpoints
- `POST /v1/events` (Event Ingestion)
- `DELETE /v1/events/:eventID` (Event Retraction)
- `GET /v1/feed/:userID` (Feed Retrieval)
- `GET /v1/notifications/:userID/unread_count` (Unread Badging)
- `POST /v1/notifications/mark_read` (Mark Notifications Read)
- `GET /v1/notifications/:userID/offline` (Offline Recovery)
- `GET /v1/ws` (WebSocket Real-Time Notification Stream)
- `GET /v1/admin/projects/:projectID/dlq` (DLQ Inspection)
- `POST /v1/admin/projects/:projectID/dlq/replay` (DLQ Replay)

### Backward Compatibility Commitments
1. **Field Addition**: New fields may be added to JSON response bodies without breaking existing clients.
2. **Field Deprecation**: No JSON response fields or query parameters will be removed without a major version bump (`v2.0.0`) and 6 months prior deprecation notice.
3. **Status Codes**: Standard HTTP status codes (`200`, `201`, `400`, `401`, `404`, `429`, `500`) are guaranteed.
