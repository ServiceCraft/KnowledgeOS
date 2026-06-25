package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/config"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/llm"
	"github.com/knowledgeos/backend/internal/llm/yandex"
	applog "github.com/knowledgeos/backend/internal/logger"
)

type LLMFactory struct {
	cfg      *config.Config
	settings *BotSettingsService
	secrets  *TenantSecretService
	client   *http.Client
}

func NewLLMFactory(cfg *config.Config, settings *BotSettingsService, secrets *TenantSecretService) *LLMFactory {
	timeout := time.Duration(cfg.YandexTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &LLMFactory{
		cfg:      cfg,
		settings: settings,
		secrets:  secrets,
		client:   &http.Client{Timeout: timeout},
	}
}

func (f *LLMFactory) ForCompany(ctx context.Context, companyID uuid.UUID) (llm.Provider, llm.Embedder, error) {
	settings, err := f.settings.Get(ctx, companyID)
	if err != nil {
		applog.From(ctx).Error().Err(err).Str("company_id", companyID.String()).Msg("llm settings load failed")
		return nil, nil, err
	}
	token, authType, err := f.credentials(ctx, companyID)
	if err != nil {
		applog.From(ctx).Debug().Err(err).Str("company_id", companyID.String()).Msg("llm credentials load failed")
		return nil, nil, err
	}
	model := f.chatModel(settings)
	docModel := f.embeddingModel(f.cfg.YandexEmbeddingDocModel, "text-embeddings-v2-doc")
	queryModel := f.embeddingModel(f.cfg.YandexEmbeddingQueryModel, "text-embeddings-v2-query")
	applog.From(ctx).Debug().
		Str("company_id", companyID.String()).
		Str("provider", string(settings.Provider)).
		Str("model_tier", string(settings.ModelTier)).
		Str("auth_type", authType).
		Str("chat_model", model).
		Str("embedding_doc_model", docModel).
		Str("embedding_query_model", queryModel).
		Msg("llm client configured")
	client := yandex.NewClient(f.client, yandex.Config{
		Endpoint:            f.cfg.YandexEndpoint,
		FolderID:            f.cfg.YandexFolderID,
		AuthToken:           token,
		AuthType:            authType,
		DefaultChatModel:    model,
		EmbeddingDocModel:   docModel,
		EmbeddingQueryModel: queryModel,
		Timeout:             time.Duration(f.cfg.YandexTimeoutSeconds) * time.Second,
		MaxRetries:          f.cfg.YandexMaxRetries,
	})
	return client, client, nil
}

func (f *LLMFactory) credentials(ctx context.Context, companyID uuid.UUID) (string, string, error) {
	statuses, err := f.secrets.ListStatus(ctx, companyID)
	if err == nil {
		for _, status := range statuses {
			if status.Kind != domain.SecretKindLLM || !status.IsSet {
				continue
			}
			token, err := f.secrets.GetPlaintext(ctx, companyID, domain.SecretKindLLM)
			return token, authTypeFromMetadata(status.Metadata), err
		}
	}
	if strings.TrimSpace(f.cfg.YandexAPIKey) != "" {
		return strings.TrimSpace(f.cfg.YandexAPIKey), "api_key", nil
	}
	if err != nil {
		return "", "", err
	}
	token, err := f.secrets.GetPlaintext(ctx, companyID, domain.SecretKindLLM)
	return token, "iam", err
}

func (f *LLMFactory) chatModel(settings *domain.BotSettings) string {
	if settings.Model != nil && strings.TrimSpace(*settings.Model) != "" {
		return strings.TrimSpace(*settings.Model)
	}
	if settings.ModelTier == domain.BotModelTierPro && f.cfg.YandexDefaultChatModelPro != "" {
		return f.cfg.YandexDefaultChatModelPro
	}
	if settings.ModelTier == domain.BotModelTierLite && f.cfg.YandexDefaultChatModelLite != "" {
		return f.cfg.YandexDefaultChatModelLite
	}
	if f.cfg.YandexFolderID == "" {
		return ""
	}
	if settings.ModelTier == domain.BotModelTierPro {
		return "gpt://" + f.cfg.YandexFolderID + "/yandexgpt/latest"
	}
	return "gpt://" + f.cfg.YandexFolderID + "/yandexgpt-lite/latest"
}

func (f *LLMFactory) embeddingModel(model, fallbackName string) string {
	model = strings.TrimSpace(model)
	if model != "" {
		if strings.Contains(model, "://") || f.cfg.YandexFolderID == "" {
			return model
		}
		return "emb://" + f.cfg.YandexFolderID + "/" + model + "/latest"
	}
	if f.cfg.YandexFolderID == "" {
		return ""
	}
	return "emb://" + f.cfg.YandexFolderID + "/" + fallbackName + "/latest"
}

func authTypeFromMetadata(metadata json.RawMessage) string {
	var meta struct {
		AuthType string `json:"auth_type"`
	}
	if err := json.Unmarshal(metadata, &meta); err != nil {
		return "iam"
	}
	if meta.AuthType != "" {
		return meta.AuthType
	}
	return "iam"
}
