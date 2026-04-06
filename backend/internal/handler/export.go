package handler

import (
	"net/http"

	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
	"github.com/knowledgeos/backend/internal/store"
)

type ExportHandler struct {
	svc   *service.ExportService
	users *store.UserStore
}

func NewExportHandler(svc *service.ExportService, users *store.UserStore) *ExportHandler {
	return &ExportHandler{svc: svc, users: users}
}

func (h *ExportHandler) Export(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())

	data, err := h.svc.Export(r.Context(), companyID)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSON(w, http.StatusOK, data)
}

func (h *ExportHandler) Import(w http.ResponseWriter, r *http.Request) {
	// Resolve company_id from DB (not JWT) to handle stale tokens
	userID := middleware.GetUserID(r.Context())
	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil || user.CompanyID == nil {
		Error(w, http.StatusUnauthorized, "could not resolve company for current user")
		return
	}
	companyID := *user.CompanyID

	var data service.ImportData
	if err := Decode(r, &data); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.Import(r.Context(), companyID, &data)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusOK, result)
}
