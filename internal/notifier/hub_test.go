package notifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"ripple/internal/model"
	"ripple/internal/store"
)

func setupTestRedisStore(t *testing.T) (*store.RedisFeedStore, func()) {
	redisURL := "redis://localhost:6382/0"
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Skip("Redis URL invalid, skipping test")
		return nil, nil
	}
	rClient := redis.NewClient(opt)
	if err := rClient.Ping(context.Background()).Err(); err != nil {
		t.Skip("Redis not accessible locally, skipping test")
		return nil, nil
	}

	redisStore := store.NewRedisFeedStore(rClient, 100)
	cleanup := func() {
		_ = rClient.FlushDB(context.Background()).Err()
		rClient.Close()
	}
	return redisStore, cleanup
}

func TestHub_WebSocketConnectionAndRealTimePush(t *testing.T) {
	redisStore, cleanup := setupTestRedisStore(t)
	if redisStore == nil {
		return
	}
	defer cleanup()

	hub := NewHub(redisStore)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	projectID := "00000000-0000-0000-0000-000000000001"
	userID := "user_ws_test"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r, projectID, userID)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial WebSocket server: %v", err)
	}
	defer ws.Close()

	// Wait for subscription registration
	time.Sleep(100 * time.Millisecond)

	// Publish a notification frame via Redis
	act := &model.Activity{
		EventID:  "evt_ws_1",
		Verb:     "post",
		ActorID:  "user_alice",
		ObjectID: "post_ws_100",
	}
	msg := &model.WSMessage{
		Type:        "notification",
		Activity:    act,
		UnreadCount: 1,
		Timestamp:   time.Now(),
	}

	err = redisStore.PublishNotification(context.Background(), projectID, userID, msg)
	if err != nil {
		t.Fatalf("PublishNotification failed: %v", err)
	}

	// Read message from WebSocket connection
	_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, p, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read WebSocket message: %v", err)
	}

	var received model.WSMessage
	if err := json.Unmarshal(p, &received); err != nil {
		t.Fatalf("Failed to unmarshal received WebSocket message: %v", err)
	}

	if received.Type != "notification" {
		t.Errorf("Expected message type 'notification', got '%s'", received.Type)
	}
	if received.Activity == nil || received.Activity.ObjectID != "post_ws_100" {
		t.Errorf("Expected activity object_id 'post_ws_100', got %v", received.Activity)
	}
}
