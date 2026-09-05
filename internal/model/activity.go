package model

import (
	"encoding/json"
	"time"
)

// Activity represents an event ingested into Ripple (Actor-Verb-Object-Target structure).
type Activity struct {
	EventID    string          `json:"event_id"`
	ProjectID  string          `json:"project_id"`
	Verb       string          `json:"verb"`
	ActorID    string          `json:"actor_id"`
	ObjectID   string          `json:"object_id"`
	TargetID   string          `json:"target_id,omitempty"`
	Recipients []string        `json:"recipients,omitempty"`
	Payload    json.RawMessage `json:"payload"`
	DedupKey   string          `json:"dedup_key,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// IngestEventRequest represents the request body for POST /v1/events.
type IngestEventRequest struct {
	Verb       string          `json:"verb"`
	ActorID    string          `json:"actor_id"`
	ObjectID   string          `json:"object_id"`
	TargetID   string          `json:"target_id,omitempty"`
	Recipients []string        `json:"recipients,omitempty"`
	Payload    json.RawMessage `json:"payload"`
	DedupKey   string          `json:"dedup_key,omitempty"`
}

// IngestEventResponse represents the response for POST /v1/events.
type IngestEventResponse struct {
	EventID string `json:"event_id"`
	Status  string `json:"status"`
}

// FeedResponse represents the response for GET /v1/feed/:userID.
type FeedResponse struct {
	Activities []Activity `json:"activities"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

// UnreadCountResponse represents the unread counts for a user.
type UnreadCountResponse struct {
	UserID      string         `json:"user_id"`
	TotalUnread int64          `json:"total_unread"`
	ByChannel   map[string]int `json:"by_channel"`
}

// UserPreferences represents recipient channel rules and quiet hours.
type UserPreferences struct {
	ProjectID  string          `json:"project_id"`
	UserID     string          `json:"user_id"`
	Channels   map[string]bool `json:"channels"`
	QuietHours QuietHours      `json:"quiet_hours"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type QuietHours struct {
	Enabled  bool   `json:"enabled"`
	StartUTC string `json:"start_utc"`
	EndUTC   string `json:"end_utc"`
}

// DLQMessage represents a failed dispatch item in the Dead Letter Queue.
type DLQMessage struct {
	ID           string          `json:"id"`
	ProjectID    string          `json:"project_id"`
	EventID      string          `json:"event_id"`
	Channel      string          `json:"channel"`
	RecipientID  string          `json:"recipient_id"`
	ErrorMessage string          `json:"error_message"`
	RetryCount   int             `json:"retry_count"`
	Payload      json.RawMessage `json:"payload"`
	CreatedAt    time.Time       `json:"created_at"`
	ReplayedAt   *time.Time      `json:"replayed_at,omitempty"`
}
