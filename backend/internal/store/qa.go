package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type QAStore struct {
	*Store
}

func NewQAStore(s *Store) *QAStore {
	return &QAStore{Store: s}
}

func (s *QAStore) List(ctx context.Context, companyID uuid.UUID, filter domain.QAPairFilter) ([]domain.QAPair, int64, error) {
	var items []domain.QAPair
	var total int64

	q := s.db.WithContext(ctx).Model(&domain.QAPair{}).Scopes(tenantScope(companyID))

	if filter.ThemeID != nil {
		q = q.Where("theme_id = ?", *filter.ThemeID)
	}
	if filter.IsFAQ != nil {
		q = q.Where("is_faq = ?", *filter.IsFAQ)
	}
	if filter.Query != "" {
		like := "%" + filter.Query + "%"
		q = q.Where("question ILIKE ? OR answer ILIKE ?", like, like)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := q.Scopes(paginate(filter.Page, filter.Limit)).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *QAStore) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.QAPair, error) {
	var item domain.QAPair
	if err := s.db.WithContext(ctx).Scopes(tenantScope(companyID)).Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *QAStore) Create(ctx context.Context, companyID uuid.UUID, qa *domain.QAPair) error {
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

func (s *QAStore) Update(ctx context.Context, companyID uuid.UUID, qa *domain.QAPair) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		qa.SyncVersion = seq
		qa.SyncOrigin = s.origin
		return tx.Model(qa).Scopes(tenantScope(companyID)).Where("id = ?", qa.ID).Updates(qa).Error
	})
}

func (s *QAStore) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		return tx.Model(&domain.QAPair{}).Scopes(tenantScope(companyID)).Where("id = ?", id).
			Updates(map[string]interface{}{"sync_version": seq, "sync_origin": s.origin, "deleted_at": gorm.Expr("now()")}).Error
	})
}

func (s *QAStore) ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]domain.QAPair, error) {
	var items []domain.QAPair
	if err := s.db.WithContext(ctx).Unscoped().Scopes(tenantScope(companyID)).
		Where("sync_version > ?", sinceVersion).
		Order("sync_version ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *QAStore) ApplyRemote(ctx context.Context, companyID uuid.UUID, qa *domain.QAPair) error {
	qa.SyncOrigin = "cloud"
	qa.CompanyID = companyID
	return s.db.WithContext(ctx).Where("id = ? AND company_id = ?", qa.ID, companyID).
		Assign(qa).FirstOrCreate(qa).Error
}
