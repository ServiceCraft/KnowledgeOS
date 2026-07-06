package channels

import (
	"encoding/json"
	"strings"

	"github.com/knowledgeos/backend/internal/domain"
)

// ResolveWebhookBaseURL picks the public HTTPS origin used in webhook URLs.
// Configured PUBLIC_WEBHOOK_BASE_URL wins; otherwise the incoming admin request
// must already be served over HTTPS (including X-Forwarded-Proto).
func ResolveWebhookBaseURL(requestBase, configured string) string {
	configured = strings.TrimRight(strings.TrimSpace(configured), "/")
	if configured != "" && strings.HasPrefix(configured, "https://") {
		return configured
	}
	requestBase = strings.TrimRight(strings.TrimSpace(requestBase), "/")
	if strings.HasPrefix(requestBase, "https://") {
		return requestBase
	}
	return ""
}

// MetadataString returns the first non-empty string metadata value for the keys.
func MetadataString(raw json.RawMessage, keys ...string) string {
	var data map[string]interface{}
	if len(raw) == 0 || json.Unmarshal(raw, &data) != nil {
		return ""
	}
	for _, key := range keys {
		if value, ok := data[key].(string); ok {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

// MissingWebhookRegistrationFields returns a user-facing reason when automatic
// registration cannot run yet. Empty string means prerequisites are satisfied.
func MissingWebhookRegistrationFields(channel domain.ChatChannel, cfg ChannelConfig, webhookURL string) string {
	if strings.TrimSpace(cfg.Token) == "" {
		return "Не задан токен бота"
	}
	if strings.TrimSpace(webhookURL) == "" {
		return "Не задан публичный HTTPS URL webhook (PUBLIC_WEBHOOK_BASE_URL или доступ к admin по HTTPS)"
	}
	switch channel {
	case domain.ChatChannelTelegram:
		if MetadataString(cfg.Metadata, "webhook_secret", "secret_token") == "" {
			return "Не задан webhook secret"
		}
	case domain.ChatChannelMAX:
		if MetadataString(cfg.Metadata, "webhook_secret", "secret") == "" {
			return "Не задан webhook secret"
		}
	case domain.ChatChannelVK:
		if MetadataString(cfg.Metadata, "secret", "webhook_secret") == "" {
			return "Не задан callback secret"
		}
		if MetadataString(cfg.Metadata, "confirmation_token") == "" {
			return "Не задан confirmation token"
		}
		if MetadataString(cfg.Metadata, "webhook_registration_url") == "" {
			return "Не задан webhook registration URL"
		}
	}
	return ""
}
