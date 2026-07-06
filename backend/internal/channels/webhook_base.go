package channels

import (
	"encoding/json"
	"net"
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

// WebhookDisplayBase picks the origin shown in admin UI for webhook URLs.
// PUBLIC_WEBHOOK_BASE_URL and HTTPS requests win; plain HTTP on a public host
// is upgraded to HTTPS so staging behind a TLS proxy shows the webhook URL
// providers expect.
func WebhookDisplayBase(requestBase, configured string) string {
	if resolved := ResolveWebhookBaseURL(requestBase, configured); resolved != "" {
		return resolved
	}
	requestBase = strings.TrimRight(strings.TrimSpace(requestBase), "/")
	if requestBase == "" {
		return ""
	}
	if strings.HasPrefix(requestBase, "http://") && shouldUpgradeHTTPDisplayToHTTPS(requestBase) {
		return "https://" + strings.TrimPrefix(requestBase, "http://")
	}
	return requestBase
}

func shouldUpgradeHTTPDisplayToHTTPS(baseURL string) bool {
	hostPort := webhookHostPortFromBaseURL(baseURL)
	if hostPort == "" || webhookHostIsLocal(hostPort) {
		return false
	}
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		// No explicit port — treat as public HTTPS endpoint.
		return !webhookHostIsLocal(hostPort)
	}
	// Custom ports (e.g. :8081) may not serve TLS; require PUBLIC_WEBHOOK_BASE_URL.
	return port == "80" && !webhookHostIsLocal(host)
}

func webhookHostPortFromBaseURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if i := strings.Index(baseURL, "://"); i >= 0 {
		baseURL = baseURL[i+3:]
	}
	if baseURL == "" {
		return ""
	}
	if j := strings.Index(baseURL, "/"); j >= 0 {
		baseURL = baseURL[:j]
	}
	return baseURL
}

func webhookHostIsLocal(hostPort string) bool {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		host = hostPort
	}
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
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

// ChannelRequiredFieldsComplete reports whether all required channel metadata fields
// are present. The encrypted bot token alone is not enough for "configured".
func ChannelRequiredFieldsComplete(channel domain.ChatChannel, metadata json.RawMessage) bool {
	switch channel {
	case domain.ChatChannelTelegram:
		return MetadataString(metadata, "webhook_secret", "secret_token") != ""
	case domain.ChatChannelMAX:
		return MetadataString(metadata, "webhook_secret", "secret") != ""
	case domain.ChatChannelVK:
		return MetadataString(metadata, "secret", "webhook_secret") != "" &&
			MetadataString(metadata, "confirmation_token") != ""
	default:
		return true
	}
}
