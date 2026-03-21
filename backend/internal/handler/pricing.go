package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
)

type PricingHandler struct {
	svc *service.PricingService
}

func NewPricingHandler(svc *service.PricingService) *PricingHandler {
	return &PricingHandler{svc: svc}
}

func (h *PricingHandler) List(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	filter := domain.PricingNodeFilter{
		Page:     intQuery(r, "page", 1),
		Limit:    intQuery(r, "limit", 50),
		NodeType: r.URL.Query().Get("node_type"),
	}
	if pid := r.URL.Query().Get("parent_id"); pid != "" {
		id, err := uuid.Parse(pid)
		if err == nil {
			filter.ParentID = &id
		}
	}

	items, total, err := h.svc.List(r.Context(), companyID, filter)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSONList(w, http.StatusOK, items, total)
}

func (h *PricingHandler) Get(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.svc.GetByID(r.Context(), companyID, id)
	if err != nil {
		Error(w, http.StatusNotFound, "pricing node not found")
		return
	}
	JSON(w, http.StatusOK, item)
}

func (h *PricingHandler) Create(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var node domain.PricingNode
	if err := Decode(r, &node); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	node.CreatedBy = &userID
	node.UpdatedBy = &userID

	if err := h.svc.Create(r.Context(), companyID, &node); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusCreated, node)
}

func (h *PricingHandler) Update(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var node domain.PricingNode
	if err := Decode(r, &node); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	node.ID = id
	node.UpdatedBy = &userID

	if err := h.svc.Update(r.Context(), companyID, &node); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusOK, node)
}

func (h *PricingHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

type moveRequest struct {
	ParentID *uuid.UUID `json:"parent_id"`
}

func (h *PricingHandler) Move(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req moveRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.Move(r.Context(), companyID, id, req.ParentID); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
