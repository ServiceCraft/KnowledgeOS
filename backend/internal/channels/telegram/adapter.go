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

// defaultAPIBase is the public Bot API origin. It can be overridden with a relay
// (e.g. a Cloudflare Worker forwarding to api.telegram.org) where the host cannot
// reach Telegram directly — see TELEGRAM_API_BASE_URL.
const defaultAPIBase = "https://api.telegram.org"

type Adapter struct {
	client      *http.Client
	apiBase     string
	relaySecret string
}

func New(client *http.Client) *Adapter {
	return NewWithBase(client, "")
}

// NewWithBase builds an adapter pointing at a custom Bot API base URL. An empty
// baseURL falls back to the public api.telegram.org. Use a relay base when the
// host's network blocks Telegram directly.
func NewWithBase(client *http.Client, baseURL string) *Adapter {
	return NewWithRelay(client, baseURL, "")
}

// NewWithRelay is NewWithBase plus an optional shared secret sent as the
// X-Relay-Secret header, matching a hardened relay worker (see
// scripts/telegram-relay). Empty secret means the header is not sent.
func NewWithRelay(client *http.Client, baseURL, relaySecret string) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = defaultAPIBase
	}
	return &Adapter{client: client, apiBase: base, relaySecret: strings.TrimSpace(relaySecret)}
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
		"allowed_updates":      []string{"message", "callback_query"},
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
	return parseTelegramUpdate(r.Body)
}

// parseTelegramUpdate decodes one Telegram update (message, edited_message or
// callback_query) into an InboundMessage. Shared by webhook and long-polling.
func parseTelegramUpdate(body []byte) (*channels.InboundMessage, *channels.WebhookResponse, error) {
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
		CallbackQuery *struct {
			ID      string `json:"id"`
			Data    string `json:"data"`
			Message *struct {
				Chat struct {
					ID int64 `json:"id"`
				} `json:"chat"`
			} `json:"message"`
		} `json:"callback_query"`
	}
	if err := json.Unmarshal(body, &update); err != nil {
		return nil, nil, statusError(http.StatusBadRequest, "invalid telegram update")
	}
	// Inline-button press.
	if cq := update.CallbackQuery; cq != nil && strings.TrimSpace(cq.Data) != "" && cq.Message != nil {
		return &channels.InboundMessage{
			Channel:        domain.ChatChannelTelegram,
			ExternalChatID: strconv.FormatInt(cq.Message.Chat.ID, 10),
			UpdateID:       strconv.FormatInt(update.UpdateID, 10),
			CallbackData:   strings.TrimSpace(cq.Data),
			CallbackID:     cq.ID,
		}, nil, nil
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

// DeleteWebhook removes any registered webhook so long-polling (getUpdates) can
// run. Webhook and getUpdates are mutually exclusive in the Bot API.
func (a *Adapter) DeleteWebhook(ctx context.Context, token string) error {
	return a.call(ctx, token, "deleteWebhook", map[string]interface{}{"drop_pending_updates": false})
}

// GetUpdates long-polls the Bot API for new updates starting at offset. It is
// used where Telegram cannot reach our webhook (e.g. inbound filtering on a
// RU-hosted VM): the bot pulls updates over its own outbound connection instead.
// The server-side timeout is kept below the HTTP client timeout so the request
// always returns cleanly.
func (a *Adapter) GetUpdates(ctx context.Context, token string, offset int64) ([]channels.PolledUpdate, int64, error) {
	payload := map[string]interface{}{
		"timeout":         5,
		"allowed_updates": []string{"message", "callback_query"},
	}
	if offset > 0 {
		payload["offset"] = offset
	}
	data, err := a.callAPI(ctx, token, "getUpdates", payload)
	if err != nil {
		return nil, 0, err
	}
	var resp struct {
		Result []json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, 0, applog.TraceErr(ctx, "telegram: decode getUpdates failed", err)
	}
	updates := make([]channels.PolledUpdate, 0, len(resp.Result))
	var maxID int64
	for _, raw := range resp.Result {
		var head struct {
			UpdateID int64 `json:"update_id"`
		}
		if err := json.Unmarshal(raw, &head); err != nil {
			continue
		}
		updates = append(updates, channels.PolledUpdate{UpdateID: head.UpdateID, Raw: raw})
		if head.UpdateID > maxID {
			maxID = head.UpdateID
		}
	}
	return updates, maxID, nil
}

func (a *Adapter) SendTyping(ctx context.Context, cfg channels.ChannelConfig, externalChatID string) error {
	return a.call(ctx, cfg.Token, "sendChatAction", map[string]interface{}{
		"chat_id": externalChatID,
		"action":  "typing",
	})
}

func (a *Adapter) SendMessage(ctx context.Context, cfg channels.ChannelConfig, msg channels.OutboundMessage) error {
	payload := map[string]interface{}{
		"chat_id": msg.ExternalChatID,
		"text":    msg.Text,
	}
	if len(msg.Buttons) > 0 {
		row := make([]map[string]interface{}, 0, len(msg.Buttons))
		for _, b := range msg.Buttons {
			row = append(row, map[string]interface{}{"text": b.Text, "callback_data": b.Data})
		}
		// One button per row reads better on mobile.
		rows := make([][]map[string]interface{}, 0, len(row))
		for _, b := range row {
			rows = append(rows, []map[string]interface{}{b})
		}
		payload["reply_markup"] = map[string]interface{}{"inline_keyboard": rows}
	}
	return a.call(ctx, cfg.Token, "sendMessage", payload)
}

// AnswerCallback acknowledges an inline-button press so the client's button
// stops showing a loading spinner. Optional text shows as a small toast.
func (a *Adapter) AnswerCallback(ctx context.Context, cfg channels.ChannelConfig, callbackID, text string) error {
	payload := map[string]interface{}{"callback_query_id": callbackID}
	if strings.TrimSpace(text) != "" {
		payload["text"] = text
	}
	return a.call(ctx, cfg.Token, "answerCallbackQuery", payload)
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/bot%s/%s", a.apiBase, token, method), bytes.NewReader(body))
	if err != nil {
		return nil, applog.TraceErr(ctx, "telegram: build request failed", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if a.relaySecret != "" {
		req.Header.Set("X-Relay-Secret", a.relaySecret)
	}
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
