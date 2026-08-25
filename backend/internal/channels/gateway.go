package channels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	applog "github.com/knowledgeos/backend/internal/logger"
	"github.com/knowledgeos/backend/internal/service"
	"gorm.io/gorm"
)

const asyncWebhookTimeout = 2 * time.Minute

type chatService interface {
	SendMessage(ctx context.Context, companyID, sessionID uuid.UUID, req service.SendChatMessageRequest) (*service.ChatExchange, error)
}

type handoffService interface {
	RecordInbound(ctx context.Context, companyID uuid.UUID, session *domain.ChatSession, content string) (*domain.ChatMessage, error)
}

type webhookDeduper interface {
	StartChannelUpdate(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel, updateID string) (bool, error)
	MarkChannelUpdateDone(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel, updateID string) error
	MarkChannelUpdateFailed(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel, updateID string, errMsg string) error
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
	Channel           domain.ChatChannel `json:"channel"`
	SecretKind        domain.SecretKind  `json:"secret_kind"`
	Configured        bool               `json:"configured"`
	Enabled           bool               `json:"enabled"`
	BotEnabled        bool               `json:"bot_enabled"`
	WebhookURL        string             `json:"webhook_url"`
	WebhookConfigured bool               `json:"webhook_configured"`
	WebhookError      string             `json:"webhook_error,omitempty"`
	Metadata          json.RawMessage    `json:"metadata"`
	UpdatedAt         *string            `json:"updated_at,omitempty"`
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

func (g *Gateway) registerWebhook(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind, baseURL string) (WebhookRegistrationResult, error) {
	adapter := g.adapterBySecretKind(kind)
	if adapter == nil {
		return WebhookRegistrationResult{}, nil
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || !strings.HasPrefix(baseURL, "https://") {
		return WebhookRegistrationResult{
			Reason: "Не задан публичный HTTPS URL webhook (PUBLIC_WEBHOOK_BASE_URL или доступ к admin по HTTPS)",
		}, nil
	}
	token, metadata, err := g.secrets.GetPlaintextWithMetadata(ctx, companyID, kind)
	if err != nil {
		return WebhookRegistrationResult{}, err
	}
	cfg := ChannelConfig{Token: token, Metadata: metadata}
	targetURL := webhookURL(baseURL, adapter.Channel(), companyID)
	if reason := MissingWebhookRegistrationFields(adapter.Channel(), cfg, targetURL); reason != "" {
		return WebhookRegistrationResult{Reason: reason}, nil
	}
	registered, err := adapter.RegisterWebhook(ctx, cfg, targetURL)
	if err != nil {
		return WebhookRegistrationResult{}, err
	}
	if !registered {
		return WebhookRegistrationResult{Reason: "Не хватает обязательных полей канала"}, nil
	}
	return WebhookRegistrationResult{Registered: true}, nil
}

// RegisterChannelWebhook (re)registers the webhook for a channel on demand,
// e.g. when triggered from the admin UI. It returns whether the upstream
// registration was actually performed (false when required fields are missing).
func (g *Gateway) RegisterChannelWebhook(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel, baseURL string) (WebhookRegistrationResult, error) {
	adapter, ok := g.adapters[channel]
	if !ok {
		return WebhookRegistrationResult{}, statusError(http.StatusBadRequest, "unsupported channel")
	}
	return g.registerWebhook(ctx, companyID, adapter.SecretKind(), baseURL)
}

// ListSubscriptions returns the raw subscriptions/webhook payload reported by the
// channel provider for the stored bot token.
func (g *Gateway) ListSubscriptions(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel) (json.RawMessage, error) {
	adapter, ok := g.adapters[channel]
	if !ok {
		return nil, statusError(http.StatusBadRequest, "unsupported channel")
	}
	lister, ok := adapter.(WebhookSubscriptionsLister)
	if !ok {
		return nil, statusError(http.StatusBadRequest, "subscriptions are not supported for this channel")
	}
	token, metadata, err := g.secrets.GetPlaintextWithMetadata(ctx, companyID, adapter.SecretKind())
	if err != nil {
		return nil, err
	}
	return lister.ListSubscriptions(ctx, ChannelConfig{Token: token, Metadata: metadata})
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
		return successResponse(channel), nil
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
		if shouldAckRejected(channel) {
			if err := g.ackWebhookUpdate(ctx, companyID, channel, inbound.UpdateID); err != nil {
				return nil, err
			}
			return successResponse(channel), nil
		}
		return nil, statusError(http.StatusConflict, "bot is disabled")
	}
	if !channelEnabled(settings.EnabledModules, channel) {
		if shouldAckRejected(channel) {
			if err := g.ackWebhookUpdate(ctx, companyID, channel, inbound.UpdateID); err != nil {
				return nil, err
			}
			return successResponse(channel), nil
		}
		return nil, statusError(http.StatusConflict, "channel is disabled")
	}
	// Claim the update for processing. The event is only marked 'done' after the
	// reply is delivered, so a transient failure leaves it reclaimable and the
	// next provider retry can reprocess instead of being dropped as a duplicate.
	shouldProcess, err := g.startWebhookUpdate(ctx, companyID, channel, inbound.UpdateID)
	if err != nil {
		return nil, err
	}
	if !shouldProcess {
		return successResponse(channel), nil
	}
	session, err := g.getOrCreateSession(ctx, companyID, channel, inbound.ExternalChatID)
	if err != nil {
		g.markWebhookFailed(ctx, companyID, channel, inbound.UpdateID, err)
		return nil, err
	}
	if shouldAckBeforeProcessing(channel) {
		g.processInboundAsync(companyID, channel, adapter, cfg, session, inbound)
		return successResponse(channel), nil
	}
	if err := g.processInbound(ctx, companyID, adapter, cfg, session, inbound); err != nil {
		g.markWebhookFailed(ctx, companyID, channel, inbound.UpdateID, err)
		return nil, err
	}
	g.markWebhookDone(ctx, companyID, channel, inbound.UpdateID)
	return successResponse(channel), nil
}

func (g *Gateway) processInboundAsync(companyID uuid.UUID, channel domain.ChatChannel, adapter Adapter, cfg ChannelConfig, session *domain.ChatSession, inbound *InboundMessage) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), asyncWebhookTimeout)
		defer cancel()
		if err := g.processInbound(ctx, companyID, adapter, cfg, session, inbound); err != nil {
			g.markWebhookFailed(ctx, companyID, channel, inbound.UpdateID, err)
			applog.From(ctx).Error().
				Err(err).
				Str("company_id", companyID.String()).
				Str("channel", string(channel)).
				Str("update_id", strings.TrimSpace(inbound.UpdateID)).
				Msg("channel webhook async processing failed")
			return
		}
		g.markWebhookDone(ctx, companyID, channel, inbound.UpdateID)
	}()
}

func (g *Gateway) processInbound(ctx context.Context, companyID uuid.UUID, adapter Adapter, cfg ChannelConfig, session *domain.ChatSession, inbound *InboundMessage) error {
	if session.State != domain.ChatStateBot {
		if g.handoff != nil && (session.State == domain.ChatStateWaitingOperator || session.State == domain.ChatStateOperator) {
			_, err := g.handoff.RecordInbound(ctx, companyID, session, inbound.Text)
			return err
		}
		return nil
	}
	_ = adapter.SendTyping(ctx, cfg, inbound.ExternalChatID)
	exchange, err := g.chat.SendMessage(ctx, companyID, session.ID, service.SendChatMessageRequest{Content: inbound.Text})
	if err != nil {
		if service.HTTPStatus(err) == http.StatusConflict {
			return nil
		}
		return err
	}
	if exchange == nil || exchange.Message == nil {
		return nil
	}
	if err := adapter.SendMessage(ctx, cfg, OutboundMessage{
		ExternalChatID: inbound.ExternalChatID,
		Text:           exchange.Message.Content,
		Sources:        exchange.Sources,
	}); err != nil {
		return err
	}
	return nil
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

func (g *Gateway) Status(ctx context.Context, companyID uuid.UUID, requestBaseURL, publicBaseURL string) ([]ChannelStatus, error) {
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
		webhookStatus := g.checkWebhook(ctx, companyID, adapter, secret, publicBaseURL)
		metadata := normalizeMetadata(secret.Metadata)
		out = append(out, ChannelStatus{
			Channel:           channel,
			SecretKind:        adapter.SecretKind(),
			Configured:        secret.IsSet && ChannelRequiredFieldsComplete(channel, metadata),
			Enabled:           settings.Enabled && channelEnabled(settings.EnabledModules, channel),
			BotEnabled:        settings.Enabled,
			WebhookURL:        webhookURL(WebhookDisplayBase(requestBaseURL, publicBaseURL), channel, companyID),
			WebhookConfigured: webhookStatus.Configured,
			WebhookError:      webhookStatus.Error,
			Metadata:          metadata,
			UpdatedAt:         updatedAt,
		})
	}
	return out, nil
}

func (g *Gateway) checkWebhook(ctx context.Context, companyID uuid.UUID, adapter Adapter, secret domain.TenantSecretStatus, baseURL string) WebhookStatus {
	if !secret.IsSet {
		return WebhookStatus{Configured: false, Error: "Секрет канала не задан"}
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || !strings.HasPrefix(baseURL, "https://") {
		return WebhookStatus{
			Configured: false,
			Error:      "Не задан публичный HTTPS URL webhook (PUBLIC_WEBHOOK_BASE_URL или доступ к admin по HTTPS)",
		}
	}
	checker, ok := adapter.(WebhookChecker)
	if !ok {
		return WebhookStatus{}
	}
	token, metadata, err := g.secrets.GetPlaintextWithMetadata(ctx, companyID, adapter.SecretKind())
	if err != nil {
		return WebhookStatus{Configured: false, Error: err.Error()}
	}
	cfg := ChannelConfig{Token: token, Metadata: metadata}
	targetURL := webhookURL(baseURL, adapter.Channel(), companyID)
	if reason := MissingWebhookRegistrationFields(adapter.Channel(), cfg, targetURL); reason != "" {
		return WebhookStatus{Configured: false, Error: reason}
	}
	status, err := checker.CheckWebhook(ctx, cfg, targetURL)
	if err != nil && status.Error == "" {
		status.Error = err.Error()
	}
	return status
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

// startWebhookUpdate claims an inbound update for processing. When the store does
// not support deduplication the update is always processed.
func (g *Gateway) startWebhookUpdate(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel, updateID string) (bool, error) {
	updateID = strings.TrimSpace(updateID)
	if updateID == "" {
		return true, nil
	}
	deduper, ok := g.chats.(webhookDeduper)
	if !ok {
		return true, nil
	}
	return deduper.StartChannelUpdate(ctx, companyID, channel, updateID)
}

// markWebhookDone marks a claimed update as processed. Failures are logged but do
// not affect the already-delivered reply.
func (g *Gateway) markWebhookDone(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel, updateID string) {
	if strings.TrimSpace(updateID) == "" {
		return
	}
	deduper, ok := g.chats.(webhookDeduper)
	if !ok {
		return
	}
	if err := deduper.MarkChannelUpdateDone(ctx, companyID, channel, updateID); err != nil {
		applog.From(ctx).Warn().Err(err).
			Str("company_id", companyID.String()).
			Str("channel", string(channel)).
			Msg("channel webhook mark done failed")
	}
}

// markWebhookFailed records a processing failure so a later delivery can reclaim
// the update instead of treating it as a processed duplicate.
func (g *Gateway) markWebhookFailed(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel, updateID string, cause error) {
	if strings.TrimSpace(updateID) == "" {
		return
	}
	deduper, ok := g.chats.(webhookDeduper)
	if !ok {
		return
	}
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	if err := deduper.MarkChannelUpdateFailed(ctx, companyID, channel, updateID, msg); err != nil {
		applog.From(ctx).Warn().Err(err).
			Str("company_id", companyID.String()).
			Str("channel", string(channel)).
			Msg("channel webhook mark failed update error")
	}
}

// ackWebhookUpdate claims and immediately marks an update done. It is used when a
// webhook is accepted but intentionally not processed (bot/channel disabled) so a
// retry is not reprocessed.
func (g *Gateway) ackWebhookUpdate(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel, updateID string) error {
	shouldProcess, err := g.startWebhookUpdate(ctx, companyID, channel, updateID)
	if err != nil {
		return err
	}
	if !shouldProcess {
		return nil
	}
	g.markWebhookDone(ctx, companyID, channel, updateID)
	return nil
}

// shouldAckBeforeProcessing reports whether the webhook must return 200 to the
// provider immediately and run the LLM in the background, delivering the reply
// via a separate sendMessage. Telegram/MAX/VK all send replies out-of-band, so
// processing synchronously would keep the provider's connection open for the
// whole tool-loop — the provider times out, retries, and duplicates the update
// (observed as pending updates and duplicated /start greetings). Ack first.
func shouldAckBeforeProcessing(channel domain.ChatChannel) bool {
	switch channel {
	case domain.ChatChannelTelegram, domain.ChatChannelMAX, domain.ChatChannelVK:
		return true
	}
	return false
}

func shouldAckRejected(channel domain.ChatChannel) bool {
	return channel == domain.ChatChannelVK
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

func successResponse(channel domain.ChatChannel) *WebhookResponse {
	if channel == domain.ChatChannelVK {
		return &WebhookResponse{Status: http.StatusOK, ContentType: "text/plain; charset=utf-8", Body: []byte("ok")}
	}
	return okResponse()
}

func okResponse() *WebhookResponse {
	return &WebhookResponse{Status: http.StatusOK, ContentType: "application/json", Body: []byte(`{"ok":true}`)}
}

func statusError(status int, msg string) error {
	return &service.UserError{Status: status, Msg: msg}
}
