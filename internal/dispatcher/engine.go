package dispatcher

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DispatcherEngine manages provider adapters and routes notification messages to appropriate channels.
type DispatcherEngine struct {
	mu       sync.RWMutex
	adapters map[string][]ProviderAdapter // key: channel name -> slice of adapters
}

// NewDispatcherEngine initializes a new DispatcherEngine.
func NewDispatcherEngine() *DispatcherEngine {
	return &DispatcherEngine{
		adapters: make(map[string][]ProviderAdapter),
	}
}

// RegisterAdapter registers a provider adapter for a specific channel.
func (e *DispatcherEngine) RegisterAdapter(adapter ProviderAdapter) {
	e.mu.Lock()
	defer e.mu.Unlock()

	channel := adapter.Channel()
	e.adapters[channel] = append(e.adapters[channel], adapter)
}

// GetAdapters returns all registered adapters for a given channel.
func (e *DispatcherEngine) GetAdapters(channel string) []ProviderAdapter {
	e.mu.RLock()
	defer e.mu.RUnlock()

	adapters, ok := e.adapters[channel]
	if !ok {
		return nil
	}
	copied := make([]ProviderAdapter, len(adapters))
	copy(copied, adapters)
	return copied
}

// Dispatch routes a message to registered provider adapter(s) for its channel.
func (e *DispatcherEngine) Dispatch(ctx context.Context, msg *NotificationMessage) (*DispatchResult, error) {
	adapters := e.GetAdapters(msg.Channel)
	if len(adapters) == 0 {
		return &DispatchResult{
			Success:   false,
			Channel:   msg.Channel,
			Error:     fmt.Sprintf("no provider adapter registered for channel: %s", msg.Channel),
			Timestamp: time.Now().UTC(),
		}, fmt.Errorf("no provider adapter registered for channel: %s", msg.Channel)
	}

	// Try registered adapters for the channel
	var lastErr error
	for _, adapter := range adapters {
		result, err := adapter.Dispatch(ctx, msg)
		if err == nil && result.Success {
			return result, nil
		}
		if err != nil {
			lastErr = err
		} else if result != nil && result.Error != "" {
			lastErr = fmt.Errorf("%s", result.Error)
		}
	}

	return &DispatchResult{
		Success:   false,
		Channel:   msg.Channel,
		Error:     fmt.Sprintf("all adapters for channel %s failed: %v", msg.Channel, lastErr),
		Timestamp: time.Now().UTC(),
	}, lastErr
}
