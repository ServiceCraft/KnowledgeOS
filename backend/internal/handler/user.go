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

type UserHandler struct {
	svc *service.UserService
}

// NewUserHandler executes the handler.NewUserHandler operation.
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) actor(r *http.Request) service.Actor {
	applog.TraceCall(r.Context(), "handler.UserHandler.actor")
	return service.Actor{
		ID:        middleware.GetUserID(r.Context()),
		Role:      middleware.GetRole(r.Context()),
		CompanyID: middleware.GetCompanyID(r.Context()),
	}
}

// List executes the handler.UserHandler.List operation.
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.UserHandler.List")
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

// Create executes the handler.UserHandler.Create operation.
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.UserHandler.Create")
	var req service.CreateUserRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.svc.Create(r.Context(), h.actor(r), req)
	if err != nil {
		ServiceError(w, r, err)
		return
	}
	JSON(w, http.StatusCreated, user)
}

// Update executes the handler.UserHandler.Update operation.
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.UserHandler.Update")
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
		ServiceError(w, r, err)
		return
	}
	JSON(w, http.StatusOK, user)
}

// Delete executes the handler.UserHandler.Delete operation.
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.UserHandler.Delete")
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.Delete(r.Context(), h.actor(r), id); err != nil {
		ServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
