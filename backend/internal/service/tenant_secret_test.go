package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type fakeSecretRepo struct {
	items map[domain.SecretKind]*domain.TenantSecret
}

func newFakeSecretRepo() *fakeSecretRepo {
	return &fakeSecretRepo{items: map[domain.SecretKind]*domain.TenantSecret{}}
}

func (r *fakeSecretRepo) List(_ context.Context, _ uuid.UUID) ([]domain.TenantSecret, error) {
	out := make([]domain.TenantSecret, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, *item)
	}
	return out, nil
}

func (r *fakeSecretRepo) Get(_ context.Context, _ uuid.UUID, kind domain.SecretKind) (*domain.TenantSecret, error) {
	item, ok := r.items[kind]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copy := *item
	return &copy, nil
}

func (r *fakeSecretRepo) Upsert(_ context.Context, _ uuid.UUID, secret *domain.TenantSecret) error {
	copy := *secret
	r.items[secret.Kind] = &copy
	return nil
}

func (r *fakeSecretRepo) Delete(_ context.Context, _ uuid.UUID, kind domain.SecretKind) error {
	delete(r.items, kind)
	return nil
}

func TestListStatusMasksSensitiveMetadata(t *testing.T) {
	repo := newFakeSecretRepo()
	repo.items[domain.SecretKindTelegram] = &domain.TenantSecret{
		Kind:     domain.SecretKindTelegram,
		Metadata: json.RawMessage(`{"webhook_secret":"super-secret-value","handoff_notification_chat_id":"12345"}`),
	}
	svc := NewTenantSecretService(repo, nil)

	statuses, err := svc.ListStatus(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("ListStatus error: %v", err)
	}
	for _, status := range statuses {
		if status.Kind != domain.SecretKindTelegram {
			continue
		}
		meta := string(status.Metadata)
		if strings.Contains(meta, "super-secret-value") {
			t.Fatalf("ListStatus leaked plaintext webhook secret: %s", meta)
		}
		if !strings.Contains(meta, MaskedSecretValue) {
			t.Fatalf("expected masked webhook secret, got: %s", meta)
		}
		if !strings.Contains(meta, "12345") {
			t.Fatalf("expected non-sensitive metadata preserved, got: %s", meta)
		}
	}
}

func TestMergeSecretMetadataPreservesMaskedSecret(t *testing.T) {
	existing := json.RawMessage(`{"webhook_secret":"real-secret","group_id":"1"}`)
	incoming := json.RawMessage(`{"webhook_secret":"` + MaskedSecretValue + `","group_id":"2"}`)

	merged, err := mergeSecretMetadata(incoming, existing)
	if err != nil {
		t.Fatalf("mergeSecretMetadata error: %v", err)
	}
	var out map[string]string
	if err := json.Unmarshal(merged, &out); err != nil {
		t.Fatalf("unmarshal merged: %v", err)
	}
	if out["webhook_secret"] != "real-secret" {
		t.Fatalf("expected preserved secret, got %q", out["webhook_secret"])
	}
	if out["group_id"] != "2" {
		t.Fatalf("expected updated non-sensitive value, got %q", out["group_id"])
	}
}

func TestMergeSecretMetadataDropsMaskedWithoutExisting(t *testing.T) {
	incoming := json.RawMessage(`{"webhook_secret":"` + MaskedSecretValue + `","group_id":"2"}`)

	merged, err := mergeSecretMetadata(incoming, nil)
	if err != nil {
		t.Fatalf("mergeSecretMetadata error: %v", err)
	}
	var out map[string]string
	if err := json.Unmarshal(merged, &out); err != nil {
		t.Fatalf("unmarshal merged: %v", err)
	}
	if _, ok := out["webhook_secret"]; ok {
		t.Fatalf("expected masked secret without existing value to be dropped, got %v", out)
	}
	if out["group_id"] != "2" {
		t.Fatalf("expected non-sensitive value preserved, got %q", out["group_id"])
	}
}
