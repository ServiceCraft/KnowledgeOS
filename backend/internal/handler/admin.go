package handler

import (
	applog "github.com/knowledgeos/backend/internal/logger"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/service"
)

type AdminHandler struct {
	svc *service.AdminService
}

// NewAdminHandler executes the handler.NewAdminHandler operation.
func NewAdminHandler(svc *service.AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// ListCompanies executes the handler.AdminHandler.ListCompanies operation.
func (h *AdminHandler) ListCompanies(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.AdminHandler.ListCompanies")
	filter := domain.CompanyFilter{
		Page:  intQuery(r, "page", 1),
		Limit: intQuery(r, "limit", 50),
	}

	items, total, err := h.svc.ListCompanies(r.Context(), filter)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSONList(w, http.StatusOK, items, total)
}

// GetCompany executes the handler.AdminHandler.GetCompany operation.
func (h *AdminHandler) GetCompany(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.AdminHandler.GetCompany")
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	company, err := h.svc.GetCompany(r.Context(), id)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusOK, company)
}

// CreateCompany executes the handler.AdminHandler.CreateCompany operation.
func (h *AdminHandler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.AdminHandler.CreateCompany")
	var company domain.Company
	if err := Decode(r, &company); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.CreateCompany(r.Context(), &company); err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusCreated, company)
}

// UpdateCompany executes the handler.AdminHandler.UpdateCompany operation.
func (h *AdminHandler) UpdateCompany(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.AdminHandler.UpdateCompany")
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var company domain.Company
	if err := Decode(r, &company); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	company.ID = id

	if err := h.svc.UpdateCompany(r.Context(), &company); err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusOK, company)
}

// DeleteCompany executes the handler.AdminHandler.DeleteCompany operation.
func (h *AdminHandler) DeleteCompany(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.AdminHandler.DeleteCompany")
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.DeleteCompany(r.Context(), id); err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CreateCompanyAdmin executes the handler.AdminHandler.CreateCompanyAdmin operation.
func (h *AdminHandler) CreateCompanyAdmin(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.AdminHandler.CreateCompanyAdmin")
	companyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req service.CreateCompanyAdminRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.svc.CreateCompanyAdmin(r.Context(), companyID, req)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusCreated, user)
}
