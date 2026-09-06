package service

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"ripple/internal/model"
	"ripple/internal/queue"
	"ripple/internal/store"
)

func setupTestDBRedisAndNATS(t *testing.T) (*store.PostgresStore, *store.RedisFeedStore, *queue.NATSQueue, func()) {
	DBTestMutex.Lock()

	dbURL := "postgres://ripple:secret@localhost:5435/ripple?sslmode=disable"
	db, err := sql.Open("postgres", dbURL)
	if err != nil || db.Ping() != nil {
		DBTestMutex.Unlock()
		t.Skip("PostgreSQL not accessible locally, skipping test")
		return nil, nil, nil, nil
	}

	redisURL := "redis://localhost:6382/0"
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		db.Close()
		DBTestMutex.Unlock()
		t.Skip("Redis URL invalid, skipping test")
		return nil, nil, nil, nil
	}
	rClient := redis.NewClient(opt)
	if err := rClient.Ping(context.Background()).Err(); err != nil {
		db.Close()
		DBTestMutex.Unlock()
		t.Skip("Redis not accessible locally, skipping test")
		return nil, nil, nil, nil
	}

	queueURL := "nats://localhost:4222"
	q, err := queue.NewNATSQueue(queueURL)
	if err != nil {
		db.Close()
		rClient.Close()
		DBTestMutex.Unlock()
		t.Skip("NATS not accessible locally, skipping test")
		return nil, nil, nil, nil
	}

	_, _ = db.Exec(`INSERT INTO projects (id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'Default Project') ON CONFLICT DO NOTHING`)
	_, _ = db.Exec(`TRUNCATE TABLE event_log, follows, users CASCADE`)
	_ = rClient.FlushDB(context.Background()).Err()

	pgStore := store.NewPostgresStore(db)
	redisStore := store.NewRedisFeedStore(rClient, 5) // Cap feed size to 5 items for testing

	cleanup := func() {
		db.Close()
		rClient.Close()
		q.Close()
		DBTestMutex.Unlock()
	}

	return pgStore, redisStore, q, cleanup
}

func TestEventService_DeduplicationLock(t *testing.T) {
	pgStore, redisStore, q, cleanup := setupTestDBRedisAndNATS(t)
	if pgStore == nil {
		return
	}
	defer cleanup()

	svc := NewEventService(pgStore, redisStore, q, 10000)
	ctx := context.Background()

	dedupKey := "unique_action_key_100"

	req1 := &model.IngestEventRequest{
		Verb:       "like",
		ActorID:    "user_alice",
		ObjectID:   "post_999",
		Recipients: []string{"user_bob"},
		DedupKey:   dedupKey,
	}

	// First ingest: should succeed & generate event ID
	act1, err := svc.IngestEvent(ctx, DefaultProjectID, req1)
	if err != nil {
		t.Fatalf("First ingest failed: %v", err)
	}

	// Second ingest with SAME dedupKey: should hit deduplication lock and return original event ID
	req2 := &model.IngestEventRequest{
		Verb:       "like",
		ActorID:    "user_alice",
		ObjectID:   "post_999",
		Recipients: []string{"user_bob"},
		DedupKey:   dedupKey,
	}

	act2, err := svc.IngestEvent(ctx, DefaultProjectID, req2)
	if err != nil {
		t.Fatalf("Second ingest failed: %v", err)
	}

	if act2.EventID != act1.EventID {
		t.Errorf("Expected duplicate ingest to return original EventID '%s', got '%s'", act1.EventID, act2.EventID)
	}
}

func TestEventService_FeedSizeCapTrimming(t *testing.T) {
	pgStore, redisStore, q, cleanup := setupTestDBRedisAndNATS(t)
	if pgStore == nil {
		return
	}
	defer cleanup()

	svc := NewEventService(pgStore, redisStore, q, 10000)
	ctx := context.Background()

	recipientID := "user_feed_cap_test"

	// Ingest 10 activities for recipientID with feed size cap = 5
	for i := 1; i <= 10; i++ {
		req := &model.IngestEventRequest{
			Verb:       "post",
			ActorID:    "user_creator",
			ObjectID:   fmt.Sprintf("item_%d", i),
			Recipients: []string{recipientID},
		}
		act, err := svc.IngestEvent(ctx, DefaultProjectID, req)
		if err != nil {
			t.Fatalf("Ingest failed at item %d: %v", i, err)
		}
		// Directly process fanout for test
		_ = svc.ProcessFanoutMessage(ctx, act)
		time.Sleep(5 * time.Millisecond)
	}

	// Get feed for recipientID
	feed, err := svc.GetFeed(ctx, DefaultProjectID, recipientID, 20, 0)
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}

	if len(feed) > 5 {
		t.Errorf("Expected feed size capped to <= 5 items, got %d items", len(feed))
	}
	if len(feed) > 0 && feed[0].ObjectID != "item_10" {
		t.Errorf("Expected newest item in feed to be 'item_10', got '%s'", feed[0].ObjectID)
	}
}
