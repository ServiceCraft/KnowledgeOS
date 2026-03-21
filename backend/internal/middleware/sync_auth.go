package middleware

import (
	"net/http"
	"strings"

	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/respond"
)

func SyncAPIKeyAuth(syncRepo domain.SyncRepository) func(http.Handler) http.Handler {
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

			keyHash := auth.HashAPIKey(parts[1])
			apiKey, err := syncRepo.GetAPIKeyByHash(r.Context(), keyHash)
			if err != nil {
				respond.Error(w, http.StatusUnauthorized, "invalid api key")
				return
			}

			_ = syncRepo.TouchAPIKey(r.Context(), apiKey.ID)

			ctx := SetCompanyID(r.Context(), apiKey.CompanyID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
