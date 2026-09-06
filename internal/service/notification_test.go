package service

import (
	"context"
	"testing"
	"time"

	"ripple/internal/model"
)

func TestEventService_UnreadCounterAndMarkReadFlow(t *testing.T) {
	pgStore, redisStore, cleanup := setupTestDBAndRedis(t)
	if pgStore == nil {
		return
	}
	defer cleanup()

	svc := NewEventService(pgStore, redisStore, nil, 10000)
	ctx := context.Background()

	followerID := "user_unread_follower"
	followeeID := "user_unread_followee"

	// 1. Add follow relationship
	if err := svc.AddFollow(ctx, DefaultProjectID, followerID, followeeID); err != nil {
		t.Fatalf("AddFollow failed: %v", err)
	}

	// 2. Initial unread count should be 0
	count, byChan, err := svc.GetUnreadCount(ctx, DefaultProjectID, followerID)
	if err != nil {
		t.Fatalf("GetUnreadCount failed: %v", err)
	}
	if count != 0 || byChan["in_app"] != 0 {
		t.Errorf("Expected initial unread count 0, got %d (byChan: %v)", count, byChan)
	}

	// 3. Ingest event from followee
	req := &model.IngestEventRequest{
		Verb:     "like",
		ActorID:  followeeID,
		ObjectID: "photo_999",
	}
	act, err := svc.IngestEvent(ctx, DefaultProjectID, req)
	if err != nil {
		t.Fatalf("IngestEvent failed: %v", err)
	}
	if act.EventID == "" {
		t.Fatal("Expected valid event ID")
	}

	time.Sleep(50 * time.Millisecond)

	// 4. Verify unread count is now 1
	count, byChan, err = svc.GetUnreadCount(ctx, DefaultProjectID, followerID)
	if err != nil {
		t.Fatalf("GetUnreadCount after ingest failed: %v", err)
	}
	if count != 1 || byChan["in_app"] != 1 {
		t.Errorf("Expected unread count 1 after ingest, got %d (byChan: %v)", count, byChan)
	}

	// 5. Mark read
	if err := svc.MarkRead(ctx, DefaultProjectID, followerID); err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}

	// 6. Verify unread count is reset to 0
	count, byChan, err = svc.GetUnreadCount(ctx, DefaultProjectID, followerID)
	if err != nil {
		t.Fatalf("GetUnreadCount after MarkRead failed: %v", err)
	}
	if count != 0 || byChan["in_app"] != 0 {
		t.Errorf("Expected unread count 0 after MarkRead, got %d (byChan: %v)", count, byChan)
	}
}

func TestEventService_OfflineNotificationRetrieval(t *testing.T) {
	pgStore, redisStore, cleanup := setupTestDBAndRedis(t)
	if pgStore == nil {
		return
	}
	defer cleanup()

	svc := NewEventService(pgStore, redisStore, nil, 10000)
	ctx := context.Background()

	userID := "user_offline_recipient"
	actorID := "user_sender"

	_ = svc.AddFollow(ctx, DefaultProjectID, userID, actorID)

	// Ingest 2 events
	_, _ = svc.IngestEvent(ctx, DefaultProjectID, &model.IngestEventRequest{
		Verb:     "comment",
		ActorID:  actorID,
		ObjectID: "comment_1",
	})
	_, _ = svc.IngestEvent(ctx, DefaultProjectID, &model.IngestEventRequest{
		Verb:     "comment",
		ActorID:  actorID,
		ObjectID: "comment_2",
	})

	time.Sleep(50 * time.Millisecond)

	offlineEvts, err := svc.GetOfflineNotifications(ctx, DefaultProjectID, userID, 10, 0)
	if err != nil {
		t.Fatalf("GetOfflineNotifications failed: %v", err)
	}

	if len(offlineEvts) != 2 {
		t.Fatalf("Expected 2 offline notifications, got %d", len(offlineEvts))
	}
}
