package auth_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"codex-switch/internal/auth"
)

func TestEmailFromAuthBytes(t *testing.T) {
	data := map[string]any{
		"auth_mode": "chatgpt",
		"tokens": map[string]any{
			"id_token": makeJWT(t, map[string]any{
				"email": "alice@example.com",
			}),
		},
	}

	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal auth json: %v", err)
	}

	email, err := auth.EmailFromAuthBytes(raw)
	if err != nil {
		t.Fatalf("EmailFromAuthBytes returned error: %v", err)
	}

	if email != "alice@example.com" {
		t.Fatalf("expected alice@example.com, got %q", email)
	}
}

func TestEmailFromAuthBytesRequiresEmailClaim(t *testing.T) {
	data := map[string]any{
		"tokens": map[string]any{
			"id_token": makeJWT(t, map[string]any{
				"sub": "user-123",
			}),
		},
	}

	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal auth json: %v", err)
	}

	if _, err := auth.EmailFromAuthBytes(raw); err == nil {
		t.Fatal("expected error when email claim is missing")
	}
}

func makeJWT(t *testing.T, claims map[string]any) string {
	t.Helper()

	header := encodeSegment(t, map[string]any{
		"alg": "none",
		"typ": "JWT",
	})
	payload := encodeSegment(t, claims)

	return header + "." + payload + "."
}

func encodeSegment(t *testing.T, value map[string]any) string {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal jwt segment: %v", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw)
}
