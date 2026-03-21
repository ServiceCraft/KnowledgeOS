package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
)

type LinkHandler struct {
	svc *service.LinkService
}

func NewLinkHandler(svc *service.LinkService) *LinkHandler {
	return &LinkHandler{svc: svc}
}

func (h *LinkHandler) List(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	sourceType := middleware.GetEntityType(r.Context())
	sourceID, err := uuid.Parse(chi.URLParam(r, "entityID"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid entity id")
		return
	}

	filter := domain.EntityLinkFilter{
		Page:  intQuery(r, "page", 1),
		Limit: intQuery(r, "limit", 50),
	}

	items, total, err := h.svc.List(r.Context(), companyID, sourceType, sourceID, filter)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSONList(w, http.StatusOK, items, total)
}

func (h *LinkHandler) Create(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	sourceType := middleware.GetEntityType(r.Context())
	sourceID, err := uuid.Parse(chi.URLParam(r, "entityID"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid entity id")
		return
	}

	var link domain.EntityLink
	if err := Decode(r, &link); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	link.SourceType = sourceType
	link.SourceID = sourceID

	if err := h.svc.Create(r.Context(), companyID, &link); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusCreated, link)
}

func (h *LinkHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
