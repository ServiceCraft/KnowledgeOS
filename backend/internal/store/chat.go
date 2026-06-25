package store

import (
	"context"
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

// UpdateSession executes the store.ChatStore.UpdateSession operation.
func (s *ChatStore) UpdateSession(ctx context.Context, companyID uuid.UUID, session *domain.ChatSession) error {
	applog.TraceCall(ctx, "store.ChatStore.UpdateSession")
	session.CompanyID = companyID
	return s.db.WithContext(ctx).
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
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(message).Error; err != nil {
			return err
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
