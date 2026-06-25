package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type BotProvider string

const (
	BotProviderYandex BotProvider = "yandex"
)

type BotModelTier string

const (
	BotModelTierLite BotModelTier = "lite"
	BotModelTierPro  BotModelTier = "pro"
)

type SecretKind string

const (
	SecretKindLLM      SecretKind = "llm"
	SecretKindTelegram SecretKind = "telegram"
	SecretKindMAX      SecretKind = "max"
	SecretKindVK       SecretKind = "vk"
	SecretKindBitrix24 SecretKind = "bitrix24"
)

type BotSettings struct {
	CompanyID      uuid.UUID       `gorm:"type:uuid;primaryKey" json:"company_id"`
	Enabled        bool            `gorm:"not null;default:false" json:"enabled"`
	Provider       BotProvider     `gorm:"type:text;not null;default:yandex" json:"provider"`
	ModelTier      BotModelTier    `gorm:"type:text;not null;default:lite" json:"model_tier"`
	Model          *string         `gorm:"type:text" json:"model,omitempty"`
	Temperature    float64         `gorm:"not null;default:0.2" json:"temperature"`
	MaxTokens      int             `gorm:"not null;default:1024" json:"max_tokens"`
	PersonaName    string          `gorm:"type:text;not null;default:Администратор" json:"persona_name"`
	PersonaTone    string          `gorm:"type:text;not null;default:friendly_professional" json:"persona_tone"`
	PersonaRules   string          `gorm:"type:text;not null;default:''" json:"persona_rules"`
	EnabledModules json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"enabled_modules"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (BotSettings) TableName() string { return "bot_settings" }

type TenantSecret struct {
	ID         uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CompanyID  uuid.UUID       `gorm:"type:uuid;not null;index" json:"company_id"`
	Kind       SecretKind      `gorm:"type:text;not null" json:"kind"`
	Ciphertext []byte          `gorm:"type:bytea;not null" json:"-"`
	Nonce      []byte          `gorm:"type:bytea;not null" json:"-"`
	Metadata   json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func (TenantSecret) TableName() string { return "tenant_secrets" }

type TenantSecretStatus struct {
	Kind      SecretKind      `json:"kind"`
	IsSet     bool            `json:"is_set"`
	Metadata  json.RawMessage `json:"metadata"`
	UpdatedAt *time.Time      `json:"updated_at,omitempty"`
}

func ValidBotProvider(provider BotProvider) bool {
	return provider == BotProviderYandex
}

func ValidBotModelTier(tier BotModelTier) bool {
	return tier == BotModelTierLite || tier == BotModelTierPro
}

func ValidSecretKind(kind SecretKind) bool {
	switch kind {
	case SecretKindLLM, SecretKindTelegram, SecretKindMAX, SecretKindVK, SecretKindBitrix24:
		return true
	default:
		return false
	}
}

func SupportedSecretKinds() []SecretKind {
	return []SecretKind{SecretKindLLM, SecretKindTelegram, SecretKindMAX, SecretKindVK, SecretKindBitrix24}
}

type BotSettingsRepository interface {
	Get(ctx context.Context, companyID uuid.UUID) (*BotSettings, error)
	Upsert(ctx context.Context, companyID uuid.UUID, settings *BotSettings) error
}

type TenantSecretRepository interface {
	List(ctx context.Context, companyID uuid.UUID) ([]TenantSecret, error)
	Get(ctx context.Context, companyID uuid.UUID, kind SecretKind) (*TenantSecret, error)
	Upsert(ctx context.Context, companyID uuid.UUID, secret *TenantSecret) error
	Delete(ctx context.Context, companyID uuid.UUID, kind SecretKind) error
}
