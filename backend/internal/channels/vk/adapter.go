package vk

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	applog "github.com/knowledgeos/backend/internal/logger"
	"math/rand"
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
	apiURL string
}

func New(client *http.Client) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Adapter{client: client, apiURL: "https://api.vk.com/method/messages.send"}
}

func (a *Adapter) Channel() domain.ChatChannel { return domain.ChatChannelVK }

func (a *Adapter) SecretKind() domain.SecretKind { return domain.SecretKindVK }

func (a *Adapter) RegisterWebhook(ctx context.Context, cfg channels.ChannelConfig, webhookURL string) (bool, error) {
	registrationURL := metadataString(cfg.Metadata, "webhook_registration_url")
	secret := metadataString(cfg.Metadata, "secret", "webhook_secret")
	confirmation := metadataString(cfg.Metadata, "confirmation_token")
	if registrationURL == "" || secret == "" || confirmation == "" || strings.TrimSpace(cfg.Token) == "" || strings.TrimSpace(webhookURL) == "" {
		return false, nil
	}
	payload := map[string]interface{}{
		"url":                webhookURL,
		"secret":             secret,
		"confirmation_token": confirmation,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, registrationURL, strings.NewReader(string(body)))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	resp, err := a.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return false, statusError(http.StatusBadGateway, "vk webhook registration failed")
	}
	return true, nil
}

func (a *Adapter) ParseInbound(r channels.WebhookRequest, cfg channels.ChannelConfig) (*channels.InboundMessage, *channels.WebhookResponse, error) {
	var update struct {
		Type    string `json:"type"`
		Secret  string `json:"secret"`
		GroupID int64  `json:"group_id"`
		Object  struct {
			Message struct {
				ID     int64  `json:"id"`
				PeerID int64  `json:"peer_id"`
				FromID int64  `json:"from_id"`
				Text   string `json:"text"`
			} `json:"message"`
		} `json:"object"`
	}
	if err := json.Unmarshal(r.Body, &update); err != nil {
		return nil, nil, statusError(http.StatusBadRequest, "invalid vk callback")
	}
	if update.Type == "confirmation" {
		token := metadataString(cfg.Metadata, "confirmation_token")
		if token == "" {
			return nil, nil, statusError(http.StatusUnauthorized, "vk confirmation token is not configured")
		}
		return nil, &channels.WebhookResponse{Status: http.StatusOK, ContentType: "text/plain; charset=utf-8", Body: []byte(token)}, nil
	}
	secret := metadataString(cfg.Metadata, "secret", "webhook_secret")
	if secret == "" {
		return nil, nil, statusError(http.StatusUnauthorized, "vk webhook secret is not configured")
	}
	if subtle.ConstantTimeCompare([]byte(update.Secret), []byte(secret)) != 1 {
		return nil, nil, statusError(http.StatusUnauthorized, "invalid vk webhook secret")
	}
	if update.Type != "message_new" || strings.TrimSpace(update.Object.Message.Text) == "" {
		return nil, &channels.WebhookResponse{Status: http.StatusOK, ContentType: "text/plain; charset=utf-8", Body: []byte("ok")}, nil
	}
	peerID := update.Object.Message.PeerID
	if peerID == 0 {
		peerID = update.Object.Message.FromID
	}
	return &channels.InboundMessage{
		Channel:        domain.ChatChannelVK,
		ExternalChatID: strconv.FormatInt(peerID, 10),
		UpdateID:       strconv.FormatInt(update.Object.Message.ID, 10),
		Text:           update.Object.Message.Text,
	}, nil, nil
}

func (a *Adapter) SendTyping(ctx context.Context, cfg channels.ChannelConfig, externalChatID string) error {
	return nil
}

func (a *Adapter) SendMessage(ctx context.Context, cfg channels.ChannelConfig, msg channels.OutboundMessage) error {
	peerID, err := strconv.ParseInt(msg.ExternalChatID, 10, 64)
	if err != nil {
		return statusError(http.StatusBadRequest, "invalid vk peer id")
	}
	form := url.Values{}
	form.Set("access_token", cfg.Token)
	form.Set("v", metadataStringDefault(cfg.Metadata, "api_version", "5.199"))
	form.Set("peer_id", strconv.FormatInt(peerID, 10))
	form.Set("random_id", strconv.FormatInt(rand.Int63(), 10))
	form.Set("message", msg.Text)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return applog.TraceErr(ctx, "vk: build request failed", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.client.Do(req)
	if err != nil {
		return applog.TraceErr(ctx, "vk: api request failed", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return statusError(http.StatusBadGateway, "vk api request failed")
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

func metadataStringDefault(raw json.RawMessage, key, fallback string) string {
	if value := metadataString(raw, key); value != "" {
		return value
	}
	return fallback
}

func statusError(status int, msg string) error {
	return &service.UserError{Status: status, Msg: msg}
}
