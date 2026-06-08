package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type QACallMentionStore struct {
	*Store
}

func NewQACallMentionStore(s *Store) *QACallMentionStore {
	return &QACallMentionStore{Store: s}
}

func (s *QACallMentionStore) ListByQA(ctx context.Context, companyID, qaID uuid.UUID, filter domain.QACallMentionFilter) ([]domain.QAPairCallMentionView, int64, error) {
	var rows []domain.QAPairCallMentionView
	var total int64

	base := s.db.WithContext(ctx).
		Table("qa_pair_call_mentions AS m").
		Joins("JOIN calls c ON c.id = m.call_id AND c.deleted_at IS NULL").
		Where("m.company_id = ? AND m.qa_pair_id = ? AND m.deleted_at IS NULL", companyID, qaID)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.Select("m.*, c.title AS call_title, c.occurred_at AS call_occurred_at").
		Order("c.occurred_at DESC NULLS LAST, m.created_at DESC").
		Scopes(paginate(filter.Page, filter.Limit)).
		Scan(&rows).Error
	return rows, total, err
}

func (s *QACallMentionStore) Create(ctx context.Context, companyID uuid.UUID, m *domain.QAPairCallMention) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		m.SyncVersion = seq
		m.SyncOrigin = s.origin
		m.CompanyID = companyID
		return tx.Create(m).Error
	})
}

func (s *QACallMentionStore) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		return tx.Model(&domain.QAPairCallMention{}).Scopes(tenantScope(companyID)).Where("id = ?", id).
			Updates(map[string]interface{}{"sync_version": seq, "sync_origin": s.origin, "deleted_at": gorm.Expr("now()")}).Error
	})
}

func (s *QACallMentionStore) ApplyRemote(ctx context.Context, companyID uuid.UUID, m *domain.QAPairCallMention) error {
	m.CompanyID = companyID
	if m.SyncOrigin == "" {
		m.SyncOrigin = "cloud"
	}
	return s.db.WithContext(ctx).Where("id = ? AND company_id = ?", m.ID, companyID).
		Assign(m).FirstOrCreate(m).Error
}

func (s *QACallMentionStore) ListAll(ctx context.Context, companyID uuid.UUID) ([]domain.QAPairCallMention, error) {
	var items []domain.QAPairCallMention
	if err := s.db.WithContext(ctx).Scopes(tenantScope(companyID)).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
