package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	applog "github.com/knowledgeos/backend/internal/logger"
	"gorm.io/gorm"
)

type BotSettingsService struct {
	settings domain.BotSettingsRepository
}

// NewBotSettingsService executes the service.NewBotSettingsService operation.
func NewBotSettingsService(settings domain.BotSettingsRepository) *BotSettingsService {
	return &BotSettingsService{settings: settings}
}

type UpdateBotSettingsRequest struct {
	Enabled        *bool                `json:"enabled"`
	Provider       *domain.BotProvider  `json:"provider"`
	ModelTier      *domain.BotModelTier `json:"model_tier"`
	Model          *string              `json:"model"`
	Temperature    *float64             `json:"temperature"`
	MaxTokens      *int                 `json:"max_tokens"`
	PersonaName    *string              `json:"persona_name"`
	PersonaTone    *string              `json:"persona_tone"`
	PersonaRules   *string              `json:"persona_rules"`
	EnabledModules json.RawMessage      `json:"enabled_modules"`

	MinRetrievalScore       *float64        `json:"min_retrieval_score"`
	MinConfidence           *float64        `json:"min_confidence"`
	AllowedThemeIDs         json.RawMessage `json:"allowed_theme_ids"`
	EscalateOnLowConfidence *bool           `json:"escalate_on_low_confidence"`
	RequireCitations        *bool           `json:"require_citations"`
}

// Get executes the service.BotSettingsService.Get operation.
func (s *BotSettingsService) Get(ctx context.Context, companyID uuid.UUID) (*domain.BotSettings, error) {
	applog.TraceCall(ctx, "service.BotSettingsService.Get")
	settings, err := s.settings.Get(ctx, companyID)
	if err == nil {
		normalizeBotSettings(settings)
		applog.From(ctx).Debug().
			Str("company_id", companyID.String()).
			Bool("enabled", settings.Enabled).
			Str("provider", string(settings.Provider)).
			Str("model_tier", string(settings.ModelTier)).
			Msg("bot settings loaded")
		return settings, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		applog.From(ctx).Error().Err(err).Str("company_id", companyID.String()).Msg("bot settings load failed")
		return nil, err
	}
	settings = DefaultBotSettings(companyID)
	if err := s.settings.Upsert(ctx, companyID, settings); err != nil {
		applog.From(ctx).Error().Err(err).Str("company_id", companyID.String()).Msg("bot default settings create failed")
		return nil, err
	}
	applog.From(ctx).Debug().Str("company_id", companyID.String()).Msg("bot default settings created")
	return settings, nil
}

// Update executes the service.BotSettingsService.Update operation.
func (s *BotSettingsService) Update(ctx context.Context, companyID uuid.UUID, req UpdateBotSettingsRequest) (*domain.BotSettings, error) {
	applog.TraceCall(ctx, "service.BotSettingsService.Update")
	current, err := s.Get(ctx, companyID)
	if err != nil {
		return nil, err
	}

	if req.Enabled != nil {
		current.Enabled = *req.Enabled
	}
	if req.Provider != nil {
		if !domain.ValidBotProvider(*req.Provider) {
			return nil, badRequest("invalid provider")
		}
		current.Provider = *req.Provider
	}
	if req.ModelTier != nil {
		if !domain.ValidBotModelTier(*req.ModelTier) {
			return nil, badRequest("invalid model_tier")
		}
		current.ModelTier = *req.ModelTier
	}
	if req.Model != nil {
		model := strings.TrimSpace(*req.Model)
		if model == "" {
			current.Model = nil
		} else {
			current.Model = &model
		}
	}
	if req.Temperature != nil {
		if *req.Temperature < 0 || *req.Temperature > 2 {
			return nil, badRequest("temperature must be between 0 and 2")
		}
		current.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		if *req.MaxTokens <= 0 || *req.MaxTokens > 8192 {
			return nil, badRequest("max_tokens must be between 1 and 8192")
		}
		current.MaxTokens = *req.MaxTokens
	}
	if req.PersonaName != nil {
		name := strings.TrimSpace(*req.PersonaName)
		if name == "" {
			return nil, badRequest("persona_name is required")
		}
		current.PersonaName = name
	}
	if req.PersonaTone != nil {
		tone := strings.TrimSpace(*req.PersonaTone)
		if tone == "" {
			return nil, badRequest("persona_tone is required")
		}
		current.PersonaTone = tone
	}
	if req.PersonaRules != nil {
		current.PersonaRules = strings.TrimSpace(*req.PersonaRules)
	}
	if len(req.EnabledModules) > 0 {
		modules, err := normalizeJSONObject(req.EnabledModules)
		if err != nil {
			return nil, badRequest("enabled_modules must be a JSON object")
		}
		current.EnabledModules = modules
	}
	if req.MinRetrievalScore != nil {
		if *req.MinRetrievalScore < 0 {
			return nil, badRequest("min_retrieval_score must be >= 0")
		}
		current.MinRetrievalScore = *req.MinRetrievalScore
	}
	if req.MinConfidence != nil {
		if *req.MinConfidence < 0 || *req.MinConfidence > 1 {
			return nil, badRequest("min_confidence must be between 0 and 1")
		}
		current.MinConfidence = *req.MinConfidence
	}
	if len(req.AllowedThemeIDs) > 0 {
		themes, err := normalizeThemeIDList(req.AllowedThemeIDs)
		if err != nil {
			return nil, badRequest("allowed_theme_ids must be an array of UUIDs")
		}
		current.AllowedThemeIDs = themes
	}
	if req.EscalateOnLowConfidence != nil {
		current.EscalateOnLowConfidence = *req.EscalateOnLowConfidence
	}
	if req.RequireCitations != nil {
		current.RequireCitations = *req.RequireCitations
	}

	if err := s.settings.Upsert(ctx, companyID, current); err != nil {
		applog.From(ctx).Error().Err(err).Str("company_id", companyID.String()).Msg("bot settings update failed")
		return nil, err
	}
	normalizeBotSettings(current)
	applog.From(ctx).Debug().
		Str("company_id", companyID.String()).
		Bool("enabled", current.Enabled).
		Str("provider", string(current.Provider)).
		Str("model_tier", string(current.ModelTier)).
		Float64("temperature", current.Temperature).
		Int("max_tokens", current.MaxTokens).
		Msg("bot settings updated")
	return current, nil
}

// DefaultBotSettings executes the service.DefaultBotSettings operation.
func DefaultBotSettings(companyID uuid.UUID) *domain.BotSettings {
	return &domain.BotSettings{
		CompanyID:      companyID,
		Enabled:        false,
		Provider:       domain.BotProviderYandex,
		ModelTier:      domain.BotModelTierLite,
		Temperature:    0.2,
		MaxTokens:      1024,
		PersonaName:    "Администратор",
		PersonaTone:    "friendly_professional",
		PersonaRules:   "",
		EnabledModules: json.RawMessage(`{}`),

		MinRetrievalScore:       0,
		MinConfidence:           0,
		AllowedThemeIDs:         json.RawMessage(`[]`),
		EscalateOnLowConfidence: false,
		RequireCitations:        false,
	}
}

func normalizeBotSettings(settings *domain.BotSettings) {
	if len(settings.EnabledModules) == 0 {
		settings.EnabledModules = json.RawMessage(`{}`)
	}
	if len(settings.AllowedThemeIDs) == 0 {
		settings.AllowedThemeIDs = json.RawMessage(`[]`)
	}
	if settings.Provider == "" {
		settings.Provider = domain.BotProviderYandex
	}
	if settings.ModelTier == "" {
		settings.ModelTier = domain.BotModelTierLite
	}
	if settings.MaxTokens == 0 {
		settings.MaxTokens = 1024
	}
	if settings.PersonaName == "" {
		settings.PersonaName = "Администратор"
	}
	if settings.PersonaTone == "" {
		settings.PersonaTone = "friendly_professional"
	}
}

func normalizeThemeIDList(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage(`[]`), nil
	}
	var ids []uuid.UUID
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, err
	}
	out, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}

func normalizeJSONObject(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage(`{}`), nil
	}
	var value map[string]interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	out, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}
