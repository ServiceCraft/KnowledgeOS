package store

import (
	"context"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type CompanyStore struct {
	*Store
}

// NewCompanyStore executes the store.NewCompanyStore operation.
func NewCompanyStore(s *Store) *CompanyStore {
	return &CompanyStore{Store: s}
}

// List executes the store.CompanyStore.List operation.
func (s *CompanyStore) List(ctx context.Context, filter domain.CompanyFilter) ([]domain.Company, int64, error) {
	applog.TraceCall(ctx, "store.CompanyStore.List")
	var items []domain.Company
	var total int64

	q := s.db.WithContext(ctx).Model(&domain.Company{})

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Scopes(paginate(filter.Page, filter.Limit)).Order("name ASC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID executes the store.CompanyStore.GetByID operation.
func (s *CompanyStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	applog.TraceCall(ctx, "store.CompanyStore.GetByID")
	var company domain.Company
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&company).Error; err != nil {
		return nil, mapErr(err)
	}
	return &company, nil
}

// Exists reports whether a company with the given ID exists.
func (s *CompanyStore) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	applog.TraceCall(ctx, "store.CompanyStore.Exists")
	var exists bool
	err := s.db.WithContext(ctx).Raw(
		`SELECT EXISTS(SELECT 1 FROM companies WHERE id = ?)`, id,
	).Scan(&exists).Error
	return exists, err
}

// Create executes the store.CompanyStore.Create operation.
func (s *CompanyStore) Create(ctx context.Context, company *domain.Company) error {
	applog.TraceCall(ctx, "store.CompanyStore.Create")
	return s.db.WithContext(ctx).Create(company).Error
}

// Update executes the store.CompanyStore.Update operation.
func (s *CompanyStore) Update(ctx context.Context, company *domain.Company) error {
	applog.TraceCall(ctx, "store.CompanyStore.Update")
	return s.db.WithContext(ctx).Save(company).Error
}

// Delete executes the store.CompanyStore.Delete operation.
func (s *CompanyStore) Delete(ctx context.Context, id uuid.UUID) error {
	applog.TraceCall(ctx, "store.CompanyStore.Delete")
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Company{}).Error
}

// Count executes the store.CompanyStore.Count operation.
func (s *CompanyStore) Count(ctx context.Context) (int64, error) {
	applog.TraceCall(ctx, "store.CompanyStore.Count")
	var count int64
	if err := s.db.WithContext(ctx).Model(&domain.Company{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
