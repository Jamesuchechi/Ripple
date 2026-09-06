package dispatcher

import (
	"context"
	"time"

	"ripple/internal/model"
)

// Channel constants
const (
	ChannelWebhook = "webhook"
	ChannelPush    = "push"
	ChannelEmail   = "email"
	ChannelSMS     = "sms"
	ChannelInApp   = "in_app"
)

// NotificationMessage represents a formatted message to be dispatched across a channel.
type NotificationMessage struct {
	ID          string            `json:"id"`
	ProjectID   string            `json:"project_id"`
	RecipientID string            `json:"recipient_id"`
	Channel     string            `json:"channel"`
	Target      string            `json:"target,omitempty"` // URL for webhook, token/device for push, email address, phone number
	Payload     *model.Activity   `json:"payload"`
	Subject     string            `json:"subject,omitempty"`
	Body        string            `json:"body,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// DispatchResult captures the outcome of a channel dispatch execution.
type DispatchResult struct {
	Success   bool      `json:"success"`
	Provider  string    `json:"provider"`
	Channel   string    `json:"channel"`
	MessageID string    `json:"message_id,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ProviderAdapter defines the common interface for all channel dispatch adapters.
type ProviderAdapter interface {
	Name() string
	Channel() string
	Dispatch(ctx context.Context, msg *NotificationMessage) (*DispatchResult, error)
}
