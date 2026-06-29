package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/respond"
)

// Tenant resolves the active company for tenant-scoped routes. Superadmins must
// send X-Company-ID. Other roles use X-Company-ID when they belong to multiple
// companies; a single assignment is auto-selected.
func Tenant(users domain.UserRepository) func(http.Handler) http.Handler {
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
				ctx := SetCompanyID(r.Context(), id)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			userID := GetUserID(r.Context())
			companyIDs, err := users.GetCompanyIDs(r.Context(), userID)
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

			ctx := SetCompanyID(r.Context(), companyID)
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
