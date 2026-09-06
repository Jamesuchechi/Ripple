package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"ripple/internal/config"
	"ripple/internal/model"
	"ripple/internal/service"
	"ripple/internal/store"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil || db.Ping() != nil {
		t.Skip("PostgreSQL not accessible locally, skipping API tests")
		return nil, nil
	}

	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		t.Skip("Redis URL invalid, skipping API tests")
		return nil, nil
	}
	rClient := redis.NewClient(redisOpt)
	if err := rClient.Ping(context.Background()).Err(); err != nil {
		t.Skip("Redis not accessible locally, skipping API tests")
		return nil, nil
	}

	_, _ = db.Exec(`INSERT INTO projects (id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'Default Project') ON CONFLICT DO NOTHING`)

	pgStore := store.NewPostgresStore(db)
	redisStore := store.NewRedisFeedStore(rClient, 1000)
	svc := service.NewEventService(pgStore, redisStore, nil, 10000)

	srv := &Server{
		cfg: cfg,
		svc: svc,
	}

	cleanup := func() {
		_ = rClient.FlushDB(context.Background()).Err()
		db.Close()
		rClient.Close()
	}

	return srv, cleanup
}

func TestAPI_IngestRetractAndFeed(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	if srv == nil {
		return
	}
	defer cleanup()

	r := chi.NewRouter()
	r.Post("/v1/events", srv.handleIngestEvent)
	r.Delete("/v1/events/{eventID}", srv.handleRetractEvent)
	r.Get("/v1/feed/{userID}", srv.handleGetFeed)
	r.Post("/v1/follows", srv.handleAddFollow)

	// 1. Create Follower relationship: user_2 follows user_1
	followReqBody, _ := json.Marshal(map[string]string{
		"follower_id": "user_2",
		"followee_id": "user_1",
	})
	reqFollow := httptest.NewRequest("POST", "/v1/follows", bytes.NewBuffer(followReqBody))
	reqFollow.Header.Set("Content-Type", "application/json")
	wFollow := httptest.NewRecorder()
	r.ServeHTTP(wFollow, reqFollow)

	if wFollow.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for add follow, got %d: %s", wFollow.Code, wFollow.Body.String())
	}

	// 2. Post Event by user_1
	eventReqBody, _ := json.Marshal(map[string]any{
		"verb":      "post",
		"actor_id":  "user_1",
		"object_id": "photo_999",
		"payload":   map[string]string{"caption": "Sunset at beach"},
	})
	reqEvent := httptest.NewRequest("POST", "/v1/events", bytes.NewBuffer(eventReqBody))
	reqEvent.Header.Set("Content-Type", "application/json")
	wEvent := httptest.NewRecorder()
	r.ServeHTTP(wEvent, reqEvent)

	if wEvent.Code != http.StatusAccepted {
		t.Fatalf("Expected 202 Accepted for ingest event, got %d: %s", wEvent.Code, wEvent.Body.String())
	}

	var ingestResp model.IngestEventResponse
	json.NewDecoder(wEvent.Body).Decode(&ingestResp)
	if ingestResp.EventID == "" || ingestResp.Status != "accepted" {
		t.Fatalf("Invalid ingest response: %+v", ingestResp)
	}

	// 3. Get Feed for user_2
	reqFeed := httptest.NewRequest("GET", "/v1/feed/user_2", nil)
	wFeed := httptest.NewRecorder()
	r.ServeHTTP(wFeed, reqFeed)

	if wFeed.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for get feed, got %d: %s", wFeed.Code, wFeed.Body.String())
	}

	var feedResp model.FeedResponse
	json.NewDecoder(wFeed.Body).Decode(&feedResp)
	if len(feedResp.Activities) != 1 {
		t.Fatalf("Expected 1 activity in feed, got %d", len(feedResp.Activities))
	}
	if feedResp.Activities[0].ObjectID != "photo_999" {
		t.Errorf("Expected object_id 'photo_999', got '%s'", feedResp.Activities[0].ObjectID)
	}

	// 4. Retract Event
	reqRetract := httptest.NewRequest("DELETE", "/v1/events/"+ingestResp.EventID, nil)
	wRetract := httptest.NewRecorder()
	r.ServeHTTP(wRetract, reqRetract)

	if wRetract.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for retract event, got %d: %s", wRetract.Code, wRetract.Body.String())
	}

	// 5. Get Feed for user_2 after retraction
	wFeedAfter := httptest.NewRecorder()
	r.ServeHTTP(wFeedAfter, reqFeed)

	var feedRespAfter model.FeedResponse
	json.NewDecoder(wFeedAfter.Body).Decode(&feedRespAfter)
	if len(feedRespAfter.Activities) != 0 {
		t.Errorf("Expected 0 activities in feed after retraction, got %d", len(feedRespAfter.Activities))
	}
}
