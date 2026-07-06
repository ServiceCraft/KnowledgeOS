package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/channels"
	"github.com/knowledgeos/backend/internal/domain"
	applog "github.com/knowledgeos/backend/internal/logger"
	"github.com/knowledgeos/backend/internal/middleware"
)

const channelWebhookMaxBody = 1 << 20

type ChannelWebhookHandler struct {
	gateway              *channels.Gateway
	publicWebhookBaseURL string
}

func NewChannelWebhookHandler(gateway *channels.Gateway, publicWebhookBaseURL string) *ChannelWebhookHandler {
	return &ChannelWebhookHandler{gateway: gateway, publicWebhookBaseURL: publicWebhookBaseURL}
}

func (h *ChannelWebhookHandler) webhookBaseURL(r *http.Request) string {
	return channels.ResolveWebhookBaseURL(requestBaseURL(r), h.publicWebhookBaseURL)
}

func (h *ChannelWebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.ChannelWebhookHandler.Handle")
	companyID, channel, ok := channelRouteParams(w, r)
	if !ok {
		return
	}
	if retryCounter := strings.TrimSpace(r.Header.Get("X-Retry-Counter")); retryCounter != "" {
		applog.From(r.Context()).Warn().
			Str("channel", string(channel)).
			Str("company_id", companyID.String()).
			Str("retry_counter", retryCounter).
			Msg("channel webhook retry received")
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, channelWebhookMaxBody))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid webhook body")
		return
	}
	ctx := middleware.SetCompanyID(r.Context(), companyID)
	response, err := h.gateway.HandleWebhook(ctx, companyID, channel, channels.WebhookRequest{
		Headers: r.Header,
		Body:    body,
		Query:   r.URL.Query(),
	})
	if err != nil {
		ServiceError(w, r, err)
		return
	}
	writeWebhookResponse(w, response)
}

func (h *ChannelWebhookHandler) Status(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.ChannelWebhookHandler.Status")
	baseURL := h.webhookBaseURL(r)
	statuses, err := h.gateway.Status(r.Context(), middleware.GetCompanyID(r.Context()), baseURL)
	if err != nil {
		ServiceError(w, r, err)
		return
	}
	JSON(w, http.StatusOK, statuses)
}

// RegisterWebhook (re)registers the webhook for a channel using the stored
// secret and metadata. Triggered from the admin UI.
func (h *ChannelWebhookHandler) RegisterWebhook(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.ChannelWebhookHandler.RegisterWebhook")
	channel, ok := adminChannelParam(w, r)
	if !ok {
		return
	}
	companyID := middleware.GetCompanyID(r.Context())
	result, err := h.gateway.RegisterChannelWebhook(r.Context(), companyID, channel, h.webhookBaseURL(r))
	if err != nil {
		ServiceError(w, r, err)
		return
	}
	JSON(w, http.StatusOK, result)
}

// Subscriptions returns the raw subscriptions/webhook payload the channel
// provider reports for the stored bot token.
func (h *ChannelWebhookHandler) Subscriptions(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.ChannelWebhookHandler.Subscriptions")
	channel, ok := adminChannelParam(w, r)
	if !ok {
		return
	}
	companyID := middleware.GetCompanyID(r.Context())
	payload, err := h.gateway.ListSubscriptions(r.Context(), companyID, channel)
	if err != nil {
		ServiceError(w, r, err)
		return
	}
	JSON(w, http.StatusOK, payload)
}

func adminChannelParam(w http.ResponseWriter, r *http.Request) (domain.ChatChannel, bool) {
	channel := domain.ChatChannel(chi.URLParam(r, "channel"))
	switch channel {
	case domain.ChatChannelTelegram, domain.ChatChannelMAX, domain.ChatChannelVK:
		return channel, true
	default:
		Error(w, http.StatusBadRequest, "invalid channel")
		return "", false
	}
}

func channelRouteParams(w http.ResponseWriter, r *http.Request) (uuid.UUID, domain.ChatChannel, bool) {
	companyID, err := uuid.Parse(chi.URLParam(r, "company_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid company id")
		return uuid.Nil, "", false
	}
	channel := domain.ChatChannel(chi.URLParam(r, "channel"))
	switch channel {
	case domain.ChatChannelTelegram, domain.ChatChannelMAX, domain.ChatChannelVK:
		return companyID, channel, true
	default:
		Error(w, http.StatusBadRequest, "invalid channel")
		return uuid.Nil, "", false
	}
}

func writeWebhookResponse(w http.ResponseWriter, response *channels.WebhookResponse) {
	if response == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if response.ContentType != "" {
		w.Header().Set("Content-Type", response.ContentType)
	}
	status := response.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if len(response.Body) > 0 {
		_, _ = w.Write(response.Body)
	}
}

func requestBaseURL(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return strings.TrimRight(scheme+"://"+host, "/")
}
