package model

import "time"


// MarkReadRequest represents the payload to mark notifications as read for a user.
type MarkReadRequest struct {
	UserID    string `json:"user_id"`
	ProjectID string `json:"project_id,omitempty"`
}

// MarkReadResponse represents the response after marking notifications as read.
type MarkReadResponse struct {
	Status string `json:"status"`
}

// WSMessage represents the framing envelope for real-time WebSocket push notifications.
type WSMessage struct {
	Type        string    `json:"type"`                  // e.g. "notification"
	Activity    *Activity `json:"activity,omitempty"`    // the event payload
	UnreadCount int64     `json:"unread_count,omitempty"`// current atomic unread count
	Timestamp   time.Time `json:"timestamp"`
}
