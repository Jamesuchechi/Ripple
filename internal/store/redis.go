package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"ripple/internal/model"
)

type RedisFeedStore struct {
	client        *redis.Client
	feedCacheSize int
}

func NewRedisFeedStore(client *redis.Client, feedCacheSize int) *RedisFeedStore {
	if feedCacheSize <= 0 {
		feedCacheSize = 1000
	}
	return &RedisFeedStore{
		client:        client,
		feedCacheSize: feedCacheSize,
	}
}

func (s *RedisFeedStore) feedKey(projectID, recipientID string) string {
	if projectID == "" {
		projectID = "00000000-0000-0000-0000-000000000001"
	}
	return fmt.Sprintf("feed:%s:%s", projectID, recipientID)
}

func (s *RedisFeedStore) dedupKey(projectID, dedupKey string) string {
	return fmt.Sprintf("dedup:%s:%s", projectID, dedupKey)
}

func (s *RedisFeedStore) AddActivityToFeed(ctx context.Context, projectID, recipientID string, activity *model.Activity) error {
	key := s.feedKey(projectID, recipientID)
	data, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("failed to marshal activity: %w", err)
	}

	score := float64(activity.CreatedAt.UnixMilli()) / 1000.0
	err = s.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
	if err != nil {
		return fmt.Errorf("redis ZADD failed for key %s: %w", key, err)
	}

	// Enforce feed size cap
	if s.feedCacheSize > 0 {
		// Keep top `feedCacheSize` newest items, trim older items
		stopRank := -int64(s.feedCacheSize + 1)
		_ = s.client.ZRemRangeByRank(ctx, key, 0, stopRank).Err()
	}

	return nil
}

func (s *RedisFeedStore) RemoveActivityFromFeed(ctx context.Context, projectID, recipientID string, eventID string) error {
	key := s.feedKey(projectID, recipientID)
	members, err := s.client.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("redis ZRANGE failed for key %s: %w", key, err)
	}

	var membersToRemove []interface{}
	for _, m := range members {
		var act model.Activity
		if err := json.Unmarshal([]byte(m), &act); err == nil {
			if act.EventID == eventID {
				membersToRemove = append(membersToRemove, m)
			}
		}
	}

	if len(membersToRemove) > 0 {
		if err := s.client.ZRem(ctx, key, membersToRemove...).Err(); err != nil {
			return fmt.Errorf("redis ZREM failed for key %s: %w", key, err)
		}
	}
	return nil
}

func (s *RedisFeedStore) GetFeed(ctx context.Context, projectID, recipientID string, limit, offset int) ([]model.Activity, error) {
	key := s.feedKey(projectID, recipientID)

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	start := int64(offset)
	stop := int64(offset + limit - 1)

	members, err := s.client.ZRevRange(ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("redis ZREVRANGE failed for key %s: %w", key, err)
	}

	activities := make([]model.Activity, 0, len(members))
	for _, m := range members {
		var act model.Activity
		if err := json.Unmarshal([]byte(m), &act); err != nil {
			continue
		}
		activities = append(activities, act)
	}
	return activities, nil
}

// SetDedupLock attempts to acquire a deduplication lock for (projectID, dedupKey).
// Returns true if acquired (key was not present), false if key was already present.
func (s *RedisFeedStore) SetDedupLock(ctx context.Context, projectID, dedupKey, eventID string, ttl time.Duration) (bool, error) {
	if dedupKey == "" {
		return true, nil
	}
	key := s.dedupKey(projectID, dedupKey)
	acquired, err := s.client.SetNX(ctx, key, eventID, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis SETNX failed for key %s: %w", key, err)
	}
	return acquired, nil
}

func (s *RedisFeedStore) GetDedupLock(ctx context.Context, projectID, dedupKey string) (string, error) {
	if dedupKey == "" {
		return "", nil
	}
	key := s.dedupKey(projectID, dedupKey)
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", fmt.Errorf("redis GET failed for key %s: %w", key, err)
	}
	return val, nil
}

func (s *RedisFeedStore) celebPostsKey(projectID, celebrityID string) string {
	return fmt.Sprintf("celeb_posts:%s:%s", projectID, celebrityID)
}

func (s *RedisFeedStore) GetCachedCelebrityPosts(ctx context.Context, projectID, celebrityID string) ([]model.Activity, error) {
	key := s.celebPostsKey(projectID, celebrityID)
	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("redis GET failed for celebrity key %s: %w", key, err)
	}

	var activities []model.Activity
	if err := json.Unmarshal([]byte(data), &activities); err != nil {
		return nil, err
	}
	return activities, nil
}

func (s *RedisFeedStore) SetCachedCelebrityPosts(ctx context.Context, projectID, celebrityID string, activities []model.Activity, ttl time.Duration) error {
	key := s.celebPostsKey(projectID, celebrityID)
	data, err := json.Marshal(activities)
	if err != nil {
		return fmt.Errorf("failed to marshal celebrity activities: %w", err)
	}

	if ttl <= 0 {
		ttl = 10 * time.Second
	}
	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis SET failed for celebrity key %s: %w", key, err)
	}
	return nil
}

func (s *RedisFeedStore) unreadKey(projectID, userID string) string {
	return fmt.Sprintf("unread:%s:%s", projectID, userID)
}

func (s *RedisFeedStore) WSPubSubChannel(projectID, userID string) string {
	return fmt.Sprintf("ws_pubsub:%s:%s", projectID, userID)
}

func (s *RedisFeedStore) IncrementUnreadCount(ctx context.Context, projectID, userID string) (int64, error) {
	key := s.unreadKey(projectID, userID)
	val, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis INCR failed for key %s: %w", key, err)
	}
	return val, nil
}

func (s *RedisFeedStore) GetUnreadCount(ctx context.Context, projectID, userID string) (int64, error) {
	key := s.unreadKey(projectID, userID)
	val, err := s.client.Get(ctx, key).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, fmt.Errorf("redis GET failed for key %s: %w", key, err)
	}
	return val, nil
}

func (s *RedisFeedStore) MarkRead(ctx context.Context, projectID, userID string) error {
	key := s.unreadKey(projectID, userID)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis DEL failed for key %s: %w", key, err)
	}
	return nil
}

func (s *RedisFeedStore) PublishNotification(ctx context.Context, projectID, userID string, msg *model.WSMessage) error {
	channel := s.WSPubSubChannel(projectID, userID)
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal WSMessage: %w", err)
	}
	if err := s.client.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("redis PUBLISH failed for channel %s: %w", channel, err)
	}
	return nil
}

func (s *RedisFeedStore) SubscribeUserNotifications(ctx context.Context, projectID, userID string) *redis.PubSub {
	channel := s.WSPubSubChannel(projectID, userID)
	return s.client.Subscribe(ctx, channel)
}

func (s *RedisFeedStore) GetClient() *redis.Client {
	return s.client
}

