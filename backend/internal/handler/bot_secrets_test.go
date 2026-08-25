package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	secretcrypto "github.com/knowledgeos/backend/internal/crypto"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
	"gorm.io/gorm"
)

func TestBotHandlerGetSecretForEditReturnsPlaintext(t *testing.T) {
	companyID := uuid.New()
	repo := &botHandlerSecretRepo{items: map[domain.SecretKind]*domain.TenantSecret{}}
	cipher, err := secretcrypto.NewCipher("test-secret-key")
	if err != nil {
		t.Fatalf("NewCipher() error = %v", err)
	}
	secrets := service.NewTenantSecretService(repo, cipher)
	if _, err := secrets.Set(context.Background(), companyID, domain.SecretKindTelegram, service.SetTenantSecretRequest{
		Value:    "real-bot-token",
		Metadata: json.RawMessage(`{"webhook_secret":"hook-secret-value"}`),
	}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	h := NewBotHandler(nil, secrets, nil)
	router := chi.NewRouter()
	router.Get("/admin/bot/secrets/{kind}/edit", func(w http.ResponseWriter, r *http.Request) {
		ctx := middleware.SetCompanyID(r.Context(), companyID)
		h.GetSecretForEdit(w, r.WithContext(ctx))
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/bot/secrets/telegram/edit", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var envelope struct {
		Data service.EditableTenantSecret `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if envelope.Data.Value != "real-bot-token" {
		t.Fatalf("expected plaintext token, got %q", envelope.Data.Value)
	}
	var meta map[string]string
	if err := json.Unmarshal(envelope.Data.Metadata, &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if meta["webhook_secret"] != "hook-secret-value" {
		t.Fatalf("expected plaintext webhook secret, got %q", meta["webhook_secret"])
	}
	if meta["webhook_secret"] == service.MaskedSecretValue {
		t.Fatal("edit endpoint must not return masked metadata")
	}
}

type botHandlerSecretRepo struct {
	items map[domain.SecretKind]*domain.TenantSecret
}

func (r *botHandlerSecretRepo) List(_ context.Context, _ uuid.UUID) ([]domain.TenantSecret, error) {
	out := make([]domain.TenantSecret, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, *item)
	}
	return out, nil
}

func (r *botHandlerSecretRepo) Get(_ context.Context, _ uuid.UUID, kind domain.SecretKind) (*domain.TenantSecret, error) {
	item, ok := r.items[kind]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copy := *item
	return &copy, nil
}

func (r *botHandlerSecretRepo) Upsert(_ context.Context, _ uuid.UUID, secret *domain.TenantSecret) error {
	copy := *secret
	r.items[secret.Kind] = &copy
	return nil
}

func (r *botHandlerSecretRepo) Delete(_ context.Context, _ uuid.UUID, kind domain.SecretKind) error {
	delete(r.items, kind)
	return nil
}
