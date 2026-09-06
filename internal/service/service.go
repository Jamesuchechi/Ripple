package service

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/google/uuid"

	"ripple/internal/auth"
	"ripple/internal/dispatcher"
	"ripple/internal/dlq"
	"ripple/internal/logger"
	"ripple/internal/metrics"
	"ripple/internal/model"
	"ripple/internal/store"
)

const DefaultProjectID = "00000000-0000-0000-0000-000000000001"

type QueuePublisher interface {
	PublishEvent(ctx context.Context, projectID string, activity *model.Activity) error
}

type EventService struct {
	pg                 *store.PostgresStore
	redis              *store.RedisFeedStore
	queue              QueuePublisher
	celebrityThreshold int
	dispatcher         *dispatcher.DispatcherEngine
}

func NewEventService(pg *store.PostgresStore, redis *store.RedisFeedStore, queue QueuePublisher, celebrityThreshold int) *EventService {
	if celebrityThreshold <= 0 {
		celebrityThreshold = 10000
	}
	return &EventService{
		pg:                 pg,
		redis:              redis,
		queue:              queue,
		celebrityThreshold: celebrityThreshold,
		dispatcher:         dispatcher.NewDispatcherEngine(),
	}
}

func (s *EventService) IngestEvent(ctx context.Context, projectID string, req *model.IngestEventRequest) (*model.Activity, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if projectID == "" {
		projectID = DefaultProjectID
	}

	eventID := uuid.New().String()

	// Check deduplication lock if DedupKey is provided
	if req.DedupKey != "" {
		acquired, err := s.redis.SetDedupLock(ctx, projectID, req.DedupKey, eventID, 24*time.Hour)
		if err != nil {
			log.Printf("Warning: failed setting dedup lock: %v", err)
		} else if !acquired {
			// Duplicate event; get existing event_id
			existingEventID, _ := s.redis.GetDedupLock(ctx, projectID, req.DedupKey)
			if existingEventID == "" {
				existingEventID = "dedup_duplicate"
			}
			return &model.Activity{
				EventID:   existingEventID,
				ProjectID: projectID,
				Verb:      req.Verb,
				ActorID:   req.ActorID,
				ObjectID:  req.ObjectID,
				DedupKey:  req.DedupKey,
			}, nil
		}
	}

	recipients := req.Recipients
	if len(recipients) == 0 {
		followers, err := s.pg.GetFollowers(ctx, projectID, req.ActorID)
		if err != nil {
			log.Printf("Warning: failed to lookup followers for actor %s: %v", req.ActorID, err)
		} else {
			recipients = followers
		}
	}

	now := time.Now().UTC()
	activity := model.Activity{
		EventID:    eventID,
		ProjectID:  projectID,
		Verb:       req.Verb,
		ActorID:    req.ActorID,
		ObjectID:   req.ObjectID,
		TargetID:   req.TargetID,
		Recipients: recipients,
		Payload:    req.Payload,
		DedupKey:   req.DedupKey,
		CreatedAt:  now,
	}

	if err := s.pg.SaveEvent(ctx, &activity); err != nil {
		return nil, fmt.Errorf("failed to save event to postgres: %w", err)
	}

	// Publish to NATS JetStream queue for asynchronous fanout
	if s.queue != nil {
		if err := s.queue.PublishEvent(ctx, projectID, &activity); err != nil {
			log.Printf("Warning: failed publishing event %s to queue: %v. Fallback to sync write.", eventID, err)
			_ = s.ProcessFanoutMessage(ctx, &activity)
		}
	} else {
		// Sync fallback when queue is not configured
		_ = s.ProcessFanoutMessage(ctx, &activity)
	}

	return &activity, nil
}

// ProcessFanoutMessage is called by worker daemon (or sync fallback) to fanout activities to recipient feeds.
func (s *EventService) ProcessFanoutMessage(ctx context.Context, activity *model.Activity) error {
	start := time.Now()
	defer func() {
		metrics.FanoutLatency.WithLabelValues(activity.ProjectID, "push").Observe(time.Since(start).Seconds())
	}()

	l := logger.FromContext(ctx)
	l.Debug("Processing fanout message", "event_id", activity.EventID, "project_id", activity.ProjectID)

	// Hybrid Fanout: Check if actor is a celebrity
	if s.pg != nil {
		isCeleb, err := s.pg.IsCelebrity(ctx, activity.ProjectID, activity.ActorID)
		if err != nil {
			log.Printf("Warning: failed checking celebrity status for %s: %v", activity.ActorID, err)
		} else if isCeleb {
			log.Printf("Actor %s is a celebrity (is_celebrity=true). Skipping write-path ZSET fanout to followers.", activity.ActorID)
			// Invalidate/refresh short-TTL celebrity cache in Redis
			if s.redis != nil {
				_ = s.redis.SetCachedCelebrityPosts(ctx, activity.ProjectID, activity.ActorID, nil, 0)
			}
			return nil
		}
	}

	recipients := activity.Recipients
	if len(recipients) == 0 && s.pg != nil {
		followers, err := s.pg.GetFollowers(ctx, activity.ProjectID, activity.ActorID)
		if err != nil {
			return fmt.Errorf("failed to lookup followers for actor %s: %w", activity.ActorID, err)
		}
		recipients = followers
	}

	if s.redis != nil {
		for _, recipientID := range recipients {
			if err := s.redis.AddActivityToFeed(ctx, activity.ProjectID, recipientID, activity); err != nil {
				log.Printf("Warning: failed fanout write to feed for recipient %s: %v", recipientID, err)
			}

			// Increment atomic unread counter and publish real-time notification frame
			unreadCount, err := s.redis.IncrementUnreadCount(ctx, activity.ProjectID, recipientID)
			if err != nil {
				log.Printf("Warning: failed to increment unread count for %s: %v", recipientID, err)
			}

			msg := &model.WSMessage{
				Type:        "notification",
				Activity:    activity,
				UnreadCount: unreadCount,
				Timestamp:   time.Now(),
			}
			if err := s.redis.PublishNotification(ctx, activity.ProjectID, recipientID, msg); err != nil {
				log.Printf("Warning: failed to publish notification for %s: %v", recipientID, err)
			}
		}
	}
	return nil
}

func (s *EventService) RetractEvent(ctx context.Context, eventID string) (*model.Activity, error) {
	activity, err := s.pg.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	if err := s.pg.RetractEvent(ctx, eventID); err != nil {
		return nil, fmt.Errorf("failed to retract event from postgres: %w", err)
	}

	recipients := activity.Recipients
	if len(recipients) == 0 {
		followers, err := s.pg.GetFollowers(ctx, activity.ProjectID, activity.ActorID)
		if err == nil {
			recipients = followers
		}
	}

	for _, recipientID := range recipients {
		if err := s.redis.RemoveActivityFromFeed(ctx, activity.ProjectID, recipientID, eventID); err != nil {
			log.Printf("Warning: failed to remove activity %s from feed of %s: %v", eventID, recipientID, err)
		}
	}

	return activity, nil
}

// GetFeed reads from Redis ZSET feed AND dynamically merges posts from followed celebrities.
func (s *EventService) GetFeed(ctx context.Context, projectID, recipientID string, limit, offset int) ([]model.Activity, error) {
	if projectID == "" {
		projectID = DefaultProjectID
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// 1. Fetch regular ZSET feed from Redis
	cachedFeed, err := s.redis.GetFeed(ctx, projectID, recipientID, limit+offset, 0)
	if err != nil {
		log.Printf("Warning: failed fetching Redis feed for %s: %v", recipientID, err)
		cachedFeed = []model.Activity{}
	}

	// 2. Fetch followed celebrity IDs
	celebIDs, err := s.pg.GetCelebrityFollowees(ctx, projectID, recipientID)
	if err != nil {
		log.Printf("Warning: failed getting celebrity followees for %s: %v", recipientID, err)
	}

	var celebActivities []model.Activity
	for _, celebID := range celebIDs {
		// Check Redis cache for celebrity posts
		cachedPosts, _ := s.redis.GetCachedCelebrityPosts(ctx, projectID, celebID)
		if cachedPosts != nil {
			celebActivities = append(celebActivities, cachedPosts...)
		} else {
			// Query PostgreSQL for recent celebrity posts
			dbPosts, dbErr := s.pg.GetRecentEventsByActors(ctx, projectID, []string{celebID}, 50)
			if dbErr == nil && len(dbPosts) > 0 {
				celebActivities = append(celebActivities, dbPosts...)
				// Cache in Redis with 10s TTL
				_ = s.redis.SetCachedCelebrityPosts(ctx, projectID, celebID, dbPosts, 10*time.Second)
			}
		}
	}

	// 3. Merge regular feed and celebrity activities
	seenEvents := make(map[string]bool)
	var merged []model.Activity

	for _, act := range cachedFeed {
		if !seenEvents[act.EventID] {
			seenEvents[act.EventID] = true
			merged = append(merged, act)
		}
	}
	for _, act := range celebActivities {
		if !seenEvents[act.EventID] {
			seenEvents[act.EventID] = true
			merged = append(merged, act)
		}
	}

	// Sort merged items descending by CreatedAt
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].CreatedAt.After(merged[j].CreatedAt)
	})

	// 4. Apply pagination
	if offset >= len(merged) {
		return []model.Activity{}, nil
	}
	end := offset + limit
	if end > len(merged) {
		end = len(merged)
	}

	return merged[offset:end], nil
}

func (s *EventService) AddFollow(ctx context.Context, projectID, followerID, followeeID string) error {
	if projectID == "" {
		projectID = DefaultProjectID
	}
	return s.pg.AddFollow(ctx, projectID, followerID, followeeID)
}

func (s *EventService) SetCelebrity(ctx context.Context, projectID, userID string, isCelebrity bool) error {
	if projectID == "" {
		projectID = DefaultProjectID
	}
	return s.pg.UpsertUser(ctx, projectID, userID, 0, isCelebrity)
}

func (s *EventService) ReconcileCelebrities(ctx context.Context, projectID string) (int64, error) {
	if projectID == "" {
		projectID = DefaultProjectID
	}
	return s.pg.ReconcileCelebrityStatus(ctx, projectID, s.celebrityThreshold)
}

func (s *EventService) GetUnreadCount(ctx context.Context, projectID, userID string) (int64, map[string]int, error) {
	if projectID == "" {
		projectID = DefaultProjectID
	}
	count, err := s.redis.GetUnreadCount(ctx, projectID, userID)
	if err != nil {
		return 0, nil, err
	}
	return count, map[string]int{"in_app": int(count)}, nil
}

func (s *EventService) MarkRead(ctx context.Context, projectID, userID string) error {
	if projectID == "" {
		projectID = DefaultProjectID
	}
	return s.redis.MarkRead(ctx, projectID, userID)
}

func (s *EventService) GetOfflineNotifications(ctx context.Context, projectID, userID string, limit, offset int) ([]model.Activity, error) {
	return s.GetFeed(ctx, projectID, userID, limit, offset)
}

func (s *EventService) SetDispatcherEngine(engine *dispatcher.DispatcherEngine) {
	s.dispatcher = engine
}

func (s *EventService) GetDispatcherEngine() *dispatcher.DispatcherEngine {
	return s.dispatcher
}

func (s *EventService) DispatchNotification(ctx context.Context, msg *dispatcher.NotificationMessage) (*dispatcher.DispatchResult, error) {
	if s.dispatcher == nil {
		return &dispatcher.DispatchResult{
			Success:   false,
			Channel:   msg.Channel,
			Error:     "dispatcher engine not configured",
			Timestamp: time.Now().UTC(),
		}, fmt.Errorf("dispatcher engine not configured")
	}
	return s.dispatcher.Dispatch(ctx, msg)
}

func (s *EventService) UpsertUserPreferences(ctx context.Context, projectID, userID string, prefBytes []byte) error {
	if projectID == "" {
		projectID = DefaultProjectID
	}
	return s.pg.UpsertUserPreferences(ctx, projectID, userID, prefBytes)
}

func (s *EventService) GetUserPreferences(ctx context.Context, projectID, userID string) ([]byte, error) {
	if projectID == "" {
		projectID = DefaultProjectID
	}
	return s.pg.GetUserPreferences(ctx, projectID, userID)
}

func (s *EventService) GetDLQMessages(ctx context.Context, projectID string, limit, offset int) ([]model.DLQMessage, error) {
	if projectID == "" {
		projectID = DefaultProjectID
	}
	return s.pg.GetDLQMessages(ctx, projectID, limit, offset)
}

func (s *EventService) ReplayDLQMessage(ctx context.Context, dlqID string) (*dispatcher.DispatchResult, error) {
	dlqRouter := dlq.NewDLQRouter(s.dispatcher, s.pg, 1, 10*time.Millisecond)
	return dlqRouter.ReplayDLQMessage(ctx, dlqID)
}

func (s *EventService) CreateAPIKey(ctx context.Context, projectID, name string) (*model.CreateAPIKeyResponse, error) {
	if projectID == "" {
		projectID = DefaultProjectID
	}
	rawKey, keyRecord, err := auth.GenerateAPIKey(projectID, name)
	if err != nil {
		return nil, err
	}
	if err := s.pg.SaveAPIKey(ctx, keyRecord); err != nil {
		return nil, fmt.Errorf("failed to save API key: %w", err)
	}
	return &model.CreateAPIKeyResponse{
		ID:        keyRecord.ID,
		ProjectID: keyRecord.ProjectID,
		Name:      keyRecord.Name,
		RawAPIKey: rawKey,
		CreatedAt: keyRecord.CreatedAt,
	}, nil
}

func (s *EventService) RevokeAPIKey(ctx context.Context, projectID, keyID string) error {
	if projectID == "" {
		projectID = DefaultProjectID
	}
	return s.pg.RevokeAPIKey(ctx, projectID, keyID)
}




