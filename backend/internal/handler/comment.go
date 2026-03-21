package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
)

type CommentHandler struct {
	svc *service.CommentService
}

func NewCommentHandler(svc *service.CommentService) *CommentHandler {
	return &CommentHandler{svc: svc}
}

func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	entityType := middleware.GetEntityType(r.Context())
	entityID, err := uuid.Parse(chi.URLParam(r, "entityID"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid entity id")
		return
	}

	filter := domain.CommentFilter{
		Page:  intQuery(r, "page", 1),
		Limit: intQuery(r, "limit", 50),
	}

	items, total, err := h.svc.List(r.Context(), companyID, entityType, entityID, filter)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSONList(w, http.StatusOK, items, total)
}

func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	userID := middleware.GetUserID(r.Context())
	entityType := middleware.GetEntityType(r.Context())
	entityID, err := uuid.Parse(chi.URLParam(r, "entityID"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid entity id")
		return
	}

	var comment domain.Comment
	if err := Decode(r, &comment); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	comment.EntityType = entityType
	comment.EntityID = entityID
	comment.AuthorID = &userID

	if err := h.svc.Create(r.Context(), companyID, &comment); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusCreated, comment)
}

func (h *CommentHandler) Update(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var comment domain.Comment
	if err := Decode(r, &comment); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	comment.ID = id

	if err := h.svc.Update(r.Context(), companyID, &comment); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusOK, comment)
}

func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
