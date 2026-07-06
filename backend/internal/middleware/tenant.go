package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/cache"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/respond"
)

// Tenant resolves the active company for tenant-scoped routes. Superadmins must
// send X-Company-ID. Other roles use X-Company-ID when they belong to multiple
// companies; a single assignment is auto-selected.
func Tenant(membership domain.UserMembershipReader, companies cache.Provider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil {
				respond.Error(w, http.StatusUnauthorized, "no claims in context")
				return
			}

			if claims.Role == domain.RoleSuperadmin {
				headerID := r.Header.Get("X-Company-ID")
				if headerID == "" {
					respond.Error(w, http.StatusForbidden, "company selection required")
					return
				}
				id, err := uuid.Parse(headerID)
				if err != nil {
					respond.Error(w, http.StatusBadRequest, "invalid X-Company-ID")
					return
				}
				exists, err := companies.CompanyExists(r.Context(), id)
				if err != nil {
					respond.Error(w, http.StatusInternalServerError, "failed to validate company")
					return
				}
				if !exists {
					respond.Error(w, http.StatusNotFound, "company not found")
					return
				}
				ctx := SetTenant(r.Context(), TenantContext{
					UserID:    GetUserID(r.Context()),
					Role:      claims.Role,
					CompanyID: id,
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			userID := GetUserID(r.Context())
			// Reject access tokens issued before the user's company assignments
			// changed so revoked access takes effect before the token expires.
			currentVersion, err := membership.GetMembershipVersion(r.Context(), userID)
			if err != nil {
				respond.Error(w, http.StatusInternalServerError, "failed to validate session")
				return
			}
			if currentVersion != claims.MembershipVersion {
				respond.Error(w, http.StatusUnauthorized, "invalid token")
				return
			}
			companyIDs, err := membership.GetCompanyIDs(r.Context(), userID)
			if err != nil {
				respond.Error(w, http.StatusInternalServerError, "failed to resolve user companies")
				return
			}
			if len(companyIDs) == 0 {
				respond.Error(w, http.StatusForbidden, "no company associated with this user")
				return
			}

			var companyID uuid.UUID
			headerID := r.Header.Get("X-Company-ID")
			if headerID != "" {
				id, err := uuid.Parse(headerID)
				if err != nil {
					respond.Error(w, http.StatusBadRequest, "invalid X-Company-ID")
					return
				}
				if !containsUUID(companyIDs, id) {
					respond.Error(w, http.StatusForbidden, "company access denied")
					return
				}
				companyID = id
			} else if len(companyIDs) == 1 {
				companyID = companyIDs[0]
			} else {
				respond.Error(w, http.StatusForbidden, "company selection required")
				return
			}

			ctx := SetTenant(r.Context(), TenantContext{
				UserID:    userID,
				Role:      claims.Role,
				CompanyID: companyID,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func containsUUID(ids []uuid.UUID, target uuid.UUID) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}
