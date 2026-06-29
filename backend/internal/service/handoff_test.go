package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

func TestHandoffServiceOperatorFlow(t *testing.T) {
	ctx := context.Background()
	companyID := uuid.New()
	sessionID := uuid.New()
	operatorID := uuid.New()
	repo := newFakeChatRepo(companyID, sessionID)
	settings := &fakeChatSettings{settings: &domain.BotSettings{
		Enabled:        true,
		EnabledModules: json.RawMessage(`{"handoff":true}`),
	}}
	outbound := &fakeHandoffOutbound{}
	svc := NewHandoffService(repo, settings, outbound, nil)

	escalated, err := svc.Escalate(ctx, companyID, sessionID, "no_context")
	if err != nil {
		t.Fatalf("Escalate() error = %v", err)
	}
	if escalated.State != domain.ChatStateWaitingOperator {
		t.Fatalf("state after escalate = %s, want waiting_operator", escalated.State)
	}

	claimed, err := svc.Claim(ctx, companyID, HandoffActor{UserID: operatorID, Role: domain.RoleViewer}, sessionID)
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if claimed.State != domain.ChatStateOperator || claimed.OperatorID == nil || *claimed.OperatorID != operatorID {
		t.Fatalf("claimed session = %+v", claimed)
	}

	message, err := svc.SendOperatorMessage(ctx, companyID, HandoffActor{UserID: operatorID, Role: domain.RoleViewer}, sessionID, SendOperatorMessageRequest{Content: "Здравствуйте, я оператор"})
	if err != nil {
		t.Fatalf("SendOperatorMessage() error = %v", err)
	}
	if message.Role != domain.ChatRoleOperator {
		t.Fatalf("message role = %s, want operator", message.Role)
	}
	if outbound.sent == nil || outbound.sent.Content != message.Content {
		t.Fatalf("outbound message = %+v", outbound.sent)
	}

	released, err := svc.ReleaseToBot(ctx, companyID, HandoffActor{UserID: operatorID, Role: domain.RoleViewer}, sessionID)
	if err != nil {
		t.Fatalf("ReleaseToBot() error = %v", err)
	}
	if released.State != domain.ChatStateBot || released.OperatorID != nil {
		t.Fatalf("released session = %+v", released)
	}
}

func TestHandoffServiceClaimIsAtomic(t *testing.T) {
	ctx := context.Background()
	companyID := uuid.New()
	sessionID := uuid.New()
	repo := newFakeChatRepo(companyID, sessionID)
	settings := &fakeChatSettings{settings: &domain.BotSettings{
		Enabled:        true,
		EnabledModules: json.RawMessage(`{"handoff":true}`),
	}}
	svc := NewHandoffService(repo, settings, nil, nil)
	if _, err := svc.Escalate(ctx, companyID, sessionID, "manual"); err != nil {
		t.Fatalf("Escalate() error = %v", err)
	}
	if _, err := svc.Claim(ctx, companyID, HandoffActor{UserID: uuid.New(), Role: domain.RoleViewer}, sessionID); err != nil {
		t.Fatalf("first Claim() error = %v", err)
	}
	if _, err := svc.Claim(ctx, companyID, HandoffActor{UserID: uuid.New(), Role: domain.RoleViewer}, sessionID); err == nil || HTTPStatus(err) != 409 {
		t.Fatalf("second Claim() error = %v, want 409", err)
	}
}

type fakeHandoffOutbound struct {
	sent *domain.ChatMessage
}

func (o *fakeHandoffOutbound) SendOperatorMessage(_ context.Context, _ uuid.UUID, _ *domain.ChatSession, message *domain.ChatMessage) error {
	copy := *message
	o.sent = &copy
	return nil
}

func (r *fakeChatRepo) TransitionSession(_ context.Context, companyID, sessionID uuid.UUID, from []domain.ChatState, to domain.ChatState, operatorID *uuid.UUID) (*domain.ChatSession, error) {
	session := r.sessions[sessionID]
	if session == nil || session.CompanyID != companyID {
		return nil, gorm.ErrRecordNotFound
	}
	allowed := len(from) == 0
	for _, state := range from {
		if session.State == state {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, gorm.ErrRecordNotFound
	}
	session.State = to
	session.OperatorID = operatorID
	copy := *session
	r.sessions[sessionID] = &copy
	return &copy, nil
}

func (r *fakeChatRepo) SessionMetrics(_ context.Context, companyID uuid.UUID, visibleTo *uuid.UUID) (*domain.ChatSessionMetrics, error) {
	out := &domain.ChatSessionMetrics{ByChannel: map[domain.ChatChannel]int64{}}
	for _, session := range r.sessions {
		if session.CompanyID != companyID {
			continue
		}
		if visibleTo != nil && session.State != domain.ChatStateWaitingOperator && (session.OperatorID == nil || *session.OperatorID != *visibleTo) {
			continue
		}
		out.Total++
		out.ByChannel[session.Channel]++
		switch session.State {
		case domain.ChatStateBot:
			out.Bot++
		case domain.ChatStateWaitingOperator:
			out.WaitingOperator++
		case domain.ChatStateOperator:
			out.Operator++
		case domain.ChatStateClosed:
			out.Closed++
		}
	}
	return out, nil
}

func TestHandoffServiceViewerCannotWriteAnotherOperatorsSession(t *testing.T) {
	ctx := context.Background()
	companyID := uuid.New()
	sessionID := uuid.New()
	operatorID := uuid.New()
	repo := newFakeChatRepo(companyID, sessionID)
	repo.sessions[sessionID].State = domain.ChatStateOperator
	repo.sessions[sessionID].OperatorID = &operatorID
	svc := NewHandoffService(repo, &fakeChatSettings{settings: &domain.BotSettings{EnabledModules: json.RawMessage(`{"handoff":true}`)}}, nil, nil)

	_, err := svc.SendOperatorMessage(ctx, companyID, HandoffActor{UserID: uuid.New(), Role: domain.RoleViewer}, sessionID, SendOperatorMessageRequest{Content: "reply"})
	if err == nil || HTTPStatus(err) != 403 {
		t.Fatalf("SendOperatorMessage() error = %v, want 403", err)
	}
}
