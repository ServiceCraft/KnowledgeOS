package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/service"
)

type AdminHandler struct {
	svc *service.AdminService
}

func NewAdminHandler(svc *service.AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) ListCompanies(w http.ResponseWriter, r *http.Request) {
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

func (h *AdminHandler) GetCompany(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	company, err := h.svc.GetCompany(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "company not found")
		return
	}
	JSON(w, http.StatusOK, company)
}

func (h *AdminHandler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	var company domain.Company
	if err := Decode(r, &company); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.CreateCompany(r.Context(), &company); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusCreated, company)
}

func (h *AdminHandler) UpdateCompany(w http.ResponseWriter, r *http.Request) {
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
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusOK, company)
}

func (h *AdminHandler) DeleteCompany(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.DeleteCompany(r.Context(), id); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) CreateCompanyAdmin(w http.ResponseWriter, r *http.Request) {
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
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusCreated, user)
}
