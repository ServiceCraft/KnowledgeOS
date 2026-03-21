package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/respond"
)

func Tenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r.Context())
		if claims == nil {
			respond.Error(w, http.StatusUnauthorized, "no claims in context")
			return
		}

		if claims.Role == domain.RoleSuperadmin {
			headerID := r.Header.Get("X-Company-ID")
			if headerID != "" {
				id, err := uuid.Parse(headerID)
				if err != nil {
					respond.Error(w, http.StatusBadRequest, "invalid X-Company-ID")
					return
				}
				ctx := SetCompanyID(r.Context(), id)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		if claims.CompanyID == nil {
			respond.Error(w, http.StatusForbidden, "no company associated with this user")
			return
		}

		ctx := SetCompanyID(r.Context(), *claims.CompanyID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
