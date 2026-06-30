package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/testutil"
)

func mkUser(role domain.Role, company uuid.UUID) *domain.User {
	id := uuid.New()
	return &domain.User{
		BaseModel:  domain.BaseModel{ID: id},
		CompanyIDs: []uuid.UUID{company},
		Email:      id.String() + "@example.com",
		Role:       role,
		IsActive:   true,
	}
}

func ptrStr(s string) *string { return &s }

func TestUserServiceRBAC(t *testing.T) {
	ctx := context.Background()
	company := uuid.New()

	admin := mkUser(domain.RoleAdmin, company)
	otherAdmin := mkUser(domain.RoleAdmin, company)
	editor := mkUser(domain.RoleEditor, company)
	superadmin := mkUser(domain.RoleSuperadmin, company)

	repo := testutil.NewFakeUserRepo(admin, otherAdmin, editor, superadmin)
	svc := NewUserService(repo, nil)

	adminActor := Actor{ID: admin.ID, Role: domain.RoleAdmin, CompanyID: company}
	superActor := Actor{ID: superadmin.ID, Role: domain.RoleSuperadmin, CompanyID: company}

	tests := []struct {
		name       string // что проверяем
		run        func() error
		wantStatus int
	}{
		{
			name:       "admin не может редактировать другого admin — 403",
			wantStatus: 403,
			run: func() error {
				_, err := svc.Update(ctx, adminActor, otherAdmin.ID, UpdateUserRequest{Email: ptrStr("changed@example.com")})
				return err
			},
		},
		{
			name:       "admin не может удалить другого admin — 403",
			wantStatus: 403,
			run: func() error {
				return svc.Delete(ctx, adminActor, otherAdmin.ID)
			},
		},
		{
			name: "admin может редактировать editor",
			run: func() error {
				_, err := svc.Update(ctx, adminActor, editor.ID, UpdateUserRequest{Email: ptrStr("editor2@example.com")})
				return err
			},
		},
		{
			name: "admin может редактировать себя",
			run: func() error {
				_, err := svc.Update(ctx, adminActor, admin.ID, UpdateUserRequest{Email: ptrStr("me@example.com")})
				return err
			},
		},
		{
			name: "superadmin может редактировать admin",
			run: func() error {
				_, err := svc.Update(ctx, superActor, admin.ID, UpdateUserRequest{Email: ptrStr("admin-by-super@example.com")})
				return err
			},
		},
		{
			name:       "admin не может назначить чужую company при create — 403",
			wantStatus: 403,
			run: func() error {
				_, err := svc.Create(ctx, adminActor, CreateUserRequest{
					Email:      "new-user@example.com",
					Password:   "password123",
					Role:       domain.RoleEditor,
					CompanyIDs: []uuid.UUID{uuid.New()},
				})
				return err
			},
		},
		{
			name:       "create без companies для non-superadmin — 400",
			wantStatus: 400,
			run: func() error {
				_, err := svc.Create(ctx, superActor, CreateUserRequest{
					Email:    "no-company@example.com",
					Password: "password123",
					Role:     domain.RoleEditor,
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if tt.wantStatus == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if HTTPStatus(err) != tt.wantStatus {
				t.Fatalf("status = %d, err = %v, want %d", HTTPStatus(err), err, tt.wantStatus)
			}
		})
	}
}
