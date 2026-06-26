package telegram

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
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
	return true, a.call(ctx, cfg.Token, "setWebhook", map[string]interface{}{
		"url":                  webhookURL,
		"secret_token":         secret,
		"allowed_updates":      []string{"message"},
		"drop_pending_updates": false,
	})
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
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/%s", token, method), bytes.NewReader(body))
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
		return statusError(http.StatusBadGateway, "telegram api request failed")
	}
	return nil
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
