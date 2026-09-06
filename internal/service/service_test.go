package service

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"ripple/internal/model"
	"ripple/internal/store"
)

var DBTestMutex sync.Mutex

func setupTestDBAndRedis(t *testing.T) (*store.PostgresStore, *store.RedisFeedStore, func()) {
	DBTestMutex.Lock()

	dbURL := "postgres://ripple:secret@localhost:5435/ripple?sslmode=disable"
	db, err := sql.Open("postgres", dbURL)
	if err != nil || db.Ping() != nil {
		DBTestMutex.Unlock()
		t.Skip("PostgreSQL not accessible locally, skipping integration test")
		return nil, nil, nil
	}

	redisURL := "redis://localhost:6382/0"
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		db.Close()
		DBTestMutex.Unlock()
		t.Skip("Redis URL invalid, skipping integration test")
		return nil, nil, nil
	}
	rClient := redis.NewClient(opt)
	if err := rClient.Ping(context.Background()).Err(); err != nil {
		db.Close()
		DBTestMutex.Unlock()
		t.Skip("Redis not accessible locally, skipping integration test")
		return nil, nil, nil
	}

	// Make sure default project exists and tables/Redis are clean
	_, _ = db.Exec(`INSERT INTO projects (id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'Default Project') ON CONFLICT DO NOTHING`)
	_, _ = db.Exec(`TRUNCATE TABLE event_log, follows, users CASCADE`)
	_ = rClient.FlushDB(context.Background()).Err()

	pgStore := store.NewPostgresStore(db)
	redisStore := store.NewRedisFeedStore(rClient, 1000)

	cleanup := func() {
		db.Close()
		rClient.Close()
		DBTestMutex.Unlock()
	}

	return pgStore, redisStore, cleanup
}

func TestEventService_IngestAndRetractFlow(t *testing.T) {
	pgStore, redisStore, cleanup := setupTestDBAndRedis(t)
	if pgStore == nil {
		return
	}
	defer cleanup()

	svc := NewEventService(pgStore, redisStore, nil, 10000)
	ctx := context.Background()

	followerID := "user_bob"
	followeeID := "user_alice"

	// 1. Add follow relationship
	err := svc.AddFollow(ctx, DefaultProjectID, followerID, followeeID)
	if err != nil {
		t.Fatalf("AddFollow failed: %v", err)
	}

	// 2. Ingest event from followee ("user_alice")
	req := &model.IngestEventRequest{
		Verb:     "post",
		ActorID:  followeeID,
		ObjectID: "post_hello_world",
	}

	act, err := svc.IngestEvent(ctx, DefaultProjectID, req)
	if err != nil {
		t.Fatalf("IngestEvent failed: %v", err)
	}

	if act.EventID == "" {
		t.Fatal("Expected non-empty EventID")
	}

	// 3. Verify event appears in recipient feed ("user_bob")
	time.Sleep(50 * time.Millisecond)
	feed, err := svc.GetFeed(ctx, DefaultProjectID, followerID, 10, 0)
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}

	if len(feed) != 1 {
		t.Fatalf("Expected 1 activity in feed, got %d", len(feed))
	}
	if feed[0].ObjectID != "post_hello_world" {
		t.Errorf("Expected object_id 'post_hello_world', got '%s'", feed[0].ObjectID)
	}

	// 4. Retract event
	retracted, err := svc.RetractEvent(ctx, act.EventID)
	if err != nil {
		t.Fatalf("RetractEvent failed: %v", err)
	}
	if retracted.EventID != act.EventID {
		t.Errorf("Retracted event ID mismatch: got %s, want %s", retracted.EventID, act.EventID)
	}

	// 5. Verify feed is empty after retraction
	feedAfter, err := svc.GetFeed(ctx, DefaultProjectID, followerID, 10, 0)
	if err != nil {
		t.Fatalf("GetFeed after retraction failed: %v", err)
	}
	if len(feedAfter) != 0 {
		t.Errorf("Expected 0 activities in feed after retraction, got %d", len(feedAfter))
	}
}

func TestEventService_ExplicitRecipientsIngestFlow(t *testing.T) {
	pgStore, redisStore, cleanup := setupTestDBAndRedis(t)
	if pgStore == nil {
		return
	}
	defer cleanup()

	svc := NewEventService(pgStore, redisStore, nil, 10000)
	ctx := context.Background()

	req := &model.IngestEventRequest{
		Verb:       "mention",
		ActorID:    "user_charlie",
		ObjectID:   "comment_456",
		Recipients: []string{"user_dave", "user_eve"},
	}

	act, err := svc.IngestEvent(ctx, DefaultProjectID, req)
	if err != nil {
		t.Fatalf("IngestEvent failed: %v", err)
	}

	// Check dave's feed
	feedDave, err := svc.GetFeed(ctx, DefaultProjectID, "user_dave", 10, 0)
	if err != nil || len(feedDave) != 1 {
		t.Fatalf("Expected 1 activity in dave's feed, got %d (err: %v)", len(feedDave), err)
	}

	// Check eve's feed
	feedEve, err := svc.GetFeed(ctx, DefaultProjectID, "user_eve", 10, 0)
	if err != nil || len(feedEve) != 1 {
		t.Fatalf("Expected 1 activity in eve's feed, got %d (err: %v)", len(feedEve), err)
	}

	// Retract
	_, err = svc.RetractEvent(ctx, act.EventID)
	if err != nil {
		t.Fatalf("RetractEvent failed: %v", err)
	}

	feedDaveAfter, _ := svc.GetFeed(ctx, DefaultProjectID, "user_dave", 10, 0)
	if len(feedDaveAfter) != 0 {
		t.Errorf("Expected 0 activities in dave's feed after retraction, got %d", len(feedDaveAfter))
	}
}
