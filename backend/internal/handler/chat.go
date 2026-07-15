package handler

import (
	"context"
	"encoding/json"
	"fmt"
	applog "github.com/knowledgeos/backend/internal/logger"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/middleware"
	"github.com/knowledgeos/backend/internal/service"
)

type ChatHandler struct {
	svc *service.ChatService
}

// NewChatHandler executes the handler.NewChatHandler operation.
func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

// CreateSession executes the handler.ChatHandler.CreateSession operation.
func (h *ChatHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.ChatHandler.CreateSession")
	var req service.CreateChatSessionRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	session, err := h.svc.CreateSession(r.Context(), middleware.GetCompanyID(r.Context()), req)
	if err != nil {
		ServiceError(w, r, err)
		return
	}
	JSON(w, http.StatusCreated, session)
}

// ListSessions executes the handler.ChatHandler.ListSessions operation.
func (h *ChatHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.ChatHandler.ListSessions")
	sessions, total, err := h.svc.ListSessions(r.Context(), middleware.GetCompanyID(r.Context()), domain.ChatSessionFilter{
		Page:  intQuery(r, "page", 1),
		Limit: intQuery(r, "limit", 30),
	})
	if err != nil {
		ServiceError(w, r, err)
		return
	}
	JSONList(w, http.StatusOK, sessions, total)
}

// GetSession executes the handler.ChatHandler.GetSession operation.
func (h *ChatHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.ChatHandler.GetSession")
	sessionID, ok := chatSessionID(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetSession(r.Context(), middleware.GetCompanyID(r.Context()), sessionID)
	if err != nil {
		ServiceError(w, r, err)
		return
	}
	JSON(w, http.StatusOK, result)
}

// DeleteSession permanently removes a chat session and all of its messages.
func (h *ChatHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.ChatHandler.DeleteSession")
	sessionID, ok := chatSessionID(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteSession(r.Context(), middleware.GetCompanyID(r.Context()), sessionID); err != nil {
		Error(w, service.HTTPStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SendMessage executes the handler.ChatHandler.SendMessage operation.
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.ChatHandler.SendMessage")
	sessionID, ok := chatSessionID(w, r)
	if !ok {
		return
	}
	var req service.SendChatMessageRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.SendMessage(r.Context(), middleware.GetCompanyID(r.Context()), sessionID, req)
	if err != nil {
		ServiceError(w, r, err)
		return
	}
	JSON(w, http.StatusOK, result)
}

// StreamMessage executes the handler.ChatHandler.StreamMessage operation.
func (h *ChatHandler) StreamMessage(w http.ResponseWriter, r *http.Request) {
	applog.TraceCall(r.Context(), "handler.ChatHandler.StreamMessage")
	sessionID, ok := chatSessionID(w, r)
	if !ok {
		return
	}
	var req service.SendChatMessageRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		Error(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	ctx := r.Context()
	companyID := middleware.GetCompanyID(ctx)
	// Keep generating and persisting even if the browser tab closes mid-stream.
	streamCtx := context.WithoutCancel(ctx)
	emit := func(event service.ChatStreamEvent) error {
		if event.Type == "" {
			event.Type = "message"
		}
		data, err := json.Marshal(event)
		if err != nil {
			return applog.TraceErr(ctx, "chat sse: marshal event failed", err)
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data); err != nil {
			applog.From(ctx).Debug().Err(err).Msg("chat sse: client disconnected")
			return nil
		}
		flusher.Flush()
		return nil
	}
	if err := h.svc.StreamMessage(streamCtx, companyID, sessionID, req, emit); err != nil {
		_ = emit(service.ChatStreamEvent{Type: "error", Error: err.Error()})
		_ = emit(service.ChatStreamEvent{Type: "done"})
	}
}

func chatSessionID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	applog.TraceCall(r.Context(), "handler.chatSessionID")
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid session id")
		return uuid.Nil, false
	}
	return id, true
}
