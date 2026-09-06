package store

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"ripple/internal/model"
)

func setupTestRedis(t *testing.T) (*RedisFeedStore, func()) {
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

	store := NewRedisFeedStore(rClient, 100)
	cleanup := func() {
		_ = rClient.FlushDB(context.Background()).Err()
		rClient.Close()
	}
	return store, cleanup
}

func TestRedisFeedStore_UnreadCounters(t *testing.T) {
	store, cleanup := setupTestRedis(t)
	if store == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	projectID := "proj_1"
	userID := "user_alice"

	// Initial count should be 0
	count, err := store.GetUnreadCount(ctx, projectID, userID)
	if err != nil {
		t.Fatalf("GetUnreadCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 unread, got %d", count)
	}

	// Increment unread count
	newCount, err := store.IncrementUnreadCount(ctx, projectID, userID)
	if err != nil {
		t.Fatalf("IncrementUnreadCount failed: %v", err)
	}
	if newCount != 1 {
		t.Errorf("Expected 1 unread after increment, got %d", newCount)
	}

	newCount, err = store.IncrementUnreadCount(ctx, projectID, userID)
	if err != nil {
		t.Fatalf("IncrementUnreadCount 2 failed: %v", err)
	}
	if newCount != 2 {
		t.Errorf("Expected 2 unread after 2nd increment, got %d", newCount)
	}

	// Get count again
	count, err = store.GetUnreadCount(ctx, projectID, userID)
	if err != nil {
		t.Fatalf("GetUnreadCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 unread, got %d", count)
	}

	// Mark read
	err = store.MarkRead(ctx, projectID, userID)
	if err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}

	count, err = store.GetUnreadCount(ctx, projectID, userID)
	if err != nil {
		t.Fatalf("GetUnreadCount after MarkRead failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 unread after MarkRead, got %d", count)
	}
}

func TestRedisFeedStore_PubSub(t *testing.T) {
	store, cleanup := setupTestRedis(t)
	if store == nil {
		return
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	projectID := "proj_1"
	userID := "user_bob"

	pubsub := store.SubscribeUserNotifications(ctx, projectID, userID)
	defer pubsub.Close()

	ch := pubsub.Channel()

	msg := &model.WSMessage{
		Type: "notification",
		Activity: &model.Activity{
			EventID:  "evt_100",
			Verb:     "like",
			ActorID:  "user_alice",
			ObjectID: "post_1",
		},
		UnreadCount: 1,
		Timestamp:   time.Now(),
	}

	time.Sleep(50 * time.Millisecond)

	err := store.PublishNotification(ctx, projectID, userID, msg)
	if err != nil {
		t.Fatalf("PublishNotification failed: %v", err)
	}

	select {
	case received := <-ch:
		if received.Payload == "" {
			t.Fatal("Expected non-empty pubsub message payload")
		}
	case <-ctx.Done():
		t.Fatal("Timed out waiting for Redis Pub/Sub message")
	}
}
