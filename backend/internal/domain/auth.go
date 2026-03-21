package domain

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash string     `gorm:"type:text;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

type APIKey struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CompanyID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"company_id"`
	KeyHash    string     `gorm:"type:text;not null" json:"-"`
	Name       string     `gorm:"type:text;not null" json:"name"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

func (APIKey) TableName() string { return "api_keys" }
