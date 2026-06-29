package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	applog "github.com/knowledgeos/backend/internal/logger"
	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
)

type HandoffHandler struct {
	svc *service.HandoffService
}

func NewHandoffHandler(svc *service.HandoffService) *HandoffHandler {
	return &HandoffHandler{svc: svc}
}

func (h *HandoffHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.HandoffHandler.ListSessions")
	filter, ok := handoffFilter(w, r)
	if !ok {
		return
	}
	sessions, total, err := h.svc.ListSessions(r.Context(), middleware.GetCompanyID(r.Context()), handoffActor(r), filter)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSONList(w, http.StatusOK, sessions, total)
}

func (h *HandoffHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.HandoffHandler.Metrics")
	metrics, err := h.svc.Metrics(r.Context(), middleware.GetCompanyID(r.Context()), handoffActor(r))
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusOK, metrics)
}

func (h *HandoffHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.HandoffHandler.GetSession")
	sessionID, ok := handoffSessionID(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetSession(r.Context(), middleware.GetCompanyID(r.Context()), handoffActor(r), sessionID)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusOK, result)
}

func (h *HandoffHandler) Claim(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.HandoffHandler.Claim")
	sessionID, ok := handoffSessionID(w, r)
	if !ok {
		return
	}
	session, err := h.svc.Claim(r.Context(), middleware.GetCompanyID(r.Context()), handoffActor(r), sessionID)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusOK, session)
}

func (h *HandoffHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.HandoffHandler.SendMessage")
	sessionID, ok := handoffSessionID(w, r)
	if !ok {
		return
	}
	var req service.SendOperatorMessageRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	message, err := h.svc.SendOperatorMessage(r.Context(), middleware.GetCompanyID(r.Context()), handoffActor(r), sessionID, req)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusCreated, message)
}

func (h *HandoffHandler) Release(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.HandoffHandler.Release")
	sessionID, ok := handoffSessionID(w, r)
	if !ok {
		return
	}
	session, err := h.svc.ReleaseToBot(r.Context(), middleware.GetCompanyID(r.Context()), handoffActor(r), sessionID)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusOK, session)
}

func (h *HandoffHandler) Close(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.HandoffHandler.Close")
	sessionID, ok := handoffSessionID(w, r)
	if !ok {
		return
	}
	session, err := h.svc.Close(r.Context(), middleware.GetCompanyID(r.Context()), handoffActor(r), sessionID)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusOK, session)
}

func (h *HandoffHandler) Escalate(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.HandoffHandler.Escalate")
	sessionID, ok := handoffSessionID(w, r)
	if !ok {
		return
	}
	var req service.EscalateSessionRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	session, err := h.svc.Escalate(r.Context(), middleware.GetCompanyID(r.Context()), sessionID, req.Reason)
	if err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	JSON(w, http.StatusOK, session)
}

func handoffActor(r *http.Request) service.HandoffActor {
	return service.HandoffActor{
		UserID: middleware.GetUserID(r.Context()),
		Role:   middleware.GetRole(r.Context()),
	}
}

func handoffSessionID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid session id")
		return uuid.Nil, false
	}
	return id, true
}

func handoffFilter(w http.ResponseWriter, r *http.Request) (service.HandoffSessionFilter, bool) {
	filter := service.HandoffSessionFilter{
		Page:  intQuery(r, "page", 1),
		Limit: intQuery(r, "limit", 30),
	}
	if raw := r.URL.Query().Get("state"); raw != "" {
		state := domain.ChatState(raw)
		if !validChatState(state) {
			Error(w, http.StatusBadRequest, "invalid state")
			return filter, false
		}
		filter.State = &state
	}
	if raw := r.URL.Query().Get("operator_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			Error(w, http.StatusBadRequest, "invalid operator_id")
			return filter, false
		}
		filter.OperatorID = &id
	}
	if raw := r.URL.Query().Get("channel"); raw != "" {
		channel := domain.ChatChannel(raw)
		if !validHandoffChannel(channel) {
			Error(w, http.StatusBadRequest, "invalid channel")
			return filter, false
		}
		filter.Channel = &channel
	}
	if raw := r.URL.Query().Get("active"); raw != "" {
		switch raw {
		case "true":
			filter.Active = true
		case "false":
			filter.Active = false
		default:
			Error(w, http.StatusBadRequest, "invalid active")
			return filter, false
		}
	}
	return filter, true
}

func validChatState(state domain.ChatState) bool {
	switch state {
	case domain.ChatStateBot, domain.ChatStateWaitingOperator, domain.ChatStateOperator, domain.ChatStateClosed:
		return true
	default:
		return false
	}
}

func validHandoffChannel(channel domain.ChatChannel) bool {
	switch channel {
	case domain.ChatChannelPlayground, domain.ChatChannelAPI, domain.ChatChannelTelegram, domain.ChatChannelMAX, domain.ChatChannelVK:
		return true
	default:
		return false
	}
}
