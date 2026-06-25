package handler

import (
	applog "github.com/knowledgeos/backend/internal/logger"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
)

type BotHandler struct {
	settings *service.BotSettingsService
	secrets  *service.TenantSecretService
}

// NewBotHandler executes the handler.NewBotHandler operation.
func NewBotHandler(settings *service.BotSettingsService, secrets *service.TenantSecretService) *BotHandler {
	return &BotHandler{settings: settings, secrets: secrets}
}

// GetSettings executes the handler.BotHandler.GetSettings operation.
func (h *BotHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.BotHandler.GetSettings")
	companyID := middleware.GetCompanyID(r.Context())
	settings, err := h.settings.Get(r.Context(), companyID)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusOK, settings)
}

// UpdateSettings executes the handler.BotHandler.UpdateSettings operation.
func (h *BotHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.BotHandler.UpdateSettings")
	var req service.UpdateBotSettingsRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	companyID := middleware.GetCompanyID(r.Context())
	settings, err := h.settings.Update(r.Context(), companyID, req)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusOK, settings)
}

// ListSecrets executes the handler.BotHandler.ListSecrets operation.
func (h *BotHandler) ListSecrets(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.BotHandler.ListSecrets")
	companyID := middleware.GetCompanyID(r.Context())
	statuses, err := h.secrets.ListStatus(r.Context(), companyID)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusOK, statuses)
}

// SetSecret executes the handler.BotHandler.SetSecret operation.
func (h *BotHandler) SetSecret(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.BotHandler.SetSecret")
	kind := domain.SecretKind(chi.URLParam(r, "kind"))
	var req service.SetTenantSecretRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	companyID := middleware.GetCompanyID(r.Context())
	status, err := h.secrets.Set(r.Context(), companyID, kind, req)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusOK, status)
}

// DeleteSecret executes the handler.BotHandler.DeleteSecret operation.
func (h *BotHandler) DeleteSecret(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.BotHandler.DeleteSecret")
	kind := domain.SecretKind(chi.URLParam(r, "kind"))
	companyID := middleware.GetCompanyID(r.Context())
	if err := h.secrets.Delete(r.Context(), companyID, kind); err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
