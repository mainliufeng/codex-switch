package profiles_test

import (
	"os"
	"path/filepath"
	"testing"

	"codex-switch/internal/profiles"
)

func TestSaveCurrentAndListProfiles(t *testing.T) {
	store := newTestStore(t)
	writeAuthFile(t, store.CodexDir, "alice@example.com")

	profile, err := store.SaveCurrent()
	if err != nil {
		t.Fatalf("SaveCurrent returned error: %v", err)
	}

	if profile.Email != "alice@example.com" {
		t.Fatalf("expected alice@example.com, got %q", profile.Email)
	}

	profilesList, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(profilesList) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profilesList))
	}

	if profilesList[0].Email != "alice@example.com" {
		t.Fatalf("expected alice@example.com in list, got %q", profilesList[0].Email)
	}
}

func TestSwitchReplacesCurrentAuthFile(t *testing.T) {
	store := newTestStore(t)
	writeAuthFile(t, store.CodexDir, "alice@example.com")

	if _, err := store.SaveCurrent(); err != nil {
		t.Fatalf("save alice profile: %v", err)
	}

	writeAuthFile(t, store.CodexDir, "bob@example.com")
	if _, err := store.SaveCurrent(); err != nil {
		t.Fatalf("save bob profile: %v", err)
	}

	if _, err := store.Switch("alice@example.com"); err != nil {
		t.Fatalf("Switch returned error: %v", err)
	}

	currentEmail, err := store.CurrentEmail()
	if err != nil {
		t.Fatalf("CurrentEmail returned error: %v", err)
	}

	if currentEmail != "alice@example.com" {
		t.Fatalf("expected alice@example.com, got %q", currentEmail)
	}
}

func TestDeleteRemovesSavedProfile(t *testing.T) {
	store := newTestStore(t)
	writeAuthFile(t, store.CodexDir, "alice@example.com")

	if _, err := store.SaveCurrent(); err != nil {
		t.Fatalf("SaveCurrent returned error: %v", err)
	}

	if err := store.Delete("alice@example.com"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	profilesList, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(profilesList) != 0 {
		t.Fatalf("expected 0 profiles, got %d", len(profilesList))
	}
}

func newTestStore(t *testing.T) *profiles.Store {
	t.Helper()

	root := t.TempDir()
	codexDir := filepath.Join(root, ".codex")
	profilesDir := filepath.Join(root, "profiles")

	if err := os.MkdirAll(codexDir, 0o700); err != nil {
		t.Fatalf("create codex dir: %v", err)
	}

	if err := os.MkdirAll(profilesDir, 0o700); err != nil {
		t.Fatalf("create profiles dir: %v", err)
	}

	return &profiles.Store{
		CodexDir:    codexDir,
		ProfilesDir: profilesDir,
	}
}
