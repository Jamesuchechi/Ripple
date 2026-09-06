package dlq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ripple/internal/dispatcher"
	"ripple/internal/model"
	"ripple/internal/store"
)

// DLQRouter manages exponential backoff retries and routes permanently failed dispatches to the DLQ table.
type DLQRouter struct {
	dispatcher *dispatcher.DispatcherEngine
	pg         *store.PostgresStore
	maxRetries int
	baseDelay  time.Duration
}

// NewDLQRouter initializes a DLQRouter instance.
func NewDLQRouter(engine *dispatcher.DispatcherEngine, pg *store.PostgresStore, maxRetries int, baseDelay time.Duration) *DLQRouter {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if baseDelay <= 0 {
		baseDelay = 100 * time.Millisecond
	}
	return &DLQRouter{
		dispatcher: engine,
		pg:         pg,
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
	}
}

// DispatchWithRetry attempts to dispatch a message with exponential backoff retries.
// If all retries fail, it routes the message to the Dead Letter Queue (DLQ).
func (r *DLQRouter) DispatchWithRetry(ctx context.Context, msg *dispatcher.NotificationMessage) (*dispatcher.DispatchResult, error) {
	if r.dispatcher == nil {
		return nil, fmt.Errorf("dispatcher engine is not configured")
	}

	var lastErr error
	var lastResult *dispatcher.DispatchResult

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := r.baseDelay * (1 << (attempt - 1))
			time.Sleep(backoff)
		}

		result, err := r.dispatcher.Dispatch(ctx, msg)
		if err == nil && result.Success {
			return result, nil
		}

		if err != nil {
			lastErr = err
		} else if result != nil && result.Error != "" {
			lastErr = fmt.Errorf("%s", result.Error)
		}
		lastResult = result
	}

	// All retries failed -> route to DLQ in Postgres
	if r.pg != nil {
		payloadBytes, _ := json.Marshal(msg)
		eventID := ""
		if msg.Payload != nil {
			eventID = msg.Payload.EventID
		}

		dlqMsg := &model.DLQMessage{
			ProjectID:    msg.ProjectID,
			EventID:      eventID,
			Channel:      msg.Channel,
			RecipientID:  msg.RecipientID,
			ErrorMessage: lastErr.Error(),
			RetryCount:   r.maxRetries,
			Payload:      payloadBytes,
		}

		if err := r.pg.SaveDLQMessage(ctx, dlqMsg); err != nil {
			log.Printf("Warning: failed to save message to DLQ table: %v", err)
		}
	}

	if lastResult != nil {
		return lastResult, lastErr
	}
	return &dispatcher.DispatchResult{
		Success:   false,
		Channel:   msg.Channel,
		Error:     lastErr.Error(),
		Timestamp: time.Now().UTC(),
	}, lastErr
}

// ReplayDLQMessage fetches a message from DLQ table, re-executes dispatch, and marks it as replayed on success.
func (r *DLQRouter) ReplayDLQMessage(ctx context.Context, dlqID string) (*dispatcher.DispatchResult, error) {
	if r.pg == nil {
		return nil, fmt.Errorf("postgres store is not configured")
	}

	dlqMsg, err := r.pg.GetDLQMessageByID(ctx, dlqID)
	if err != nil {
		return nil, fmt.Errorf("failed to get DLQ message %s: %w", dlqID, err)
	}
	if dlqMsg == nil {
		return nil, fmt.Errorf("DLQ message %s not found", dlqID)
	}

	var msg dispatcher.NotificationMessage
	if err := json.Unmarshal(dlqMsg.Payload, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DLQ payload: %w", err)
	}

	result, err := r.dispatcher.Dispatch(ctx, &msg)
	if err != nil || !result.Success {
		return result, fmt.Errorf("replay dispatch failed: %v", err)
	}

	if err := r.pg.MarkDLQMessageReplayed(ctx, dlqID); err != nil {
		log.Printf("Warning: failed to mark DLQ message %s as replayed: %v", dlqID, err)
	}

	return result, nil
}
