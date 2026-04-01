package profiles_test

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeAuthFile(t *testing.T, codexDir string, email string) {
	t.Helper()

	payload, err := json.Marshal(map[string]any{
		"email": email,
	})
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
	if err := os.WriteFile(authPath, raw, 0o600); err != nil {
		t.Fatalf("write auth file: %v", err)
	}
}
