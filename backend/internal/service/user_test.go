package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

// fakeUserRepo is a minimal in-memory domain.UserRepository for service tests.
type fakeUserRepo struct {
	byID map[uuid.UUID]*domain.User
}

func newFakeUserRepo(users ...*domain.User) *fakeUserRepo {
	r := &fakeUserRepo{byID: map[uuid.UUID]*domain.User{}}
	for _, u := range users {
		r.byID[u.ID] = u
	}
	return r
}

func (r *fakeUserRepo) List(context.Context, uuid.UUID, domain.UserFilter) ([]domain.User, int64, error) {
	return nil, 0, nil
}
func (r *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if u, ok := r.byID[id]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range r.byID {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeUserRepo) Create(_ context.Context, u *domain.User) error { r.byID[u.ID] = u; return nil }
func (r *fakeUserRepo) Update(_ context.Context, u *domain.User) error { r.byID[u.ID] = u; return nil }
func (r *fakeUserRepo) Delete(_ context.Context, id uuid.UUID) error   { delete(r.byID, id); return nil }

func mkUser(role domain.Role, company uuid.UUID) *domain.User {
	id := uuid.New()
	return &domain.User{
		BaseModel: domain.BaseModel{ID: id},
		CompanyID: &company,
		Email:     id.String() + "@example.com",
		Role:      role,
		IsActive:  true,
	}
}

func ptrStr(s string) *string { return &s }

func TestAdminCannotManageOtherAdmins(t *testing.T) {
	ctx := context.Background()
	company := uuid.New()

	admin := mkUser(domain.RoleAdmin, company)
	otherAdmin := mkUser(domain.RoleAdmin, company)
	editor := mkUser(domain.RoleEditor, company)
	superadmin := mkUser(domain.RoleSuperadmin, company)

	repo := newFakeUserRepo(admin, otherAdmin, editor, superadmin)
	svc := NewUserService(repo, nil)

	adminActor := Actor{ID: admin.ID, Role: domain.RoleAdmin, CompanyID: company}
	superActor := Actor{ID: superadmin.ID, Role: domain.RoleSuperadmin, CompanyID: company}

	newEmail := "changed@example.com"

	// Admin editing ANOTHER admin -> 403.
	if _, err := svc.Update(ctx, adminActor, otherAdmin.ID, UpdateUserRequest{Email: ptrStr(newEmail)}); HTTPStatus(err) != 403 {
		t.Fatalf("admin editing other admin: want 403, got %v (err=%v)", HTTPStatus(err), err)
	}
	// Admin deleting ANOTHER admin -> 403.
	if err := svc.Delete(ctx, adminActor, otherAdmin.ID); HTTPStatus(err) != 403 {
		t.Fatalf("admin deleting other admin: want 403, got %v (err=%v)", HTTPStatus(err), err)
	}

	// Admin editing a lower-privilege user -> allowed.
	if _, err := svc.Update(ctx, adminActor, editor.ID, UpdateUserRequest{Email: ptrStr("editor2@example.com")}); err != nil {
		t.Fatalf("admin editing editor: unexpected error %v", err)
	}

	// Admin editing OWN account (email) -> allowed.
	if _, err := svc.Update(ctx, adminActor, admin.ID, UpdateUserRequest{Email: ptrStr("me@example.com")}); err != nil {
		t.Fatalf("admin editing self: unexpected error %v", err)
	}

	// Superadmin editing an admin -> allowed.
	if _, err := svc.Update(ctx, superActor, admin.ID, UpdateUserRequest{Email: ptrStr("admin-by-super@example.com")}); err != nil {
		t.Fatalf("superadmin editing admin: unexpected error %v", err)
	}
}
