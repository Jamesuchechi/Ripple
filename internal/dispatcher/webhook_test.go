package dispatcher

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ripple/internal/model"
)

func TestWebhookAdapter_DispatchAndHMACVerification(t *testing.T) {
	secret := "whsec_test_secret_key_12345"

	var receivedBody []byte
	var receivedSigHeader string
	var receivedTimestamp string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSigHeader = r.Header.Get("X-Ripple-Signature")
		receivedTimestamp = r.Header.Get("X-Ripple-Timestamp")

		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed reading request body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"received"}`))
	}))
	defer server.Close()

	adapter := NewWebhookAdapter(secret, 2*time.Second)

	msg := &NotificationMessage{
		ID:          "msg_123",
		ProjectID:   "proj_test",
		RecipientID: "user_456",
		Channel:     ChannelWebhook,
		Target:      server.URL,
		Payload: &model.Activity{
			EventID:  "evt_789",
			Verb:     "like",
			ActorID:  "user_alice",
			ObjectID: "post_001",
		},
		CreatedAt: time.Now().UTC(),
	}

	result, err := adapter.Dispatch(context.Background(), msg)
	if err != nil {
		t.Fatalf("Webhook dispatch failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected result.Success = true, got false")
	}
	if result.Provider != "webhook_hmac_sha256" {
		t.Errorf("Expected provider 'webhook_hmac_sha256', got '%s'", result.Provider)
	}

	// Verify HMAC-SHA256 signature using helper
	valid := VerifyWebhookSignature([]byte(secret), receivedBody, receivedTimestamp, receivedSigHeader)
	if !valid {
		t.Errorf("VerifyWebhookSignature returned false for valid webhook signature header: %s", receivedSigHeader)
	}

	// Test invalid signature check
	invalid := VerifyWebhookSignature([]byte("wrong_secret"), receivedBody, receivedTimestamp, receivedSigHeader)
	if invalid {
		t.Errorf("VerifyWebhookSignature returned true for wrong secret key")
	}
}

func TestWebhookAdapter_Non2xxError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer server.Close()

	adapter := NewWebhookAdapter("secret", 2*time.Second)

	msg := &NotificationMessage{
		ID:      "msg_err",
		Channel: ChannelWebhook,
		Target:  server.URL,
	}

	result, err := adapter.Dispatch(context.Background(), msg)
	if err == nil {
		t.Errorf("Expected error for 500 status code, got nil")
	}
	if result.Success {
		t.Errorf("Expected result.Success = false for 500 status code")
	}
}
