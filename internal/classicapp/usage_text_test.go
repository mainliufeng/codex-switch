package classicapp

import (
	"testing"
	"time"

	"codex-switch/internal/core"
)

func TestFormatProgressBar(t *testing.T) {
	tests := []struct {
		name      string
		remaining int
		expected  string
	}{
		{name: "zero", remaining: 0, expected: "░░░░░░░░░░"},
		{name: "mid", remaining: 50, expected: "█████░░░░░"},
		{name: "full", remaining: 100, expected: "██████████"},
		{name: "clamp_low", remaining: -5, expected: "░░░░░░░░░░"},
		{name: "clamp_high", remaining: 200, expected: "██████████"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := formatProgressBar(test.remaining); actual != test.expected {
				t.Fatalf("expected %q, got %q", test.expected, actual)
			}
		})
	}
}

func TestBuildSwitchEntriesIncludesClickableAccountRowsAndUsageRows(t *testing.T) {
	updatedAt := time.Date(2026, 4, 1, 21, 18, 0, 0, time.FixedZone("CST", 8*60*60))
	overview := core.Overview{
		Accounts: []core.Account{
			{
				Email:      "alice@example.com",
				Current:    true,
				UsageState: "ready",
				Usage: &core.UsageState{
					PlanType:          "plus",
					UpdatedAt:         updatedAt,
					FiveHourRemaining: 50,
					WeeklyRemaining:   80,
				},
			},
			{
				Email:      "bob@example.com",
				Current:    false,
				CanSwitch:  true,
				UsageState: "error",
				UsageError: "codex CLI not found",
			},
		},
	}

	entries := buildSwitchEntries(overview)
	if len(entries) != 7 {
		t.Fatalf("expected 7 switch rows, got %d", len(entries))
	}

	if entries[0].Title != "[当前] alice@example.com" {
		t.Fatalf("unexpected current title row: %q", entries[0].Title)
	}
	if !entries[0].Disabled {
		t.Fatal("expected current account row to be disabled")
	}
	if entries[1].Title != "５ｈ  █████░░░░░  50%" {
		t.Fatalf("unexpected five-hour row: %q", entries[1].Title)
	}
	if entries[1].Tooltip != "5h 剩余 50%" {
		t.Fatalf("unexpected five-hour tooltip: %q", entries[1].Tooltip)
	}
	if entries[2].Title != "周窗  ████████░░  80%" {
		t.Fatalf("unexpected weekly row: %q", entries[2].Title)
	}
	if entries[2].Tooltip != "周剩余 80%" {
		t.Fatalf("unexpected weekly tooltip: %q", entries[2].Tooltip)
	}
	if entries[3].Title != " " {
		t.Fatalf("expected separator row, got %q", entries[3].Title)
	}
	if entries[4].Title != "[可切换] bob@example.com" {
		t.Fatalf("unexpected second account title: %q", entries[4].Title)
	}
	if entries[4].Disabled {
		t.Fatal("expected switchable account row to stay enabled")
	}
	if entries[5].Title != "５ｈ  暂不可用" {
		t.Fatalf("unexpected loading/error five-hour row: %q", entries[5].Title)
	}
	if entries[6].Title != "周窗  暂不可用" {
		t.Fatalf("unexpected loading/error weekly row: %q", entries[6].Title)
	}
	if entries[6].Tooltip != "codex CLI not found" {
		t.Fatalf("unexpected error tooltip: %q", entries[6].Tooltip)
	}
}
