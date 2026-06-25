package handler

import (
	applog "github.com/knowledgeos/backend/internal/logger"
	"net/http"

	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
)

type SyncHandler struct {
	svc *service.SyncService
}

// NewSyncHandler executes the handler.NewSyncHandler operation.
func NewSyncHandler(svc *service.SyncService) *SyncHandler {
	return &SyncHandler{svc: svc}
}

// Status executes the handler.SyncHandler.Status operation.
func (h *SyncHandler) Status(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.SyncHandler.Status")
	companyID := middleware.GetCompanyID(r.Context())

	status, err := h.svc.Status(r.Context(), companyID)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSON(w, http.StatusOK, status)
}

// Push executes the handler.SyncHandler.Push operation.
func (h *SyncHandler) Push(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.SyncHandler.Push")
	companyID := middleware.GetCompanyID(r.Context())

	var payload service.SyncPushPayload
	if err := Decode(r, &payload); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.ApplyPull(r.Context(), companyID, &payload); err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Pull executes the handler.SyncHandler.Pull operation.
func (h *SyncHandler) Pull(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.SyncHandler.Pull")
	companyID := middleware.GetCompanyID(r.Context())

	payload, err := h.svc.GatherPush(r.Context(), companyID)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSON(w, http.StatusOK, payload)
}
