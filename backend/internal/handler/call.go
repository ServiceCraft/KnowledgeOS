package handler

import (
	applog "github.com/knowledgeos/backend/internal/logger"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
)

type CallHandler struct {
	svc *service.CallService
}

// NewCallHandler executes the handler.NewCallHandler operation.
func NewCallHandler(svc *service.CallService) *CallHandler {
	return &CallHandler{svc: svc}
}

// GetCall executes the handler.CallHandler.GetCall operation.
func (h *CallHandler) GetCall(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.CallHandler.GetCall")
	companyID := middleware.GetCompanyID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	call, err := h.svc.GetByID(r.Context(), companyID, id)
	if err != nil {
		Error(w, http.StatusNotFound, "call not found")
		return
	}
	JSON(w, http.StatusOK, call)
}

// ListMentionsForQA executes the handler.CallHandler.ListMentionsForQA operation.
func (h *CallHandler) ListMentionsForQA(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.CallHandler.ListMentionsForQA")
	companyID := middleware.GetCompanyID(r.Context())
	qaID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	filter := domain.QACallMentionFilter{
		Page:  intQuery(r, "page", 1),
		Limit: intQuery(r, "limit", 100),
	}

	items, total, err := h.svc.ListMentionsForQA(r.Context(), companyID, qaID, filter)
	if err != nil {
		Error(w, http.StatusNotFound, err.Error())
		return
	}
	JSONList(w, http.StatusOK, items, total)
}
