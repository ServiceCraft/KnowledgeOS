package channels

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/service"
	"gorm.io/gorm"
)

func TestGatewayHandleWebhookCreatesSessionAndSendsReply(t *testing.T) {
	companyID := uuid.New()
	repo := &fakeRepo{sessions: map[uuid.UUID]*domain.ChatSession{}}
	adapter := &fakeAdapter{channel: domain.ChatChannelTelegram, kind: domain.SecretKindTelegram}
	gateway := NewGateway(
		repo,
		&fakeChat{reply: "Ответ"},
		&fakeSettings{enabledModules: json.RawMessage(`{"channels":{"telegram":true}}`)},
		&fakeSecrets{},
		adapter,
	)

	resp, err := gateway.HandleWebhook(context.Background(), companyID, domain.ChatChannelTelegram, WebhookRequest{Headers: http.Header{}, Body: []byte(`{}`)})
	if err != nil {
		t.Fatalf("HandleWebhook returned error: %v", err)
	}
	if resp == nil || resp.Status != http.StatusOK {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if len(repo.sessions) != 1 {
		t.Fatalf("expected one session, got %d", len(repo.sessions))
	}
	if adapter.sent == nil || adapter.sent.Text != "Ответ" {
		t.Fatalf("expected outbound reply, got %#v", adapter.sent)
	}
}

func TestGatewayHandleWebhookSkipsDisabledChannel(t *testing.T) {
	companyID := uuid.New()
	gateway := NewGateway(
		&fakeRepo{sessions: map[uuid.UUID]*domain.ChatSession{}},
		&fakeChat{reply: "Ответ"},
		&fakeSettings{enabledModules: json.RawMessage(`{"channels":{"telegram":false}}`)},
		&fakeSecrets{},
		&fakeAdapter{channel: domain.ChatChannelTelegram, kind: domain.SecretKindTelegram},
	)

	_, err := gateway.HandleWebhook(context.Background(), companyID, domain.ChatChannelTelegram, WebhookRequest{})
	if err == nil || service.HTTPStatus(err) != http.StatusConflict {
		t.Fatalf("expected conflict for disabled channel, got %v", err)
	}
}

func TestGatewayHandleWebhookRecordsInboundDuringHandoff(t *testing.T) {
	companyID := uuid.New()
	externalID := "external-1"
	sessionID := uuid.New()
	repo := &fakeRepo{sessions: map[uuid.UUID]*domain.ChatSession{
		sessionID: {
			BaseModel:      domain.BaseModel{ID: sessionID},
			CompanyID:      companyID,
			Channel:        domain.ChatChannelTelegram,
			ExternalChatID: &externalID,
			State:          domain.ChatStateOperator,
		},
	}}
	chat := &fakeChat{reply: "Ответ"}
	handoff := &fakeHandoff{}
	gateway := NewGateway(
		repo,
		chat,
		&fakeSettings{enabledModules: json.RawMessage(`{"channels":{"telegram":true}}`)},
		&fakeSecrets{},
		&fakeAdapter{channel: domain.ChatChannelTelegram, kind: domain.SecretKindTelegram},
	)
	gateway.SetHandoff(handoff)

	resp, err := gateway.HandleWebhook(context.Background(), companyID, domain.ChatChannelTelegram, WebhookRequest{})
	if err != nil {
		t.Fatalf("HandleWebhook returned error: %v", err)
	}
	if resp == nil || resp.Status != http.StatusOK {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if chat.calls != 0 {
		t.Fatalf("bot was called during handoff")
	}
	if handoff.text != "Привет" || handoff.sessionID != sessionID {
		t.Fatalf("handoff inbound = session %s text %q", handoff.sessionID, handoff.text)
	}
}

type fakeAdapter struct {
	channel domain.ChatChannel
	kind    domain.SecretKind
	sent    *OutboundMessage
}

func (a *fakeAdapter) Channel() domain.ChatChannel   { return a.channel }
func (a *fakeAdapter) SecretKind() domain.SecretKind { return a.kind }
func (a *fakeAdapter) RegisterWebhook(context.Context, ChannelConfig, string) (bool, error) {
	return true, nil
}
func (a *fakeAdapter) ParseInbound(WebhookRequest, ChannelConfig) (*InboundMessage, *WebhookResponse, error) {
	return &InboundMessage{Channel: a.channel, ExternalChatID: "external-1", Text: "Привет"}, nil, nil
}
func (a *fakeAdapter) SendTyping(context.Context, ChannelConfig, string) error { return nil }
func (a *fakeAdapter) SendMessage(_ context.Context, _ ChannelConfig, msg OutboundMessage) error {
	a.sent = &msg
	return nil
}

type fakeChat struct {
	reply string
	calls int
}

func (c *fakeChat) SendMessage(_ context.Context, companyID, sessionID uuid.UUID, req service.SendChatMessageRequest) (*service.ChatExchange, error) {
	c.calls++
	return &service.ChatExchange{
		Message: &domain.ChatMessage{CompanyID: companyID, SessionID: sessionID, Role: domain.ChatRoleAssistant, Content: c.reply},
	}, nil
}

type fakeHandoff struct {
	sessionID uuid.UUID
	text      string
}

func (h *fakeHandoff) RecordInbound(_ context.Context, _ uuid.UUID, session *domain.ChatSession, content string) (*domain.ChatMessage, error) {
	h.sessionID = session.ID
	h.text = content
	return &domain.ChatMessage{SessionID: session.ID, Role: domain.ChatRoleUser, Content: content}, nil
}

type fakeSettings struct {
	enabledModules json.RawMessage
}

func (s *fakeSettings) Get(context.Context, uuid.UUID) (*domain.BotSettings, error) {
	return &domain.BotSettings{Enabled: true, EnabledModules: s.enabledModules}, nil
}

type fakeSecrets struct{}

func (s *fakeSecrets) ListStatus(context.Context, uuid.UUID) ([]domain.TenantSecretStatus, error) {
	return nil, nil
}
func (s *fakeSecrets) GetPlaintextWithMetadata(context.Context, uuid.UUID, domain.SecretKind) (string, json.RawMessage, error) {
	return "token", json.RawMessage(`{}`), nil
}

type fakeRepo struct {
	sessions map[uuid.UUID]*domain.ChatSession
}

func (r *fakeRepo) CreateSession(_ context.Context, companyID uuid.UUID, session *domain.ChatSession) error {
	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}
	session.CompanyID = companyID
	copy := *session
	r.sessions[session.ID] = &copy
	return nil
}
func (r *fakeRepo) ListSessions(context.Context, uuid.UUID, domain.ChatSessionFilter) ([]domain.ChatSession, int64, error) {
	return nil, 0, nil
}
func (r *fakeRepo) GetSession(_ context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.ChatSession, error) {
	session := r.sessions[id]
	if session == nil || session.CompanyID != companyID {
		return nil, gorm.ErrRecordNotFound
	}
	copy := *session
	return &copy, nil
}
func (r *fakeRepo) GetSessionByExternal(_ context.Context, companyID uuid.UUID, channel domain.ChatChannel, externalChatID string) (*domain.ChatSession, error) {
	for _, session := range r.sessions {
		if session.CompanyID == companyID && session.Channel == channel && session.ExternalChatID != nil && *session.ExternalChatID == externalChatID {
			copy := *session
			return &copy, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeRepo) UpdateSession(context.Context, uuid.UUID, *domain.ChatSession) error { return nil }
func (r *fakeRepo) AppendMessage(context.Context, uuid.UUID, *domain.ChatMessage) error { return nil }
func (r *fakeRepo) ListMessages(context.Context, uuid.UUID, uuid.UUID, int) ([]domain.ChatMessage, error) {
	return nil, nil
}
