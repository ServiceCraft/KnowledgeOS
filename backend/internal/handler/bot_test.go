package handler

import (
	"encoding/json"
	"testing"
)

func TestEnsureTelegramWebhookSecretGeneratesWhenMissing(t *testing.T) {
	for _, in := range []string{``, `{}`, `{"auth_type":"api_key"}`} {
		out, err := ensureTelegramWebhookSecret(json.RawMessage(in))
		if err != nil {
			t.Fatalf("ensureTelegramWebhookSecret(%q) error = %v", in, err)
		}
		var m map[string]any
		if err := json.Unmarshal(out, &m); err != nil {
			t.Fatalf("output not an object: %v", err)
		}
		secret, _ := m["webhook_secret"].(string)
		if len(secret) != 64 {
			t.Fatalf("webhook_secret = %q, want 64 hex chars", secret)
		}
	}
}

func TestEnsureTelegramWebhookSecretPreservesExisting(t *testing.T) {
	out, err := ensureTelegramWebhookSecret(json.RawMessage(`{"webhook_secret":"keepme"}`))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("not an object: %v", err)
	}
	if m["webhook_secret"] != "keepme" {
		t.Fatalf("webhook_secret = %v, want keepme", m["webhook_secret"])
	}
}

func TestEnsureTelegramWebhookSecretLeavesNonObjectUntouched(t *testing.T) {
	in := json.RawMessage(`[]`)
	out, err := ensureTelegramWebhookSecret(in)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if string(out) != "[]" {
		t.Fatalf("non-object metadata altered: %q", string(out))
	}
}

func TestRandomWebhookSecretIsUnique(t *testing.T) {
	a, err := randomWebhookSecret()
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	b, err := randomWebhookSecret()
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if a == b {
		t.Fatal("randomWebhookSecret() returned identical values")
	}
}
