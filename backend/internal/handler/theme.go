package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
)

type ThemeHandler struct {
	svc *service.ThemeService
}

func NewThemeHandler(svc *service.ThemeService) *ThemeHandler {
	return &ThemeHandler{svc: svc}
}

func (h *ThemeHandler) List(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	filter := domain.ThemeFilter{
		Page:  intQuery(r, "page", 1),
		Limit: intQuery(r, "limit", 50),
		Query: r.URL.Query().Get("query"),
	}

	items, total, err := h.svc.List(r.Context(), companyID, filter)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSONList(w, http.StatusOK, items, total)
}

func (h *ThemeHandler) Get(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.svc.GetByID(r.Context(), companyID, id)
	if err != nil {
		Error(w, http.StatusNotFound, "theme not found")
		return
	}
	JSON(w, http.StatusOK, item)
}

func (h *ThemeHandler) Create(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var theme domain.Theme
	if err := Decode(r, &theme); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	theme.CreatedBy = &userID
	theme.UpdatedBy = &userID

	if err := h.svc.Create(r.Context(), companyID, &theme); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusCreated, theme)
}

func (h *ThemeHandler) Update(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var theme domain.Theme
	if err := Decode(r, &theme); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	theme.ID = id
	theme.UpdatedBy = &userID

	if err := h.svc.Update(r.Context(), companyID, &theme); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusOK, theme)
}

func (h *ThemeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.Delete(r.Context(), companyID, id); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ThemeHandler) ListQA(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	themeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	filter := domain.QAPairFilter{
		Page:  intQuery(r, "page", 1),
		Limit: intQuery(r, "limit", 50),
	}

	items, total, err := h.svc.ListQA(r.Context(), companyID, themeID, filter)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSONList(w, http.StatusOK, items, total)
}
