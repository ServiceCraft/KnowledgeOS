package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/domain"
)

type AdminService struct {
	companies domain.CompanyRepository
	users     domain.UserRepository
	syncRepo  domain.SyncRepository
}

func NewAdminService(companies domain.CompanyRepository, users domain.UserRepository, syncRepo domain.SyncRepository) *AdminService {
	return &AdminService{companies: companies, users: users, syncRepo: syncRepo}
}

func (s *AdminService) ListCompanies(ctx context.Context, filter domain.CompanyFilter) ([]domain.Company, int64, error) {
	return s.companies.List(ctx, filter)
}

func (s *AdminService) GetCompany(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	return s.companies.GetByID(ctx, id)
}

func (s *AdminService) CreateCompany(ctx context.Context, company *domain.Company) error {
	if company.Name == "" {
		return errors.New("name is required")
	}
	if err := s.companies.Create(ctx, company); err != nil {
		return err
	}
	return s.syncRepo.InitSequence(ctx, company.ID)
}

func (s *AdminService) UpdateCompany(ctx context.Context, company *domain.Company) error {
	if _, err := s.companies.GetByID(ctx, company.ID); err != nil {
		return errors.New("company not found")
	}
	return s.companies.Update(ctx, company)
}

func (s *AdminService) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	if _, err := s.companies.GetByID(ctx, id); err != nil {
		return errors.New("company not found")
	}
	return s.companies.Delete(ctx, id)
}

type CreateCompanyAdminRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *AdminService) CreateCompanyAdmin(ctx context.Context, companyID uuid.UUID, req CreateCompanyAdminRequest) (*domain.User, error) {
	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email and password are required")
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		CompanyID:    &companyID,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         domain.RoleAdmin,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
