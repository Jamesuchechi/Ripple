package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"ripple/internal/logger"
	"ripple/internal/store"
)

type contextKey string

const ProjectIDContextKey contextKey = "projectID"

// AuthMiddleware creates an HTTP middleware that validates API keys and attaches project_id, trace_id, and request_id to request context.
func AuthMiddleware(pg *store.PostgresStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			traceID := r.Header.Get("X-Trace-ID")
			if traceID == "" {
				traceID = logger.GenerateTraceID()
			}
			ctx = logger.WithTraceID(ctx, traceID)

			reqID := r.Header.Get("X-Request-ID")
			if reqID != "" {
				ctx = logger.WithRequestID(ctx, reqID)
			}

			w.Header().Set("X-Trace-ID", traceID)

			var rawKey string
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				rawKey = strings.TrimPrefix(authHeader, "Bearer ")
			} else if apiKeyHeader := r.Header.Get("X-API-Key"); apiKeyHeader != "" {
				rawKey = apiKeyHeader
			}

			projectID := r.Header.Get("X-Project-ID")

			if rawKey != "" && pg != nil {
				keyHash := HashAPIKey(rawKey)
				apiKey, err := pg.GetAPIKeyByHash(ctx, keyHash)
				if err != nil || apiKey == nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(map[string]string{"error": "invalid or revoked API key"})
					return
				}
				if apiKey.RevokedAt != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(map[string]string{"error": "API key has been revoked"})
					return
				}
				projectID = apiKey.ProjectID
			}

			if projectID != "" {
				ctx = context.WithValue(ctx, ProjectIDContextKey, projectID)
			}

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// GetProjectID retrieves the authenticated project ID from context.
func GetProjectID(ctx context.Context) string {
	if val, ok := ctx.Value(ProjectIDContextKey).(string); ok && val != "" {
		return val
	}
	return ""
}
