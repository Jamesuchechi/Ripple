package dlq

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"ripple/internal/dispatcher"
	"ripple/internal/store"
)

func setupTestDBForDLQ(t *testing.T) (*store.PostgresStore, func()) {
	dbURL := "postgres://ripple:secret@localhost:5435/ripple?sslmode=disable"
	db, err := sql.Open("postgres", dbURL)
	if err != nil || db.Ping() != nil {
		t.Skip("PostgreSQL not accessible locally, skipping integration test")
		return nil, nil
	}

	_, _ = db.Exec(`INSERT INTO projects (id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'Default Project') ON CONFLICT DO NOTHING`)
	_, _ = db.Exec(`TRUNCATE TABLE dlq_messages CASCADE`)

	pgStore := store.NewPostgresStore(db)
	cleanup := func() {
		db.Close()
	}
	return pgStore, cleanup
}

func TestDLQRouter_ExponentialBackoffAndRoutingToDLQ(t *testing.T) {
	pgStore, cleanup := setupTestDBForDLQ(t)
	if pgStore == nil {
		return
	}
	defer cleanup()

	// Server that fails every request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	engine := dispatcher.NewDispatcherEngine()
	webhook := dispatcher.NewWebhookAdapter("secret", 1*time.Second)
	engine.RegisterAdapter(webhook)

	router := NewDLQRouter(engine, pgStore, 2, 10*time.Millisecond)

	msg := &dispatcher.NotificationMessage{
		ID:          "msg_dlq_test",
		ProjectID:   "00000000-0000-0000-0000-000000000001",
		RecipientID: "user_dlq_recipient",
		Channel:     dispatcher.ChannelWebhook,
		Target:      server.URL,
	}

	result, err := router.DispatchWithRetry(context.Background(), msg)
	if err == nil {
		t.Fatalf("Expected error after 2 retries, got nil")
	}
	if result.Success {
		t.Fatalf("Expected result.Success = false")
	}

	// Verify message was stored in PostgreSQL dlq_messages table
	dlqMsgs, err := pgStore.GetDLQMessages(context.Background(), "00000000-0000-0000-0000-000000000001", 10, 0)
	if err != nil {
		t.Fatalf("GetDLQMessages failed: %v", err)
	}
	if len(dlqMsgs) != 1 {
		t.Fatalf("Expected 1 DLQ message, got %d", len(dlqMsgs))
	}
	if dlqMsgs[0].RecipientID != "user_dlq_recipient" {
		t.Errorf("Expected recipient user_dlq_recipient, got %s", dlqMsgs[0].RecipientID)
	}
}

func TestDLQRouter_ReplayDLQMessage(t *testing.T) {
	pgStore, cleanup := setupTestDBForDLQ(t)
	if pgStore == nil {
		return
	}
	defer cleanup()

	attemptCount := 0
	// Server fails initially, then succeeds on replay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount <= 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	engine := dispatcher.NewDispatcherEngine()
	webhook := dispatcher.NewWebhookAdapter("secret", 1*time.Second)
	engine.RegisterAdapter(webhook)

	router := NewDLQRouter(engine, pgStore, 2, 10*time.Millisecond)

	msg := &dispatcher.NotificationMessage{
		ID:          "msg_replay_test",
		ProjectID:   "00000000-0000-0000-0000-000000000001",
		RecipientID: "user_replay_recipient",
		Channel:     dispatcher.ChannelWebhook,
		Target:      server.URL,
	}

	// Initial dispatch fails and enters DLQ
	_, _ = router.DispatchWithRetry(context.Background(), msg)

	dlqMsgs, _ := pgStore.GetDLQMessages(context.Background(), "00000000-0000-0000-0000-000000000001", 10, 0)
	if len(dlqMsgs) != 1 {
		t.Fatalf("Expected 1 DLQ message, got %d", len(dlqMsgs))
	}

	dlqID := dlqMsgs[0].ID

	// Replay DLQ message (attemptCount will be 4 -> server returns 200 OK)
	resReplay, err := router.ReplayDLQMessage(context.Background(), dlqID)
	if err != nil {
		t.Fatalf("ReplayDLQMessage failed: %v", err)
	}
	if !resReplay.Success {
		t.Errorf("Expected replay success")
	}

	// Verify replayed_at timestamp is set in DB
	replayedMsg, err := pgStore.GetDLQMessageByID(context.Background(), dlqID)
	if err != nil {
		t.Fatalf("GetDLQMessageByID failed: %v", err)
	}
	if replayedMsg.ReplayedAt == nil {
		t.Errorf("Expected ReplayedAt timestamp to be set")
	}
}
