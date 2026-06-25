package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm/clause"
)

type BotSettingsStore struct {
	*Store
}

func NewBotSettingsStore(s *Store) *BotSettingsStore {
	return &BotSettingsStore{Store: s}
}

func (s *BotSettingsStore) Get(ctx context.Context, companyID uuid.UUID) (*domain.BotSettings, error) {
	var settings domain.BotSettings
	if err := s.db.WithContext(ctx).Where("company_id = ?", companyID).First(&settings).Error; err != nil {
		return nil, err
	}
	return &settings, nil
}

func (s *BotSettingsStore) Upsert(ctx context.Context, companyID uuid.UUID, settings *domain.BotSettings) error {
	settings.CompanyID = companyID
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "company_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"enabled",
			"provider",
			"model_tier",
			"model",
			"temperature",
			"max_tokens",
			"persona_name",
			"persona_tone",
			"persona_rules",
			"enabled_modules",
			"updated_at",
		}),
	}).Create(settings).Error
}
