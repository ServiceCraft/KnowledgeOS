package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

type CompanyStore struct {
	*Store
}

func NewCompanyStore(s *Store) *CompanyStore {
	return &CompanyStore{Store: s}
}

func (s *CompanyStore) List(ctx context.Context, filter domain.CompanyFilter) ([]domain.Company, int64, error) {
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

func (s *CompanyStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	var company domain.Company
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&company).Error; err != nil {
		return nil, err
	}
	return &company, nil
}

func (s *CompanyStore) Create(ctx context.Context, company *domain.Company) error {
	return s.db.WithContext(ctx).Create(company).Error
}

func (s *CompanyStore) Update(ctx context.Context, company *domain.Company) error {
	return s.db.WithContext(ctx).Save(company).Error
}

func (s *CompanyStore) Delete(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Company{}).Error
}

func (s *CompanyStore) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&domain.Company{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
