package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"ripple/internal/model"
)

// BatchAggregator handles time-windowed sliding aggregation in Redis.
type BatchAggregator struct {
	client *redis.Client
}

func NewBatchAggregator(client *redis.Client) *BatchAggregator {
	return &BatchAggregator{client: client}
}

func (b *BatchAggregator) batchKey(projectID, recipientID, verb string) string {
	return fmt.Sprintf("batch:%s:%s:%s", projectID, recipientID, verb)
}

// AddActivity adds an activity to the recipient's sliding aggregation batch.
func (b *BatchAggregator) AddActivity(ctx context.Context, projectID, recipientID, verb string, activity *model.Activity, window time.Duration) (int64, error) {
	key := b.batchKey(projectID, recipientID, verb)
	data, err := json.Marshal(activity)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal activity: %w", err)
	}

	score := float64(activity.CreatedAt.UnixMilli()) / 1000.0
	err = b.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
	if err != nil {
		return 0, fmt.Errorf("redis ZADD failed for key %s: %w", key, err)
	}

	if window <= 0 {
		window = 5 * time.Minute
	}
	_ = b.client.Expire(ctx, key, window).Err()

	count, err := b.client.ZCard(ctx, key).Result()
	if err != nil {
		return 1, nil
	}
	return count, nil
}

// GetBatch retrieves all activities in the current sliding window for a recipient and verb.
func (b *BatchAggregator) GetBatch(ctx context.Context, projectID, recipientID, verb string) ([]model.Activity, error) {
	key := b.batchKey(projectID, recipientID, verb)
	members, err := b.client.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("redis ZRANGE failed for key %s: %w", key, err)
	}

	activities := make([]model.Activity, 0, len(members))
	for _, m := range members {
		var act model.Activity
		if err := json.Unmarshal([]byte(m), &act); err == nil {
			activities = append(activities, act)
		}
	}
	return activities, nil
}

// ClearBatch removes the aggregation batch key after delivery.
func (b *BatchAggregator) ClearBatch(ctx context.Context, projectID, recipientID, verb string) error {
	key := b.batchKey(projectID, recipientID, verb)
	return b.client.Del(ctx, key).Err()
}
