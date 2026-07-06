package telegram

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	applog "github.com/knowledgeos/backend/internal/logger"
	"io"
	"net/http"
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

func (a *Adapter) Channel() domain.ChatChannel { return domain.ChatChannelTelegram }

func (a *Adapter) SecretKind() domain.SecretKind { return domain.SecretKindTelegram }

func (a *Adapter) RegisterWebhook(ctx context.Context, cfg channels.ChannelConfig, webhookURL string) (bool, error) {
	secret := metadataString(cfg.Metadata, "webhook_secret", "secret_token")
	if secret == "" || strings.TrimSpace(cfg.Token) == "" || strings.TrimSpace(webhookURL) == "" {
		return false, nil
	}
	return true, a.registerWebhook(ctx, cfg.Token, webhookURL, secret)
}

func (a *Adapter) registerWebhook(ctx context.Context, token, webhookURL, secret string) error {
	return a.call(ctx, token, "setWebhook", map[string]interface{}{
		"url":                  webhookURL,
		"secret_token":         secret,
		"allowed_updates":      []string{"message"},
		"drop_pending_updates": false,
	})
}

func (a *Adapter) CheckWebhook(ctx context.Context, cfg channels.ChannelConfig, webhookURL string) (channels.WebhookStatus, error) {
	expectedURL := strings.TrimRight(strings.TrimSpace(webhookURL), "/")
	if strings.TrimSpace(cfg.Token) == "" {
		return channels.WebhookStatus{Configured: false, Error: "Токен Telegram не задан"}, nil
	}
	if expectedURL == "" {
		return channels.WebhookStatus{Configured: false, Error: "Адрес webhook на сервере не задан"}, nil
	}
	data, err := a.ListSubscriptions(ctx, cfg)
	if err != nil {
		return channels.WebhookStatus{Configured: false, Error: "Не удалось проверить webhook в Telegram"}, err
	}
	registeredURLs := channels.ExtractSubscriptionURLs(data)
	if len(registeredURLs) == 0 {
		return channels.WebhookStatus{Configured: false, Error: "Webhook не зарегистрирован в Telegram"}, nil
	}
	if !channels.SubscriptionsContainURL(data, expectedURL) {
		return channels.WebhookStatus{Configured: false, Error: "Webhook в Telegram настроен для другого адреса"}, nil
	}
	if msg := telegramLastDeliveryError(data); msg != "" {
		return channels.WebhookStatus{Configured: false, Error: "Telegram сообщает об ошибке доставки: " + msg}, nil
	}
	return channels.WebhookStatus{Configured: true}, nil
}

// ListSubscriptions returns the current webhook registration reported by the
// Telegram Bot API (getWebhookInfo), which is Telegram's analogue of a
// subscriptions list.
func (a *Adapter) ListSubscriptions(ctx context.Context, cfg channels.ChannelConfig) (json.RawMessage, error) {
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, statusError(http.StatusBadRequest, "Токен Telegram не задан")
	}
	data, err := a.callAPI(ctx, cfg.Token, "getWebhookInfo", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func (a *Adapter) ParseInbound(r channels.WebhookRequest, cfg channels.ChannelConfig) (*channels.InboundMessage, *channels.WebhookResponse, error) {
	secret := metadataString(cfg.Metadata, "webhook_secret", "secret_token")
	if secret == "" {
		return nil, nil, statusError(http.StatusUnauthorized, "telegram webhook secret is not configured")
	}
	if subtle.ConstantTimeCompare([]byte(r.Headers.Get("X-Telegram-Bot-Api-Secret-Token")), []byte(secret)) != 1 {
		return nil, nil, statusError(http.StatusUnauthorized, "invalid telegram webhook secret")
	}
	var update struct {
		UpdateID int64 `json:"update_id"`
		Message  *struct {
			Text string `json:"text"`
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
		EditedMessage *struct {
			Text string `json:"text"`
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"edited_message"`
	}
	if err := json.Unmarshal(r.Body, &update); err != nil {
		return nil, nil, statusError(http.StatusBadRequest, "invalid telegram update")
	}
	msg := update.Message
	if msg == nil {
		msg = update.EditedMessage
	}
	if msg == nil || strings.TrimSpace(msg.Text) == "" {
		return nil, nil, nil
	}
	return &channels.InboundMessage{
		Channel:        domain.ChatChannelTelegram,
		ExternalChatID: strconv.FormatInt(msg.Chat.ID, 10),
		UpdateID:       strconv.FormatInt(update.UpdateID, 10),
		Text:           msg.Text,
	}, nil, nil
}

func (a *Adapter) SendTyping(ctx context.Context, cfg channels.ChannelConfig, externalChatID string) error {
	return a.call(ctx, cfg.Token, "sendChatAction", map[string]interface{}{
		"chat_id": externalChatID,
		"action":  "typing",
	})
}

func (a *Adapter) SendMessage(ctx context.Context, cfg channels.ChannelConfig, msg channels.OutboundMessage) error {
	return a.call(ctx, cfg.Token, "sendMessage", map[string]interface{}{
		"chat_id": msg.ExternalChatID,
		"text":    msg.Text,
	})
}

func (a *Adapter) call(ctx context.Context, token, method string, payload map[string]interface{}) error {
	_, err := a.callAPI(ctx, token, method, payload)
	return err
}

func telegramLastDeliveryError(raw json.RawMessage) string {
	var resp struct {
		Result struct {
			LastErrorMessage string `json:"last_error_message"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return ""
	}
	return strings.TrimSpace(resp.Result.LastErrorMessage)
}

func (a *Adapter) callAPI(ctx context.Context, token, method string, payload map[string]interface{}) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, applog.TraceErr(ctx, "telegram: marshal request failed", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/%s", token, method), bytes.NewReader(body))
	if err != nil {
		return nil, applog.TraceErr(ctx, "telegram: build request failed", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, applog.TraceErr(ctx, "telegram: api request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, applog.TraceErr(ctx, "telegram: read response failed", err)
	}
	if resp.StatusCode >= 300 {
		return nil, statusError(http.StatusBadGateway, "telegram api request failed")
	}
	var apiResp struct {
		OK          *bool  `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(data, &apiResp); err == nil && apiResp.OK != nil && !*apiResp.OK {
		return nil, statusError(http.StatusBadGateway, "telegram api request failed")
	}
	return data, nil
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

func statusError(status int, msg string) error {
	return &service.UserError{Status: status, Msg: msg}
}
