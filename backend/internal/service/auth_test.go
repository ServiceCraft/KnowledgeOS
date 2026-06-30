package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/testutil"
)

type fakeTokenRepo struct {
	byHash map[string]*domain.RefreshToken
}

func newFakeTokenRepo() *fakeTokenRepo {
	return &fakeTokenRepo{byHash: map[string]*domain.RefreshToken{}}
}

func (r *fakeTokenRepo) SaveRefreshToken(_ context.Context, token *domain.RefreshToken) error {
	r.byHash[token.TokenHash] = token
	return nil
}
func (r *fakeTokenRepo) GetRefreshToken(_ context.Context, tokenHash string) (*domain.RefreshToken, error) {
	if t, ok := r.byHash[tokenHash]; ok {
		return t, nil
	}
	return nil, domain.ErrNotFound
}
func (r *fakeTokenRepo) RevokeRefreshToken(_ context.Context, tokenHash string) error {
	delete(r.byHash, tokenHash)
	return nil
}
func (r *fakeTokenRepo) RevokeAllUserTokens(context.Context, uuid.UUID) error { return nil }

func TestAuthServiceLoginAndRefresh(t *testing.T) {
	ctx := context.Background()
	company := uuid.New()
	user := &domain.User{
		BaseModel:    domain.BaseModel{ID: uuid.New()},
		Email:        "user@example.com",
		PasswordHash: mustHash(t, "password123"),
		Role:         domain.RoleEditor,
		IsActive:     true,
		CompanyIDs:   []uuid.UUID{company},
	}
	users := testutil.NewFakeUserRepo(user)
	tokens := newFakeTokenRepo()
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	svc := NewAuthService(users, nil, tokens, jwtMgr, 24*time.Hour, 100)

	tests := []struct {
		name       string // что проверяем
		run        func(t *testing.T) error
		wantStatus int
	}{
		{
			name: "valid credentials — tokens и company_ids в ответе",
			run: func(t *testing.T) error {
				resp, err := svc.Login(ctx, LoginRequest{Email: user.Email, Password: "password123"})
				if err != nil {
					return err
				}
				if resp.AccessToken == "" || resp.RefreshToken == "" {
					t.Fatalf("expected tokens")
				}
				if len(resp.User.CompanyIDs) != 1 {
					t.Fatalf("company_ids = %v", resp.User.CompanyIDs)
				}
				return nil
			},
		},
		{
			name:       "wrong password — 401 invalid credentials",
			wantStatus: 401,
			run: func(t *testing.T) error {
				_, err := svc.Login(ctx, LoginRequest{Email: user.Email, Password: "wrong"})
				return err
			},
		},
		{
			name:       "unknown email — 401 invalid credentials",
			wantStatus: 401,
			run: func(t *testing.T) error {
				_, err := svc.Login(ctx, LoginRequest{Email: "missing@example.com", Password: "password123"})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(t)
			if tt.wantStatus == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if HTTPStatus(err) != tt.wantStatus {
				t.Fatalf("status = %d, err = %v", HTTPStatus(err), err)
			}
		})
	}

	login, err := svc.Login(ctx, LoginRequest{Email: user.Email, Password: "password123"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	refreshResp, err := svc.Refresh(ctx, RefreshRequest{RefreshToken: login.RefreshToken})
	if err != nil {
		t.Fatalf("refresh success: %v", err)
	}
	if refreshResp.AccessToken == "" {
		t.Fatalf("expected refreshed access token")
	}
}

func mustHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	return hash
}
