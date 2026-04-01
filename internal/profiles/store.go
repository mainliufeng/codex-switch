package profiles

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"codex-switch/internal/auth"
)

const authFileName = "auth.json"

type Profile struct {
	Email     string
	Path      string
	UpdatedAt time.Time
}

type Store struct {
	CodexDir    string
	ProfilesDir string
}

func NewDefaultStore() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(homeDir, ".local", "share")
	}

	store := &Store{
		CodexDir:    filepath.Join(homeDir, ".codex"),
		ProfilesDir: filepath.Join(dataHome, "codex-switch", "profiles"),
	}

	if err := os.MkdirAll(store.CodexDir, 0o700); err != nil {
		return nil, fmt.Errorf("create codex directory: %w", err)
	}

	if err := os.MkdirAll(store.ProfilesDir, 0o700); err != nil {
		return nil, fmt.Errorf("create profiles directory: %w", err)
	}

	return store, nil
}

func (s *Store) SaveCurrent() (Profile, error) {
	authBytes, err := os.ReadFile(s.authPath())
	if err != nil {
		return Profile{}, fmt.Errorf("read current auth.json: %w", err)
	}

	email, err := auth.EmailFromAuthBytes(authBytes)
	if err != nil {
		return Profile{}, fmt.Errorf("read current email: %w", err)
	}

	profilePath := s.profilePath(email)
	if err := writeFileAtomically(profilePath, authBytes, 0o600); err != nil {
		return Profile{}, fmt.Errorf("save profile %s: %w", email, err)
	}

	info, err := os.Stat(profilePath)
	if err != nil {
		return Profile{}, fmt.Errorf("stat saved profile %s: %w", email, err)
	}

	return Profile{
		Email:     email,
		Path:      profilePath,
		UpdatedAt: info.ModTime(),
	}, nil
}

func (s *Store) List() ([]Profile, error) {
	entries, err := os.ReadDir(s.ProfilesDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read profiles directory: %w", err)
	}

	profiles := make([]Profile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(s.ProfilesDir, entry.Name())
		authBytes, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read profile %s: %w", path, err)
		}

		email, err := auth.EmailFromAuthBytes(authBytes)
		if err != nil {
			return nil, fmt.Errorf("read email from %s: %w", path, err)
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("stat profile %s: %w", path, err)
		}

		profiles = append(profiles, Profile{
			Email:     email,
			Path:      path,
			UpdatedAt: info.ModTime(),
		})
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Email < profiles[j].Email
	})

	return profiles, nil
}

func (s *Store) Switch(email string) (Profile, error) {
	email = normalizeEmail(email)
	if email == "" {
		return Profile{}, errors.New("email is required")
	}

	authBytes, err := os.ReadFile(s.profilePath(email))
	if err != nil {
		return Profile{}, fmt.Errorf("read saved profile %s: %w", email, err)
	}

	if _, err := auth.EmailFromAuthBytes(authBytes); err != nil {
		return Profile{}, fmt.Errorf("validate saved profile %s: %w", email, err)
	}

	if err := writeFileAtomically(s.authPath(), authBytes, 0o600); err != nil {
		return Profile{}, fmt.Errorf("replace current auth.json: %w", err)
	}

	info, err := os.Stat(s.profilePath(email))
	if err != nil {
		return Profile{}, fmt.Errorf("stat saved profile %s: %w", email, err)
	}

	return Profile{
		Email:     email,
		Path:      s.profilePath(email),
		UpdatedAt: info.ModTime(),
	}, nil
}

func (s *Store) Delete(email string) error {
	email = normalizeEmail(email)
	if email == "" {
		return errors.New("email is required")
	}

	if err := os.Remove(s.profilePath(email)); err != nil {
		return fmt.Errorf("delete saved profile %s: %w", email, err)
	}

	return nil
}

func (s *Store) CurrentEmail() (string, error) {
	authBytes, err := os.ReadFile(s.authPath())
	if err != nil {
		return "", fmt.Errorf("read current auth.json: %w", err)
	}

	email, err := auth.EmailFromAuthBytes(authBytes)
	if err != nil {
		return "", fmt.Errorf("read current email: %w", err)
	}

	return email, nil
}

func (s *Store) authPath() string {
	return filepath.Join(s.CodexDir, authFileName)
}

func (s *Store) profilePath(email string) string {
	return filepath.Join(s.ProfilesDir, normalizeEmail(email)+".json")
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func writeFileAtomically(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return err
	}

	if err := tempFile.Chmod(mode); err != nil {
		_ = tempFile.Close()
		return err
	}

	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, path)
}
