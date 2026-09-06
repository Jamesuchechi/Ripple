package auth_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"ripple/internal/model"
	"ripple/internal/service"
	"ripple/internal/store"
)

func setupTestDBAndRedisForSecurity(t *testing.T) (*store.PostgresStore, *store.RedisFeedStore, func()) {
	dbURL := "postgres://ripple:secret@localhost:5435/ripple?sslmode=disable"
	db, err := sql.Open("postgres", dbURL)
	if err != nil || db.Ping() != nil {
		t.Skip("PostgreSQL not accessible locally, skipping security test")
		return nil, nil, nil
	}

	redisURL := "redis://localhost:6382/0"
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		db.Close()
		t.Skip("Redis URL invalid, skipping security test")
		return nil, nil, nil
	}
	rClient := redis.NewClient(opt)
	if err := rClient.Ping(context.Background()).Err(); err != nil {
		db.Close()
		t.Skip("Redis not accessible locally, skipping security test")
		return nil, nil, nil
	}

	_, _ = db.Exec(`INSERT INTO projects (id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'Tenant A') ON CONFLICT DO NOTHING`)
	_, _ = db.Exec(`INSERT INTO projects (id, name) VALUES ('00000000-0000-0000-0000-000000000002', 'Tenant B') ON CONFLICT DO NOTHING`)
	_, _ = db.Exec(`TRUNCATE TABLE event_log, follows, users, user_preferences, api_keys CASCADE`)
	_ = rClient.FlushDB(context.Background()).Err()

	pgStore := store.NewPostgresStore(db)
	redisStore := store.NewRedisFeedStore(rClient, 1000)

	cleanup := func() {
		db.Close()
		rClient.Close()
	}
	return pgStore, redisStore, cleanup
}

func TestMultiTenantSecurityIsolation(t *testing.T) {
	pgStore, redisStore, cleanup := setupTestDBAndRedisForSecurity(t)
	if pgStore == nil {
		return
	}
	defer cleanup()

	svc := service.NewEventService(pgStore, redisStore, nil, 10000)
	ctx := context.Background()

	tenantA := "00000000-0000-0000-0000-000000000001"
	tenantB := "00000000-0000-0000-0000-000000000002"

	userID := "user_shared_id"

	// 1. Tenant A ingests event for user_shared_id
	_, err := svc.IngestEvent(ctx, tenantA, &model.IngestEventRequest{
		Verb:       "like",
		ActorID:    "actor_a",
		ObjectID:   "post_tenant_a",
		Recipients: []string{userID},
	})
	if err != nil {
		t.Fatalf("Tenant A IngestEvent failed: %v", err)
	}

	// 2. Tenant B ingests event for user_shared_id
	_, err = svc.IngestEvent(ctx, tenantB, &model.IngestEventRequest{
		Verb:       "comment",
		ActorID:    "actor_b",
		ObjectID:   "post_tenant_b",
		Recipients: []string{userID},
	})
	if err != nil {
		t.Fatalf("Tenant B IngestEvent failed: %v", err)
	}

	// 3. Query feed for Tenant A -> must NOT contain Tenant B activity
	feedA, err := svc.GetFeed(ctx, tenantA, userID, 10, 0)
	if err != nil {
		t.Fatalf("GetFeed Tenant A failed: %v", err)
	}
	for _, act := range feedA {
		if act.ObjectID == "post_tenant_b" {
			t.Fatalf("SECURITY VIOLATION: Tenant A feed contains Tenant B activity: %v", act)
		}
	}

	// 4. Query feed for Tenant B -> must NOT contain Tenant A activity
	feedB, err := svc.GetFeed(ctx, tenantB, userID, 10, 0)
	if err != nil {
		t.Fatalf("GetFeed Tenant B failed: %v", err)
	}
	for _, act := range feedB {
		if act.ObjectID == "post_tenant_a" {
			t.Fatalf("SECURITY VIOLATION: Tenant B feed contains Tenant A activity: %v", act)
		}
	}

	// 5. User preferences isolation
	_ = svc.UpsertUserPreferences(ctx, tenantA, userID, []byte(`{"channels":{"email":true}}`))
	_ = svc.UpsertUserPreferences(ctx, tenantB, userID, []byte(`{"channels":{"email":false}}`))

	prefA, _ := svc.GetUserPreferences(ctx, tenantA, userID)
	prefB, _ := svc.GetUserPreferences(ctx, tenantB, userID)

	if string(prefA) == string(prefB) {
		t.Fatalf("SECURITY VIOLATION: User preferences leaked across tenants")
	}
}
