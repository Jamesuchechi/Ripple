package dispatcher

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// WebhookPayload represents the standard JSON payload sent via webhook.
type WebhookPayload struct {
	EventID     string      `json:"event_id"`
	ProjectID   string      `json:"project_id"`
	RecipientID string      `json:"recipient_id"`
	Payload     interface{} `json:"payload"`
	Timestamp   int64       `json:"timestamp"`
}

// WebhookAdapter dispatches notifications to HTTP webhook endpoints with HMAC-SHA256 signatures.
type WebhookAdapter struct {
	secret     []byte
	httpClient *http.Client
	timeout    time.Duration
}

// NewWebhookAdapter creates a new WebhookAdapter with given secret key.
func NewWebhookAdapter(secret string, timeout time.Duration) *WebhookAdapter {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &WebhookAdapter{
		secret: []byte(secret),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

func (w *WebhookAdapter) Name() string {
	return "webhook_hmac_sha256"
}

func (w *WebhookAdapter) Channel() string {
	return ChannelWebhook
}

// GenerateSignature generates an HMAC-SHA256 hex signature for payload at a given timestamp.
func GenerateSignature(secret []byte, body []byte, timestamp string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyWebhookSignature verifies an incoming X-Ripple-Signature header against secret, body, and timestamp.
func VerifyWebhookSignature(secret []byte, body []byte, timestamp string, signatureHeader string) bool {
	if signatureHeader == "" || timestamp == "" {
		return false
	}

	parts := strings.Split(signatureHeader, ",")
	var v1Sig string
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) == 2 && kv[0] == "v1" {
			v1Sig = kv[1]
			break
		}
	}

	if v1Sig == "" {
		return false
	}

	expectedSig := GenerateSignature(secret, body, timestamp)
	return hmac.Equal([]byte(v1Sig), []byte(expectedSig))
}

func (w *WebhookAdapter) Dispatch(ctx context.Context, msg *NotificationMessage) (*DispatchResult, error) {
	if msg.Target == "" {
		return &DispatchResult{
			Success:   false,
			Provider:  w.Name(),
			Channel:   w.Channel(),
			Error:     "webhook target URL is empty",
			Timestamp: time.Now().UTC(),
		}, fmt.Errorf("webhook target URL is empty")
	}

	now := time.Now().UTC()
	tsStr := strconv.FormatInt(now.Unix(), 10)

	payload := WebhookPayload{
		EventID:     msg.ID,
		ProjectID:   msg.ProjectID,
		RecipientID: msg.RecipientID,
		Payload:     msg.Payload,
		Timestamp:   now.Unix(),
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return &DispatchResult{
			Success:   false,
			Provider:  w.Name(),
			Channel:   w.Channel(),
			Error:     fmt.Sprintf("failed to marshal webhook payload: %v", err),
			Timestamp: now,
		}, err
	}

	sigHex := GenerateSignature(w.secret, bodyBytes, tsStr)
	sigHeader := fmt.Sprintf("t=%s,v1=%s", tsStr, sigHex)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, msg.Target, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return &DispatchResult{
			Success:   false,
			Provider:  w.Name(),
			Channel:   w.Channel(),
			Error:     fmt.Sprintf("failed to create http request: %v", err),
			Timestamp: now,
		}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Ripple-Signature", sigHeader)
	req.Header.Set("X-Ripple-Timestamp", tsStr)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return &DispatchResult{
			Success:   false,
			Provider:  w.Name(),
			Channel:   w.Channel(),
			Error:     fmt.Sprintf("webhook HTTP POST failed: %v", err),
			Timestamp: time.Now().UTC(),
		}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("webhook returned non-2xx status code: %d", resp.StatusCode)
		return &DispatchResult{
			Success:   false,
			Provider:  w.Name(),
			Channel:   w.Channel(),
			Error:     errMsg,
			Timestamp: time.Now().UTC(),
		}, fmt.Errorf("%s", errMsg)
	}

	return &DispatchResult{
		Success:   true,
		Provider:  w.Name(),
		Channel:   w.Channel(),
		MessageID: msg.ID,
		Timestamp: time.Now().UTC(),
	}, nil
}
