package handler

import (
	"bytes"
	"context"
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

// YClientsBranchInfo is one filial discovered from a partner token.
type YClientsBranchInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

// YClientsBranchFetcher auto-discovers the filials for a YClients partner token
// so the admin does not enter them by hand. It is best-effort: when it fails the
// secret is still saved with whatever the admin provided.
type YClientsBranchFetcher func(ctx context.Context, token, baseURL, groupID, companyID string) ([]YClientsBranchInfo, error)

type BotHandler struct {
	settings         *service.BotSettingsService
	secrets          *service.TenantSecretService
	yclientsBranches YClientsBranchFetcher
}

// NewBotHandler executes the handler.NewBotHandler operation.
func NewBotHandler(settings *service.BotSettingsService, secrets *service.TenantSecretService, yclientsBranches YClientsBranchFetcher) *BotHandler {
	return &BotHandler{settings: settings, secrets: secrets, yclientsBranches: yclientsBranches}
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
	// YClients: auto-discover the clinic's filials from the partner token so the
	// admin only enters the key (plus optionally one company_id/group_id to scope
	// the chain). Best-effort — a fetch failure must not block saving the key.
	if kind == domain.SecretKindYClients {
		req.Metadata = h.enrichYClientsBranches(r.Context(), req.Value, req.Metadata)
	}
	companyID := middleware.GetCompanyID(r.Context())
	status, err := h.secrets.Set(r.Context(), companyID, kind, req)
	if err != nil {
		ServiceError(w, r, err)
		return
	}
	JSON(w, http.StatusOK, status)
}

// enrichYClientsBranches fetches the filials for the given partner token and
// writes them into the secret metadata under "branches". It never fails the
// request: on any error it returns the metadata unchanged.
func (h *BotHandler) enrichYClientsBranches(ctx context.Context, token string, metadata json.RawMessage) json.RawMessage {
	if h.yclientsBranches == nil || strings.TrimSpace(token) == "" {
		return metadata
	}
	meta := map[string]json.RawMessage{}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &meta); err != nil {
			return metadata
		}
	}
	baseURL := metaScalar(meta["api_base"])
	groupID := metaScalar(meta["group_id"])
	companyID := metaScalar(meta["company_id"])
	branches, err := h.yclientsBranches(ctx, strings.TrimSpace(token), baseURL, groupID, companyID)
	if err != nil {
		applog.From(ctx).Warn().Err(err).Msg("yclients: auto-fetch branches failed; saving key without branch list")
		return metadata
	}
	if len(branches) == 0 {
		return metadata
	}
	raw, err := json.Marshal(branches)
	if err != nil {
		return metadata
	}
	meta["branches"] = raw
	out, err := json.Marshal(meta)
	if err != nil {
		return metadata
	}
	return out
}

// metaScalar coerces a JSON string or number metadata value to a trimmed string.
func metaScalar(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	return ""
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
