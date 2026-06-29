package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/domain"
)

func TestTenantRequiresCompanyHeaderForSuperadmin(t *testing.T) {
	called := false
	handler := Tenant(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
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
	handler := Tenant(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if got := GetCompanyID(r.Context()); got != companyID {
			t.Fatalf("company id = %s, want %s", got, companyID)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/qa", nil)
	req.Header.Set("X-Company-ID", companyID.String())
	req = req.WithContext(SetClaims(req.Context(), &auth.Claims{Role: domain.RoleSuperadmin}))

	handler.ServeHTTP(httptest.NewRecorder(), req)
}

func TestTenantIgnoresCompanyHeaderForRegularUser(t *testing.T) {
	companyID := uuid.New()
	otherCompanyID := uuid.New()
	handler := Tenant(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if got := GetCompanyID(r.Context()); got != companyID {
			t.Fatalf("company id = %s, want %s", got, companyID)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/qa", nil)
	req.Header.Set("X-Company-ID", otherCompanyID.String())
	req = req.WithContext(SetClaims(req.Context(), &auth.Claims{Role: domain.RoleAdmin, CompanyID: &companyID}))

	handler.ServeHTTP(httptest.NewRecorder(), req)
}
