package service

import (
	"context"
	"errors"
	applog "github.com/knowledgeos/backend/internal/logger"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/domain"
)

type AuthService struct {
	users     domain.UserRepository
	companies domain.CompanyRepository
	syncRepo  domain.SyncRepository
	jwtMgr    *auth.JWTManager
}

// NewAuthService executes the service.NewAuthService operation.
func NewAuthService(users domain.UserRepository, companies domain.CompanyRepository, syncRepo domain.SyncRepository, jwtMgr *auth.JWTManager) *AuthService {
	return &AuthService{users: users, companies: companies, syncRepo: syncRepo, jwtMgr: jwtMgr}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    int64     `json:"expires_at"`
	User         LoginUser `json:"user"`
}

type LoginUser struct {
	ID         string   `json:"id"`
	Email      string   `json:"email"`
	Role       string   `json:"role"`
	CompanyIDs []string `json:"company_ids,omitempty"`
}

// Login executes the service.AuthService.Login operation.
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	applog.TraceCall(ctx, "service.AuthService.Login")
	user, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !auth.ComparePassword(user.PasswordHash, req.Password) {
		return nil, errors.New("invalid credentials")
	}

	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	pair, refreshHash, err := s.jwtMgr.Issue(user)
	if err != nil {
		return nil, err
	}

	token := &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	if err := s.syncRepo.SaveRefreshToken(ctx, token); err != nil {
		return nil, err
	}

	lu, err := s.buildLoginUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt,
		User:         lu,
	}, nil
}

// Refresh executes the service.AuthService.Refresh operation.
func (s *AuthService) Refresh(ctx context.Context, req RefreshRequest) (*LoginResponse, error) {
	applog.TraceCall(ctx, "service.AuthService.Refresh")
	oldHash := auth.HashToken(req.RefreshToken)
	stored, err := s.syncRepo.GetRefreshToken(ctx, oldHash)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	if time.Now().After(stored.ExpiresAt) {
		return nil, errors.New("refresh token expired")
	}

	if err := s.syncRepo.RevokeRefreshToken(ctx, oldHash); err != nil {
		return nil, err
	}

	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	pair, newHash, err := s.jwtMgr.Issue(user)
	if err != nil {
		return nil, err
	}

	newToken := &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: newHash,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	if err := s.syncRepo.SaveRefreshToken(ctx, newToken); err != nil {
		return nil, err
	}

	lu, err := s.buildLoginUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt,
		User:         lu,
	}, nil
}

// ListAccessibleCompanies returns companies the caller may access.
func (s *AuthService) ListAccessibleCompanies(ctx context.Context, userID uuid.UUID, role domain.Role) ([]domain.Company, error) {
	applog.TraceCall(ctx, "service.AuthService.ListAccessibleCompanies")
	if role == domain.RoleSuperadmin {
		items, _, err := s.companies.List(ctx, domain.CompanyFilter{Page: 1, Limit: 1000})
		return items, err
	}
	return s.users.ListCompaniesForUser(ctx, userID)
}

// Logout executes the service.AuthService.Logout operation.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	applog.TraceCall(ctx, "service.AuthService.Logout")
	hash := auth.HashToken(refreshToken)
	return s.syncRepo.RevokeRefreshToken(ctx, hash)
}

func (s *AuthService) buildLoginUser(ctx context.Context, user *domain.User) (LoginUser, error) {
	lu := LoginUser{
		ID:    user.ID.String(),
		Email: user.Email,
		Role:  string(user.Role),
	}
	if user.Role == domain.RoleSuperadmin {
		return lu, nil
	}
	ids, err := s.users.GetCompanyIDs(ctx, user.ID)
	if err != nil {
		return LoginUser{}, err
	}
	for _, id := range ids {
		lu.CompanyIDs = append(lu.CompanyIDs, id.String())
	}
	return lu, nil
}
