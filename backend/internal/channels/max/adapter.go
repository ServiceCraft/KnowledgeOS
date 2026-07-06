package max

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	applog "github.com/knowledgeos/backend/internal/logger"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/knowledgeos/backend/internal/channels"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/service"
)

type Adapter struct {
	client *http.Client
}

func New(client *http.Client) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Adapter{client: client}
}

func (a *Adapter) Channel() domain.ChatChannel { return domain.ChatChannelMAX }

func (a *Adapter) SecretKind() domain.SecretKind { return domain.SecretKindMAX }

func (a *Adapter) RegisterWebhook(ctx context.Context, cfg channels.ChannelConfig, webhookURL string) (bool, error) {
	registrationURL := metadataStringDefault(cfg.Metadata, "webhook_registration_url", defaultSubscriptionsURL)
	secret := metadataString(cfg.Metadata, "webhook_secret", "secret")
	if secret == "" || strings.TrimSpace(cfg.Token) == "" || strings.TrimSpace(webhookURL) == "" {
		return false, nil
	}
	if err := channels.ValidateOutboundURL(registrationURL); err != nil {
		return false, statusError(http.StatusBadRequest, "max webhook registration url is not allowed")
	}
	payload := map[string]interface{}{
		"url":          webhookURL,
		"secret":       secret,
		"update_types": []string{"message_created", "bot_started"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, registrationURL, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", cfg.Token)
	resp, err := a.client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return false, statusError(http.StatusBadGateway, "max webhook registration failed")
	}
	return true, nil
}

const defaultSubscriptionsURL = "https://platform-api2.max.ru/subscriptions"

// ListSubscriptions fetches the webhook subscriptions MAX has registered for the
// bot token. It calls GET <subscriptions_url> with the bot token in the
// Authorization header, mirroring the MAX platform API.
func (a *Adapter) ListSubscriptions(ctx context.Context, cfg channels.ChannelConfig) (json.RawMessage, error) {
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, statusError(http.StatusBadRequest, "Токен MAX не задан")
	}
	endpoint := metadataStringDefault(cfg.Metadata, "subscriptions_url", defaultSubscriptionsURL)
	if err := channels.ValidateOutboundURL(endpoint); err != nil {
		return nil, statusError(http.StatusBadRequest, "max subscriptions url is not allowed")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, applog.TraceErr(ctx, "max: build subscriptions request failed", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", cfg.Token)
	resp, err := a.client.Do(req)
	if err != nil {
		_ = applog.TraceErr(ctx, "max: subscriptions request failed", err)
		return nil, statusError(http.StatusBadGateway, "max subscriptions request failed")
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, applog.TraceErr(ctx, "max: read subscriptions response failed", err)
	}
	if resp.StatusCode >= 300 {
		return nil, statusError(http.StatusBadGateway, "max subscriptions request failed")
	}
	if !json.Valid(data) {
		return nil, statusError(http.StatusBadGateway, "max subscriptions response is not valid json")
	}
	return json.RawMessage(data), nil
}

func (a *Adapter) ParseInbound(r channels.WebhookRequest, cfg channels.ChannelConfig) (*channels.InboundMessage, *channels.WebhookResponse, error) {
	secret := metadataString(cfg.Metadata, "webhook_secret", "secret")
	if secret == "" {
		return nil, nil, statusError(http.StatusUnauthorized, "max webhook secret is not configured")
	}
	got := r.Headers.Get("X-Max-Bot-Api-Secret")
	if got == "" {
		got = r.Headers.Get("X-MAX-Bot-Api-Secret")
	}
	if subtle.ConstantTimeCompare([]byte(got), []byte(secret)) != 1 {
		return nil, nil, statusError(http.StatusUnauthorized, "invalid max webhook secret")
	}
	var update struct {
		UpdateID string `json:"update_id"`
		Type     string `json:"type"`
		Message  *struct {
			ID   string `json:"id"`
			Text string `json:"text"`
			Body *struct {
				Text string `json:"text"`
			} `json:"body"`
			ChatID interface{} `json:"chat_id"`
			Chat   *struct {
				ID interface{} `json:"id"`
			} `json:"chat"`
			Recipient *struct {
				ChatID interface{} `json:"chat_id"`
			} `json:"recipient"`
		} `json:"message"`
	}
	if err := json.Unmarshal(r.Body, &update); err != nil {
		return nil, nil, statusError(http.StatusBadRequest, "invalid max update")
	}
	if update.Message == nil {
		return nil, nil, nil
	}
	text := strings.TrimSpace(update.Message.Text)
	if text == "" && update.Message.Body != nil {
		text = strings.TrimSpace(update.Message.Body.Text)
	}
	if text == "" {
		return nil, nil, nil
	}
	chatID := stringify(update.Message.ChatID)
	if chatID == "" && update.Message.Chat != nil {
		chatID = stringify(update.Message.Chat.ID)
	}
	if chatID == "" && update.Message.Recipient != nil {
		chatID = stringify(update.Message.Recipient.ChatID)
	}
	if chatID == "" {
		return nil, nil, statusError(http.StatusBadRequest, "max chat id is required")
	}
	updateID := update.UpdateID
	if updateID == "" {
		updateID = update.Message.ID
	}
	return &channels.InboundMessage{
		Channel:        domain.ChatChannelMAX,
		ExternalChatID: chatID,
		UpdateID:       updateID,
		Text:           text,
	}, nil, nil
}

func (a *Adapter) SendTyping(ctx context.Context, cfg channels.ChannelConfig, externalChatID string) error {
	return nil
}

func (a *Adapter) SendMessage(ctx context.Context, cfg channels.ChannelConfig, msg channels.OutboundMessage) error {
	endpoint := metadataString(cfg.Metadata, "send_url")
	if endpoint == "" {
		base := metadataStringDefault(cfg.Metadata, "api_base", "https://botapi.max.ru")
		endpoint = strings.TrimRight(base, "/") + "/messages"
	}
	// Guard against SSRF: the endpoint is derived from tenant-controlled metadata
	// and the bot token is attached as a query param, so it must be a public https
	// target before we build or send the request.
	if err := channels.ValidateOutboundURL(endpoint); err != nil {
		return statusError(http.StatusBadRequest, "max api endpoint is not allowed")
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return applog.TraceErr(ctx, "max: parse send url failed", err)
	}
	q := u.Query()
	if q.Get("access_token") == "" {
		q.Set("access_token", cfg.Token)
	}
	u.RawQuery = q.Encode()
	payload := map[string]interface{}{
		"chat_id": msg.ExternalChatID,
		"text":    msg.Text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return applog.TraceErr(ctx, "max: marshal request failed", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return applog.TraceErr(ctx, "max: build request failed", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return applog.TraceErr(ctx, "max: api request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return statusError(http.StatusBadGateway, "max api request failed")
	}
	return nil
}

func stringify(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case json.Number:
		return v.String()
	default:
		if v == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func metadataString(raw json.RawMessage, keys ...string) string {
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

func metadataStringDefault(raw json.RawMessage, key, fallback string) string {
	if value := metadataString(raw, key); value != "" {
		return value
	}
	return fallback
}

func statusError(status int, msg string) error {
	return &service.UserError{Status: status, Msg: msg}
}
