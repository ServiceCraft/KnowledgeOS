package store

import (
	"context"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type QAStore struct {
	*Store
}

// NewQAStore executes the store.NewQAStore operation.
func NewQAStore(s *Store) *QAStore {
	return &QAStore{Store: s}
}

// List executes the store.QAStore.List operation.
func (s *QAStore) List(ctx context.Context, companyID uuid.UUID, filter domain.QAPairFilter) ([]domain.QAPair, int64, error) {
	applog.TraceCall(ctx, "store.QAStore.List")
	var items []domain.QAPair
	var total int64

	q := s.db.WithContext(ctx).Model(&domain.QAPair{}).Scopes(tenantScope(companyID))

	if filter.ThemeID != nil {
		q = q.Where("theme_id = ?", *filter.ThemeID)
	}
	if filter.IsFAQ != nil {
		q = q.Where("is_faq = ?", *filter.IsFAQ)
	}
	if filter.AIStatus != nil {
		q = q.Where("ai_status = ?", *filter.AIStatus)
	}
	if filter.Query != "" {
		like := "%" + filter.Query + "%"
		q = q.Where("question ILIKE ? OR answer ILIKE ?", like, like)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderClause := "created_at DESC"
	switch filter.Sort {
	case "frequency":
		orderClause = "frequency ASC, created_at DESC"
	case "-frequency":
		orderClause = "frequency DESC, created_at DESC"
	case "created_at":
		orderClause = "created_at ASC"
	case "-created_at":
		orderClause = "created_at DESC"
	}

	if err := q.Scopes(paginate(filter.Page, filter.Limit)).Order(orderClause).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID executes the store.QAStore.GetByID operation.
func (s *QAStore) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.QAPair, error) {
	applog.TraceCall(ctx, "store.QAStore.GetByID")
	var item domain.QAPair
	if err := s.db.WithContext(ctx).Scopes(tenantScope(companyID)).Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// Create executes the store.QAStore.Create operation.
func (s *QAStore) Create(ctx context.Context, companyID uuid.UUID, qa *domain.QAPair) error {
	applog.TraceCall(ctx, "store.QAStore.Create")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		qa.SyncVersion = seq
		qa.SyncOrigin = s.origin
		qa.CompanyID = companyID
		return tx.Create(qa).Error
	})
}

// Update executes the store.QAStore.Update operation.
func (s *QAStore) Update(ctx context.Context, companyID uuid.UUID, qa *domain.QAPair) error {
	applog.TraceCall(ctx, "store.QAStore.Update")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		qa.SyncVersion = seq
		qa.SyncOrigin = s.origin
		return tx.Model(qa).Scopes(tenantScope(companyID)).Where("id = ?", qa.ID).
			Select("question", "answer", "is_faq", "is_locked", "frequency", "theme_id", "ai_answer", "ai_status", "ai_reviewed_by", "ai_reviewed_at", "sync_version", "sync_origin", "updated_by").
			Updates(qa).Error
	})
}

// Delete executes the store.QAStore.Delete operation.
func (s *QAStore) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	applog.TraceCall(ctx, "store.QAStore.Delete")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		return tx.Model(&domain.QAPair{}).Scopes(tenantScope(companyID)).Where("id = ?", id).
			Updates(map[string]interface{}{"sync_version": seq, "sync_origin": s.origin, "deleted_at": gorm.Expr("now()")}).Error
	})
}

// ListSince executes the store.QAStore.ListSince operation.
func (s *QAStore) ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]domain.QAPair, error) {
	applog.TraceCall(ctx, "store.QAStore.ListSince")
	var items []domain.QAPair
	if err := s.db.WithContext(ctx).Unscoped().Scopes(tenantScope(companyID)).
		Where("sync_version > ?", sinceVersion).
		Order("sync_version ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ApplyRemote executes the store.QAStore.ApplyRemote operation.
func (s *QAStore) ApplyRemote(ctx context.Context, companyID uuid.UUID, qa *domain.QAPair) error {
	applog.TraceCall(ctx, "store.QAStore.ApplyRemote")
	qa.SyncOrigin = "cloud"
	qa.CompanyID = companyID
	return s.db.WithContext(ctx).Where("id = ? AND company_id = ?", qa.ID, companyID).
		Assign(qa).FirstOrCreate(qa).Error
}
