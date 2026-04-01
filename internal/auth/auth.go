package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type authFile struct {
	Tokens struct {
		IDToken string `json:"id_token"`
	} `json:"tokens"`
}

type idTokenClaims struct {
	Email string `json:"email"`
}

func EmailFromAuthBytes(data []byte) (string, error) {
	var auth authFile
	if err := json.Unmarshal(data, &auth); err != nil {
		return "", fmt.Errorf("parse auth.json: %w", err)
	}

	if auth.Tokens.IDToken == "" {
		return "", errors.New("auth.json does not contain tokens.id_token")
	}

	parts := strings.Split(auth.Tokens.IDToken, ".")
	if len(parts) < 2 {
		return "", errors.New("id_token is not a valid JWT")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decode id_token payload: %w", err)
	}

	var claims idTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("parse id_token claims: %w", err)
	}

	email := strings.TrimSpace(strings.ToLower(claims.Email))
	if email == "" {
		return "", errors.New("id_token does not contain email claim")
	}

	return email, nil
}
