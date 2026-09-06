package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements a Redis-backed sliding window rate limiter.
type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

func (r *RateLimiter) key(projectID, identifier string) string {
	return fmt.Sprintf("ratelimit:%s:%s", projectID, identifier)
}

// Allow checks if a request is allowed under rate limit settings.
func (r *RateLimiter) Allow(ctx context.Context, projectID, identifier string, limit int, window time.Duration) (bool, int, time.Duration, error) {
	if r.client == nil || limit <= 0 {
		return true, limit, 0, nil
	}

	key := r.key(projectID, identifier)
	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	ttl := pipe.TTL(ctx, key)
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return true, limit, 0, nil
	}

	count := incr.Val()
	currTTL := ttl.Val()

	if count == 1 || currTTL <= 0 {
		r.client.Expire(ctx, key, window)
		currTTL = window
	}

	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	allowed := count <= int64(limit)
	return allowed, remaining, currTTL, nil
}

// RateLimitMiddleware creates an HTTP middleware enforcing rate limits per project ID.
func RateLimitMiddleware(rClient *redis.Client, limitPerSec int) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(rClient)
	if limitPerSec <= 0 {
		limitPerSec = 100
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			projectID := GetProjectID(r.Context())
			if projectID == "" {
				projectID = r.Header.Get("X-Project-ID")
			}
			if projectID == "" {
				projectID = "global"
			}

			allowed, remaining, resetIn, _ := limiter.Allow(r.Context(), projectID, "api", limitPerSec, 1*time.Second)

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limitPerSec))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(resetIn).Unix(), 10))

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
