package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/testutil"
)

type fakeCompanyRepo struct {
	byID map[uuid.UUID]*domain.Company
}

func (r *fakeCompanyRepo) List(context.Context, domain.CompanyFilter) ([]domain.Company, int64, error) {
	return nil, 0, nil
}
func (r *fakeCompanyRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Company, error) {
	if c, ok := r.byID[id]; ok {
		return c, nil
	}
	return nil, domain.ErrNotFound
}
func (r *fakeCompanyRepo) Create(_ context.Context, c *domain.Company) error {
	r.byID[c.ID] = c
	return nil
}
func (r *fakeCompanyRepo) Update(_ context.Context, c *domain.Company) error {
	r.byID[c.ID] = c
	return nil
}
func (r *fakeCompanyRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.byID, id)
	return nil
}
func (r *fakeCompanyRepo) Count(context.Context) (int64, error) { return int64(len(r.byID)), nil }
func (r *fakeCompanyRepo) Exists(_ context.Context, id uuid.UUID) (bool, error) {
	_, ok := r.byID[id]
	return ok, nil
}

type fakeSyncRepo struct{}

func (fakeSyncRepo) GetSequence(context.Context, uuid.UUID) (*domain.SyncSequence, error) {
	return nil, nil
}
func (fakeSyncRepo) InitSequence(context.Context, uuid.UUID) error { return nil }
func (fakeSyncRepo) GetWatermark(context.Context, uuid.UUID) (*domain.SyncWatermark, error) {
	return nil, nil
}
func (fakeSyncRepo) UpdateWatermark(context.Context, *domain.SyncWatermark) error { return nil }
func (fakeSyncRepo) GetStatus(context.Context, uuid.UUID) (*domain.SyncStatus, error) {
	return nil, nil
}
func (fakeSyncRepo) UpdateStatus(context.Context, *domain.SyncStatus) error { return nil }
func (fakeSyncRepo) CreateLog(context.Context, *domain.SyncLog) error       { return nil }
func (fakeSyncRepo) SaveRefreshToken(context.Context, *domain.RefreshToken) error {
	return nil
}
func (fakeSyncRepo) GetRefreshToken(context.Context, string) (*domain.RefreshToken, error) {
	return nil, domain.ErrNotFound
}
func (fakeSyncRepo) RevokeRefreshToken(context.Context, string) error { return nil }
func (fakeSyncRepo) RevokeAllUserTokens(context.Context, uuid.UUID) error {
	return nil
}
func (fakeSyncRepo) GetAPIKeyByHash(context.Context, string) (*domain.APIKey, error) {
	return nil, domain.ErrNotFound
}
func (fakeSyncRepo) CreateAPIKey(context.Context, *domain.APIKey) error { return nil }
func (fakeSyncRepo) TouchAPIKey(context.Context, uuid.UUID) error       { return nil }

func TestAdminServiceValidation(t *testing.T) {
	ctx := context.Background()
	companyID := uuid.New()
	companies := &fakeCompanyRepo{byID: map[uuid.UUID]*domain.Company{
		companyID: {BaseModel: domain.BaseModel{ID: companyID}, Name: "Acme"},
	}}
	users := testutil.NewFakeUserRepo()
	svc := NewAdminService(companies, users, fakeSyncRepo{}, nil)

	tests := []struct {
		name       string // что проверяем
		run        func() error
		wantStatus int
	}{
		{
			name:       "create company empty name — 400",
			wantStatus: 400,
			run: func() error {
				return svc.CreateCompany(ctx, &domain.Company{Name: ""})
			},
		},
		{
			name:       "update/delete not found — 404",
			wantStatus: 404,
			run: func() error {
				return svc.UpdateCompany(ctx, &domain.Company{BaseModel: domain.BaseModel{ID: uuid.New()}, Name: "X"})
			},
		},
		{
			name: "create company admin duplicate email — 409",
			run: func() error {
				existing := &domain.User{
					BaseModel: domain.BaseModel{ID: uuid.New()},
					Email:     "admin@example.com",
					Role:      domain.RoleAdmin,
					IsActive:  true,
				}
				users.ByID[existing.ID] = existing
				_, err := svc.CreateCompanyAdmin(ctx, companyID, CreateCompanyAdminRequest{
					Email:    existing.Email,
					Password: "password123",
				})
				return err
			},
			wantStatus: 409,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if HTTPStatus(err) != tt.wantStatus {
				t.Fatalf("status = %d, err = %v, want %d", HTTPStatus(err), err, tt.wantStatus)
			}
		})
	}
}
