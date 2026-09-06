package logger

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
)

type contextKey string

const (
	TraceIDKey   contextKey = "trace_id"
	RequestIDKey contextKey = "request_id"
)

var defaultLogger *slog.Logger

func init() {
	defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(defaultLogger)
}

// InitLogger initializes the global logger with custom level and JSON output.
func InitLogger(level slog.Level) *slog.Logger {
	defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(defaultLogger)
	return defaultLogger
}

// GenerateTraceID generates a 16-byte random hex string for distributed tracing.
func GenerateTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b)
}

// WithTraceID attaches trace_id to context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// TraceIDFromContext extracts trace_id from context.
func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(TraceIDKey).(string); ok && v != "" {
		return v
	}
	return ""
}

// WithRequestID attaches request_id to context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// RequestIDFromContext extracts request_id from context.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(RequestIDKey).(string); ok && v != "" {
		return v
	}
	return ""
}

// FromContext returns an *slog.Logger enriched with trace_id and request_id if present in ctx.
func FromContext(ctx context.Context) *slog.Logger {
	l := defaultLogger
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		l = l.With(slog.String("trace_id", traceID))
	}
	if reqID := RequestIDFromContext(ctx); reqID != "" {
		l = l.With(slog.String("request_id", reqID))
	}
	return l
}
