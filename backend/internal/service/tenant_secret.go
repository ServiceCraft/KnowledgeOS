package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	secretcrypto "github.com/knowledgeos/backend/internal/crypto"
	"github.com/knowledgeos/backend/internal/domain"
	applog "github.com/knowledgeos/backend/internal/logger"
	"gorm.io/gorm"
)

type TenantSecretService struct {
	secrets domain.TenantSecretRepository
	cipher  *secretcrypto.Cipher
}

func NewTenantSecretService(secrets domain.TenantSecretRepository, cipher *secretcrypto.Cipher) *TenantSecretService {
	return &TenantSecretService{secrets: secrets, cipher: cipher}
}

type SetTenantSecretRequest struct {
	Value    string          `json:"value"`
	Metadata json.RawMessage `json:"metadata"`
}

func (s *TenantSecretService) ListStatus(ctx context.Context, companyID uuid.UUID) ([]domain.TenantSecretStatus, error) {
	items, err := s.secrets.List(ctx, companyID)
	if err != nil {
		applog.From(ctx).Error().Err(err).Str("company_id", companyID.String()).Msg("tenant secret status list failed")
		return nil, err
	}

	byKind := map[domain.SecretKind]domain.TenantSecret{}
	for _, item := range items {
		byKind[item.Kind] = item
	}

	statuses := make([]domain.TenantSecretStatus, 0, len(domain.SupportedSecretKinds()))
	for _, kind := range domain.SupportedSecretKinds() {
		status := domain.TenantSecretStatus{
			Kind:     kind,
			IsSet:    false,
			Metadata: json.RawMessage(`{}`),
		}
		if item, ok := byKind[kind]; ok {
			updatedAt := item.UpdatedAt
			status.IsSet = true
			status.Metadata = normalizeJSONRaw(item.Metadata)
			status.UpdatedAt = &updatedAt
		}
		statuses = append(statuses, status)
	}
	applog.From(ctx).Debug().
		Str("company_id", companyID.String()).
		Int("total", len(statuses)).
		Msg("tenant secret statuses listed")
	return statuses, nil
}

func (s *TenantSecretService) Set(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind, req SetTenantSecretRequest) (*domain.TenantSecretStatus, error) {
	if !domain.ValidSecretKind(kind) {
		return nil, badRequest("invalid secret kind")
	}
	value := strings.TrimSpace(req.Value)
	if value == "" {
		return nil, badRequest("secret value is required")
	}
	if s.cipher == nil {
		return nil, badRequest("SECRETS_ENCRYPTION_KEY is not configured")
	}
	metadata, err := normalizeJSONObject(req.Metadata)
	if err != nil {
		return nil, badRequest("metadata must be a JSON object")
	}
	ciphertext, nonce, err := s.cipher.Encrypt([]byte(value))
	if err != nil {
		return nil, err
	}
	secret := &domain.TenantSecret{
		Kind:       kind,
		Ciphertext: ciphertext,
		Nonce:      nonce,
		Metadata:   metadata,
	}
	if err := s.secrets.Upsert(ctx, companyID, secret); err != nil {
		applog.From(ctx).Error().Err(err).
			Str("company_id", companyID.String()).
			Str("kind", string(kind)).
			Msg("tenant secret upsert failed")
		return nil, err
	}
	saved, err := s.secrets.Get(ctx, companyID, kind)
	if err != nil {
		applog.From(ctx).Error().Err(err).
			Str("company_id", companyID.String()).
			Str("kind", string(kind)).
			Msg("tenant secret reload failed")
		return nil, err
	}
	applog.From(ctx).Debug().
		Str("company_id", companyID.String()).
		Str("kind", string(kind)).
		Msg("tenant secret saved")
	return secretStatus(saved), nil
}

func (s *TenantSecretService) Delete(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind) error {
	if !domain.ValidSecretKind(kind) {
		return badRequest("invalid secret kind")
	}
	if err := s.secrets.Delete(ctx, companyID, kind); err != nil {
		applog.From(ctx).Error().Err(err).
			Str("company_id", companyID.String()).
			Str("kind", string(kind)).
			Msg("tenant secret delete failed")
		return err
	}
	applog.From(ctx).Debug().
		Str("company_id", companyID.String()).
		Str("kind", string(kind)).
		Msg("tenant secret deleted")
	return nil
}

func (s *TenantSecretService) GetPlaintext(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind) (string, error) {
	if !domain.ValidSecretKind(kind) {
		return "", badRequest("invalid secret kind")
	}
	if s.cipher == nil {
		return "", badRequest("SECRETS_ENCRYPTION_KEY is not configured")
	}
	secret, err := s.secrets.Get(ctx, companyID, kind)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", notFound("secret not found")
		}
		return "", err
	}
	plaintext, err := s.cipher.Decrypt(secret.Ciphertext, secret.Nonce)
	if err != nil {
		applog.From(ctx).Error().Err(err).
			Str("company_id", companyID.String()).
			Str("kind", string(kind)).
			Msg("tenant secret decrypt failed")
		return "", err
	}
	applog.From(ctx).Debug().
		Str("company_id", companyID.String()).
		Str("kind", string(kind)).
		Msg("tenant secret decrypted for internal use")
	return string(plaintext), nil
}

func secretStatus(secret *domain.TenantSecret) *domain.TenantSecretStatus {
	updatedAt := secret.UpdatedAt
	return &domain.TenantSecretStatus{
		Kind:      secret.Kind,
		IsSet:     true,
		Metadata:  normalizeJSONRaw(secret.Metadata),
		UpdatedAt: &updatedAt,
	}
}

func normalizeJSONRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}
