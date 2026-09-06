package dispatcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDispatcherEngine_MultiChannelRouting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	engine := NewDispatcherEngine()

	webhookAdapter := NewWebhookAdapter("secret", 2*time.Second)
	fcmAdapter := NewFCMAdapter("fcm_server_key")
	apnsAdapter := NewAPNsAdapter("com.app.topic")
	sendGridAdapter := NewEmailAdapter("sendgrid", "sg_api_key", "no-reply@ripple.io")
	twilioAdapter := NewSMSAdapter("account_sid", "auth_token", "+18005550199")

	engine.RegisterAdapter(webhookAdapter)
	engine.RegisterAdapter(fcmAdapter)
	engine.RegisterAdapter(apnsAdapter)
	engine.RegisterAdapter(sendGridAdapter)
	engine.RegisterAdapter(twilioAdapter)

	ctx := context.Background()

	// 1. Test Webhook channel dispatch
	resWebhook, err := engine.Dispatch(ctx, &NotificationMessage{
		ID:      "101",
		Channel: ChannelWebhook,
		Target:  server.URL,
	})
	if err != nil || !resWebhook.Success || resWebhook.Provider != "webhook_hmac_sha256" {
		t.Fatalf("Webhook dispatch failed: %v (res: %+v)", err, resWebhook)
	}

	// 2. Test Push channel dispatch
	resPush, err := engine.Dispatch(ctx, &NotificationMessage{
		ID:      "102",
		Channel: ChannelPush,
		Target:  "device_token_abc",
	})
	if err != nil || !resPush.Success || resPush.Provider != "fcm_push" {
		t.Fatalf("Push dispatch failed: %v (res: %+v)", err, resPush)
	}

	// 3. Test Email channel dispatch
	resEmail, err := engine.Dispatch(ctx, &NotificationMessage{
		ID:      "103",
		Channel: ChannelEmail,
		Target:  "user@example.com",
	})
	if err != nil || !resEmail.Success || resEmail.Provider != "email_sendgrid" {
		t.Fatalf("Email dispatch failed: %v (res: %+v)", err, resEmail)
	}

	// 4. Test SMS channel dispatch
	resSMS, err := engine.Dispatch(ctx, &NotificationMessage{
		ID:      "104",
		Channel: ChannelSMS,
		Target:  "+15551234567",
	})
	if err != nil || !resSMS.Success || resSMS.Provider != "sms_twilio" {
		t.Fatalf("SMS dispatch failed: %v (res: %+v)", err, resSMS)
	}
}

func TestDispatcherEngine_UnregisteredChannel(t *testing.T) {
	engine := NewDispatcherEngine()
	ctx := context.Background()

	_, err := engine.Dispatch(ctx, &NotificationMessage{
		ID:      "999",
		Channel: "unknown_channel",
	})
	if err == nil {
		t.Errorf("Expected error when dispatching to unregistered channel, got nil")
	}
}

func TestPushAdapters_FCMAndAPNs(t *testing.T) {
	ctx := context.Background()

	fcm := NewFCMAdapter("key")
	resFCM, err := fcm.Dispatch(ctx, &NotificationMessage{
		ID:      "fcm_1",
		Channel: ChannelPush,
		Target:  "token_123",
	})
	if err != nil || !resFCM.Success {
		t.Errorf("FCM dispatch failed: %v", err)
	}

	apns := NewAPNsAdapter("topic")
	resAPNs, err := apns.Dispatch(ctx, &NotificationMessage{
		ID:      "apns_1",
		Channel: ChannelPush,
		Target:  "token_456",
	})
	if err != nil || !resAPNs.Success {
		t.Errorf("APNs dispatch failed: %v", err)
	}
}

func TestEmailAdapters_SendGridPostmarkSES(t *testing.T) {
	ctx := context.Background()
	msg := &NotificationMessage{
		ID:      "email_1",
		Channel: ChannelEmail,
		Target:  "test@example.com",
	}

	sg := NewEmailAdapter("sendgrid", "key", "")
	resSG, err := sg.Dispatch(ctx, msg)
	if err != nil || !resSG.Success || resSG.Provider != "email_sendgrid" {
		t.Errorf("SendGrid email dispatch failed: %v", err)
	}

	postmark := NewEmailAdapter("postmark", "key", "")
	resPM, err := postmark.Dispatch(ctx, msg)
	if err != nil || !resPM.Success || resPM.Provider != "email_postmark" {
		t.Errorf("Postmark email dispatch failed: %v", err)
	}

	ses := NewEmailAdapter("aws_ses", "key", "")
	resSES, err := ses.Dispatch(ctx, msg)
	if err != nil || !resSES.Success || resSES.Provider != "email_aws_ses" {
		t.Errorf("AWS SES email dispatch failed: %v", err)
	}
}

func TestSMSAdapter_Twilio(t *testing.T) {
	ctx := context.Background()
	twilio := NewSMSAdapter("sid", "token", "+18005550199")

	res, err := twilio.Dispatch(ctx, &NotificationMessage{
		ID:      "sms_1",
		Channel: ChannelSMS,
		Target:  "+15550001111",
	})
	if err != nil || !res.Success || res.Provider != "sms_twilio" {
		t.Errorf("Twilio SMS dispatch failed: %v", err)
	}
}
