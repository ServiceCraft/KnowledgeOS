package channels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	applog "github.com/knowledgeos/backend/internal/logger"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/service"
	"gorm.io/gorm"
)

type chatService interface {
	SendMessage(ctx context.Context, companyID, sessionID uuid.UUID, req service.SendChatMessageRequest) (*service.ChatExchange, error)
}

type handoffService interface {
	RecordInbound(ctx context.Context, companyID uuid.UUID, session *domain.ChatSession, content string) (*domain.ChatMessage, error)
}

type settingsService interface {
	Get(ctx context.Context, companyID uuid.UUID) (*domain.BotSettings, error)
}

type secretService interface {
	ListStatus(ctx context.Context, companyID uuid.UUID) ([]domain.TenantSecretStatus, error)
	GetPlaintextWithMetadata(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind) (string, json.RawMessage, error)
}

type Gateway struct {
	chats    domain.ChatRepository
	chat     chatService
	settings settingsService
	secrets  secretService
	handoff  handoffService
	adapters map[domain.ChatChannel]Adapter
}

type ChannelStatus struct {
	Channel    domain.ChatChannel `json:"channel"`
	SecretKind domain.SecretKind  `json:"secret_kind"`
	Configured bool               `json:"configured"`
	Enabled    bool               `json:"enabled"`
	BotEnabled bool               `json:"bot_enabled"`
	WebhookURL string             `json:"webhook_url"`
	Metadata   json.RawMessage    `json:"metadata"`
	UpdatedAt  *string            `json:"updated_at,omitempty"`
}

func NewGateway(chats domain.ChatRepository, chat chatService, settings settingsService, secrets secretService, adapters ...Adapter) *Gateway {
	byChannel := make(map[domain.ChatChannel]Adapter, len(adapters))
	for _, adapter := range adapters {
		if adapter != nil {
			byChannel[adapter.Channel()] = adapter
		}
	}
	return &Gateway{chats: chats, chat: chat, settings: settings, secrets: secrets, adapters: byChannel}
}

func (g *Gateway) SetHandoff(handoff handoffService) {
	g.handoff = handoff
}

func (g *Gateway) EnsureWebhook(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind, baseURL string) (bool, error) {
	adapter := g.adapterBySecretKind(kind)
	if adapter == nil {
		return false, nil
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || !strings.HasPrefix(baseURL, "https://") {
		return false, nil
	}
	token, metadata, err := g.secrets.GetPlaintextWithMetadata(ctx, companyID, kind)
	if err != nil {
		return false, err
	}
	cfg := ChannelConfig{Token: token, Metadata: metadata}
	return adapter.RegisterWebhook(ctx, cfg, webhookURL(baseURL, adapter.Channel(), companyID))
}

func (g *Gateway) HandleWebhook(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel, r WebhookRequest) (*WebhookResponse, error) {
	adapter, ok := g.adapters[channel]
	if !ok {
		return nil, statusError(http.StatusBadRequest, "unsupported channel")
	}
	token, metadata, err := g.secrets.GetPlaintextWithMetadata(ctx, companyID, adapter.SecretKind())
	if err != nil {
		return nil, err
	}
	cfg := ChannelConfig{Token: token, Metadata: metadata}
	inbound, immediate, err := adapter.ParseInbound(r, cfg)
	if err != nil {
		return nil, err
	}
	if immediate != nil {
		return immediate, nil
	}
	if inbound == nil || strings.TrimSpace(inbound.Text) == "" {
		return okResponse(), nil
	}
	if inbound.Channel == "" {
		inbound.Channel = channel
	}
	if inbound.Channel != channel {
		return nil, statusError(http.StatusBadRequest, "webhook channel mismatch")
	}
	if strings.TrimSpace(inbound.ExternalChatID) == "" {
		return nil, statusError(http.StatusBadRequest, "external chat id is required")
	}
	settings, err := g.settings.Get(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled {
		return nil, statusError(http.StatusConflict, "bot is disabled")
	}
	if !channelEnabled(settings.EnabledModules, channel) {
		return nil, statusError(http.StatusConflict, "channel is disabled")
	}
	session, err := g.getOrCreateSession(ctx, companyID, channel, inbound.ExternalChatID)
	if err != nil {
		return nil, err
	}
	if session.State != domain.ChatStateBot {
		if g.handoff != nil && (session.State == domain.ChatStateWaitingOperator || session.State == domain.ChatStateOperator) {
			if _, err := g.handoff.RecordInbound(ctx, companyID, session, inbound.Text); err != nil {
				return nil, err
			}
		}
		return okResponse(), nil
	}
	_ = adapter.SendTyping(ctx, cfg, inbound.ExternalChatID)
	exchange, err := g.chat.SendMessage(ctx, companyID, session.ID, service.SendChatMessageRequest{Content: inbound.Text})
	if err != nil {
		if service.HTTPStatus(err) == http.StatusConflict {
			return okResponse(), nil
		}
		return nil, err
	}
	if exchange == nil || exchange.Message == nil {
		return okResponse(), nil
	}
	if err := adapter.SendMessage(ctx, cfg, OutboundMessage{
		ExternalChatID: inbound.ExternalChatID,
		Text:           exchange.Message.Content,
		Sources:        exchange.Sources,
	}); err != nil {
		return nil, err
	}
	return okResponse(), nil
}

func (g *Gateway) SendOperatorMessage(ctx context.Context, companyID uuid.UUID, session *domain.ChatSession, message *domain.ChatMessage) error {
	if session == nil || message == nil {
		return nil
	}
	if session.Channel == domain.ChatChannelPlayground || session.Channel == domain.ChatChannelAPI || session.ExternalChatID == nil {
		return nil
	}
	adapter, ok := g.adapters[session.Channel]
	if !ok {
		return statusError(http.StatusBadRequest, "unsupported channel")
	}
	token, metadata, err := g.secrets.GetPlaintextWithMetadata(ctx, companyID, adapter.SecretKind())
	if err != nil {
		return applog.TraceErr(ctx, "channel gateway: load channel secret failed", err)
	}
	return adapter.SendMessage(ctx, ChannelConfig{Token: token, Metadata: metadata}, OutboundMessage{
		ExternalChatID: *session.ExternalChatID,
		Text:           message.Content,
	})
}

func (g *Gateway) NotifyHandoff(ctx context.Context, companyID uuid.UUID, session *domain.ChatSession, reason string, lastQuestion string) error {
	adapter, ok := g.adapters[domain.ChatChannelTelegram]
	if !ok {
		return nil
	}
	token, metadata, err := g.secrets.GetPlaintextWithMetadata(ctx, companyID, adapter.SecretKind())
	if err != nil {
		if service.HTTPStatus(err) == http.StatusNotFound {
			return nil
		}
		return applog.TraceErr(ctx, "channel gateway: load handoff notification secret failed", err)
	}
	chatID := handoffNotificationChatID(metadata)
	if chatID == "" {
		return nil
	}
	return adapter.SendMessage(ctx, ChannelConfig{Token: token, Metadata: metadata}, OutboundMessage{
		ExternalChatID: chatID,
		Text:           handoffNotificationText(session, reason, lastQuestion),
	})
}

func (g *Gateway) Status(ctx context.Context, companyID uuid.UUID, baseURL string) ([]ChannelStatus, error) {
	settings, err := g.settings.Get(ctx, companyID)
	if err != nil {
		return nil, err
	}
	secrets, err := g.secrets.ListStatus(ctx, companyID)
	if err != nil {
		return nil, err
	}
	byKind := make(map[domain.SecretKind]domain.TenantSecretStatus, len(secrets))
	for _, item := range secrets {
		byKind[item.Kind] = item
	}
	channels := []domain.ChatChannel{domain.ChatChannelTelegram, domain.ChatChannelMAX, domain.ChatChannelVK}
	out := make([]ChannelStatus, 0, len(channels))
	for _, channel := range channels {
		adapter, ok := g.adapters[channel]
		if !ok {
			continue
		}
		secret := byKind[adapter.SecretKind()]
		var updatedAt *string
		if secret.UpdatedAt != nil {
			v := secret.UpdatedAt.Format(time.RFC3339)
			updatedAt = &v
		}
		out = append(out, ChannelStatus{
			Channel:    channel,
			SecretKind: adapter.SecretKind(),
			Configured: secret.IsSet,
			Enabled:    settings.Enabled && channelEnabled(settings.EnabledModules, channel),
			BotEnabled: settings.Enabled,
			WebhookURL: webhookURL(baseURL, channel, companyID),
			Metadata:   normalizeMetadata(secret.Metadata),
			UpdatedAt:  updatedAt,
		})
	}
	return out, nil
}

func (g *Gateway) getOrCreateSession(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel, externalChatID string) (*domain.ChatSession, error) {
	session, err := g.chats.GetSessionByExternal(ctx, companyID, channel, externalChatID)
	if err == nil {
		return session, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	session = &domain.ChatSession{
		Channel:        channel,
		ExternalChatID: &externalChatID,
		State:          domain.ChatStateBot,
	}
	if err := g.chats.CreateSession(ctx, companyID, session); err != nil {
		if existing, lookupErr := g.chats.GetSessionByExternal(ctx, companyID, channel, externalChatID); lookupErr == nil {
			return existing, nil
		}
		return nil, err
	}
	return session, nil
}

func (g *Gateway) adapterBySecretKind(kind domain.SecretKind) Adapter {
	for _, adapter := range g.adapters {
		if adapter.SecretKind() == kind {
			return adapter
		}
	}
	return nil
}

func channelEnabled(raw json.RawMessage, channel domain.ChatChannel) bool {
	if len(raw) == 0 {
		return false
	}
	var modules map[string]json.RawMessage
	if err := json.Unmarshal(raw, &modules); err != nil {
		return false
	}
	value, ok := modules["channels"]
	if !ok || len(value) == 0 {
		return false
	}
	var all bool
	if err := json.Unmarshal(value, &all); err == nil {
		return all
	}
	var perChannel map[string]bool
	if err := json.Unmarshal(value, &perChannel); err != nil {
		return false
	}
	return perChannel[string(channel)]
}

func handoffNotificationChatID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var metadata map[string]interface{}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return ""
	}
	for _, key := range []string{"handoff_notification_chat_id", "notification_chat_id", "handoff_group_chat_id"} {
		if value, ok := metadata[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	if nested, ok := metadata["handoff"].(map[string]interface{}); ok {
		if value, ok := nested["notification_chat_id"].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func handoffNotificationText(session *domain.ChatSession, reason string, lastQuestion string) string {
	title := "Новый диалог"
	if session != nil && strings.TrimSpace(session.Title) != "" {
		title = strings.TrimSpace(session.Title)
	}
	question := strings.TrimSpace(lastQuestion)
	if len([]rune(question)) > 240 {
		question = string([]rune(question)[:240]) + "..."
	}
	parts := []string{
		"Требуется оператор",
		fmt.Sprintf("Диалог: %s", title),
	}
	if session != nil {
		parts = append(parts, fmt.Sprintf("Канал: %s", session.Channel))
	}
	if strings.TrimSpace(reason) != "" {
		parts = append(parts, fmt.Sprintf("Причина: %s", reason))
	}
	if question != "" {
		parts = append(parts, fmt.Sprintf("Последний вопрос: %s", question))
	}
	return strings.Join(parts, "\n")
}

func webhookURL(baseURL string, channel domain.ChatChannel, companyID uuid.UUID) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return fmt.Sprintf("/api/v1/webhooks/%s/%s", channel, companyID)
	}
	return fmt.Sprintf("%s/api/v1/webhooks/%s/%s", baseURL, channel, companyID)
}

func normalizeMetadata(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

func okResponse() *WebhookResponse {
	return &WebhookResponse{Status: http.StatusOK, ContentType: "application/json", Body: []byte(`{"ok":true}`)}
}

func statusError(status int, msg string) error {
	return &service.UserError{Status: status, Msg: msg}
}
