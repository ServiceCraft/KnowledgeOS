package middleware

import (
	"net/http"

	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/respond"
)

func RequireRole(roles ...domain.Role) func(http.Handler) http.Handler {
	allowed := map[domain.Role]bool{}
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil {
				respond.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			if !allowed[claims.Role] {
				respond.Error(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
