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

// NewTenantSecretService executes the service.NewTenantSecretService operation.
func NewTenantSecretService(secrets domain.TenantSecretRepository, cipher *secretcrypto.Cipher) *TenantSecretService {
	return &TenantSecretService{secrets: secrets, cipher: cipher}
}

type SetTenantSecretRequest struct {
	Value    string          `json:"value"`
	Metadata json.RawMessage `json:"metadata"`
}

// ListStatus executes the service.TenantSecretService.ListStatus operation.
func (s *TenantSecretService) ListStatus(ctx context.Context, companyID uuid.UUID) ([]domain.TenantSecretStatus, error) {
	applog.TraceCall(ctx, "service.TenantSecretService.ListStatus")
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

// Set executes the service.TenantSecretService.Set operation.
func (s *TenantSecretService) Set(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind, req SetTenantSecretRequest) (*domain.TenantSecretStatus, error) {
	applog.TraceCall(ctx, "service.TenantSecretService.Set")
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

// Delete executes the service.TenantSecretService.Delete operation.
func (s *TenantSecretService) Delete(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind) error {
	applog.TraceCall(ctx, "service.TenantSecretService.Delete")
	if !domain.ValidSecretKind(kind) {
		return badRequest("invalid secret kind")
	}
	if err := s.secrets.Delete(ctx, companyID, kind); err != nil {
		applog.From(ctx).Error().Err(err).
			Str("company_id", companyID.String()).
			Str("kind", string(kind)).
			Msg("tenant secret delete failed")
		return applog.TraceErr(ctx, "delete tenant secret failed", err)
	}
	applog.From(ctx).Debug().
		Str("company_id", companyID.String()).
		Str("kind", string(kind)).
		Msg("tenant secret deleted")
	return nil
}

// GetPlaintext executes the service.TenantSecretService.GetPlaintext operation.
func (s *TenantSecretService) GetPlaintext(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind) (string, error) {
	applog.TraceCall(ctx, "service.TenantSecretService.GetPlaintext")
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

// GetPlaintextWithMetadata decrypts a tenant secret and returns its public
// metadata for internal integrations that need both the token and webhook config.
func (s *TenantSecretService) GetPlaintextWithMetadata(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind) (string, json.RawMessage, error) {
	applog.TraceCall(ctx, "service.TenantSecretService.GetPlaintextWithMetadata")
	if !domain.ValidSecretKind(kind) {
		return "", nil, badRequest("invalid secret kind")
	}
	if s.cipher == nil {
		return "", nil, badRequest("SECRETS_ENCRYPTION_KEY is not configured")
	}
	secret, err := s.secrets.Get(ctx, companyID, kind)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, notFound("secret not found")
		}
		return "", nil, err
	}
	plaintext, err := s.cipher.Decrypt(secret.Ciphertext, secret.Nonce)
	if err != nil {
		applog.From(ctx).Error().Err(err).
			Str("company_id", companyID.String()).
			Str("kind", string(kind)).
			Msg("tenant secret decrypt failed")
		return "", nil, err
	}
	return string(plaintext), normalizeJSONRaw(secret.Metadata), nil
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
