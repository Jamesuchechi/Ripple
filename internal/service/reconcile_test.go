package service

import (
	"context"
	"testing"
	"time"

	"ripple/internal/model"
)

func TestEventService_HybridFanoutCelebritySplit(t *testing.T) {
	pgStore, redisStore, q, cleanup := setupTestDBRedisAndNATS(t)
	if pgStore == nil {
		return
	}
	defer cleanup()

	svc := NewEventService(pgStore, redisStore, q, 10) // Threshold = 10
	ctx := context.Background()

	celebID := "user_taylor_swift"
	normalActorID := "user_normal"
	followerID := "user_fan"

	// Add follow relationships
	_ = svc.AddFollow(ctx, DefaultProjectID, followerID, celebID)
	_ = svc.AddFollow(ctx, DefaultProjectID, followerID, normalActorID)

	// Mark taylor_swift as celebrity
	err := svc.SetCelebrity(ctx, DefaultProjectID, celebID, true)
	if err != nil {
		t.Fatalf("SetCelebrity failed: %v", err)
	}

	// 1. Ingest event from normal user
	reqNormal := &model.IngestEventRequest{
		Verb:     "post",
		ActorID:  normalActorID,
		ObjectID: "normal_post_1",
	}
	actNormal, err := svc.IngestEvent(ctx, DefaultProjectID, reqNormal)
	if err != nil {
		t.Fatalf("IngestEvent normal user failed: %v", err)
	}
	_ = svc.ProcessFanoutMessage(ctx, actNormal)

	// 2. Ingest event from celebrity user
	reqCeleb := &model.IngestEventRequest{
		Verb:     "post",
		ActorID:  celebID,
		ObjectID: "celeb_concert_photo",
	}
	actCeleb, err := svc.IngestEvent(ctx, DefaultProjectID, reqCeleb)
	if err != nil {
		t.Fatalf("IngestEvent celebrity failed: %v", err)
	}
	_ = svc.ProcessFanoutMessage(ctx, actCeleb)

	// 3. Verify Redis ZSET feed for followerID only contains normal_post_1 (since celebrity write fanout was skipped)
	redisFeedOnly, err := redisStore.GetFeed(ctx, DefaultProjectID, followerID, 10, 0)
	if err != nil {
		t.Fatalf("GetFeed from Redis failed: %v", err)
	}
	if len(redisFeedOnly) != 1 || redisFeedOnly[0].ObjectID != "normal_post_1" {
		t.Errorf("Expected Redis ZSET feed to contain only 'normal_post_1', got %v", redisFeedOnly)
	}

	// 4. Verify GetFeed read-path merges celebrity post dynamically
	time.Sleep(20 * time.Millisecond)
	hybridFeed, err := svc.GetFeed(ctx, DefaultProjectID, followerID, 10, 0)
	if err != nil {
		t.Fatalf("GetFeed hybrid read path failed: %v", err)
	}

	if len(hybridFeed) != 2 {
		t.Fatalf("Expected hybrid feed to contain 2 activities, got %d", len(hybridFeed))
	}

	// Newest post should be celeb_concert_photo
	if hybridFeed[0].ObjectID != "celeb_concert_photo" {
		t.Errorf("Expected newest item to be 'celeb_concert_photo', got '%s'", hybridFeed[0].ObjectID)
	}
}

func TestEventService_ReconcileCelebrities(t *testing.T) {
	pgStore, redisStore, q, cleanup := setupTestDBRedisAndNATS(t)
	if pgStore == nil {
		return
	}
	defer cleanup()

	svc := NewEventService(pgStore, redisStore, q, 3) // Threshold = 3
	ctx := context.Background()

	celebCandidate := "user_rising_star"

	// Create 3 followers for celebCandidate
	for i := 1; i <= 3; i++ {
		_ = svc.AddFollow(ctx, DefaultProjectID, model.Activity{}.EventID+string(rune(i)), celebCandidate)
	}

	// Reconcile
	reconciled, err := svc.ReconcileCelebrities(ctx, DefaultProjectID)
	if err != nil {
		t.Fatalf("ReconcileCelebrities failed: %v", err)
	}
	if reconciled == 0 {
		t.Logf("Reconcile returned 0 rows affected")
	}

	// Check if celebCandidate is now a celebrity
	isCeleb, err := pgStore.IsCelebrity(ctx, DefaultProjectID, celebCandidate)
	if err != nil {
		t.Fatalf("IsCelebrity failed: %v", err)
	}
	if !isCeleb {
		t.Errorf("Expected user_rising_star with 3 followers to become celebrity (threshold=3), but is_celebrity=false")
	}
}
