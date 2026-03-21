package service

import (
	"context"
	"errors"
	"time"

	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/domain"
)

type AuthService struct {
	users    domain.UserRepository
	syncRepo domain.SyncRepository
	jwtMgr   *auth.JWTManager
}

func NewAuthService(users domain.UserRepository, syncRepo domain.SyncRepository, jwtMgr *auth.JWTManager) *AuthService {
	return &AuthService{users: users, syncRepo: syncRepo, jwtMgr: jwtMgr}
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
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	CompanyID *string `json:"company_id,omitempty"`
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	user, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !auth.ComparePassword(user.PasswordHash, req.Password) {
		return nil, errors.New("invalid credentials")
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

	lu := LoginUser{
		ID:    user.ID.String(),
		Email: user.Email,
		Role:  string(user.Role),
	}
	if user.CompanyID != nil {
		cid := user.CompanyID.String()
		lu.CompanyID = &cid
	}

	return &LoginResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt,
		User:         lu,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, req RefreshRequest) (*LoginResponse, error) {
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

	lu := LoginUser{
		ID:    user.ID.String(),
		Email: user.Email,
		Role:  string(user.Role),
	}
	if user.CompanyID != nil {
		cid := user.CompanyID.String()
		lu.CompanyID = &cid
	}

	return &LoginResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt,
		User:         lu,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	hash := auth.HashToken(refreshToken)
	return s.syncRepo.RevokeRefreshToken(ctx, hash)
}
