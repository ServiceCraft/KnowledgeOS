package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	applog "github.com/knowledgeos/backend/internal/logger"
	"gorm.io/gorm"
)

type HandoffActor struct {
	UserID uuid.UUID
	Role   domain.Role
}

type HandoffSessionFilter struct {
	Page       int
	Limit      int
	State      *domain.ChatState
	OperatorID *uuid.UUID
	Channel    *domain.ChatChannel
	Active     bool
}

type SendOperatorMessageRequest struct {
	Content string `json:"content"`
}

type EscalateSessionRequest struct {
	Reason string `json:"reason"`
}

type handoffSettingsProvider interface {
	Get(ctx context.Context, companyID uuid.UUID) (*domain.BotSettings, error)
}

type handoffChatRepository interface {
	domain.ChatRepository
	TransitionSession(ctx context.Context, companyID, sessionID uuid.UUID, from []domain.ChatState, to domain.ChatState, operatorID *uuid.UUID) (*domain.ChatSession, error)
	SessionMetrics(ctx context.Context, companyID uuid.UUID, visibleTo *uuid.UUID) (*domain.ChatSessionMetrics, error)
}

type handoffOutbound interface {
	SendOperatorMessage(ctx context.Context, companyID uuid.UUID, session *domain.ChatSession, message *domain.ChatMessage) error
}

type handoffNotifier interface {
	NotifyHandoff(ctx context.Context, companyID uuid.UUID, session *domain.ChatSession, reason string, lastQuestion string) error
}

type HandoffService struct {
	chats    handoffChatRepository
	settings handoffSettingsProvider
	outbound handoffOutbound
	notifier handoffNotifier
}

func NewHandoffService(chats handoffChatRepository, settings handoffSettingsProvider, outbound handoffOutbound, notifier handoffNotifier) *HandoffService {
	return &HandoffService{chats: chats, settings: settings, outbound: outbound, notifier: notifier}
}

func (s *HandoffService) Enabled(ctx context.Context, companyID uuid.UUID) bool {
	settings, err := s.settings.Get(ctx, companyID)
	if err != nil {
		return false
	}
	return handoffEnabled(settings.EnabledModules)
}

func (s *HandoffService) ListSessions(ctx context.Context, companyID uuid.UUID, actor HandoffActor, filter HandoffSessionFilter) ([]domain.ChatSession, int64, error) {
	applog.TraceCall(ctx, "service.HandoffService.ListSessions")
	repoFilter := domain.ChatSessionFilter{
		Page:       filter.Page,
		Limit:      filter.Limit,
		State:      filter.State,
		OperatorID: filter.OperatorID,
		Channel:    filter.Channel,
		Active:     filter.Active,
	}
	if !isHandoffManager(actor.Role) {
		repoFilter.VisibleTo = &actor.UserID
	}
	return s.chats.ListSessions(ctx, companyID, repoFilter)
}

func (s *HandoffService) Metrics(ctx context.Context, companyID uuid.UUID, actor HandoffActor) (*domain.ChatSessionMetrics, error) {
	applog.TraceCall(ctx, "service.HandoffService.Metrics")
	var visibleTo *uuid.UUID
	if !isHandoffManager(actor.Role) {
		visibleTo = &actor.UserID
	}
	return s.chats.SessionMetrics(ctx, companyID, visibleTo)
}

func (s *HandoffService) GetSession(ctx context.Context, companyID uuid.UUID, actor HandoffActor, sessionID uuid.UUID) (*domain.ChatSessionWithMessages, error) {
	applog.TraceCall(ctx, "service.HandoffService.GetSession")
	session, err := s.chats.GetSession(ctx, companyID, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFound("chat session not found")
		}
		return nil, err
	}
	if err := authorizeHandoffRead(actor, session); err != nil {
		return nil, err
	}
	messages, err := s.chats.ListMessages(ctx, companyID, sessionID, 200)
	if err != nil {
		return nil, err
	}
	return &domain.ChatSessionWithMessages{Session: session, Messages: messages}, nil
}

func (s *HandoffService) Escalate(ctx context.Context, companyID, sessionID uuid.UUID, reason string) (*domain.ChatSession, error) {
	applog.TraceCall(ctx, "service.HandoffService.Escalate")
	if !s.Enabled(ctx, companyID) {
		session, err := s.chats.GetSession(ctx, companyID, sessionID)
		if err != nil {
			return nil, err
		}
		return session, nil
	}
	session, err := s.chats.TransitionSession(ctx, companyID, sessionID, []domain.ChatState{domain.ChatStateBot}, domain.ChatStateWaitingOperator, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			current, getErr := s.chats.GetSession(ctx, companyID, sessionID)
			if getErr == nil && (current.State == domain.ChatStateWaitingOperator || current.State == domain.ChatStateOperator) {
				return current, nil
			}
			return nil, conflict("chat session cannot be escalated from current state")
		}
		return nil, err
	}
	if s.notifier != nil {
		lastQuestion := latestUserQuestion(ctx, s.chats, companyID, sessionID)
		if err := s.notifier.NotifyHandoff(ctx, companyID, session, reason, lastQuestion); err != nil {
			applog.From(ctx).Warn().Err(err).Str("session_id", sessionID.String()).Msg("handoff notification failed")
		}
	}
	return session, nil
}

func (s *HandoffService) Claim(ctx context.Context, companyID uuid.UUID, actor HandoffActor, sessionID uuid.UUID) (*domain.ChatSession, error) {
	applog.TraceCall(ctx, "service.HandoffService.Claim")
	if actor.UserID == uuid.Nil {
		return nil, forbidden("operator identity is required")
	}
	session, err := s.chats.TransitionSession(ctx, companyID, sessionID, []domain.ChatState{domain.ChatStateWaitingOperator}, domain.ChatStateOperator, &actor.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, conflict("chat session is not waiting for operator")
		}
		return nil, err
	}
	return session, nil
}

func (s *HandoffService) SendOperatorMessage(ctx context.Context, companyID uuid.UUID, actor HandoffActor, sessionID uuid.UUID, req SendOperatorMessageRequest) (*domain.ChatMessage, error) {
	applog.TraceCall(ctx, "service.HandoffService.SendOperatorMessage")
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, badRequest("content is required")
	}
	session, err := s.chats.GetSession(ctx, companyID, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFound("chat session not found")
		}
		return nil, err
	}
	if session.State != domain.ChatStateOperator {
		return nil, conflict("chat session is not handled by operator")
	}
	if err := authorizeAssignedOperator(actor, session); err != nil {
		return nil, err
	}
	message := &domain.ChatMessage{
		SessionID: sessionID,
		Role:      domain.ChatRoleOperator,
		Content:   content,
	}
	// Persist only after outbound delivery succeeds, so failed sends do not
	// appear in the operator transcript as delivered replies.
	if s.outbound != nil {
		if err := s.outbound.SendOperatorMessage(ctx, companyID, session, message); err != nil {
			return nil, err
		}
	}
	if err := s.chats.AppendMessage(ctx, companyID, message); err != nil {
		return nil, err
	}
	return message, nil
}

func (s *HandoffService) RecordInbound(ctx context.Context, companyID uuid.UUID, session *domain.ChatSession, content string) (*domain.ChatMessage, error) {
	applog.TraceCall(ctx, "service.HandoffService.RecordInbound")
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, badRequest("content is required")
	}
	message := &domain.ChatMessage{
		SessionID: session.ID,
		Role:      domain.ChatRoleUser,
		Content:   content,
	}
	if err := s.chats.AppendMessage(ctx, companyID, message); err != nil {
		return nil, err
	}
	if session.State == domain.ChatStateWaitingOperator && s.notifier != nil {
		if err := s.notifier.NotifyHandoff(ctx, companyID, session, "new_user_message", content); err != nil {
			applog.From(ctx).Warn().Err(err).Str("session_id", session.ID.String()).Msg("handoff notification failed")
		}
	}
	return message, nil
}

func (s *HandoffService) ReleaseToBot(ctx context.Context, companyID uuid.UUID, actor HandoffActor, sessionID uuid.UUID) (*domain.ChatSession, error) {
	applog.TraceCall(ctx, "service.HandoffService.ReleaseToBot")
	session, err := s.chats.GetSession(ctx, companyID, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFound("chat session not found")
		}
		return nil, err
	}
	if err := authorizeAssignedOperator(actor, session); err != nil {
		return nil, err
	}
	return s.chats.TransitionSession(ctx, companyID, sessionID, []domain.ChatState{domain.ChatStateOperator}, domain.ChatStateBot, nil)
}

func (s *HandoffService) Close(ctx context.Context, companyID uuid.UUID, actor HandoffActor, sessionID uuid.UUID) (*domain.ChatSession, error) {
	applog.TraceCall(ctx, "service.HandoffService.Close")
	session, err := s.chats.GetSession(ctx, companyID, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFound("chat session not found")
		}
		return nil, err
	}
	if err := authorizeAssignedOperator(actor, session); err != nil {
		return nil, err
	}
	return s.chats.TransitionSession(ctx, companyID, sessionID, []domain.ChatState{domain.ChatStateWaitingOperator, domain.ChatStateOperator, domain.ChatStateBot}, domain.ChatStateClosed, nil)
}

func handoffEnabled(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var modules map[string]json.RawMessage
	if err := json.Unmarshal(raw, &modules); err != nil {
		return false
	}
	value, ok := modules["handoff"]
	if !ok || len(value) == 0 {
		return false
	}
	var enabled bool
	return json.Unmarshal(value, &enabled) == nil && enabled
}

func authorizeHandoffRead(actor HandoffActor, session *domain.ChatSession) error {
	if isHandoffManager(actor.Role) {
		return nil
	}
	if session.State == domain.ChatStateWaitingOperator {
		return nil
	}
	if session.OperatorID != nil && *session.OperatorID == actor.UserID {
		return nil
	}
	return forbidden("insufficient permissions for chat session")
}

func authorizeAssignedOperator(actor HandoffActor, session *domain.ChatSession) error {
	if isHandoffManager(actor.Role) {
		return nil
	}
	if session.OperatorID == nil || *session.OperatorID != actor.UserID {
		return forbidden("chat session is assigned to another operator")
	}
	return nil
}

func isHandoffManager(role domain.Role) bool {
	return role == domain.RoleAdmin || role == domain.RoleSuperadmin
}

func latestUserQuestion(ctx context.Context, chats domain.ChatRepository, companyID, sessionID uuid.UUID) string {
	messages, err := chats.ListMessages(ctx, companyID, sessionID, 20)
	if err != nil {
		return ""
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == domain.ChatRoleUser {
			return messages[i].Content
		}
	}
	return ""
}
