package preferences

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ripple/internal/store"
)

type QuietHours struct {
	Enabled   bool   `json:"enabled"`
	StartTime string `json:"start_time"` // e.g. "22:00"
	EndTime   string `json:"end_time"`   // e.g. "08:00"
	Timezone  string `json:"timezone"`   // e.g. "America/New_York" or "UTC"
}

type UserPreferences struct {
	ProjectID       string          `json:"project_id"`
	UserID          string          `json:"user_id"`
	EnabledChannels map[string]bool `json:"channels,omitempty"`
	DisabledVerbs   map[string]bool `json:"disabled_verbs,omitempty"`
	QuietHours      QuietHours      `json:"quiet_hours"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type PreferenceEvaluator struct {
	pg *store.PostgresStore
}

func NewPreferenceEvaluator(pg *store.PostgresStore) *PreferenceEvaluator {
	return &PreferenceEvaluator{pg: pg}
}

// ShouldDeliver evaluates if a notification should be delivered to a recipient based on their preferences.
func (e *PreferenceEvaluator) ShouldDeliver(ctx context.Context, projectID, userID, channel, verb string, now time.Time) (bool, string) {
	if e.pg == nil {
		return true, ""
	}

	prefBytes, err := e.pg.GetUserPreferences(ctx, projectID, userID)
	if err != nil || len(prefBytes) == 0 {
		return true, ""
	}

	var pref UserPreferences
	if err := json.Unmarshal(prefBytes, &pref); err != nil {
		return true, ""
	}

	// Check channel preferences
	if pref.EnabledChannels != nil {
		if enabled, exists := pref.EnabledChannels[channel]; exists && !enabled {
			return false, fmt.Sprintf("channel %s disabled by user", channel)
		}
	}

	// Check verb preferences
	if pref.DisabledVerbs != nil {
		if disabled, exists := pref.DisabledVerbs[verb]; exists && disabled {
			return false, fmt.Sprintf("verb %s disabled by user", verb)
		}
	}

	// Check quiet hours
	if IsInQuietHours(now, pref.QuietHours) {
		return false, "recipient in quiet hours (Do-Not-Disturb)"
	}

	return true, ""
}

// IsInQuietHours checks if the current time falls within recipient quiet hours.
func IsInQuietHours(now time.Time, qh QuietHours) bool {
	if !qh.Enabled || qh.StartTime == "" || qh.EndTime == "" {
		return false
	}

	tz := qh.Timezone
	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	localNow := now.In(loc)
	currentMin := localNow.Hour()*60 + localNow.Minute()

	startMin, okStart := parseTimeToMinutes(qh.StartTime)
	endMin, okEnd := parseTimeToMinutes(qh.EndTime)
	if !okStart || !okEnd {
		return false
	}

	// Overnight window (e.g. 22:00 to 08:00)
	if startMin > endMin {
		return currentMin >= startMin || currentMin < endMin
	}

	// Same-day window (e.g. 13:00 to 15:00)
	return currentMin >= startMin && currentMin < endMin
}

func parseTimeToMinutes(tStr string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(tStr), ":")
	if len(parts) != 2 {
		return 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}
