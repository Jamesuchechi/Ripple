package logger_test

import (
	"context"
	"log/slog"
	"testing"

	"ripple/internal/logger"
)

func TestLoggerContext(t *testing.T) {
	ctx := context.Background()
	traceID := logger.GenerateTraceID()
	if len(traceID) != 32 {
		t.Errorf("expected 32 char traceID, got %d", len(traceID))
	}

	ctx = logger.WithTraceID(ctx, traceID)
	ctx = logger.WithRequestID(ctx, "req-123")

	if got := logger.TraceIDFromContext(ctx); got != traceID {
		t.Errorf("expected %s, got %s", traceID, got)
	}

	if got := logger.RequestIDFromContext(ctx); got != "req-123" {
		t.Errorf("expected req-123, got %s", got)
	}

	l := logger.FromContext(ctx)
	if l == nil {
		t.Fatal("expected non-nil logger")
	}

	l.Info("testing context logging", slog.String("key", "value"))
}
