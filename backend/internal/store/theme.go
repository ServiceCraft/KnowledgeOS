package store

import (
	"context"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type ThemeStore struct {
	*Store
}

// NewThemeStore executes the store.NewThemeStore operation.
func NewThemeStore(s *Store) *ThemeStore {
	return &ThemeStore{Store: s}
}

// List executes the store.ThemeStore.List operation.
func (s *ThemeStore) List(ctx context.Context, companyID uuid.UUID, filter domain.ThemeFilter) ([]domain.Theme, int64, error) {
	applog.TraceCall(ctx, "store.ThemeStore.List")
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

// GetByID executes the store.ThemeStore.GetByID operation.
func (s *ThemeStore) GetByID(ctx context.Context, companyID uuid.UUID, id uuid.UUID) (*domain.Theme, error) {
	applog.TraceCall(ctx, "store.ThemeStore.GetByID")
	var item domain.Theme
	if err := s.db.WithContext(ctx).Scopes(tenantScope(companyID)).Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// Create executes the store.ThemeStore.Create operation.
func (s *ThemeStore) Create(ctx context.Context, companyID uuid.UUID, theme *domain.Theme) error {
	applog.TraceCall(ctx, "store.ThemeStore.Create")
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

// Update executes the store.ThemeStore.Update operation.
func (s *ThemeStore) Update(ctx context.Context, companyID uuid.UUID, theme *domain.Theme) error {
	applog.TraceCall(ctx, "store.ThemeStore.Update")
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

// Delete executes the store.ThemeStore.Delete operation.
func (s *ThemeStore) Delete(ctx context.Context, companyID uuid.UUID, id uuid.UUID) error {
	applog.TraceCall(ctx, "store.ThemeStore.Delete")
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var seq int64
		if err := tx.Raw("UPDATE sync_sequence SET current_seq = current_seq + 1 WHERE company_id = ? RETURNING current_seq", companyID).Scan(&seq).Error; err != nil {
			return err
		}
		return tx.Model(&domain.Theme{}).Scopes(tenantScope(companyID)).Where("id = ?", id).
			Updates(map[string]interface{}{"sync_version": seq, "sync_origin": s.origin, "deleted_at": gorm.Expr("now()")}).Error
	})
}

// ListSince executes the store.ThemeStore.ListSince operation.
func (s *ThemeStore) ListSince(ctx context.Context, companyID uuid.UUID, sinceVersion int64) ([]domain.Theme, error) {
	applog.TraceCall(ctx, "store.ThemeStore.ListSince")
	var items []domain.Theme
	if err := s.db.WithContext(ctx).Unscoped().Scopes(tenantScope(companyID)).
		Where("sync_version > ?", sinceVersion).
		Order("sync_version ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ApplyRemote executes the store.ThemeStore.ApplyRemote operation.
func (s *ThemeStore) ApplyRemote(ctx context.Context, companyID uuid.UUID, theme *domain.Theme) error {
	applog.TraceCall(ctx, "store.ThemeStore.ApplyRemote")
	theme.SyncOrigin = "cloud"
	theme.CompanyID = companyID
	return s.db.WithContext(ctx).Where("id = ? AND company_id = ?", theme.ID, companyID).
		Assign(theme).FirstOrCreate(theme).Error
}
