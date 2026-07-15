package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	applog "github.com/knowledgeos/backend/internal/logger"
	"net/http"
	"strings"

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
		ServiceError(w, r, err)
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
		ServiceError(w, r, err)
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
		ServiceError(w, r, err)
		return
	}
	JSON(w, http.StatusOK, statuses)
}

// GetSecretForEdit returns the decrypted secret for the admin editor.
func (h *BotHandler) GetSecretForEdit(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.BotHandler.GetSecretForEdit")
	kind := domain.SecretKind(chi.URLParam(r, "kind"))
	if !domain.ValidSecretKind(kind) {
		Error(w, http.StatusBadRequest, "invalid secret kind")
		return
	}
	companyID := middleware.GetCompanyID(r.Context())
	secret, err := h.secrets.GetEditable(r.Context(), companyID, kind)
	if err != nil {
		ServiceError(w, r, err)
		return
	}
	JSON(w, http.StatusOK, secret)
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
	// Telegram verifies webhooks via a secret token. Auto-generate one if the
	// admin did not supply it, so registration and verification share a secret
	// without manual setup.
	if kind == domain.SecretKindTelegram {
		meta, err := ensureTelegramWebhookSecret(req.Metadata)
		if err != nil {
			Error(w, http.StatusInternalServerError, "failed to generate webhook secret")
			return
		}
		req.Metadata = meta
	}
	companyID := middleware.GetCompanyID(r.Context())
	status, err := h.secrets.Set(r.Context(), companyID, kind, req)
	if err != nil {
		ServiceError(w, r, err)
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
		ServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ensureTelegramWebhookSecret injects a random `webhook_secret` into the
// secret metadata when one is not already present. Invalid (non-object)
// metadata is returned untouched so the service layer can reject it.
func ensureTelegramWebhookSecret(metadata json.RawMessage) (json.RawMessage, error) {
	m := map[string]any{}
	if trimmed := bytes.TrimSpace(metadata); len(trimmed) > 0 {
		if err := json.Unmarshal(trimmed, &m); err != nil {
			return metadata, nil // not an object; let TenantSecretService.Set reject it
		}
	}
	if s, _ := m["webhook_secret"].(string); strings.TrimSpace(s) != "" {
		return metadata, nil
	}
	secret, err := randomWebhookSecret()
	if err != nil {
		return nil, err
	}
	m["webhook_secret"] = secret
	out, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// randomWebhookSecret returns a 64-char hex token (Telegram allows 1-256 chars
// of A-Z, a-z, 0-9, _ and -).
func randomWebhookSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func isChannelSecret(kind domain.SecretKind) bool {
	switch kind {
	case domain.SecretKindTelegram, domain.SecretKindMAX, domain.SecretKindVK:
		return true
	default:
		return false
	}
}
