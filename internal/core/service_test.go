package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"codex-switch/internal/profiles"
	"codex-switch/internal/usage"
)

type fakeUsageReader struct {
	results map[string]*usage.Snapshot
	errs    map[string]error
}

func (f fakeUsageReader) FetchFromAuthFile(_ context.Context, authPath string) (*usage.Snapshot, error) {
	if err := f.errs[authPath]; err != nil {
		return nil, err
	}
	if snapshot := f.results[authPath]; snapshot != nil {
		return snapshot, nil
	}
	return nil, errors.New("missing snapshot")
}

func TestBuildAccountTargetsCurrentWinsAndComesFirst(t *testing.T) {
	targets := buildAccountTargets(
		"bob@example.com",
		[]profiles.Profile{
			{Email: "alice@example.com", Path: "/profiles/alice.json"},
			{Email: "bob@example.com", Path: "/profiles/bob.json"},
		},
		"/home/user/.codex/auth.json",
	)

	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}

	if targets[0].Email != "bob@example.com" || !targets[0].Current {
		t.Fatalf("expected current target first, got %+v", targets[0])
	}

	if targets[0].AuthPath != "/home/user/.codex/auth.json" {
		t.Fatalf("expected current auth path to win, got %q", targets[0].AuthPath)
	}

	if targets[1].Email != "alice@example.com" {
		t.Fatalf("expected alice second, got %+v", targets[1])
	}
}

func TestBuildOverviewIncludesUsageAndErrors(t *testing.T) {
	store := &profiles.Store{
		CodexDir:    "/home/user/.codex",
		ProfilesDir: "/profiles",
	}

	refreshedAt := time.Date(2026, 4, 1, 20, 15, 0, 0, time.UTC)
	fiveHourResetAt := refreshedAt.Add(2*time.Hour + 18*time.Minute)
	weeklyResetAt := refreshedAt.Add(7 * 24 * time.Hour)
	overview := buildOverview(
		store,
		"bob@example.com",
		[]profiles.Profile{
			{Email: "alice@example.com", Path: "/profiles/alice.json"},
		},
		map[string]accountUsageResult{
			"bob@example.com": {
				Snapshot: &usage.Snapshot{
					Email:             "bob@example.com",
					PlanType:          "plus",
					FiveHourRemaining: 92,
					WeeklyRemaining:   88,
					UpdatedAt:         refreshedAt,
					FiveHourResetAt:   fiveHourResetAt,
					WeeklyResetAt:     weeklyResetAt,
				},
			},
			"alice@example.com": {
				Error: "read rate limits: boom",
			},
		},
		refreshedAt,
	)

	if overview.CurrentEmail != "bob@example.com" {
		t.Fatalf("unexpected current email: %q", overview.CurrentEmail)
	}

	if overview.ProfilesDir != "/profiles" {
		t.Fatalf("unexpected profiles dir: %q", overview.ProfilesDir)
	}

	if len(overview.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(overview.Accounts))
	}

	current := overview.Accounts[0]
	if !current.Current || !current.CanSave {
		t.Fatalf("expected current account metadata, got %+v", current)
	}
	if current.Usage == nil || current.Usage.PlanType != "plus" {
		t.Fatalf("expected current usage snapshot, got %+v", current.Usage)
	}
	if !current.Usage.FiveHourResetAt.Equal(fiveHourResetAt) {
		t.Fatalf("expected five-hour reset time to propagate, got %+v", current.Usage)
	}
	if !current.Usage.WeeklyResetAt.Equal(weeklyResetAt) {
		t.Fatalf("expected weekly reset time to propagate, got %+v", current.Usage)
	}

	other := overview.Accounts[1]
	if other.Current || other.CanSave {
		t.Fatalf("expected saved secondary account, got %+v", other)
	}
	if other.UsageState != "error" || other.UsageError == "" {
		t.Fatalf("expected usage error, got %+v", other)
	}
}

func TestServiceOverviewReadsSavedProfilesAndCurrentAuth(t *testing.T) {
	fixtureDir := t.TempDir()
	store := &profiles.Store{
		CodexDir:    filepath.Join(fixtureDir, ".codex"),
		ProfilesDir: filepath.Join(fixtureDir, "profiles"),
	}

	writeAuthFile(t, store.CodexDir, "bob@example.com")
	if _, err := store.SaveCurrent(); err != nil {
		t.Fatalf("save current profile: %v", err)
	}
	writeAuthFile(t, store.CodexDir, "bob@example.com")
	writeSavedProfile(t, store, "alice@example.com")

	service := NewService(store, fakeUsageReader{
		results: map[string]*usage.Snapshot{
			filepath.Join(store.CodexDir, "auth.json"): {
				Email:             "bob@example.com",
				PlanType:          "plus",
				FiveHourRemaining: 95,
				WeeklyRemaining:   90,
				UpdatedAt:         time.Now(),
				FiveHourResetAt:   time.Now().Add(30 * time.Minute),
				WeeklyResetAt:     time.Now().Add(72 * time.Hour),
			},
			filepath.Join(store.ProfilesDir, "alice@example.com.json"): {
				Email:             "alice@example.com",
				PlanType:          "plus",
				FiveHourRemaining: 80,
				WeeklyRemaining:   70,
				UpdatedAt:         time.Now(),
				FiveHourResetAt:   time.Now().Add(90 * time.Minute),
				WeeklyResetAt:     time.Now().Add(24 * time.Hour),
			},
		},
	})

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}

	if len(overview.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(overview.Accounts))
	}
	if !overview.Accounts[0].Current || overview.Accounts[1].Current {
		t.Fatalf("expected current account first, got %+v", overview.Accounts)
	}
}

func TestServiceOverviewWithoutUsageReturnsAccountsImmediately(t *testing.T) {
	fixtureDir := t.TempDir()
	store := &profiles.Store{
		CodexDir:    filepath.Join(fixtureDir, ".codex"),
		ProfilesDir: filepath.Join(fixtureDir, "profiles"),
	}

	writeAuthFile(t, store.CodexDir, "bob@example.com")
	if _, err := store.SaveCurrent(); err != nil {
		t.Fatalf("save current profile: %v", err)
	}
	writeSavedProfile(t, store, "alice@example.com")
	writeAuthFile(t, store.CodexDir, "bob@example.com")

	service := NewService(store, fakeUsageReader{})

	overview, err := service.OverviewWithoutUsage()
	if err != nil {
		t.Fatalf("OverviewWithoutUsage returned error: %v", err)
	}

	if len(overview.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(overview.Accounts))
	}
	if overview.Accounts[0].UsageState != "idle" || overview.Accounts[1].UsageState != "idle" {
		t.Fatalf("expected idle usage states, got %+v", overview.Accounts)
	}
	if !overview.Accounts[0].Current || !overview.Accounts[1].CanSwitch {
		t.Fatalf("expected immediate switch metadata, got %+v", overview.Accounts)
	}
}

func writeSavedProfile(t *testing.T, store *profiles.Store, email string) {
	t.Helper()
	writeAuthFile(t, store.CodexDir, email)
	if _, err := store.SaveCurrent(); err != nil {
		t.Fatalf("save profile %s: %v", email, err)
	}
}

func writeAuthFile(t *testing.T, codexDir string, email string) {
	t.Helper()

	payload, err := json.Marshal(map[string]any{"email": email})
	if err != nil {
		t.Fatalf("marshal jwt payload: %v", err)
	}

	header, err := json.Marshal(map[string]any{
		"alg": "none",
		"typ": "JWT",
	})
	if err != nil {
		t.Fatalf("marshal jwt header: %v", err)
	}

	raw, err := json.Marshal(map[string]any{
		"auth_mode": "chatgpt",
		"tokens": map[string]any{
			"id_token": base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload) + ".",
		},
	})
	if err != nil {
		t.Fatalf("marshal auth json: %v", err)
	}

	authPath := filepath.Join(codexDir, "auth.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o700); err != nil {
		t.Fatalf("create auth dir: %v", err)
	}
	if err := os.WriteFile(authPath, raw, 0o600); err != nil {
		t.Fatalf("write auth file: %v", err)
	}
}
