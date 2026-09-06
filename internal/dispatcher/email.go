package dispatcher

import (
	"context"
	"fmt"
	"time"
)

// EmailAdapter dispatches notifications via Email providers (SendGrid, Postmark, AWS SES).
type EmailAdapter struct {
	providerName string // "sendgrid", "postmark", "aws_ses"
	apiKey       string
	fromEmail    string
}

func NewEmailAdapter(providerName, apiKey, fromEmail string) *EmailAdapter {
	if fromEmail == "" {
		fromEmail = "notifications@ripple.io"
	}
	return &EmailAdapter{
		providerName: providerName,
		apiKey:       apiKey,
		fromEmail:    fromEmail,
	}
}

func (e *EmailAdapter) Name() string { return fmt.Sprintf("email_%s", e.providerName) }
func (e *EmailAdapter) Channel() string { return ChannelEmail }

func (e *EmailAdapter) Dispatch(ctx context.Context, msg *NotificationMessage) (*DispatchResult, error) {
	if msg.Target == "" {
		return &DispatchResult{
			Success:   false,
			Provider:  e.Name(),
			Channel:   e.Channel(),
			Error:     "recipient email target address is empty",
			Timestamp: time.Now().UTC(),
		}, fmt.Errorf("recipient email target address is empty")
	}

	// Simulates formatted email payload dispatch to SendGrid / Postmark / SES API
	messageID := fmt.Sprintf("%s_%s", e.providerName, msg.ID)
	return &DispatchResult{
		Success:   true,
		Provider:  e.Name(),
		Channel:   e.Channel(),
		MessageID: messageID,
		Timestamp: time.Now().UTC(),
	}, nil
}
