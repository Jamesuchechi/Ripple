package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"ripple/internal/model"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) SaveEvent(ctx context.Context, activity *model.Activity) error {
	query := `
		INSERT INTO event_log (id, project_id, verb, actor_id, object_id, target_id, recipients, payload, dedup_key, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'accepted', $10)
	`
	payloadBytes := activity.Payload
	if len(payloadBytes) == 0 {
		payloadBytes = json.RawMessage("{}")
	}

	var recipientsArray interface{}
	if len(activity.Recipients) > 0 {
		recipientsArray = pq.Array(activity.Recipients)
	} else {
		recipientsArray = nil
	}

	var dedupKey *string
	if activity.DedupKey != "" {
		dedupKey = &activity.DedupKey
	}

	var targetID *string
	if activity.TargetID != "" {
		targetID = &activity.TargetID
	}

	_, err := s.db.ExecContext(ctx, query,
		activity.EventID,
		activity.ProjectID,
		activity.Verb,
		activity.ActorID,
		activity.ObjectID,
		targetID,
		recipientsArray,
		payloadBytes,
		dedupKey,
		activity.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert event into event_log: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetEventByID(ctx context.Context, eventID string) (*model.Activity, error) {
	query := `
		SELECT id, project_id, verb, actor_id, object_id, COALESCE(target_id, ''), recipients, payload, COALESCE(dedup_key, ''), created_at
		FROM event_log
		WHERE id = $1 AND status != 'retracted'
	`
	var act model.Activity
	var recipients []string
	var payloadBytes []byte

	err := s.db.QueryRowContext(ctx, query, eventID).Scan(
		&act.EventID,
		&act.ProjectID,
		&act.Verb,
		&act.ActorID,
		&act.ObjectID,
		&act.TargetID,
		pq.Array(&recipients),
		&payloadBytes,
		&act.DedupKey,
		&act.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrEventNotFound
		}
		return nil, fmt.Errorf("failed to query event: %w", err)
	}
	act.Recipients = recipients
	act.Payload = json.RawMessage(payloadBytes)
	return &act, nil
}

func (s *PostgresStore) RetractEvent(ctx context.Context, eventID string) error {
	query := `UPDATE event_log SET status = 'retracted' WHERE id = $1`
	res, err := s.db.ExecContext(ctx, query, eventID)
	if err != nil {
		return fmt.Errorf("failed to retract event: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return model.ErrEventNotFound
	}
	return nil
}

func (s *PostgresStore) AddFollow(ctx context.Context, projectID, followerID, followeeID string) error {
	query := `
		INSERT INTO follows (project_id, follower_id, followee_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (project_id, follower_id, followee_id) DO NOTHING
	`
	_, err := s.db.ExecContext(ctx, query, projectID, followerID, followeeID)
	if err != nil {
		return fmt.Errorf("failed to add follow relationship: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetFollowers(ctx context.Context, projectID, followeeID string) ([]string, error) {
	query := `
		SELECT follower_id
		FROM follows
		WHERE project_id = $1 AND followee_id = $2
	`
	rows, err := s.db.QueryContext(ctx, query, projectID, followeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query followers: %w", err)
	}
	defer rows.Close()

	var followers []string
	for rows.Next() {
		var followerID string
		if err := rows.Scan(&followerID); err != nil {
			return nil, err
		}
		followers = append(followers, followerID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return followers, nil
}

func (s *PostgresStore) UpsertUser(ctx context.Context, projectID, externalUserID string, followerCount int, isCelebrity bool) error {
	query := `
		INSERT INTO users (project_id, external_user_id, follower_count, is_celebrity)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (project_id, external_user_id) DO UPDATE
		SET follower_count = EXCLUDED.follower_count, is_celebrity = EXCLUDED.is_celebrity
	`
	_, err := s.db.ExecContext(ctx, query, projectID, externalUserID, followerCount, isCelebrity)
	if err != nil {
		return fmt.Errorf("failed to upsert user %s: %w", externalUserID, err)
	}
	return nil
}

func (s *PostgresStore) IsCelebrity(ctx context.Context, projectID, actorID string) (bool, error) {
	query := `
		SELECT is_celebrity
		FROM users
		WHERE project_id = $1 AND external_user_id = $2
	`
	var isCeleb bool
	err := s.db.QueryRowContext(ctx, query, projectID, actorID).Scan(&isCeleb)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to query is_celebrity status for %s: %w", actorID, err)
	}
	return isCeleb, nil
}

func (s *PostgresStore) GetCelebrityFollowees(ctx context.Context, projectID, followerID string) ([]string, error) {
	query := `
		SELECT f.followee_id
		FROM follows f
		JOIN users u ON f.project_id = u.project_id AND f.followee_id = u.external_user_id
		WHERE f.project_id = $1 AND f.follower_id = $2 AND u.is_celebrity = true
	`
	rows, err := s.db.QueryContext(ctx, query, projectID, followerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query celebrity followees: %w", err)
	}
	defer rows.Close()

	var celebIDs []string
	for rows.Next() {
		var followeeID string
		if err := rows.Scan(&followeeID); err != nil {
			return nil, err
		}
		celebIDs = append(celebIDs, followeeID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return celebIDs, nil
}

func (s *PostgresStore) GetRecentEventsByActors(ctx context.Context, projectID string, actorIDs []string, limit int) ([]model.Activity, error) {
	if len(actorIDs) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, project_id, verb, actor_id, object_id, COALESCE(target_id, ''), recipients, payload, COALESCE(dedup_key, ''), created_at
		FROM event_log
		WHERE project_id = $1 AND actor_id = ANY($2) AND status != 'retracted'
		ORDER BY created_at DESC
		LIMIT $3
	`
	rows, err := s.db.QueryContext(ctx, query, projectID, pq.Array(actorIDs), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent events for celebrity actors: %w", err)
	}
	defer rows.Close()

	var activities []model.Activity
	for rows.Next() {
		var act model.Activity
		var recipients []string
		var payloadBytes []byte
		if err := rows.Scan(
			&act.EventID,
			&act.ProjectID,
			&act.Verb,
			&act.ActorID,
			&act.ObjectID,
			&act.TargetID,
			pq.Array(&recipients),
			&payloadBytes,
			&act.DedupKey,
			&act.CreatedAt,
		); err != nil {
			return nil, err
		}
		act.Recipients = recipients
		act.Payload = json.RawMessage(payloadBytes)
		activities = append(activities, act)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return activities, nil
}

func (s *PostgresStore) ReconcileCelebrityStatus(ctx context.Context, projectID string, threshold int) (int64, error) {
	// First upsert any followed users into users table if not existing
	insertMissingQuery := `
		INSERT INTO users (project_id, external_user_id, follower_count, is_celebrity)
		SELECT f.project_id, f.followee_id, COUNT(f.follower_id), (COUNT(f.follower_id) >= $2)
		FROM follows f
		WHERE f.project_id = $1
		GROUP BY f.project_id, f.followee_id
		ON CONFLICT (project_id, external_user_id) DO UPDATE
		SET follower_count = EXCLUDED.follower_count,
		    is_celebrity = EXCLUDED.is_celebrity
	`
	res, err := s.db.ExecContext(ctx, insertMissingQuery, projectID, threshold)
	if err != nil {
		return 0, fmt.Errorf("failed to reconcile celebrity status: %w", err)
	}
	rows, _ := res.RowsAffected()
	return rows, nil
}

func (s *PostgresStore) UpsertTemplate(ctx context.Context, tmpl *model.NotificationTemplate) error {
	query := `
		INSERT INTO templates (id, project_id, name, event_type, channel, subject_template, body_template, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (project_id, event_type, channel) DO UPDATE
		SET name = EXCLUDED.name,
		    subject_template = EXCLUDED.subject_template,
		    body_template = EXCLUDED.body_template,
		    updated_at = NOW()
	`
	if tmpl.ID == "" {
		tmpl.ID = uuid.New().String()
	}
	_, err := s.db.ExecContext(ctx, query, tmpl.ID, tmpl.ProjectID, tmpl.Name, tmpl.Verb, tmpl.Channel, tmpl.SubjectTemplate, tmpl.BodyTemplate)
	if err != nil {
		return fmt.Errorf("failed to upsert template: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetTemplate(ctx context.Context, projectID, verb, channel string) (*model.NotificationTemplate, error) {
	query := `
		SELECT id, project_id, COALESCE(name, ''), event_type, channel, COALESCE(subject_template, ''), body_template, updated_at
		FROM templates
		WHERE project_id = $1 AND event_type = $2 AND channel = $3
	`
	var tmpl model.NotificationTemplate
	err := s.db.QueryRowContext(ctx, query, projectID, verb, channel).Scan(
		&tmpl.ID,
		&tmpl.ProjectID,
		&tmpl.Name,
		&tmpl.Verb,
		&tmpl.Channel,
		&tmpl.SubjectTemplate,
		&tmpl.BodyTemplate,
		&tmpl.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	return &tmpl, nil
}

func (s *PostgresStore) UpsertUserPreferences(ctx context.Context, projectID, externalUserID string, prefBytes []byte) error {
	// Ensure user exists
	_, _ = s.db.ExecContext(ctx, `INSERT INTO users (project_id, external_user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, projectID, externalUserID)

	query := `
		INSERT INTO user_preferences (project_id, user_id, preferences, updated_at)
		SELECT $1, u.id, $3::jsonb, NOW()
		FROM users u WHERE u.project_id = $1 AND u.external_user_id = $2
		ON CONFLICT (project_id, user_id) DO UPDATE
		SET preferences = EXCLUDED.preferences, updated_at = NOW()
	`
	_, err := s.db.ExecContext(ctx, query, projectID, externalUserID, prefBytes)
	if err != nil {
		return fmt.Errorf("failed to upsert user preferences: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetUserPreferences(ctx context.Context, projectID, externalUserID string) ([]byte, error) {
	query := `
		SELECT up.preferences
		FROM user_preferences up
		JOIN users u ON up.project_id = u.project_id AND up.user_id = u.id
		WHERE up.project_id = $1 AND u.external_user_id = $2
	`
	var prefBytes []byte
	err := s.db.QueryRowContext(ctx, query, projectID, externalUserID).Scan(&prefBytes)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user preferences: %w", err)
	}
	return prefBytes, nil
}

func (s *PostgresStore) SaveDLQMessage(ctx context.Context, dlq *model.DLQMessage) error {
	if dlq.ID == "" {
		dlq.ID = uuid.New().String()
	}
	query := `
		INSERT INTO dlq_messages (id, project_id, event_id, channel, recipient_id, error_message, retry_count, payload, created_at)
		VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, NOW())
	`
	payloadBytes := dlq.Payload
	if len(payloadBytes) == 0 {
		payloadBytes = []byte("{}")
	}
	_, err := s.db.ExecContext(ctx, query, dlq.ID, dlq.ProjectID, dlq.EventID, dlq.Channel, dlq.RecipientID, dlq.ErrorMessage, dlq.RetryCount, payloadBytes)
	if err != nil {
		return fmt.Errorf("failed to save DLQ message: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetDLQMessages(ctx context.Context, projectID string, limit, offset int) ([]model.DLQMessage, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	query := `
		SELECT id, project_id, COALESCE(event_id::text, ''), channel, recipient_id, error_message, retry_count, payload, created_at, replayed_at
		FROM dlq_messages
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := s.db.QueryContext(ctx, query, projectID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query DLQ messages: %w", err)
	}
	defer rows.Close()

	var messages []model.DLQMessage
	for rows.Next() {
		var msg model.DLQMessage
		var payloadBytes []byte
		var replayedAt sql.NullTime
		if err := rows.Scan(
			&msg.ID,
			&msg.ProjectID,
			&msg.EventID,
			&msg.Channel,
			&msg.RecipientID,
			&msg.ErrorMessage,
			&msg.RetryCount,
			&payloadBytes,
			&msg.CreatedAt,
			&replayedAt,
		); err != nil {
			return nil, err
		}
		msg.Payload = payloadBytes
		if replayedAt.Valid {
			msg.ReplayedAt = &replayedAt.Time
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *PostgresStore) GetDLQMessageByID(ctx context.Context, id string) (*model.DLQMessage, error) {
	query := `
		SELECT id, project_id, COALESCE(event_id::text, ''), channel, recipient_id, error_message, retry_count, payload, created_at, replayed_at
		FROM dlq_messages
		WHERE id = $1
	`
	var msg model.DLQMessage
	var payloadBytes []byte
	var replayedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&msg.ID,
		&msg.ProjectID,
		&msg.EventID,
		&msg.Channel,
		&msg.RecipientID,
		&msg.ErrorMessage,
		&msg.RetryCount,
		&payloadBytes,
		&msg.CreatedAt,
		&replayedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get DLQ message: %w", err)
	}
	msg.Payload = payloadBytes
	if replayedAt.Valid {
		msg.ReplayedAt = &replayedAt.Time
	}
	return &msg, nil
}

func (s *PostgresStore) MarkDLQMessageReplayed(ctx context.Context, id string) error {
	query := `UPDATE dlq_messages SET replayed_at = NOW() WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark DLQ message replayed: %w", err)
	}
	return nil
}

func (s *PostgresStore) SaveAPIKey(ctx context.Context, key *model.APIKey) error {
	if key.ID == "" {
		key.ID = uuid.New().String()
	}
	query := `
		INSERT INTO api_keys (id, project_id, key_hash, name, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	_, err := s.db.ExecContext(ctx, query, key.ID, key.ProjectID, key.KeyHash, key.Name)
	if err != nil {
		return fmt.Errorf("failed to save API key: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetAPIKeyByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	query := `
		SELECT id, project_id, key_hash, name, created_at, revoked_at
		FROM api_keys
		WHERE key_hash = $1
	`
	var key model.APIKey
	var revokedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, keyHash).Scan(
		&key.ID,
		&key.ProjectID,
		&key.KeyHash,
		&key.Name,
		&key.CreatedAt,
		&revokedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get API key by hash: %w", err)
	}
	if revokedAt.Valid {
		key.RevokedAt = &revokedAt.Time
	}
	return &key, nil
}

func (s *PostgresStore) RevokeAPIKey(ctx context.Context, projectID, keyID string) error {
	query := `UPDATE api_keys SET revoked_at = NOW() WHERE project_id = $1 AND id = $2`
	_, err := s.db.ExecContext(ctx, query, projectID, keyID)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}
	return nil
}




