package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/channels"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/service"
	"gorm.io/gorm"
)

func TestChannelWebhookHandlerHandle(t *testing.T) {
	companyID := uuid.New()
	gateway := channels.NewGateway(
		&handlerFakeRepo{sessions: map[uuid.UUID]*domain.ChatSession{}},
		&handlerFakeChat{},
		&handlerFakeSettings{},
		&handlerFakeSecrets{},
		&handlerFakeAdapter{},
	)
	h := NewChannelWebhookHandler(gateway)
	router := chi.NewRouter()
	router.Post("/webhooks/{channel}/{company_id}", h.Handle)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/telegram/"+companyID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

type handlerFakeAdapter struct{}

func (a *handlerFakeAdapter) Channel() domain.ChatChannel   { return domain.ChatChannelTelegram }
func (a *handlerFakeAdapter) SecretKind() domain.SecretKind { return domain.SecretKindTelegram }
func (a *handlerFakeAdapter) ParseInbound(channels.WebhookRequest, channels.ChannelConfig) (*channels.InboundMessage, *channels.WebhookResponse, error) {
	return &channels.InboundMessage{Channel: domain.ChatChannelTelegram, ExternalChatID: "chat-1", Text: "hello"}, nil, nil
}
func (a *handlerFakeAdapter) SendTyping(context.Context, channels.ChannelConfig, string) error {
	return nil
}
func (a *handlerFakeAdapter) SendMessage(context.Context, channels.ChannelConfig, channels.OutboundMessage) error {
	return nil
}

type handlerFakeChat struct{}

func (c *handlerFakeChat) SendMessage(_ context.Context, companyID, sessionID uuid.UUID, _ service.SendChatMessageRequest) (*service.ChatExchange, error) {
	return &service.ChatExchange{Message: &domain.ChatMessage{CompanyID: companyID, SessionID: sessionID, Content: "reply"}}, nil
}

type handlerFakeSettings struct{}

func (s *handlerFakeSettings) Get(context.Context, uuid.UUID) (*domain.BotSettings, error) {
	return &domain.BotSettings{Enabled: true, EnabledModules: json.RawMessage(`{"channels":{"telegram":true}}`)}, nil
}

type handlerFakeSecrets struct{}

func (s *handlerFakeSecrets) ListStatus(context.Context, uuid.UUID) ([]domain.TenantSecretStatus, error) {
	return nil, nil
}
func (s *handlerFakeSecrets) GetPlaintextWithMetadata(context.Context, uuid.UUID, domain.SecretKind) (string, json.RawMessage, error) {
	return "token", json.RawMessage(`{}`), nil
}

type handlerFakeRepo struct {
	sessions map[uuid.UUID]*domain.ChatSession
}

func (r *handlerFakeRepo) CreateSession(_ context.Context, companyID uuid.UUID, session *domain.ChatSession) error {
	session.ID = uuid.New()
	session.CompanyID = companyID
	copy := *session
	r.sessions[session.ID] = &copy
	return nil
}
func (r *handlerFakeRepo) ListSessions(context.Context, uuid.UUID, domain.ChatSessionFilter) ([]domain.ChatSession, int64, error) {
	return nil, 0, nil
}
func (r *handlerFakeRepo) GetSession(context.Context, uuid.UUID, uuid.UUID) (*domain.ChatSession, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *handlerFakeRepo) GetSessionByExternal(_ context.Context, companyID uuid.UUID, channel domain.ChatChannel, externalChatID string) (*domain.ChatSession, error) {
	for _, session := range r.sessions {
		if session.CompanyID == companyID && session.Channel == channel && session.ExternalChatID != nil && *session.ExternalChatID == externalChatID {
			copy := *session
			return &copy, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *handlerFakeRepo) UpdateSession(context.Context, uuid.UUID, *domain.ChatSession) error {
	return nil
}
func (r *handlerFakeRepo) AppendMessage(context.Context, uuid.UUID, *domain.ChatMessage) error {
	return nil
}
func (r *handlerFakeRepo) ListMessages(context.Context, uuid.UUID, uuid.UUID, int) ([]domain.ChatMessage, error) {
	return nil, nil
}
