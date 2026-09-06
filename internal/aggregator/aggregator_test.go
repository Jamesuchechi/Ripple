package aggregator

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"ripple/internal/model"
	"ripple/internal/store"
)

func setupTestDBAndRedisForAggregator(t *testing.T) (*store.PostgresStore, *redis.Client, func()) {
	dbURL := "postgres://ripple:secret@localhost:5435/ripple?sslmode=disable"
	db, err := sql.Open("postgres", dbURL)
	if err != nil || db.Ping() != nil {
		t.Skip("PostgreSQL not accessible locally, skipping test")
		return nil, nil, nil
	}

	redisURL := "redis://localhost:6382/0"
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		db.Close()
		t.Skip("Redis URL invalid, skipping test")
		return nil, nil, nil
	}
	rClient := redis.NewClient(opt)
	if err := rClient.Ping(context.Background()).Err(); err != nil {
		db.Close()
		t.Skip("Redis not accessible locally, skipping test")
		return nil, nil, nil
	}

	_, _ = db.Exec(`INSERT INTO projects (id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'Default Project') ON CONFLICT DO NOTHING`)
	_, _ = db.Exec(`ALTER TABLE templates ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_templates_proj_type_chan ON templates (project_id, event_type, channel)`)
	_, _ = db.Exec(`TRUNCATE TABLE templates CASCADE`)
	_ = rClient.FlushDB(context.Background()).Err()

	pgStore := store.NewPostgresStore(db)
	cleanup := func() {
		db.Close()
		rClient.Close()
	}
	return pgStore, rClient, cleanup
}

func TestTemplateEngine_Render(t *testing.T) {
	engine := NewTemplateEngine()

	ctx := &TemplateContext{
		ActorName:  "Alice",
		Count:      15,
		OtherCount: 14,
		Verb:       "like",
		ObjectID:   "photo_123",
	}

	tmplStr := "{{ actor_name }} and {{ other_count }} others liked your post {{ object_id }}"
	rendered := engine.Render(tmplStr, ctx)

	expected := "Alice and 14 others liked your post photo_123"
	if rendered != expected {
		t.Errorf("Expected '%s', got '%s'", expected, rendered)
	}
}

func TestCoalescer_CoalesceLogic(t *testing.T) {
	coalescer := NewCoalescer()

	// 1. Single activity
	acts1 := []model.Activity{
		{EventID: "e1", ActorID: "Alice", Verb: "like", ObjectID: "p1"},
	}
	res1 := coalescer.Coalesce(acts1, "", "")
	if res1.SummaryText != "Alice liked your post" {
		t.Errorf("Expected 'Alice liked your post', got '%s'", res1.SummaryText)
	}

	// 2. Two activities
	acts2 := []model.Activity{
		{EventID: "e1", ActorID: "Alice", Verb: "like", ObjectID: "p1"},
		{EventID: "e2", ActorID: "Bob", Verb: "like", ObjectID: "p1"},
	}
	res2 := coalescer.Coalesce(acts2, "", "")
	if res2.SummaryText != "Alice and Bob liked your post" {
		t.Errorf("Expected 'Alice and Bob liked your post', got '%s'", res2.SummaryText)
	}

	// 3. 15 activities ("Alice and 14 others liked your post")
	acts15 := make([]model.Activity, 15)
	acts15[0] = model.Activity{EventID: "e0", ActorID: "Alice", Verb: "like", ObjectID: "p1"}
	for i := 1; i < 15; i++ {
		acts15[i] = model.Activity{EventID: string(rune(i)), ActorID: "User", Verb: "like", ObjectID: "p1"}
	}
	res15 := coalescer.Coalesce(acts15, "New likes from {{ actor_name }}", "{{ actor_name }} and {{ other_count }} others liked your post")
	if res15.SummaryText != "Alice and 14 others liked your post" {
		t.Errorf("Expected 'Alice and 14 others liked your post', got '%s'", res15.SummaryText)
	}
	if res15.Subject != "New likes from Alice" {
		t.Errorf("Expected subject 'New likes from Alice', got '%s'", res15.Subject)
	}
	if res15.Body != "Alice and 14 others liked your post" {
		t.Errorf("Expected body 'Alice and 14 others liked your post', got '%s'", res15.Body)
	}
}

func TestBatchAggregator_SlidingWindow(t *testing.T) {
	pgStore, rClient, cleanup := setupTestDBAndRedisForAggregator(t)
	if pgStore == nil {
		return
	}
	defer cleanup()

	agg := NewBatchAggregator(rClient)
	ctx := context.Background()

	projectID := "00000000-0000-0000-0000-000000000001"
	recipientID := "user_recipient"
	verb := "like"

	// Add 3 activities to batch
	for i := 1; i <= 3; i++ {
		act := &model.Activity{
			EventID:   string(rune(i)),
			ProjectID: projectID,
			Verb:      verb,
			ActorID:   "user_actor",
			CreatedAt: time.Now(),
		}
		count, err := agg.AddActivity(ctx, projectID, recipientID, verb, act, 1*time.Minute)
		if err != nil {
			t.Fatalf("AddActivity failed: %v", err)
		}
		if count != int64(i) {
			t.Errorf("Expected batch count %d, got %d", i, count)
		}
	}

	// Retrieve batch
	batch, err := agg.GetBatch(ctx, projectID, recipientID, verb)
	if err != nil {
		t.Fatalf("GetBatch failed: %v", err)
	}
	if len(batch) != 3 {
		t.Fatalf("Expected batch size 3, got %d", len(batch))
	}

	// Clear batch
	err = agg.ClearBatch(ctx, projectID, recipientID, verb)
	if err != nil {
		t.Fatalf("ClearBatch failed: %v", err)
	}

	batchAfter, _ := agg.GetBatch(ctx, projectID, recipientID, verb)
	if len(batchAfter) != 0 {
		t.Errorf("Expected 0 items in batch after ClearBatch, got %d", len(batchAfter))
	}
}

func TestPostgresStore_TemplateCRUD(t *testing.T) {
	pgStore, _, cleanup := setupTestDBAndRedisForAggregator(t)
	if pgStore == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	projectID := "00000000-0000-0000-0000-000000000001"

	tmpl := &model.NotificationTemplate{
		ProjectID:       projectID,
		Name:            "Like Aggregation Email",
		Verb:            "like",
		Channel:         "email",
		SubjectTemplate: "New likes from {{ actor_name }}",
		BodyTemplate:    "{{ actor_name }} and {{ other_count }} others liked your post",
	}

	// 1. Upsert template
	err := pgStore.UpsertTemplate(ctx, tmpl)
	if err != nil {
		t.Fatalf("UpsertTemplate failed: %v", err)
	}

	// 2. Fetch template
	fetched, err := pgStore.GetTemplate(ctx, projectID, "like", "email")
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}
	if fetched == nil {
		t.Fatal("Expected non-nil template")
	}
	if fetched.SubjectTemplate != "New likes from {{ actor_name }}" {
		t.Errorf("Expected subject template 'New likes from {{ actor_name }}', got '%s'", fetched.SubjectTemplate)
	}
}
