package store

import (
	"context"
	"strings"

	applog "github.com/knowledgeos/backend/internal/logger"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type ChatStore struct {
	*Store
}

// NewChatStore executes the store.NewChatStore operation.
func NewChatStore(s *Store) *ChatStore {
	return &ChatStore{Store: s}
}

// CreateSession executes the store.ChatStore.CreateSession operation.
func (s *ChatStore) CreateSession(ctx context.Context, companyID uuid.UUID, session *domain.ChatSession) error {
	applog.TraceCall(ctx, "store.ChatStore.CreateSession")
	session.CompanyID = companyID
	if session.Channel == "" {
		session.Channel = domain.ChatChannelPlayground
	}
	if session.State == "" {
		session.State = domain.ChatStateBot
	}
	return s.db.WithContext(ctx).Create(session).Error
}

// ListSessions executes the store.ChatStore.ListSessions operation.
func (s *ChatStore) ListSessions(ctx context.Context, companyID uuid.UUID, filter domain.ChatSessionFilter) ([]domain.ChatSession, int64, error) {
	applog.TraceCall(ctx, "store.ChatStore.ListSessions")
	var items []domain.ChatSession
	q := s.db.WithContext(ctx).Model(&domain.ChatSession{}).Scopes(tenantScope(companyID))
	if filter.State != nil {
		q = q.Where("state = ?", *filter.State)
	}
	if filter.Active {
		q = q.Where("state IN ?", []domain.ChatState{domain.ChatStateWaitingOperator, domain.ChatStateOperator})
	}
	if filter.OperatorID != nil {
		q = q.Where("operator_id = ?", *filter.OperatorID)
	}
	if filter.Channel != nil {
		q = q.Where("channel = ?", *filter.Channel)
	}
	if filter.VisibleTo != nil {
		q = q.Where("(state = ? OR operator_id = ?)", domain.ChatStateWaitingOperator, *filter.VisibleTo)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("last_message_at DESC NULLS LAST, created_at DESC").
		Scopes(paginate(filter.Page, filter.Limit)).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *ChatStore) SessionMetrics(ctx context.Context, companyID uuid.UUID, visibleTo *uuid.UUID) (*domain.ChatSessionMetrics, error) {
	applog.TraceCall(ctx, "store.ChatStore.SessionMetrics")
	base := s.db.WithContext(ctx).Model(&domain.ChatSession{}).Scopes(tenantScope(companyID))
	if visibleTo != nil {
		base = base.Where("(state = ? OR operator_id = ?)", domain.ChatStateWaitingOperator, *visibleTo)
	}

	out := &domain.ChatSessionMetrics{ByChannel: map[domain.ChatChannel]int64{}}
	type stateRow struct {
		State domain.ChatState
		Count int64
	}
	var stateRows []stateRow
	if err := base.Session(&gorm.Session{}).
		Select("state, count(*) as count").
		Group("state").
		Scan(&stateRows).Error; err != nil {
		return nil, err
	}
	for _, row := range stateRows {
		out.Total += row.Count
		switch row.State {
		case domain.ChatStateBot:
			out.Bot = row.Count
		case domain.ChatStateWaitingOperator:
			out.WaitingOperator = row.Count
		case domain.ChatStateOperator:
			out.Operator = row.Count
		case domain.ChatStateClosed:
			out.Closed = row.Count
		}
	}

	type channelRow struct {
		Channel domain.ChatChannel
		Count   int64
	}
	var channelRows []channelRow
	if err := base.Session(&gorm.Session{}).
		Where("state IN ?", []domain.ChatState{domain.ChatStateWaitingOperator, domain.ChatStateOperator}).
		Select("channel, count(*) as count").
		Group("channel").
		Scan(&channelRows).Error; err != nil {
		return nil, err
	}
	for _, row := range channelRows {
		out.ByChannel[row.Channel] = row.Count
	}
	return out, nil
}

// GetSession executes the store.ChatStore.GetSession operation.
func (s *ChatStore) GetSession(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.ChatSession, error) {
	applog.TraceCall(ctx, "store.ChatStore.GetSession")
	var session domain.ChatSession
	if err := s.db.WithContext(ctx).
		Scopes(tenantScope(companyID)).
		Where("id = ?", id).
		First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// GetSessionByExternal executes the store.ChatStore.GetSessionByExternal operation.
func (s *ChatStore) GetSessionByExternal(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel, externalChatID string) (*domain.ChatSession, error) {
	applog.TraceCall(ctx, "store.ChatStore.GetSessionByExternal")
	var session domain.ChatSession
	if err := s.db.WithContext(ctx).
		Scopes(tenantScope(companyID)).
		Where("channel = ? AND external_chat_id = ? AND state <> ?", channel, externalChatID, domain.ChatStateClosed).
		Order("created_at DESC").
		First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// staleWebhookProcessingTTL bounds how long a webhook event may stay in the
// 'processing' state before another delivery is allowed to reclaim it. It guards
// against a crashed worker leaving an event stuck forever.
const staleWebhookProcessingTTL = 5 * time.Minute

// StartChannelUpdate claims a channel webhook update for processing. It returns
// true when the caller should process the update (new event, previously failed,
// or a stale 'processing' event) and false when the update is a duplicate that
// is already done or actively being processed. An empty updateID cannot be
// deduplicated, so it is always processed.
func (s *ChatStore) StartChannelUpdate(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel, updateID string) (bool, error) {
	applog.TraceCall(ctx, "store.ChatStore.StartChannelUpdate")
	updateID = strings.TrimSpace(updateID)
	if updateID == "" {
		return true, nil
	}
	staleCutoff := time.Now().Add(-staleWebhookProcessingTTL)
	result := s.db.WithContext(ctx).Exec(
		`INSERT INTO channel_webhook_events (company_id, channel, update_id, status, attempts, updated_at)
		 VALUES (?, ?, ?, 'processing', 1, now())
		 ON CONFLICT (company_id, channel, update_id) DO UPDATE SET
		     status = 'processing',
		     attempts = channel_webhook_events.attempts + 1,
		     last_error = NULL,
		     updated_at = now()
		 WHERE channel_webhook_events.status = 'failed'
		    OR (channel_webhook_events.status = 'processing' AND channel_webhook_events.updated_at < ?)`,
		companyID, channel, updateID, staleCutoff,
	)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// MarkChannelUpdateDone marks a claimed webhook update as fully processed.
func (s *ChatStore) MarkChannelUpdateDone(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel, updateID string) error {
	applog.TraceCall(ctx, "store.ChatStore.MarkChannelUpdateDone")
	updateID = strings.TrimSpace(updateID)
	if updateID == "" {
		return nil
	}
	return s.db.WithContext(ctx).Exec(
		`UPDATE channel_webhook_events
		 SET status = 'done', processed_at = now(), last_error = NULL, updated_at = now()
		 WHERE company_id = ? AND channel = ? AND update_id = ?`,
		companyID, channel, updateID,
	).Error
}

// MarkChannelUpdateFailed records that processing failed so a later delivery can
// reclaim the update instead of treating it as a processed duplicate.
func (s *ChatStore) MarkChannelUpdateFailed(ctx context.Context, companyID uuid.UUID, channel domain.ChatChannel, updateID string, errMsg string) error {
	applog.TraceCall(ctx, "store.ChatStore.MarkChannelUpdateFailed")
	updateID = strings.TrimSpace(updateID)
	if updateID == "" {
		return nil
	}
	if len(errMsg) > 1000 {
		errMsg = errMsg[:1000]
	}
	return s.db.WithContext(ctx).Exec(
		`UPDATE channel_webhook_events
		 SET status = 'failed', last_error = ?, updated_at = now()
		 WHERE company_id = ? AND channel = ? AND update_id = ?`,
		errMsg, companyID, channel, updateID,
	).Error
}

// UpdateSession executes the store.ChatStore.UpdateSession operation.
func (s *ChatStore) UpdateSession(ctx context.Context, companyID uuid.UUID, session *domain.ChatSession) error {
	applog.TraceCall(ctx, "store.ChatStore.UpdateSession")
	session.CompanyID = companyID
	return s.db.WithContext(ctx).
		Model(&domain.ChatSession{}).
		Scopes(tenantScope(companyID)).
		Where("id = ?", session.ID).
		Updates(map[string]interface{}{
			"channel":          session.Channel,
			"external_chat_id": session.ExternalChatID,
			"state":            session.State,
			"operator_id":      session.OperatorID,
			"title":            session.Title,
			"last_message_at":  session.LastMessageAt,
		}).Error
}

// DeleteSession permanently removes a session and (via the ON DELETE CASCADE on
// chat_messages.session_id) all of its messages. Scoped to the company.
func (s *ChatStore) DeleteSession(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	applog.TraceCall(ctx, "store.ChatStore.DeleteSession")
	res := s.db.WithContext(ctx).
		Scopes(tenantScope(companyID)).
		Where("id = ?", id).
		Delete(&domain.ChatSession{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// TransitionSession atomically changes a session state when its current state
// is one of the allowed source states.
func (s *ChatStore) TransitionSession(ctx context.Context, companyID, sessionID uuid.UUID, from []domain.ChatState, to domain.ChatState, operatorID *uuid.UUID) (*domain.ChatSession, error) {
	applog.TraceCall(ctx, "store.ChatStore.TransitionSession")
	if len(from) == 0 {
		from = []domain.ChatState{domain.ChatStateBot, domain.ChatStateWaitingOperator, domain.ChatStateOperator}
	}
	var session domain.ChatSession
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"state":       to,
			"operator_id": operatorID,
		}
		result := tx.Model(&domain.ChatSession{}).
			Where("company_id = ? AND id = ? AND state IN ?", companyID, sessionID, from).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.
			Scopes(tenantScope(companyID)).
			Where("id = ?", sessionID).
			First(&session).Error
	})
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// AppendMessage executes the store.ChatStore.AppendMessage operation.
func (s *ChatStore) AppendMessage(ctx context.Context, companyID uuid.UUID, message *domain.ChatMessage) error {
	applog.TraceCall(ctx, "store.ChatStore.AppendMessage")
	message.CompanyID = companyID
	if len(message.ToolCalls) == 0 {
		message.ToolCalls = []byte("[]")
	}
	if len(message.Sources) == 0 {
		message.Sources = []byte("[]")
	}
	if len(message.CitedSourceIDs) == 0 {
		message.CitedSourceIDs = []byte("[]")
	}
	if message.GuardrailAction == "" {
		message.GuardrailAction = domain.GuardrailActionAnswer
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(message).Error; err != nil {
			return applog.TraceErr(ctx, "chat: insert message", err)
		}
		now := time.Now()
		updates := map[string]interface{}{"last_message_at": now}
		if message.Role == domain.ChatRoleUser {
			title := message.Content
			if len([]rune(title)) > 80 {
				title = string([]rune(title)[:80])
			}
			updates["title"] = title
		}
		return tx.Model(&domain.ChatSession{}).
			Where("company_id = ? AND id = ?", companyID, message.SessionID).
			Updates(updates).Error
	})
}

// ListMessages executes the store.ChatStore.ListMessages operation.
func (s *ChatStore) ListMessages(ctx context.Context, companyID uuid.UUID, sessionID uuid.UUID, limit int) ([]domain.ChatMessage, error) {
	applog.TraceCall(ctx, "store.ChatStore.ListMessages")
	if limit <= 0 {
		limit = 100
	}
	var items []domain.ChatMessage
	if err := s.db.WithContext(ctx).
		Where("company_id = ? AND session_id = ?", companyID, sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	return items, nil
}
