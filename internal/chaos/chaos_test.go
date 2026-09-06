package chaos_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ripple/internal/dispatcher"
	"ripple/internal/dlq"
	"ripple/internal/logger"
	"ripple/internal/metrics"
	"ripple/internal/model"
	"ripple/internal/service"
)

type FailingAdapter struct {
	failCount int
	maxFails  int
}

func (f *FailingAdapter) Name() string    { return "failing_stub" }
func (f *FailingAdapter) Channel() string { return "webhook" }
func (f *FailingAdapter) Dispatch(ctx context.Context, msg *dispatcher.NotificationMessage) (*dispatcher.DispatchResult, error) {
	if f.failCount < f.maxFails {
		f.failCount++
		return nil, errors.New("simulated network connection drop")
	}
	return &dispatcher.DispatchResult{
		Success:   true,
		Provider:  f.Name(),
		Channel:   f.Channel(),
		MessageID: "msg-chaos-123",
		Timestamp: time.Now().UTC(),
	}, nil
}

func TestChaosWorkerInterruptionAndRetry(t *testing.T) {
	ctx := context.Background()
	traceID := logger.GenerateTraceID()
	ctx = logger.WithTraceID(ctx, traceID)

	engine := dispatcher.NewDispatcherEngine()
	adapter := &FailingAdapter{maxFails: 2} // Fails twice, succeeds on 3rd attempt
	engine.RegisterAdapter(adapter)

	router := dlq.NewDLQRouter(engine, nil, 3, 10*time.Millisecond)

	msg := &dispatcher.NotificationMessage{
		ID:          "chaos-msg-1",
		ProjectID:   "proj_chaos",
		RecipientID: "user_chaos",
		Channel:     "webhook",
		Target:      "https://example.com/webhook",
		Payload: &model.Activity{
			EventID:   "evt_chaos_1",
			ProjectID: "proj_chaos",
			TraceID:   traceID,
			Verb:      "post.created",
		},
		CreatedAt: time.Now().UTC(),
	}

	result, err := router.DispatchWithRetry(ctx, msg)
	if err != nil {
		t.Fatalf("expected retry to eventually succeed, got err: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected successful dispatch after retries")
	}

	if adapter.failCount != 2 {
		t.Errorf("expected 2 failure attempts before success, got %d", adapter.failCount)
	}
}

func TestChaosPermanentWorkerFailureRoutingToDLQ(t *testing.T) {
	ctx := context.Background()
	engine := dispatcher.NewDispatcherEngine()
	adapter := &FailingAdapter{maxFails: 10} // Always fails beyond maxRetries
	engine.RegisterAdapter(adapter)

	router := dlq.NewDLQRouter(engine, nil, 2, 10*time.Millisecond)

	msg := &dispatcher.NotificationMessage{
		ID:          "chaos-dlq-msg",
		ProjectID:   "proj_chaos",
		RecipientID: "user_chaos",
		Channel:     "webhook",
		Payload: &model.Activity{
			EventID: "evt_chaos_dlq",
		},
	}

	_, err := router.DispatchWithRetry(ctx, msg)
	if err == nil {
		t.Fatalf("expected error on permanent failure, got nil")
	}
}

func TestChaosRedisDisconnectGracefulFallback(t *testing.T) {
	ctx := context.Background()
	// Create an EventService without live Redis connection (nil Redis store)
	svc := service.NewEventService(nil, nil, nil, 1000)

	act := &model.Activity{
		EventID:    "evt_no_redis",
		ProjectID:  "proj_chaos",
		Verb:       "like",
		ActorID:    "user_1",
		ObjectID:   "post_1",
		Recipients: []string{"user_2"},
	}

	// ProcessFanoutMessage should handle missing/failing Redis gracefully without crashing
	metrics.QueueDepth.WithLabelValues("proj_chaos", "fanout_queue").Inc()
	err := svc.ProcessFanoutMessage(ctx, act)
	metrics.QueueDepth.WithLabelValues("proj_chaos", "fanout_queue").Dec()

	if err != nil {
		t.Fatalf("expected graceful handling of missing store, got: %v", err)
	}
}
