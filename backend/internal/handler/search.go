package handler

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
)

type SearchHandler struct {
	svc *service.SearchService
}

func NewSearchHandler(svc *service.SearchService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())

	filter := domain.SearchFilter{
		Query: r.URL.Query().Get("query"),
		Page:  intQuery(r, "page", 1),
		Limit: intQuery(r, "limit", 20),
	}

	if types := r.URL.Query().Get("types"); types != "" {
		filter.Types = strings.Split(types, ",")
	}

	if tid := r.URL.Query().Get("theme_id"); tid != "" {
		id, err := uuid.Parse(tid)
		if err == nil {
			filter.ThemeID = &id
		}
	}

	results, total, err := h.svc.Search(r.Context(), companyID, filter)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSONList(w, http.StatusOK, results, total)
}
