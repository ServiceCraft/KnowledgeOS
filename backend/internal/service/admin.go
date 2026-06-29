package service

import (
	"context"
	"errors"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/domain"
)

type AdminService struct {
	companies domain.CompanyRepository
	users     domain.UserRepository
	syncRepo  domain.SyncRepository
}

// NewAdminService executes the service.NewAdminService operation.
func NewAdminService(companies domain.CompanyRepository, users domain.UserRepository, syncRepo domain.SyncRepository) *AdminService {
	return &AdminService{companies: companies, users: users, syncRepo: syncRepo}
}

// ListCompanies executes the service.AdminService.ListCompanies operation.
func (s *AdminService) ListCompanies(ctx context.Context, filter domain.CompanyFilter) ([]domain.Company, int64, error) {
	applog.TraceCall(ctx, "service.AdminService.ListCompanies")
	return s.companies.List(ctx, filter)
}

// GetCompany executes the service.AdminService.GetCompany operation.
func (s *AdminService) GetCompany(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	applog.TraceCall(ctx, "service.AdminService.GetCompany")
	return s.companies.GetByID(ctx, id)
}

// CreateCompany executes the service.AdminService.CreateCompany operation.
func (s *AdminService) CreateCompany(ctx context.Context, company *domain.Company) error {
	applog.TraceCall(ctx, "service.AdminService.CreateCompany")
	if company.Name == "" {
		return errors.New("name is required")
	}
	if err := s.companies.Create(ctx, company); err != nil {
		return applog.TraceErr(ctx, "create company failed", err)
	}
	return s.syncRepo.InitSequence(ctx, company.ID)
}

// UpdateCompany executes the service.AdminService.UpdateCompany operation.
func (s *AdminService) UpdateCompany(ctx context.Context, company *domain.Company) error {
	applog.TraceCall(ctx, "service.AdminService.UpdateCompany")
	if _, err := s.companies.GetByID(ctx, company.ID); err != nil {
		return errors.New("company not found")
	}
	return s.companies.Update(ctx, company)
}

// DeleteCompany executes the service.AdminService.DeleteCompany operation.
func (s *AdminService) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	applog.TraceCall(ctx, "service.AdminService.DeleteCompany")
	if _, err := s.companies.GetByID(ctx, id); err != nil {
		return errors.New("company not found")
	}
	return s.companies.Delete(ctx, id)
}

type CreateCompanyAdminRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// CreateCompanyAdmin executes the service.AdminService.CreateCompanyAdmin operation.
func (s *AdminService) CreateCompanyAdmin(ctx context.Context, companyID uuid.UUID, req CreateCompanyAdminRequest) (*domain.User, error) {
	applog.TraceCall(ctx, "service.AdminService.CreateCompanyAdmin")
	email, err := ValidateNewCredentials(req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: hash,
		Role:         domain.RoleAdmin,
		IsActive:     true,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	if err := s.users.SetCompanyIDs(ctx, user.ID, []uuid.UUID{companyID}); err != nil {
		return nil, err
	}
	user.CompanyIDs = []uuid.UUID{companyID}
	return user, nil
}
