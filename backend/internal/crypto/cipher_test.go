package crypto

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestCipherEncryptDecrypt(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	c, err := NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher() error = %v", err)
	}

	plaintext := []byte("secret")
	ciphertext, nonce, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if bytes.Contains(ciphertext, plaintext) {
		t.Fatal("ciphertext contains plaintext")
	}
	if len(nonce) != 12 {
		t.Fatalf("nonce size = %d, want 12", len(nonce))
	}

	got, err := c.Decrypt(ciphertext, nonce)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("Decrypt() = %q, want %q", got, plaintext)
	}
}

func TestCipherUsesDifferentNonces(t *testing.T) {
	c, err := NewCipher("developer-friendly-key")
	if err != nil {
		t.Fatalf("NewCipher() error = %v", err)
	}

	_, nonce1, err := c.Encrypt([]byte("secret"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	_, nonce2, err := c.Encrypt([]byte("secret"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if bytes.Equal(nonce1, nonce2) {
		t.Fatal("nonces should differ")
	}
}

func TestCipherRejectsEmptyKey(t *testing.T) {
	if _, err := NewCipher(""); err != ErrEmptyMasterKey {
		t.Fatalf("NewCipher(\"\") error = %v, want %v", err, ErrEmptyMasterKey)
	}
}
