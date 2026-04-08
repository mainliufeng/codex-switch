package usage

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultTimeout = 15 * time.Second
)

type Snapshot struct {
	Email             string
	PlanType          string
	FiveHourRemaining int
	WeeklyRemaining   int
	FiveHourResetAt   time.Time
	WeeklyResetAt     time.Time
	UpdatedAt         time.Time
}

type Fetcher struct {
	Executable string
}

func NewFetcher() *Fetcher {
	return &Fetcher{}
}

func (f *Fetcher) FetchFromAuthFile(ctx context.Context, authPath string) (*Snapshot, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	resolvedExec, err := f.resolveExecutable()
	if err != nil {
		return nil, err
	}

	tempHome, err := os.MkdirTemp("", "codex-switch-home-")
	if err != nil {
		return nil, fmt.Errorf("create temp home: %w", err)
	}
	defer os.RemoveAll(tempHome)

	tempCodexDir := filepath.Join(tempHome, ".codex")
	if err := os.MkdirAll(tempCodexDir, 0o700); err != nil {
		return nil, fmt.Errorf("create temp codex dir: %w", err)
	}

	authBytes, err := os.ReadFile(authPath)
	if err != nil {
		return nil, fmt.Errorf("read auth file %s: %w", authPath, err)
	}
	if err := os.WriteFile(filepath.Join(tempCodexDir, "auth.json"), authBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write temp auth.json: %w", err)
	}

	client, err := startRPCClient(ctx, resolvedExec, tempHome)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if _, err := client.Request(ctx, 1, "initialize", map[string]any{
		"clientInfo": map[string]any{
			"name":    "codex-switch",
			"version": "0.1.0",
		},
		"capabilities": map[string]any{},
	}); err != nil {
		return nil, fmt.Errorf("initialize app-server: %w", err)
	}

	if err := client.Notify("initialized", map[string]any{}); err != nil {
		return nil, fmt.Errorf("notify initialized: %w", err)
	}

	accountRaw, err := client.Request(ctx, 2, "account/read", map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("read account: %w", err)
	}

	limitsRaw, err := client.Request(ctx, 3, "account/rateLimits/read", map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("read rate limits: %w", err)
	}

	var account accountResult
	if err := decodeResult(accountRaw, &account); err != nil {
		return nil, fmt.Errorf("decode account: %w", err)
	}

	var rateLimits rateLimitsResult
	if err := decodeResult(limitsRaw, &rateLimits); err != nil {
		return nil, fmt.Errorf("decode rate limits: %w", err)
	}

	return buildSnapshot(account, rateLimits, time.Now()), nil
}

type accountResult struct {
	Account struct {
		Email string `json:"email"`
	} `json:"account"`
}

type rateLimitsResult struct {
	RateLimits struct {
		PlanType  string          `json:"planType"`
		Primary   rateLimitWindow `json:"primary"`
		Secondary rateLimitWindow `json:"secondary"`
	} `json:"rateLimits"`
}

type rateLimitWindow struct {
	UsedPercent int   `json:"usedPercent"`
	ResetsAt    int64 `json:"resetsAt"`
}

type rpcClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
}

func startRPCClient(ctx context.Context, executable string, home string) (*rpcClient, error) {
	cmd := exec.CommandContext(ctx, executable, "-s", "read-only", "-a", "untrusted", "app-server")
	env := os.Environ()
	env = replaceEnv(env, "HOME", home)
	env = ensurePathContains(env, filepath.Dir(executable))
	cmd.Env = env

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("open stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start codex app-server: %w", err)
	}

	go io.Copy(io.Discard, stderrPipe)

	return &rpcClient{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdoutPipe),
	}, nil
}

func (c *rpcClient) Request(ctx context.Context, id int, method string, params map[string]any) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.send(map[string]any{
		"id":     id,
		"method": method,
		"params": params,
	}); err != nil {
		return nil, err
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line, err := c.stdout.ReadBytes('\n')
		if err != nil {
			return nil, err
		}

		var msg map[string]any
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		if _, ok := msg["method"]; ok && msg["id"] == nil {
			continue
		}

		msgID, ok := parseID(msg["id"])
		if !ok || msgID != id {
			continue
		}

		if errPayload, ok := msg["error"].(map[string]any); ok {
			if message, ok := errPayload["message"].(string); ok {
				return nil, errors.New(message)
			}
		}

		return msg, nil
	}
}

func (c *rpcClient) Notify(method string, params map[string]any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.send(map[string]any{
		"method": method,
		"params": params,
	})
}

func (c *rpcClient) send(payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if _, err := c.stdin.Write(append(raw, '\n')); err != nil {
		return err
	}

	return nil
}

func (c *rpcClient) Close() {
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_, _ = c.cmd.Process.Wait()
	}
}

func decodeResult(message map[string]any, out any) error {
	result, ok := message["result"]
	if !ok {
		return errors.New("missing result field")
	}

	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return json.Unmarshal(raw, out)
}

func parseID(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	default:
		return 0, false
	}
}

func (f *Fetcher) resolveExecutable() (string, error) {
	if f.Executable != "" {
		return f.Executable, nil
	}

	if path, err := exec.LookPath("codex"); err == nil {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	candidates := []string{
		filepath.Join(homeDir, ".npm-global", "bin", "codex"),
		filepath.Join(homeDir, ".local", "bin", "codex"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", errors.New("codex CLI not found")
}

func replaceEnv(env []string, key string, value string) []string {
	prefix := key + "="
	for index, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[index] = prefix + value
			return env
		}
	}

	return append(env, prefix+value)
}

func ensurePathContains(env []string, extra string) []string {
	if extra == "" {
		return env
	}

	currentPath := ""
	for _, entry := range env {
		if strings.HasPrefix(entry, "PATH=") {
			currentPath = strings.TrimPrefix(entry, "PATH=")
			break
		}
	}

	if currentPath == "" {
		return append(env, "PATH="+extra)
	}

	parts := strings.Split(currentPath, string(os.PathListSeparator))
	for _, part := range parts {
		if part == extra {
			return env
		}
	}

	return replaceEnv(env, "PATH", extra+string(os.PathListSeparator)+currentPath)
}

func buildSnapshot(account accountResult, rateLimits rateLimitsResult, updatedAt time.Time) *Snapshot {
	return &Snapshot{
		Email:             account.Account.Email,
		PlanType:          rateLimits.RateLimits.PlanType,
		FiveHourRemaining: max(0, 100-rateLimits.RateLimits.Primary.UsedPercent),
		WeeklyRemaining:   max(0, 100-rateLimits.RateLimits.Secondary.UsedPercent),
		FiveHourResetAt:   parseResetTime(rateLimits.RateLimits.Primary.ResetsAt),
		WeeklyResetAt:     parseResetTime(rateLimits.RateLimits.Secondary.ResetsAt),
		UpdatedAt:         updatedAt,
	}
}

func parseResetTime(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}

	if value >= 1_000_000_000_000 {
		return time.UnixMilli(value)
	}

	return time.Unix(value, 0)
}
