package store

import (
	"context"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm/clause"
)

type BotSettingsStore struct {
	*Store
}

// NewBotSettingsStore executes the store.NewBotSettingsStore operation.
func NewBotSettingsStore(s *Store) *BotSettingsStore {
	return &BotSettingsStore{Store: s}
}

// Get executes the store.BotSettingsStore.Get operation.
func (s *BotSettingsStore) Get(ctx context.Context, companyID uuid.UUID) (*domain.BotSettings, error) {
	applog.TraceCall(ctx, "store.BotSettingsStore.Get")
	var settings domain.BotSettings
	if err := s.db.WithContext(ctx).Where("company_id = ?", companyID).First(&settings).Error; err != nil {
		return nil, err
	}
	return &settings, nil
}

// Upsert executes the store.BotSettingsStore.Upsert operation.
func (s *BotSettingsStore) Upsert(ctx context.Context, companyID uuid.UUID, settings *domain.BotSettings) error {
	applog.TraceCall(ctx, "store.BotSettingsStore.Upsert")
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
			"min_retrieval_score",
			"min_confidence",
			"allowed_theme_ids",
			"escalate_on_low_confidence",
			"require_citations",
			"updated_at",
		}),
	}).Create(settings).Error
}
