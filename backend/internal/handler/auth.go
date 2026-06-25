package handler

import (
	applog "github.com/knowledgeos/backend/internal/logger"
	"net/http"

	"github.com/knowledgeos/backend/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler executes the handler.NewAuthHandler operation.
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Login executes the handler.AuthHandler.Login operation.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.AuthHandler.Login")
	var req service.LoginRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pair, err := h.svc.Login(r.Context(), req)
	if err != nil {
		Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	JSON(w, http.StatusOK, pair)
}

// Refresh executes the handler.AuthHandler.Refresh operation.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.AuthHandler.Refresh")
	var req service.RefreshRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pair, err := h.svc.Refresh(r.Context(), req)
	if err != nil {
		Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	JSON(w, http.StatusOK, pair)
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout executes the handler.AuthHandler.Logout operation.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.AuthHandler.Logout")
	var req logoutRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
