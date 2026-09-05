package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"ripple/internal/config"
)

func main() {
	cfg := config.Load()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Liveness & Readiness probes
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})

	// Public API v1 stub routes
	r.Route("/v1", func(r chi.Router) {
		r.Post("/events", handleIngestEvent)
		r.Get("/feed/{userID}", handleGetFeed)
		r.Get("/notifications/{userID}/unread_count", handleGetUnreadCount)
		r.Post("/notifications/mark_read", handleMarkRead)
	})

	addr := fmt.Sprintf(":%s", cfg.APIPort)
	log.Printf("Ripple Ingest/Read API listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}

func handleIngestEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"event_id": "evt_stub_123",
		"status":   "accepted",
	})
}

func handleGetFeed(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"user_id":    userID,
		"activities": []any{},
	})
}

func handleGetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"user_id":      userID,
		"total_unread": 0,
		"by_channel":   map[string]int{"in_app": 0},
	})
}

func handleMarkRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}
