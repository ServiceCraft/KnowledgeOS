package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateAPIKey executes the auth.GenerateAPIKey operation.
func GenerateAPIKey() (plaintext string, hash string, err error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate api key: %w", err)
	}
	plaintext = "kos_live_" + hex.EncodeToString(raw)
	hash = HashToken(plaintext)
	return plaintext, hash, nil
}

// HashAPIKey executes the auth.HashAPIKey operation.
func HashAPIKey(key string) string {
	return HashToken(key)
}
