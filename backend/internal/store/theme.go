package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type ThemeStore struct {
	*Store
}

func NewThemeStore(s *Store) *ThemeStore {
	return &ThemeStore{Store: s}
}

func (s *ThemeStore) List(ctx context.Context, companyID uuid.UUID, filter domain.ThemeFilter) ([]domain.Theme, int64, error) {
	var items []domain.Theme
	var total int64

	q := s.db.WithContext(ctx).Model(&domain.Theme{}).Scopes(tenantScope(companyID))

	if filter.Query != "" {
		q = q.Where("name ILIKE ?", "%"+filter.Query+"%")
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Scopes(paginate(filter.Page, filter.Limit)).Order("name ASC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *ThemeStore) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.Theme, error) {
	var item domain.Theme
	if err := s.db.WithContext(ctx).Scopes(tenantScope(companyID)).Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *ThemeStore) Create(ctx context.Context, companyID uuid.UUID, theme *domain.Theme) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		theme.SyncVersion = seq
		theme.SyncOrigin = s.origin
		theme.CompanyID = companyID
		return tx.Create(theme).Error
	})
}

func (s *ThemeStore) Update(ctx context.Context, companyID uuid.UUID, theme *domain.Theme) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		theme.SyncVersion = seq
		theme.SyncOrigin = s.origin
		return tx.Model(theme).Scopes(tenantScope(companyID)).Where("id = ?", theme.ID).Updates(theme).Error
	})
}

func (s *ThemeStore) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		return tx.Model(&domain.Theme{}).Scopes(tenantScope(companyID)).Where("id = ?", id).
			Updates(map[string]interface{}{"sync_version": seq, "sync_origin": s.origin, "deleted_at": gorm.Expr("now()")}).Error
	})
}

func (s *ThemeStore) ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]domain.Theme, error) {
	var items []domain.Theme
	if err := s.db.WithContext(ctx).Unscoped().Scopes(tenantScope(companyID)).
		Where("sync_version > ?", sinceVersion).
		Order("sync_version ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *ThemeStore) ApplyRemote(ctx context.Context, companyID uuid.UUID, theme *domain.Theme) error {
	theme.SyncOrigin = "cloud"
	theme.CompanyID = companyID
	return s.db.WithContext(ctx).Where("id = ? AND company_id = ?", theme.ID, companyID).
		Assign(theme).FirstOrCreate(theme).Error
}
