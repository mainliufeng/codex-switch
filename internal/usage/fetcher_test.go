package usage

import (
	"testing"
	"time"
)

func TestBuildSnapshotIncludesResetTimes(t *testing.T) {
	updatedAt := time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC)
	fiveHourResetAt := updatedAt.Add(2 * time.Hour)
	weeklyResetAt := updatedAt.Add(7 * 24 * time.Hour)

	snapshot := buildSnapshot(
		accountResult{
			Account: struct {
				Email string `json:"email"`
			}{
				Email: "alice@example.com",
			},
		},
		rateLimitsResult{
			RateLimits: struct {
				PlanType  string          `json:"planType"`
				Primary   rateLimitWindow `json:"primary"`
				Secondary rateLimitWindow `json:"secondary"`
			}{
				PlanType: "plus",
				Primary: rateLimitWindow{
					UsedPercent: 26,
					ResetsAt:    fiveHourResetAt.Unix(),
				},
				Secondary: rateLimitWindow{
					UsedPercent: 37,
					ResetsAt:    weeklyResetAt.Unix(),
				},
			},
		},
		updatedAt,
	)

	if snapshot.Email != "alice@example.com" {
		t.Fatalf("unexpected snapshot email: %q", snapshot.Email)
	}
	if snapshot.FiveHourRemaining != 74 {
		t.Fatalf("unexpected five-hour remaining: %d", snapshot.FiveHourRemaining)
	}
	if snapshot.WeeklyRemaining != 63 {
		t.Fatalf("unexpected weekly remaining: %d", snapshot.WeeklyRemaining)
	}
	if !snapshot.FiveHourResetAt.Equal(fiveHourResetAt) {
		t.Fatalf("unexpected five-hour reset time: %v", snapshot.FiveHourResetAt)
	}
	if !snapshot.WeeklyResetAt.Equal(weeklyResetAt) {
		t.Fatalf("unexpected weekly reset time: %v", snapshot.WeeklyResetAt)
	}
}

func TestParseResetTimeHandlesSecondsAndMilliseconds(t *testing.T) {
	base := time.Unix(1_800_000_000, 0).UTC()

	if got := parseResetTime(base.Unix()); !got.Equal(base) {
		t.Fatalf("expected seconds timestamp, got %v", got)
	}

	if got := parseResetTime(base.UnixMilli()); !got.Equal(base) {
		t.Fatalf("expected millisecond timestamp, got %v", got)
	}

	if got := parseResetTime(0); !got.IsZero() {
		t.Fatalf("expected zero timestamp to stay zero, got %v", got)
	}
}
