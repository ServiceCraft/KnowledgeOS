package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type LinkStore struct {
	*Store
}

func NewLinkStore(s *Store) *LinkStore {
	return &LinkStore{Store: s}
}

func (s *LinkStore) List(ctx context.Context, companyID uuid.UUID, sourceType string, sourceID uuid.UUID, filter domain.EntityLinkFilter) ([]domain.EntityLink, int64, error) {
	var items []domain.EntityLink
	var total int64

	q := s.db.WithContext(ctx).Model(&domain.EntityLink{}).Scopes(tenantScope(companyID)).
		Where("source_type = ? AND source_id = ?", sourceType, sourceID)

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Scopes(paginate(filter.Page, filter.Limit)).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *LinkStore) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.EntityLink, error) {
	var item domain.EntityLink
	if err := s.db.WithContext(ctx).Scopes(tenantScope(companyID)).Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *LinkStore) Create(ctx context.Context, companyID uuid.UUID, link *domain.EntityLink) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		link.SyncVersion = seq
		link.SyncOrigin = s.origin
		link.CompanyID = companyID
		return tx.Create(link).Error
	})
}

func (s *LinkStore) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		return tx.Model(&domain.EntityLink{}).Scopes(tenantScope(companyID)).Where("id = ?", id).
			Updates(map[string]interface{}{"sync_version": seq, "sync_origin": s.origin, "deleted_at": gorm.Expr("now()")}).Error
	})
}

func (s *LinkStore) ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]domain.EntityLink, error) {
	var items []domain.EntityLink
	if err := s.db.WithContext(ctx).Unscoped().Scopes(tenantScope(companyID)).
		Where("sync_version > ?", sinceVersion).
		Order("sync_version ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *LinkStore) ApplyRemote(ctx context.Context, companyID uuid.UUID, link *domain.EntityLink) error {
	link.SyncOrigin = "cloud"
	link.CompanyID = companyID
	return s.db.WithContext(ctx).Where("id = ? AND company_id = ?", link.ID, companyID).
		Assign(link).FirstOrCreate(link).Error
}
