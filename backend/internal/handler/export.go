package handler

import (
	applog "github.com/knowledgeos/backend/internal/logger"
	"net/http"

	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
	"github.com/knowledgeos/backend/internal/store"
)

type ExportHandler struct {
	svc   *service.ExportService
	users *store.UserStore
}

// NewExportHandler executes the handler.NewExportHandler operation.
func NewExportHandler(svc *service.ExportService, users *store.UserStore) *ExportHandler {
	return &ExportHandler{svc: svc, users: users}
}

// Export executes the handler.ExportHandler.Export operation.
func (h *ExportHandler) Export(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.ExportHandler.Export")
	companyID := middleware.GetCompanyID(r.Context())

	data, err := h.svc.Export(r.Context(), companyID)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSON(w, http.StatusOK, data)
}

// Import executes the handler.ExportHandler.Import operation.
func (h *ExportHandler) Import(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.ExportHandler.Import")
	companyID := middleware.GetCompanyID(r.Context())

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
