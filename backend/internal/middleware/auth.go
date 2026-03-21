package middleware

import (
	"net/http"
	"strings"

	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/respond"
)

func JWTAuth(jwtMgr *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				respond.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				respond.Error(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}

			claims, err := jwtMgr.Validate(parts[1])
			if err != nil {
				respond.Error(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := SetClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
