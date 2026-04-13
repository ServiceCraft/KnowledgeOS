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
	filter.Sort = r.URL.Query().Get("sort")
	if aiStatus := r.URL.Query().Get("ai_status"); aiStatus != "" {
		filter.AIStatus = &aiStatus
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

	var patch map[string]interface{}
	if err := Decode(r, &patch); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := h.svc.GetByID(r.Context(), companyID, id)
	if err != nil {
		Error(w, http.StatusNotFound, "qa pair not found")
		return
	}

	if v, ok := patch["question"].(string); ok {
		existing.Question = v
	}
	if v, ok := patch["answer"].(string); ok {
		existing.Answer = v
	}
	if v, ok := patch["is_faq"].(bool); ok {
		existing.IsFAQ = v
	}
	if v, ok := patch["is_locked"].(bool); ok {
		existing.IsLocked = v
	}
	if v, ok := patch["frequency"].(float64); ok {
		existing.Frequency = int(v)
	}
	if v, ok := patch["theme_id"].(string); ok {
		tid, err := uuid.Parse(v)
		if err == nil {
			existing.ThemeID = &tid
		}
	}
	if _, ok := patch["theme_id"]; ok && patch["theme_id"] == nil {
		existing.ThemeID = nil
	}
	if v, ok := patch["ai_answer"].(string); ok {
		existing.AIAnswer = &v
	}
	if v, ok := patch["ai_status"].(string); ok {
		existing.AIStatus = &v
	}
	existing.UpdatedBy = &userID

	if err := h.svc.Update(r.Context(), companyID, existing); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusOK, existing)
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

func (h *QAHandler) Review(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Action       string `json:"action"`
		EditedAnswer string `json:"edited_answer"`
	}
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.ReviewAIAnswer(r.Context(), companyID, id, userID, req.Action, req.EditedAnswer)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusOK, result)
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
