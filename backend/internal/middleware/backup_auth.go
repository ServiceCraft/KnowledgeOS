package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/knowledgeos/backend/internal/respond"
)

// BackupAPIKeyAuth guards the snapshot endpoint with a dedicated bearer key.
// The key is resolved per-request via keyProvider (typically reading an env
// variable) so it can be rotated or revoked without restarting the application:
// clearing the source disables the endpoint entirely (503).
func BackupAPIKeyAuth(keyProvider func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expected := strings.TrimSpace(keyProvider())
			if expected == "" {
				respond.Error(w, http.StatusServiceUnavailable, "backup endpoint is disabled")
				return
			}

			header := r.Header.Get("Authorization")
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				respond.Error(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}

			if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(parts[1])), []byte(expected)) != 1 {
				respond.Error(w, http.StatusUnauthorized, "invalid api key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
