package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) actor(r *http.Request) service.Actor {
	return service.Actor{
		ID:        middleware.GetUserID(r.Context()),
		Role:      middleware.GetRole(r.Context()),
		CompanyID: middleware.GetCompanyID(r.Context()),
	}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.UserFilter{
		Page:  intQuery(r, "page", 1),
		Limit: intQuery(r, "limit", 20),
		Query: r.URL.Query().Get("q"),
		Sort:  r.URL.Query().Get("sort"),
	}
	if roleParam := r.URL.Query().Get("role"); roleParam != "" {
		role := domain.Role(roleParam)
		if !domain.ValidRole(role) {
			Error(w, http.StatusBadRequest, "invalid role filter")
			return
		}
		filter.Role = &role
	}

	companyID := middleware.GetCompanyID(r.Context())
	items, total, err := h.svc.List(r.Context(), companyID, filter)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	JSONList(w, http.StatusOK, items, total)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req service.CreateUserRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.svc.Create(r.Context(), h.actor(r), req)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusCreated, user)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req service.UpdateUserRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.svc.Update(r.Context(), h.actor(r), id, req)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusOK, user)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.Delete(r.Context(), h.actor(r), id); err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
