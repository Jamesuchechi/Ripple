package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"ripple/internal/auth"
	"ripple/internal/config"
	"ripple/internal/model"
	"ripple/internal/queue"
	"ripple/internal/service"
	"ripple/internal/store"
)

type Server struct {
	cfg *config.Config
	svc *service.EventService
}

func main() {
	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpt)
	defer redisClient.Close()

	pgStore := store.NewPostgresStore(db)
	redisStore := store.NewRedisFeedStore(redisClient, cfg.FeedCacheSize)

	var natsQueue service.QueuePublisher
	q, err := queue.NewNATSQueue(cfg.QueueURL)
	if err != nil {
		log.Printf("Warning: NATS queue connection failed (%v). Operating in direct sync fallback mode.", err)
	} else {
		defer q.Close()
		natsQueue = q
		log.Printf("NATS JetStream queue producer connected successfully to %s", cfg.QueueURL)
	}

	svc := service.NewEventService(pgStore, redisStore, natsQueue, cfg.CelebrityThreshold)

	srv := &Server{
		cfg: cfg,
		svc: svc,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Liveness & Readiness probes
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := redisClient.Ping(r.Context()).Err(); err != nil {
			http.Error(w, "Redis unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})

	// Public API v1 routes
	r.Route("/v1", func(r chi.Router) {
		r.Use(auth.AuthMiddleware(pgStore))
		r.Use(auth.RateLimitMiddleware(redisClient, 100))

		r.Post("/events", srv.handleIngestEvent)
		r.Delete("/events/{eventID}", srv.handleRetractEvent)
		r.Get("/feed/{userID}", srv.handleGetFeed)
		r.Post("/follows", srv.handleAddFollow)
		r.Post("/admin/reconcile-celebrities", srv.handleReconcileCelebrities)

		r.Get("/notifications/{userID}/unread_count", srv.handleGetUnreadCount)
		r.Post("/notifications/mark_read", srv.handleMarkRead)
		r.Get("/notifications/{userID}/offline", srv.handleGetOfflineNotifications)

		r.Get("/admin/projects/{projectID}/dlq", srv.handleGetDLQMessages)
		r.Post("/admin/projects/{projectID}/dlq/replay", srv.handleReplayDLQMessage)
		r.Post("/admin/api-keys", srv.handleCreateAPIKey)
		r.Delete("/admin/api-keys/{keyID}", srv.handleRevokeAPIKey)

		r.Post("/users/{userID}/preferences", srv.handleUpdateUserPreferences)
		r.Get("/users/{userID}/preferences", srv.handleGetUserPreferences)
	})

	addr := fmt.Sprintf(":%s", cfg.APIPort)
	log.Printf("Ripple Ingest/Read API listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}

func (s *Server) handleIngestEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req model.IngestEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request JSON body"})
		return
	}

	projectID := r.Header.Get("X-Project-ID")
	activity, err := s.svc.IngestEvent(r.Context(), projectID, &req)
	if err != nil {
		if errors.Is(err, model.ErrMissingVerb) || errors.Is(err, model.ErrMissingActorID) ||
			errors.Is(err, model.ErrMissingObjectID) || errors.Is(err, model.ErrInvalidPayload) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(model.IngestEventResponse{
		EventID: activity.EventID,
		Status:  "accepted",
	})
}

func (s *Server) handleRetractEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	eventID := chi.URLParam(r, "eventID")

	activity, err := s.svc.RetractEvent(r.Context(), eventID)
	if err != nil {
		if errors.Is(err, model.ErrEventNotFound) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "event not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"event_id": activity.EventID,
		"status":   "retracted",
	})
}

func (s *Server) handleGetFeed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := chi.URLParam(r, "userID")
	projectID := r.Header.Get("X-Project-ID")

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	activities, err := s.svc.GetFeed(r.Context(), projectID, userID, limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if activities == nil {
		activities = []model.Activity{}
	}

	json.NewEncoder(w).Encode(model.FeedResponse{
		Activities: activities,
	})
}

func (s *Server) handleReconcileCelebrities(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	projectID := r.Header.Get("X-Project-ID")

	reconciled, err := s.svc.ReconcileCelebrities(r.Context(), projectID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"status":          "success",
		"users_reconciled": reconciled,
	})
}

func (s *Server) handleAddFollow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		FollowerID string `json:"follower_id"`
		FolloweeID string `json:"followee_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FollowerID == "" || req.FolloweeID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "follower_id and followee_id are required"})
		return
	}

	projectID := r.Header.Get("X-Project-ID")
	if err := s.svc.AddFollow(r.Context(), projectID, req.FollowerID, req.FolloweeID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleGetUnreadCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := chi.URLParam(r, "userID")
	projectID := r.Header.Get("X-Project-ID")

	totalUnread, byChan, err := s.svc.GetUnreadCount(r.Context(), projectID, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(model.UnreadCountResponse{
		UserID:      userID,
		TotalUnread: totalUnread,
		ByChannel:   byChan,
	})
}

func (s *Server) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req model.MarkReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "user_id is required"})
		return
	}

	projectID := r.Header.Get("X-Project-ID")
	if req.ProjectID != "" {
		projectID = req.ProjectID
	}

	if err := s.svc.MarkRead(r.Context(), projectID, req.UserID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(model.MarkReadResponse{
		Status: "success",
	})
}

func (s *Server) handleGetOfflineNotifications(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := chi.URLParam(r, "userID")
	projectID := r.Header.Get("X-Project-ID")

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	activities, err := s.svc.GetOfflineNotifications(r.Context(), projectID, userID, limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if activities == nil {
		activities = []model.Activity{}
	}

	json.NewEncoder(w).Encode(model.FeedResponse{
		Activities: activities,
	})
}

func (s *Server) handleGetDLQMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	projectID := chi.URLParam(r, "projectID")

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	messages, err := s.svc.GetDLQMessages(r.Context(), projectID, limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if messages == nil {
		messages = []model.DLQMessage{}
	}

	json.NewEncoder(w).Encode(model.DLQListResponse{
		Messages: messages,
		Total:    len(messages),
	})
}

func (s *Server) handleReplayDLQMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req model.DLQReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MessageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "message_id is required"})
		return
	}

	result, err := s.svc.ReplayDLQMessage(r.Context(), req.MessageID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(model.DLQReplayResponse{
		Status:    "replayed",
		MessageID: result.MessageID,
	})
}

func (s *Server) handleUpdateUserPreferences(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := chi.URLParam(r, "userID")
	projectID := r.Header.Get("X-Project-ID")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if err := s.svc.UpsertUserPreferences(r.Context(), projectID, userID, bodyBytes); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleGetUserPreferences(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := chi.URLParam(r, "userID")
	projectID := r.Header.Get("X-Project-ID")

	prefBytes, err := s.svc.GetUserPreferences(r.Context(), projectID, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if len(prefBytes) == 0 {
		prefBytes = []byte("{}")
	}

	w.WriteHeader(http.StatusOK)
	w.Write(prefBytes)
}

func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req model.CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	projectID := auth.GetProjectID(r.Context())
	if req.ProjectID != "" {
		projectID = req.ProjectID
	}

	resp, err := s.svc.CreateAPIKey(r.Context(), projectID, req.Name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	keyID := chi.URLParam(r, "keyID")
	projectID := auth.GetProjectID(r.Context())

	if err := s.svc.RevokeAPIKey(r.Context(), projectID, keyID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "revoked"})
}



