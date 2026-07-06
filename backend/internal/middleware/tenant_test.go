package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/cache"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/testutil"
)

func TestTenantMiddlewareRejectsStaleMembershipVersion(t *testing.T) {
	userID := uuid.New()
	companyA := uuid.New()
	// Server-side version is 2 (membership changed since the token was issued).
	user := &domain.User{BaseModel: domain.BaseModel{ID: userID}, Role: domain.RoleAdmin, MembershipVersion: 2}
	membership := testutil.NewFakeUserRepo(user)
	membership.CompanyIDs[userID] = []uuid.UUID{companyA}
	cacheProvider := cache.NewMemoryProvider(&testutil.FakeCompanyChecker{}, 0, 0)

	called := false
	handler := Tenant(membership, cacheProvider)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/qa", nil)
	req = req.WithContext(SetClaims(req.Context(), &auth.Claims{
		RegisteredClaims:  jwt.RegisteredClaims{Subject: userID.String()},
		Role:              domain.RoleAdmin,
		MembershipVersion: 1, // stale
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if called {
		t.Fatalf("handler should not run for stale membership version")
	}
}

func TestTenantMiddleware(t *testing.T) {
	companyA := uuid.New()
	companyB := uuid.New()
	userID := uuid.New()
	superCompany := uuid.New()

	tests := []struct {
		name        string // что проверяем
		role        domain.Role
		userID      uuid.UUID
		companies   []uuid.UUID
		companyMap  map[uuid.UUID]bool
		header      string
		wantStatus  int
		wantCompany uuid.UUID
		wantCalled  bool
	}{
		{
			name:       "superadmin без заголовка — 403, tenant не резолвится",
			role:       domain.RoleSuperadmin,
			wantStatus: http.StatusForbidden,
		},
		{
			name:        "superadmin с валидным header и существующей company — company в context",
			role:        domain.RoleSuperadmin,
			companyMap:  map[uuid.UUID]bool{superCompany: true},
			header:      superCompany.String(),
			wantStatus:  http.StatusOK,
			wantCompany: superCompany,
			wantCalled:  true,
		},
		{
			name:       "superadmin с несуществующей company — 404",
			role:       domain.RoleSuperadmin,
			companyMap: map[uuid.UUID]bool{superCompany: false},
			header:     superCompany.String(),
			wantStatus: http.StatusNotFound,
		},
		{
			name:        "user с одной company — auto-select без header",
			role:        domain.RoleAdmin,
			userID:      userID,
			companies:   []uuid.UUID{companyA},
			wantStatus:  http.StatusOK,
			wantCompany: companyA,
			wantCalled:  true,
		},
		{
			name:       "user с двумя companies без header — 403",
			role:       domain.RoleAdmin,
			userID:     userID,
			companies:  []uuid.UUID{companyA, companyB},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "user с header чужой company — 403 access denied",
			role:       domain.RoleAdmin,
			userID:     userID,
			companies:  []uuid.UUID{companyA, companyB},
			header:     uuid.New().String(),
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "user без companies — 403",
			role:       domain.RoleAdmin,
			userID:     userID,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "invalid UUID в header — 400",
			role:       domain.RoleAdmin,
			userID:     userID,
			companies:  []uuid.UUID{companyA},
			header:     "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "claims == nil — 401",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:        "multi-company user с header выбирает company B",
			role:        domain.RoleAdmin,
			userID:      userID,
			companies:   []uuid.UUID{companyA, companyB},
			header:      companyB.String(),
			wantStatus:  http.StatusOK,
			wantCompany: companyB,
			wantCalled:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			membership := &testutil.FakeUserRepo{CompanyIDs: map[uuid.UUID][]uuid.UUID{}}
			if tt.userID != uuid.Nil {
				membership.CompanyIDs[tt.userID] = tt.companies
			}
			checker := &testutil.FakeCompanyChecker{ExistsMap: tt.companyMap}
			cacheProvider := cache.NewMemoryProvider(checker, 0, 0)

			called := false
			handler := Tenant(membership, cacheProvider)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				called = true
				if got := GetCompanyID(r.Context()); got != tt.wantCompany {
					t.Fatalf("company id = %s, want %s", got, tt.wantCompany)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, "/qa", nil)
			if tt.header != "" {
				req.Header.Set("X-Company-ID", tt.header)
			}
			if tt.role != "" || tt.userID != uuid.Nil {
				req = req.WithContext(SetClaims(req.Context(), &auth.Claims{
					RegisteredClaims: jwt.RegisteredClaims{Subject: tt.userID.String()},
					Role:             tt.role,
				}))
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if called != tt.wantCalled {
				t.Fatalf("next called = %v, want %v", called, tt.wantCalled)
			}
		})
	}
}
