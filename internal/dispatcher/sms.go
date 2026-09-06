package dispatcher

import (
	"context"
	"fmt"
	"time"
)

// SMSAdapter dispatches SMS notifications via Twilio or equivalent providers.
type SMSAdapter struct {
	accountSid string
	authToken  string
	fromNumber string
}

func NewSMSAdapter(accountSid, authToken, fromNumber string) *SMSAdapter {
	return &SMSAdapter{
		accountSid: accountSid,
		authToken:  authToken,
		fromNumber: fromNumber,
	}
}

func (s *SMSAdapter) Name() string { return "sms_twilio" }
func (s *SMSAdapter) Channel() string { return ChannelSMS }

func (s *SMSAdapter) Dispatch(ctx context.Context, msg *NotificationMessage) (*DispatchResult, error) {
	if msg.Target == "" {
		return &DispatchResult{
			Success:   false,
			Provider:  s.Name(),
			Channel:   s.Channel(),
			Error:     "recipient phone number target is empty",
			Timestamp: time.Now().UTC(),
		}, fmt.Errorf("recipient phone number target is empty")
	}

	// Simulates formatted SMS dispatch (Twilio REST API request)
	return &DispatchResult{
		Success:   true,
		Provider:  s.Name(),
		Channel:   s.Channel(),
		MessageID: fmt.Sprintf("twilio_%s", msg.ID),
		Timestamp: time.Now().UTC(),
	}, nil
}
