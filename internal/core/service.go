package core

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"codex-switch/internal/profiles"
	"codex-switch/internal/usage"
)

type UsageReader interface {
	FetchFromAuthFile(ctx context.Context, authPath string) (*usage.Snapshot, error)
}

type Service struct {
	store  *profiles.Store
	reader UsageReader
}

type Overview struct {
	CurrentEmail string    `json:"currentEmail,omitempty"`
	ProfilesDir  string    `json:"profilesDir"`
	Accounts     []Account `json:"accounts"`
	RefreshedAt  time.Time `json:"refreshedAt"`
}

type Account struct {
	Email      string      `json:"email"`
	Current    bool        `json:"current"`
	CanSave    bool        `json:"canSave"`
	CanSwitch  bool        `json:"canSwitch"`
	CanDelete  bool        `json:"canDelete"`
	UsageState string      `json:"usageState"`
	UsageError string      `json:"usageError,omitempty"`
	Usage      *UsageState `json:"usage,omitempty"`
}

type UsageState struct {
	PlanType          string    `json:"planType,omitempty"`
	UpdatedAt         time.Time `json:"updatedAt,omitempty"`
	FiveHourRemaining int       `json:"fiveHourRemaining"`
	WeeklyRemaining   int       `json:"weeklyRemaining"`
}

type accountTarget struct {
	Email    string
	AuthPath string
	Current  bool
}

type accountUsageResult struct {
	Snapshot *usage.Snapshot
	Error    string
}

func NewService(store *profiles.Store, reader UsageReader) *Service {
	return &Service{
		store:  store,
		reader: reader,
	}
}

func (s *Service) Overview(ctx context.Context) (Overview, error) {
	currentEmail, _ := s.store.CurrentEmail()
	savedProfiles, err := s.store.List()
	if err != nil {
		return Overview{}, err
	}

	results := make(map[string]accountUsageResult)
	targets := buildAccountTargets(currentEmail, savedProfiles, filepath.Join(s.store.CodexDir, "auth.json"))
	for _, target := range targets {
		snapshot, fetchErr := s.reader.FetchFromAuthFile(ctx, target.AuthPath)
		if fetchErr != nil {
			results[target.Email] = accountUsageResult{Error: fetchErr.Error()}
			continue
		}
		results[target.Email] = accountUsageResult{Snapshot: snapshot}
	}

	return buildOverview(s.store, currentEmail, savedProfiles, results, time.Now()), nil
}

func (s *Service) OverviewWithoutUsage() (Overview, error) {
	currentEmail, _ := s.store.CurrentEmail()
	savedProfiles, err := s.store.List()
	if err != nil {
		return Overview{}, err
	}

	return buildOverview(s.store, currentEmail, savedProfiles, nil, time.Now()), nil
}

func (s *Service) SaveCurrent(ctx context.Context) (Overview, error) {
	if _, err := s.store.SaveCurrent(); err != nil {
		return Overview{}, err
	}
	return s.Overview(ctx)
}

func (s *Service) Switch(ctx context.Context, email string) (Overview, error) {
	if _, err := s.store.Switch(email); err != nil {
		return Overview{}, err
	}
	return s.Overview(ctx)
}

func (s *Service) Delete(ctx context.Context, email string) (Overview, error) {
	if err := s.store.Delete(email); err != nil {
		return Overview{}, err
	}
	return s.Overview(ctx)
}

func buildOverview(store *profiles.Store, currentEmail string, savedProfiles []profiles.Profile, results map[string]accountUsageResult, refreshedAt time.Time) Overview {
	targets := buildAccountTargets(currentEmail, savedProfiles, filepath.Join(store.CodexDir, "auth.json"))
	accounts := make([]Account, 0, len(targets))
	for _, target := range targets {
		account := Account{
			Email:      target.Email,
			Current:    target.Current,
			CanSave:    target.Current,
			CanSwitch:  !target.Current,
			CanDelete:  !target.Current,
			UsageState: "idle",
		}

		if result, ok := results[target.Email]; ok {
			switch {
			case result.Snapshot != nil:
				account.UsageState = "ready"
				account.Usage = &UsageState{
					PlanType:          result.Snapshot.PlanType,
					UpdatedAt:         result.Snapshot.UpdatedAt,
					FiveHourRemaining: result.Snapshot.FiveHourRemaining,
					WeeklyRemaining:   result.Snapshot.WeeklyRemaining,
				}
			case result.Error != "":
				account.UsageState = "error"
				account.UsageError = result.Error
			}
		}

		accounts = append(accounts, account)
	}

	return Overview{
		CurrentEmail: currentEmail,
		ProfilesDir:  store.ProfilesDir,
		Accounts:     accounts,
		RefreshedAt:  refreshedAt,
	}
}

func buildAccountTargets(currentEmail string, saved []profiles.Profile, currentAuthPath string) []accountTarget {
	byEmail := make(map[string]accountTarget)
	normalizedCurrent := normalizeEmail(currentEmail)
	if normalizedCurrent != "" {
		byEmail[normalizedCurrent] = accountTarget{
			Email:    normalizedCurrent,
			AuthPath: currentAuthPath,
			Current:  true,
		}
	}

	for _, profile := range saved {
		normalized := normalizeEmail(profile.Email)
		if normalized == "" {
			continue
		}

		existing, ok := byEmail[normalized]
		if ok && existing.Current {
			continue
		}

		byEmail[normalized] = accountTarget{
			Email:    normalized,
			AuthPath: profile.Path,
			Current:  existing.Current,
		}
	}

	targets := make([]accountTarget, 0, len(byEmail))
	for _, target := range byEmail {
		targets = append(targets, target)
	}

	slices.SortFunc(targets, func(a, b accountTarget) int {
		if a.Current != b.Current {
			if a.Current {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Email, b.Email)
	})

	return targets
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func RequireEmail(value string) (string, error) {
	email := normalizeEmail(value)
	if email == "" {
		return "", fmt.Errorf("email is required")
	}
	return email, nil
}
