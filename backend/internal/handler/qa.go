package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
)

type QAHandler struct {
	svc *service.QAService
}

func NewQAHandler(svc *service.QAService) *QAHandler {
	return &QAHandler{svc: svc}
}

func (h *QAHandler) List(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	filter := domain.QAPairFilter{
		Page:  intQuery(r, "page", 1),
		Limit: intQuery(r, "limit", 50),
		Query: r.URL.Query().Get("query"),
	}
	if tid := r.URL.Query().Get("theme_id"); tid != "" {
		id, err := uuid.Parse(tid)
		if err == nil {
			filter.ThemeID = &id
		}
	}
	if faq := r.URL.Query().Get("is_faq"); faq != "" {
		val := faq == "true"
		filter.IsFAQ = &val
	}

	items, total, err := h.svc.List(r.Context(), companyID, filter)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSONList(w, http.StatusOK, items, total)
}

func (h *QAHandler) Get(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.svc.GetByID(r.Context(), companyID, id)
	if err != nil {
		Error(w, http.StatusNotFound, "qa pair not found")
		return
	}
	JSON(w, http.StatusOK, item)
}

func (h *QAHandler) Create(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var qa domain.QAPair
	if err := Decode(r, &qa); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	qa.CreatedBy = &userID
	qa.UpdatedBy = &userID

	if err := h.svc.Create(r.Context(), companyID, &qa); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusCreated, qa)
}

func (h *QAHandler) Update(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var qa domain.QAPair
	if err := Decode(r, &qa); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	qa.ID = id
	qa.UpdatedBy = &userID

	if err := h.svc.Update(r.Context(), companyID, &qa); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusOK, qa)
}

func (h *QAHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func intQuery(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
