package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	aes256KeySize = 32
	gcmNonceSize  = 12
)

var ErrEmptyMasterKey = errors.New("secrets encryption key is empty")

type Cipher struct {
	key []byte
}

// NewCipher executes the crypto.NewCipher operation.
func NewCipher(master string) (*Cipher, error) {
	key, err := normalizeKey(master)
	if err != nil {
		return nil, err
	}
	return &Cipher{key: key}, nil
}

// Encrypt executes the crypto.Cipher.Encrypt operation.
func (c *Cipher) Encrypt(plaintext []byte) ([]byte, []byte, error) {
	if c == nil {
		return nil, nil, ErrEmptyMasterKey
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, nil, fmt.Errorf("create aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create gcm: %w", err)
	}
	nonce := make([]byte, gcmNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}
	return gcm.Seal(nil, nonce, plaintext, nil), nonce, nil
}

// Decrypt executes the crypto.Cipher.Decrypt operation.
func (c *Cipher) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
	if c == nil {
		return nil, ErrEmptyMasterKey
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid nonce size: got %d", len(nonce))
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: %w", err)
	}
	return plaintext, nil
}

func normalizeKey(master string) ([]byte, error) {
	master = strings.TrimSpace(master)
	if master == "" {
		return nil, ErrEmptyMasterKey
	}

	if key, err := base64.StdEncoding.DecodeString(master); err == nil && len(key) == aes256KeySize {
		return key, nil
	}
	if key, err := hex.DecodeString(master); err == nil && len(key) == aes256KeySize {
		return key, nil
	}

	sum := sha256.Sum256([]byte(master))
	return sum[:], nil
}
