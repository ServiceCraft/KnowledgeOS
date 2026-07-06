package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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

// EditableTenantSecret is returned to the admin UI when opening the secret editor.
// Unlike TenantSecretStatus, it includes the decrypted token and unmasked metadata.
type EditableTenantSecret struct {
	Kind     domain.SecretKind `json:"kind"`
	IsSet    bool              `json:"is_set"`
	Value    string            `json:"value"`
	Metadata json.RawMessage   `json:"metadata"`
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
			status.Metadata = publicSecretMetadata(item.Metadata)
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

// GetEditable returns the decrypted secret and full metadata for the admin editor.
func (s *TenantSecretService) GetEditable(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind) (*EditableTenantSecret, error) {
	applog.TraceCall(ctx, "service.TenantSecretService.GetEditable")
	if !domain.ValidSecretKind(kind) {
		return nil, badRequest("invalid secret kind")
	}
	value, metadata, err := s.GetPlaintextWithMetadata(ctx, companyID, kind)
	if err != nil {
		if HTTPStatus(err) == http.StatusNotFound {
			return &EditableTenantSecret{
				Kind:     kind,
				IsSet:    false,
				Value:    "",
				Metadata: json.RawMessage(`{}`),
			}, nil
		}
		return nil, err
	}
	return &EditableTenantSecret{
		Kind:     kind,
		IsSet:    true,
		Value:    value,
		Metadata: normalizeJSONRaw(metadata),
	}, nil
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
	// Preserve existing sensitive metadata when the client sends the masked
	// sentinel or omits a sensitive key, so unrelated edits do not wipe secrets.
	var existing *domain.TenantSecret
	if loaded, getErr := s.secrets.Get(ctx, companyID, kind); getErr == nil {
		existing = loaded
	} else if !errors.Is(getErr, gorm.ErrRecordNotFound) {
		return nil, getErr
	}
	var existingMetadata json.RawMessage
	if existing != nil {
		existingMetadata = existing.Metadata
	}
	metadata, err = mergeSecretMetadata(metadata, existingMetadata)
	if err != nil {
		return nil, badRequest("metadata must be a JSON object")
	}
	var ciphertext, nonce []byte
	if value == MaskedSecretValue {
		if existing == nil {
			return nil, badRequest("secret value is required")
		}
		ciphertext = existing.Ciphertext
		nonce = existing.Nonce
	} else {
		ciphertext, nonce, err = s.cipher.Encrypt([]byte(value))
		if err != nil {
			return nil, err
		}
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
		Metadata:  publicSecretMetadata(secret.Metadata),
		UpdatedAt: &updatedAt,
	}
}

func normalizeJSONRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

// MaskedSecretValue is returned for sensitive metadata keys so the UI can show
// that a value is configured without exposing the real secret. Clients must not
// persist this sentinel back as a real value.
const MaskedSecretValue = "********"

// publicSecretMetadata returns metadata safe to expose to API clients: sensitive
// keys (webhook secrets, callback secrets, confirmation tokens, etc.) are
// replaced with MaskedSecretValue while non-sensitive config (URLs, chat IDs,
// group IDs) is preserved. Internal callers that need the real values must use
// GetPlaintextWithMetadata instead.
func publicSecretMetadata(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		// Not a JSON object we can inspect; drop it rather than risk leaking.
		return json.RawMessage(`{}`)
	}
	for key := range fields {
		if isSensitiveMetadataKey(key) {
			masked, _ := json.Marshal(MaskedSecretValue)
			fields[key] = masked
		}
	}
	out, err := json.Marshal(fields)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return out
}

// mergeSecretMetadata combines client-supplied metadata with the currently
// stored metadata. Non-sensitive keys are taken from the incoming payload (so
// they can be updated or cleared). Sensitive keys retain their existing stored
// value when the client sends the masked sentinel or omits them, and are dropped
// entirely if masked with no prior value to restore.
func mergeSecretMetadata(incoming, existing json.RawMessage) (json.RawMessage, error) {
	inMap := map[string]json.RawMessage{}
	if len(incoming) > 0 {
		if err := json.Unmarshal(incoming, &inMap); err != nil {
			return nil, err
		}
	}
	exMap := map[string]json.RawMessage{}
	if len(existing) > 0 {
		_ = json.Unmarshal(existing, &exMap)
	}
	for key, exVal := range exMap {
		if !isSensitiveMetadataKey(key) {
			continue
		}
		if inVal, ok := inMap[key]; !ok || isMaskedRawValue(inVal) {
			inMap[key] = exVal
		}
	}
	for key, inVal := range inMap {
		if isSensitiveMetadataKey(key) && isMaskedRawValue(inVal) {
			delete(inMap, key)
		}
	}
	out, err := json.Marshal(inMap)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func isMaskedRawValue(raw json.RawMessage) bool {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return false
	}
	return s == MaskedSecretValue
}

func isSensitiveMetadataKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	for _, marker := range []string{"secret", "password", "token", "api_key", "access_key", "private"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
