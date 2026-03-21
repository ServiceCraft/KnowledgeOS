package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/knowledgeos/backend/internal/respond"
)

var validEntityTypes = map[string]bool{
	"qa":      true,
	"article": true,
	"pricing": true,
}

func EntityType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		et := chi.URLParam(r, "entityType")
		if !validEntityTypes[et] {
			respond.Error(w, http.StatusBadRequest, "invalid entity type: must be qa, article, or pricing")
			return
		}
		ctx := SetEntityType(r.Context(), et)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
