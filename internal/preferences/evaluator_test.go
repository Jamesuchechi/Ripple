package preferences

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestIsInQuietHours_OvernightWindow(t *testing.T) {
	qh := QuietHours{
		Enabled:   true,
		StartTime: "22:00",
		EndTime:   "08:00",
		Timezone:  "UTC",
	}

	// 23:30 UTC -> inside quiet hours
	t1 := time.Date(2026, 9, 6, 23, 30, 0, 0, time.UTC)
	if !IsInQuietHours(t1, qh) {
		t.Errorf("Expected 23:30 UTC to be in quiet hours (22:00-08:00)")
	}

	// 03:15 UTC -> inside quiet hours
	t2 := time.Date(2026, 9, 6, 3, 15, 0, 0, time.UTC)
	if !IsInQuietHours(t2, qh) {
		t.Errorf("Expected 03:15 UTC to be in quiet hours (22:00-08:00)")
	}

	// 12:00 UTC -> outside quiet hours
	t3 := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	if IsInQuietHours(t3, qh) {
		t.Errorf("Expected 12:00 UTC to be outside quiet hours (22:00-08:00)")
	}
}

func TestIsInQuietHours_SameDayWindow(t *testing.T) {
	qh := QuietHours{
		Enabled:   true,
		StartTime: "13:00",
		EndTime:   "15:00",
		Timezone:  "UTC",
	}

	// 14:00 UTC -> inside quiet hours
	t1 := time.Date(2026, 9, 6, 14, 0, 0, 0, time.UTC)
	if !IsInQuietHours(t1, qh) {
		t.Errorf("Expected 14:00 UTC to be inside quiet hours (13:00-15:00)")
	}

	// 16:00 UTC -> outside quiet hours
	t2 := time.Date(2026, 9, 6, 16, 0, 0, 0, time.UTC)
	if IsInQuietHours(t2, qh) {
		t.Errorf("Expected 16:00 UTC to be outside quiet hours (13:00-15:00)")
	}
}

func TestIsInQuietHours_Disabled(t *testing.T) {
	qh := QuietHours{
		Enabled:   false,
		StartTime: "22:00",
		EndTime:   "08:00",
		Timezone:  "UTC",
	}

	t1 := time.Date(2026, 9, 6, 23, 30, 0, 0, time.UTC)
	if IsInQuietHours(t1, qh) {
		t.Errorf("Expected IsInQuietHours = false when Enabled = false")
	}
}

func TestPreferenceEvaluator_ShouldDeliver(t *testing.T) {
	evaluator := NewPreferenceEvaluator(nil)

	// No store configured should allow delivery
	deliver, reason := evaluator.ShouldDeliver(context.Background(), "proj_1", "user_1", "email", "like", time.Now())
	if !deliver || reason != "" {
		t.Errorf("Expected deliver=true, got deliver=%v, reason=%s", deliver, reason)
	}
}

func TestUserPreferences_JSONSerialization(t *testing.T) {
	pref := UserPreferences{
		ProjectID: "proj_1",
		UserID:    "user_1",
		EnabledChannels: map[string]bool{
			"email": true,
			"push":  false,
		},
		DisabledVerbs: map[string]bool{
			"like": true,
		},
		QuietHours: QuietHours{
			Enabled:   true,
			StartTime: "22:00",
			EndTime:   "08:00",
			Timezone:  "America/New_York",
		},
	}

	data, err := json.Marshal(pref)
	if err != nil {
		t.Fatalf("Failed to marshal UserPreferences: %v", err)
	}

	var unmarshaled UserPreferences
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal UserPreferences: %v", err)
	}

	if unmarshaled.EnabledChannels["push"] != false {
		t.Errorf("Expected push channel disabled")
	}
	if unmarshaled.QuietHours.Timezone != "America/New_York" {
		t.Errorf("Expected timezone 'America/New_York', got '%s'", unmarshaled.QuietHours.Timezone)
	}
}
