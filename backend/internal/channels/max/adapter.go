package max

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
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
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
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
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
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
