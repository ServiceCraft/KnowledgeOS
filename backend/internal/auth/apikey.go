package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateAPIKey() (plaintext string, hash string, err error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate api key: %w", err)
	}
	plaintext = "kos_live_" + hex.EncodeToString(raw)
	hash = HashToken(plaintext)
	return plaintext, hash, nil
}

func HashAPIKey(key string) string {
	return HashToken(key)
}
