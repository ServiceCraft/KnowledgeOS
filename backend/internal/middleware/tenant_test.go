package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type tenantTestUsers struct {
	companyIDs map[uuid.UUID][]uuid.UUID
}

func (r *tenantTestUsers) List(context.Context, uuid.UUID, domain.UserFilter) ([]domain.User, int64, error) {
	return nil, 0, nil
}
func (r *tenantTestUsers) GetByID(context.Context, uuid.UUID) (*domain.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *tenantTestUsers) GetByEmail(context.Context, string) (*domain.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *tenantTestUsers) Create(context.Context, *domain.User) error { return nil }
func (r *tenantTestUsers) Update(context.Context, *domain.User) error { return nil }
func (r *tenantTestUsers) Delete(context.Context, uuid.UUID) error    { return nil }
func (r *tenantTestUsers) GetCompanyIDs(_ context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return append([]uuid.UUID(nil), r.companyIDs[userID]...), nil
}
func (r *tenantTestUsers) SetCompanyIDs(context.Context, uuid.UUID, []uuid.UUID) error { return nil }
func (r *tenantTestUsers) HasCompany(_ context.Context, userID, companyID uuid.UUID) (bool, error) {
	for _, id := range r.companyIDs[userID] {
		if id == companyID {
			return true, nil
		}
	}
	return false, nil
}
func (r *tenantTestUsers) ListCompaniesForUser(context.Context, uuid.UUID) ([]domain.Company, error) {
	return nil, nil
}

func TestTenantRequiresCompanyHeaderForSuperadmin(t *testing.T) {
	called := false
	users := &tenantTestUsers{}
	handler := Tenant(users)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/qa", nil)
	req = req.WithContext(SetClaims(req.Context(), &auth.Claims{Role: domain.RoleSuperadmin}))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("next handler was called")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestTenantUsesCompanyHeaderForSuperadmin(t *testing.T) {
	companyID := uuid.New()
	users := &tenantTestUsers{}
	handler := Tenant(users)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if got := GetCompanyID(r.Context()); got != companyID {
			t.Fatalf("company id = %s, want %s", got, companyID)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/qa", nil)
	req.Header.Set("X-Company-ID", companyID.String())
	req = req.WithContext(SetClaims(req.Context(), &auth.Claims{Role: domain.RoleSuperadmin}))

	handler.ServeHTTP(httptest.NewRecorder(), req)
}

func TestTenantUsesAssignedCompanyForSingleCompanyUser(t *testing.T) {
	companyID := uuid.New()
	userID := uuid.New()
	users := &tenantTestUsers{companyIDs: map[uuid.UUID][]uuid.UUID{userID: {companyID}}}
	handler := Tenant(users)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if got := GetCompanyID(r.Context()); got != companyID {
			t.Fatalf("company id = %s, want %s", got, companyID)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/qa", nil)
	req = req.WithContext(SetClaims(req.Context(), &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: userID.String()},
		Role:             domain.RoleAdmin,
	}))

	handler.ServeHTTP(httptest.NewRecorder(), req)
}

func TestTenantUsesHeaderForMultiCompanyUser(t *testing.T) {
	companyA := uuid.New()
	companyB := uuid.New()
	userID := uuid.New()
	users := &tenantTestUsers{companyIDs: map[uuid.UUID][]uuid.UUID{userID: {companyA, companyB}}}
	handler := Tenant(users)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if got := GetCompanyID(r.Context()); got != companyB {
			t.Fatalf("company id = %s, want %s", got, companyB)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/qa", nil)
	req.Header.Set("X-Company-ID", companyB.String())
	req = req.WithContext(SetClaims(req.Context(), &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: userID.String()},
		Role:             domain.RoleAdmin,
	}))

	handler.ServeHTTP(httptest.NewRecorder(), req)
}
