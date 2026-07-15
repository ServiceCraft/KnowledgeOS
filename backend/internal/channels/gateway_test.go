package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

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

func TestGatewayHandleWebhookSkipsDuplicateUpdate(t *testing.T) {
	companyID := uuid.New()
	repo := &fakeRepo{sessions: map[uuid.UUID]*domain.ChatSession{}, updates: map[string]bool{}}
	adapter := &fakeAdapter{channel: domain.ChatChannelTelegram, kind: domain.SecretKindTelegram, updateID: "evt-1"}
	chat := &fakeChat{reply: "Ответ"}
	gateway := NewGateway(
		repo,
		chat,
		&fakeSettings{enabledModules: json.RawMessage(`{"channels":{"telegram":true}}`)},
		&fakeSecrets{},
		adapter,
	)

	for i := 0; i < 2; i++ {
		resp, err := gateway.HandleWebhook(context.Background(), companyID, domain.ChatChannelTelegram, WebhookRequest{})
		if err != nil {
			t.Fatalf("HandleWebhook returned error: %v", err)
		}
		if resp == nil || resp.Status != http.StatusOK {
			t.Fatalf("unexpected response: %#v", resp)
		}
	}
	if chat.calls != 1 {
		t.Fatalf("expected one bot call for duplicate update, got %d", chat.calls)
	}
	if adapter.sendCalls != 1 {
		t.Fatalf("expected one outbound message for duplicate update, got %d", adapter.sendCalls)
	}
}

func TestGatewayHandleWebhookRetriesAfterTransientFailure(t *testing.T) {
	companyID := uuid.New()
	repo := &fakeRepo{sessions: map[uuid.UUID]*domain.ChatSession{}, updates: map[string]bool{}}
	adapter := &fakeAdapter{channel: domain.ChatChannelTelegram, kind: domain.SecretKindTelegram, updateID: "evt-1"}
	chat := &fakeChat{reply: "Ответ", failTimes: 1}
	gateway := NewGateway(
		repo,
		chat,
		&fakeSettings{enabledModules: json.RawMessage(`{"channels":{"telegram":true}}`)},
		&fakeSecrets{},
		adapter,
	)

	// First delivery fails during bot processing and must not be recorded as done.
	if _, err := gateway.HandleWebhook(context.Background(), companyID, domain.ChatChannelTelegram, WebhookRequest{}); err == nil {
		t.Fatalf("expected error on first (failing) delivery")
	}
	key := repo.updateKey(companyID, domain.ChatChannelTelegram, "evt-1")
	if repo.statuses[key] != "failed" {
		t.Fatalf("expected failed status after transient error, got %q", repo.statuses[key])
	}

	// Provider retry of the same update must be reprocessed, not dropped as duplicate.
	resp, err := gateway.HandleWebhook(context.Background(), companyID, domain.ChatChannelTelegram, WebhookRequest{})
	if err != nil {
		t.Fatalf("HandleWebhook retry returned error: %v", err)
	}
	if resp == nil || resp.Status != http.StatusOK {
		t.Fatalf("unexpected retry response: %#v", resp)
	}
	if chat.calls != 2 {
		t.Fatalf("expected two bot calls (fail + retry), got %d", chat.calls)
	}
	if adapter.sendCalls != 1 || adapter.sent == nil || adapter.sent.Text != "Ответ" {
		t.Fatalf("expected reply delivered on retry, got %d sends %#v", adapter.sendCalls, adapter.sent)
	}
	if repo.statuses[key] != "done" {
		t.Fatalf("expected done status after successful retry, got %q", repo.statuses[key])
	}
}

func TestGatewayHandleWebhookAcknowledgesVKBeforeBotProcessing(t *testing.T) {
	companyID := uuid.New()
	repo := &fakeRepo{sessions: map[uuid.UUID]*domain.ChatSession{}, updates: map[string]bool{}}
	adapter := &fakeAdapter{channel: domain.ChatChannelVK, kind: domain.SecretKindVK, updateID: "evt-1"}
	release := make(chan struct{})
	processed := make(chan struct{}, 1)
	chat := &fakeChat{reply: "Ответ", block: release, done: processed}
	gateway := NewGateway(
		repo,
		chat,
		&fakeSettings{enabledModules: json.RawMessage(`{"channels":{"vk":true}}`)},
		&fakeSecrets{},
		adapter,
	)

	result := make(chan error, 1)
	go func() {
		resp, err := gateway.HandleWebhook(context.Background(), companyID, domain.ChatChannelVK, WebhookRequest{})
		if err != nil {
			result <- err
			return
		}
		if resp == nil || resp.Status != http.StatusOK {
			result <- fmt.Errorf("unexpected response: %#v", resp)
			return
		}
		if string(resp.Body) != "ok" || resp.ContentType != "text/plain; charset=utf-8" {
			result <- fmt.Errorf("unexpected vk ack response: %#v", resp)
			return
		}
		result <- nil
	}()

	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("HandleWebhook waited for bot processing")
	}

	close(release)
	select {
	case <-processed:
	case <-time.After(time.Second):
		t.Fatal("async bot processing did not complete")
	}
}

func TestGatewayHandleWebhookAcknowledgesDisabledVKChannel(t *testing.T) {
	companyID := uuid.New()
	repo := &fakeRepo{sessions: map[uuid.UUID]*domain.ChatSession{}, updates: map[string]bool{}}
	adapter := &fakeAdapter{channel: domain.ChatChannelVK, kind: domain.SecretKindVK, updateID: "evt-1"}
	chat := &fakeChat{reply: "Ответ"}
	gateway := NewGateway(
		repo,
		chat,
		&fakeSettings{enabledModules: json.RawMessage(`{"channels":{"vk":false}}`)},
		&fakeSecrets{},
		adapter,
	)

	resp, err := gateway.HandleWebhook(context.Background(), companyID, domain.ChatChannelVK, WebhookRequest{})
	if err != nil {
		t.Fatalf("HandleWebhook returned error: %v", err)
	}
	if resp == nil || resp.Status != http.StatusOK || string(resp.Body) != "ok" {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if chat.calls != 0 {
		t.Fatalf("bot was called for disabled channel")
	}
}

type fakeAdapter struct {
	channel   domain.ChatChannel
	kind      domain.SecretKind
	sent      *OutboundMessage
	updateID  string
	sendCalls int
}

func (a *fakeAdapter) Channel() domain.ChatChannel   { return a.channel }
func (a *fakeAdapter) SecretKind() domain.SecretKind { return a.kind }
func (a *fakeAdapter) RegisterWebhook(context.Context, ChannelConfig, string) (bool, error) {
	return true, nil
}
func (a *fakeAdapter) ParseInbound(WebhookRequest, ChannelConfig) (*InboundMessage, *WebhookResponse, error) {
	return &InboundMessage{Channel: a.channel, ExternalChatID: "external-1", UpdateID: a.updateID, Text: "Привет"}, nil, nil
}
func (a *fakeAdapter) SendTyping(context.Context, ChannelConfig, string) error { return nil }
func (a *fakeAdapter) SendMessage(_ context.Context, _ ChannelConfig, msg OutboundMessage) error {
	a.sent = &msg
	a.sendCalls++
	return nil
}

type fakeChat struct {
	reply     string
	calls     int
	block     <-chan struct{}
	done      chan<- struct{}
	failTimes int
}

func (c *fakeChat) SendMessage(_ context.Context, companyID, sessionID uuid.UUID, req service.SendChatMessageRequest) (*service.ChatExchange, error) {
	c.calls++
	if c.block != nil {
		<-c.block
	}
	if c.done != nil {
		c.done <- struct{}{}
	}
	if c.calls <= c.failTimes {
		return nil, fmt.Errorf("transient failure %d", c.calls)
	}
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
	updates  map[string]bool
	statuses map[string]string
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
func (r *fakeRepo) DeleteSession(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	if _, ok := r.sessions[id]; !ok {
		return gorm.ErrRecordNotFound
	}
	delete(r.sessions, id)
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
func (r *fakeRepo) updateKey(companyID uuid.UUID, channel domain.ChatChannel, updateID string) string {
	return companyID.String() + "|" + string(channel) + "|" + updateID
}

func (r *fakeRepo) StartChannelUpdate(_ context.Context, companyID uuid.UUID, channel domain.ChatChannel, updateID string) (bool, error) {
	if updateID == "" {
		return true, nil
	}
	if r.updates == nil {
		r.updates = map[string]bool{}
	}
	if r.statuses == nil {
		r.statuses = map[string]string{}
	}
	key := r.updateKey(companyID, channel, updateID)
	switch r.statuses[key] {
	case "done", "processing":
		return false, nil
	default:
		r.statuses[key] = "processing"
		r.updates[key] = true
		return true, nil
	}
}

func (r *fakeRepo) MarkChannelUpdateDone(_ context.Context, companyID uuid.UUID, channel domain.ChatChannel, updateID string) error {
	if updateID == "" {
		return nil
	}
	if r.statuses == nil {
		r.statuses = map[string]string{}
	}
	r.statuses[r.updateKey(companyID, channel, updateID)] = "done"
	return nil
}

func (r *fakeRepo) MarkChannelUpdateFailed(_ context.Context, companyID uuid.UUID, channel domain.ChatChannel, updateID string, _ string) error {
	if updateID == "" {
		return nil
	}
	if r.statuses == nil {
		r.statuses = map[string]string{}
	}
	r.statuses[r.updateKey(companyID, channel, updateID)] = "failed"
	return nil
}
func (r *fakeRepo) UpdateSession(context.Context, uuid.UUID, *domain.ChatSession) error { return nil }
func (r *fakeRepo) AppendMessage(context.Context, uuid.UUID, *domain.ChatMessage) error { return nil }
func (r *fakeRepo) ListMessages(context.Context, uuid.UUID, uuid.UUID, int) ([]domain.ChatMessage, error) {
	return nil, nil
}
