package dispatcher

import (
	"context"
	"fmt"
	"time"
)

// FCMAdapter handles Firebase Cloud Messaging push notifications.
type FCMAdapter struct {
	serverKey string
}

func NewFCMAdapter(serverKey string) *FCMAdapter {
	return &FCMAdapter{serverKey: serverKey}
}

func (f *FCMAdapter) Name() string { return "fcm_push" }
func (f *FCMAdapter) Channel() string { return ChannelPush }

func (f *FCMAdapter) Dispatch(ctx context.Context, msg *NotificationMessage) (*DispatchResult, error) {
	if msg.Target == "" {
		return &DispatchResult{
			Success:   false,
			Provider:  f.Name(),
			Channel:   f.Channel(),
			Error:     "FCM device token target is empty",
			Timestamp: time.Now().UTC(),
		}, fmt.Errorf("FCM device token target is empty")
	}

	// Simulates FCM API payload format
	// e.g. POST https://fcm.googleapis.com/fcm/send with Authorization: key=serverKey
	return &DispatchResult{
		Success:   true,
		Provider:  f.Name(),
		Channel:   f.Channel(),
		MessageID: fmt.Sprintf("fcm_%s", msg.ID),
		Timestamp: time.Now().UTC(),
	}, nil
}

// APNsAdapter handles Apple Push Notification service (APNs) push notifications.
type APNsAdapter struct {
	topic string
}

func NewAPNsAdapter(topic string) *APNsAdapter {
	return &APNsAdapter{topic: topic}
}

func (a *APNsAdapter) Name() string { return "apns_push" }
func (a *APNsAdapter) Channel() string { return ChannelPush }

func (a *APNsAdapter) Dispatch(ctx context.Context, msg *NotificationMessage) (*DispatchResult, error) {
	if msg.Target == "" {
		return &DispatchResult{
			Success:   false,
			Provider:  a.Name(),
			Channel:   a.Channel(),
			Error:     "APNs device token target is empty",
			Timestamp: time.Now().UTC(),
		}, fmt.Errorf("APNs device token target is empty")
	}

	// Simulates APNs HTTP/2 request format
	return &DispatchResult{
		Success:   true,
		Provider:  a.Name(),
		Channel:   a.Channel(),
		MessageID: fmt.Sprintf("apns_%s", msg.ID),
		Timestamp: time.Now().UTC(),
	}, nil
}
