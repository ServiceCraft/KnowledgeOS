package testutil

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

// FakeUserRepo is an in-memory UserRepository for tests.
type FakeUserRepo struct {
	ByID       map[uuid.UUID]*domain.User
	CompanyIDs map[uuid.UUID][]uuid.UUID
}

// NewFakeUserRepo builds a repo seeded with users and their company assignments.
func NewFakeUserRepo(users ...*domain.User) *FakeUserRepo {
	r := &FakeUserRepo{
		ByID:       map[uuid.UUID]*domain.User{},
		CompanyIDs: map[uuid.UUID][]uuid.UUID{},
	}
	for _, u := range users {
		r.ByID[u.ID] = u
		if len(u.CompanyIDs) > 0 {
			r.CompanyIDs[u.ID] = append([]uuid.UUID(nil), u.CompanyIDs...)
		}
	}
	return r
}

func (r *FakeUserRepo) List(context.Context, uuid.UUID, domain.UserFilter) ([]domain.User, int64, error) {
	return nil, 0, nil
}
func (r *FakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if u, ok := r.ByID[id]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}
func (r *FakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range r.ByID {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (r *FakeUserRepo) Create(_ context.Context, u *domain.User) error {
	r.ByID[u.ID] = u
	return nil
}
func (r *FakeUserRepo) CreateWithCompanies(_ context.Context, u *domain.User, companyIDs []uuid.UUID) error {
	r.ByID[u.ID] = u
	r.CompanyIDs[u.ID] = append([]uuid.UUID(nil), companyIDs...)
	return nil
}
func (r *FakeUserRepo) Update(_ context.Context, u *domain.User) error { r.ByID[u.ID] = u; return nil }
func (r *FakeUserRepo) Delete(_ context.Context, id uuid.UUID) error   { delete(r.ByID, id); return nil }
func (r *FakeUserRepo) GetCompanyIDs(_ context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return append([]uuid.UUID(nil), r.CompanyIDs[userID]...), nil
}
func (r *FakeUserRepo) SetCompanyIDs(_ context.Context, userID uuid.UUID, companyIDs []uuid.UUID) error {
	r.CompanyIDs[userID] = append([]uuid.UUID(nil), companyIDs...)
	return nil
}
func (r *FakeUserRepo) HasCompany(_ context.Context, userID, companyID uuid.UUID) (bool, error) {
	for _, id := range r.CompanyIDs[userID] {
		if id == companyID {
			return true, nil
		}
	}
	return false, nil
}
func (r *FakeUserRepo) ListCompaniesForUser(context.Context, uuid.UUID) ([]domain.Company, error) {
	return nil, nil
}

// FakeCompanyChecker implements CompanyExistenceChecker for tests.
type FakeCompanyChecker struct {
	ExistsMap map[uuid.UUID]bool
}

func (f *FakeCompanyChecker) Exists(_ context.Context, id uuid.UUID) (bool, error) {
	if f.ExistsMap == nil {
		return true, nil
	}
	return f.ExistsMap[id], nil
}
