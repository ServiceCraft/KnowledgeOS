package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
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
	adapter := &fakeAdapter{channel: domain.ChatChannelTelegram, kind: domain.SecretKindTelegram, updateID: "evt-1"}
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
	// The session is created synchronously; the reply is delivered by the async
	// worker, so wait for it to finish before asserting the outbound message.
	if !repo.waitStatus(repo.updateKey(companyID, domain.ChatChannelTelegram, "evt-1"), "done", time.Second) {
		t.Fatalf("expected async processing to finish (status done)")
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
		&fakeAdapter{channel: domain.ChatChannelTelegram, kind: domain.SecretKindTelegram, updateID: "evt-1"},
	)
	gateway.SetHandoff(handoff)

	resp, err := gateway.HandleWebhook(context.Background(), companyID, domain.ChatChannelTelegram, WebhookRequest{})
	if err != nil {
		t.Fatalf("HandleWebhook returned error: %v", err)
	}
	if resp == nil || resp.Status != http.StatusOK {
		t.Fatalf("unexpected response: %#v", resp)
	}
	// Inbound recording during handoff runs in the async worker now.
	if !repo.waitStatus(repo.updateKey(companyID, domain.ChatChannelTelegram, "evt-1"), "done", time.Second) {
		t.Fatalf("expected async processing to finish (status done)")
	}
	if chat.calls != 0 {
		t.Fatalf("bot was called during handoff")
	}
	if handoff.text != "Привет" || handoff.sessionID != sessionID {
		t.Fatalf("handoff inbound = session %s text %q", handoff.sessionID, handoff.text)
	}
}

func TestProcessInboundWaitingOperatorOffersChoice(t *testing.T) {
	companyID := uuid.New()
	externalID := "external-1"
	sessionID := uuid.New()
	session := &domain.ChatSession{
		BaseModel:      domain.BaseModel{ID: sessionID},
		CompanyID:      companyID,
		Channel:        domain.ChatChannelTelegram,
		ExternalChatID: &externalID,
		State:          domain.ChatStateWaitingOperator,
	}
	adapter := &fakeAdapter{channel: domain.ChatChannelTelegram, kind: domain.SecretKindTelegram}
	handoff := &fakeHandoff{}
	gateway := NewGateway(&fakeRepo{sessions: map[uuid.UUID]*domain.ChatSession{}}, &fakeChat{}, &fakeSettings{}, &fakeSecrets{}, adapter)
	gateway.SetHandoff(handoff)

	inbound := &InboundMessage{Channel: domain.ChatChannelTelegram, ExternalChatID: externalID, Text: "новый вопрос про прививки"}
	if err := gateway.processInbound(context.Background(), companyID, adapter, ChannelConfig{}, session, inbound); err != nil {
		t.Fatalf("processInbound error: %v", err)
	}
	// The new message must still reach the operator.
	if handoff.text != "новый вопрос про прививки" {
		t.Fatalf("inbound not recorded for operator: %q", handoff.text)
	}
	// And the client must be offered the two choice buttons.
	if adapter.sent == nil || len(adapter.sent.Buttons) != 2 {
		t.Fatalf("expected 2 choice buttons, got %#v", adapter.sent)
	}
	if adapter.sent.Buttons[0].Data != callbackWaitOperator || adapter.sent.Buttons[1].Data != callbackBackToBot {
		t.Fatalf("unexpected button data: %#v", adapter.sent.Buttons)
	}
}

func TestProcessInboundBackToBotCallback(t *testing.T) {
	companyID := uuid.New()
	externalID := "external-1"
	sessionID := uuid.New()
	session := &domain.ChatSession{
		BaseModel:      domain.BaseModel{ID: sessionID},
		CompanyID:      companyID,
		Channel:        domain.ChatChannelTelegram,
		ExternalChatID: &externalID,
		State:          domain.ChatStateWaitingOperator,
	}
	adapter := &fakeAdapter{channel: domain.ChatChannelTelegram, kind: domain.SecretKindTelegram}
	handoff := &fakeHandoff{}
	gateway := NewGateway(&fakeRepo{sessions: map[uuid.UUID]*domain.ChatSession{}}, &fakeChat{}, &fakeSettings{}, &fakeSecrets{}, adapter)
	gateway.SetHandoff(handoff)

	inbound := &InboundMessage{Channel: domain.ChatChannelTelegram, ExternalChatID: externalID, CallbackData: callbackBackToBot, CallbackID: "cb-1"}
	if err := gateway.processInbound(context.Background(), companyID, adapter, ChannelConfig{}, session, inbound); err != nil {
		t.Fatalf("processInbound error: %v", err)
	}
	if handoff.returnedToID != sessionID {
		t.Fatalf("expected ReturnToBot on session %s, got %s", sessionID, handoff.returnedToID)
	}
	if adapter.sent == nil || len(adapter.sent.Buttons) != 0 {
		t.Fatalf("expected a plain confirmation message, got %#v", adapter.sent)
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
	// Telegram acks first and processes asynchronously; the duplicate is dropped
	// synchronously at claim time, so only one async worker ever runs.
	key := repo.updateKey(companyID, domain.ChatChannelTelegram, "evt-1")
	if !repo.waitStatus(key, "done", time.Second) {
		t.Fatalf("expected async processing to finish (status done)")
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

	// Telegram acks first, so HandleWebhook returns 200 immediately; the first
	// (failing) delivery is handled by the async worker and must leave the update
	// reclaimable (status failed), not done.
	if _, err := gateway.HandleWebhook(context.Background(), companyID, domain.ChatChannelTelegram, WebhookRequest{}); err != nil {
		t.Fatalf("HandleWebhook returned error: %v", err)
	}
	key := repo.updateKey(companyID, domain.ChatChannelTelegram, "evt-1")
	if !repo.waitStatus(key, "failed", time.Second) {
		t.Fatalf("expected failed status after transient error")
	}
	if chat.calls != 1 {
		t.Fatalf("expected one bot call after first (failing) delivery, got %d", chat.calls)
	}

	// Provider retry of the same update must be reprocessed, not dropped as duplicate.
	resp, err := gateway.HandleWebhook(context.Background(), companyID, domain.ChatChannelTelegram, WebhookRequest{})
	if err != nil {
		t.Fatalf("HandleWebhook retry returned error: %v", err)
	}
	if resp == nil || resp.Status != http.StatusOK {
		t.Fatalf("unexpected retry response: %#v", resp)
	}
	if !repo.waitStatus(key, "done", time.Second) {
		t.Fatalf("expected done status after successful retry")
	}
	if chat.calls != 2 {
		t.Fatalf("expected two bot calls (fail + retry), got %d", chat.calls)
	}
	if adapter.sendCalls != 1 || adapter.sent == nil || adapter.sent.Text != "Ответ" {
		t.Fatalf("expected reply delivered on retry, got %d sends %#v", adapter.sendCalls, adapter.sent)
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
	sessionID    uuid.UUID
	text         string
	returnedToID uuid.UUID
}

func (h *fakeHandoff) RecordInbound(_ context.Context, _ uuid.UUID, session *domain.ChatSession, content string) (*domain.ChatMessage, error) {
	h.sessionID = session.ID
	h.text = content
	return &domain.ChatMessage{SessionID: session.ID, Role: domain.ChatRoleUser, Content: content}, nil
}

func (h *fakeHandoff) ReturnToBot(_ context.Context, _ uuid.UUID, sessionID uuid.UUID) (*domain.ChatSession, error) {
	h.returnedToID = sessionID
	return &domain.ChatSession{BaseModel: domain.BaseModel{ID: sessionID}, State: domain.ChatStateBot}, nil
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
	mu       sync.Mutex
	sessions map[uuid.UUID]*domain.ChatSession
	updates  map[string]bool
	statuses map[string]string
}

// waitStatus blocks until the update reaches the wanted terminal status or the
// deadline passes. Locking the same mutex the async worker uses to set the
// status also publishes the worker's prior writes (chat/adapter counters), so
// tests can read them race-free once the terminal status is observed.
func (r *fakeRepo) waitStatus(key, want string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		got := r.statuses[key]
		r.mu.Unlock()
		if got == want {
			return true
		}
		time.Sleep(2 * time.Millisecond)
	}
	return false
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
	r.mu.Lock()
	defer r.mu.Unlock()
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
	r.mu.Lock()
	defer r.mu.Unlock()
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
	r.mu.Lock()
	defer r.mu.Unlock()
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
