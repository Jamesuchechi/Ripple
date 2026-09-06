package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"

	"ripple/internal/config"
	"ripple/internal/notifier"
	"ripple/internal/service"
	"ripple/internal/store"
)

func main() {
	cfg := config.Load()

	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	var redisStore *store.RedisFeedStore
	if err != nil {
		log.Printf("Warning: invalid Redis URL (%v). Running WebSocket hub without Redis Pub/Sub.", err)
	} else {
		redisClient := redis.NewClient(redisOpt)
		defer redisClient.Close()
		redisStore = store.NewRedisFeedStore(redisClient, cfg.FeedCacheSize)
		log.Printf("Notifier connected to Redis at %s", cfg.RedisURL)
	}

	hub := notifier.NewHub(redisStore)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Get("/v1/ws/{userID}", func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "userID")
		projectID := r.Header.Get("X-Project-ID")
		if projectID == "" {
			projectID = r.URL.Query().Get("project_id")
		}
		if projectID == "" {
			projectID = service.DefaultProjectID
		}
		hub.HandleWebSocket(w, r, projectID, userID)
	})

	addr := fmt.Sprintf(":%s", cfg.NotifierPort)
	log.Printf("Ripple Real-time WebSocket Notifier listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Notifier server failed: %v", err)
	}
}
