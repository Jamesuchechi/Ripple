package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestAPIKey_GenerationAndHashing(t *testing.T) {
	projectID := "proj_123"
	name := "Test Key"

	rawKey, keyRecord, err := GenerateAPIKey(projectID, name)
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}

	if !strings.HasPrefix(rawKey, APIKeyPrefix) {
		t.Errorf("Expected raw API key to start with '%s', got '%s'", APIKeyPrefix, rawKey)
	}

	if keyRecord.ProjectID != projectID || keyRecord.Name != name {
		t.Errorf("APIKey struct fields mismatch: %+v", keyRecord)
	}

	// Verify SHA-256 hash
	expectedHash := HashAPIKey(rawKey)
	if keyRecord.KeyHash != expectedHash {
		t.Errorf("Hash mismatch: got %s, expected %s", keyRecord.KeyHash, expectedHash)
	}
}

func TestRateLimiter_AllowAndExceed(t *testing.T) {
	redisURL := "redis://localhost:6382/0"
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Skip("Redis URL invalid, skipping test")
		return
	}
	rClient := redis.NewClient(opt)
	if err := rClient.Ping(context.Background()).Err(); err != nil {
		t.Skip("Redis not accessible locally, skipping test")
		return
	}
	defer rClient.Close()

	limiter := NewRateLimiter(rClient)
	ctx := context.Background()
	projectID := "proj_rate_test"
	identifier := "test_client"

	// Limit = 2 per second
	limit := 2
	window := 1 * time.Second

	// 1st request -> allowed
	allowed, rem, _, err := limiter.Allow(ctx, projectID, identifier, limit, window)
	if err != nil || !allowed || rem != 1 {
		t.Errorf("1st request failed: allowed=%v, rem=%d, err=%v", allowed, rem, err)
	}

	// 2nd request -> allowed
	allowed, rem, _, err = limiter.Allow(ctx, projectID, identifier, limit, window)
	if err != nil || !allowed || rem != 0 {
		t.Errorf("2nd request failed: allowed=%v, rem=%d, err=%v", allowed, rem, err)
	}

	// 3rd request -> rate limit exceeded
	allowed, rem, _, err = limiter.Allow(ctx, projectID, identifier, limit, window)
	if allowed {
		t.Errorf("Expected 3rd request to be blocked by rate limit, but was allowed")
	}
	if rem != 0 {
		t.Errorf("Expected remaining 0, got %d", rem)
	}
}

func TestAuthMiddleware_ContextExtraction(t *testing.T) {
	middleware := AuthMiddleware(nil)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projID := GetProjectID(r.Context())
		w.Write([]byte(projID))
	}))

	req := httptest.NewRequest("GET", "/v1/test", nil)
	req.Header.Set("X-Project-ID", "proj_header_123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "proj_header_123" {
		t.Errorf("Expected project ID 'proj_header_123', got '%s'", rec.Body.String())
	}
}
