package handler

import (
	applog "github.com/knowledgeos/backend/internal/logger"
	"net/http"

	"github.com/knowledgeos/backend/internal/respond"
	"github.com/knowledgeos/backend/internal/service"
)

// JSON executes the handler.JSON operation.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	respond.JSON(w, status, data)
}

// JSONList executes the handler.JSONList operation.
func JSONList(w http.ResponseWriter, status int, data interface{}, total int64) {
	respond.JSONList(w, status, data, total)
}

// Error executes the handler.Error operation.
func Error(w http.ResponseWriter, status int, msg string) {
	respond.Error(w, status, msg)
}

// ServiceError writes a client-safe error for a service failure. Known
// UserError/AuthError messages are preserved; unexpected internal errors are
// reduced to a generic message and logged with full detail server-side.
func ServiceError(w http.ResponseWriter, r *http.Request, err error) {
	status, msg := service.SafeError(err)
	if status >= http.StatusInternalServerError {
		applog.From(r.Context()).Error().Err(err).Int("status", status).Msg("request failed")
	}
	respond.Error(w, status, msg)
}

// Decode executes the handler.Decode operation.
func Decode(r *http.Request, v interface{}) error {
	applog.TraceCall(r.Context(), "handler.Decode")
	return respond.Decode(r, v)
}
