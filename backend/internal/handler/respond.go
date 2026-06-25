package handler

import (
	applog "github.com/knowledgeos/backend/internal/logger"
	"net/http"

	"github.com/knowledgeos/backend/internal/respond"
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

// Decode executes the handler.Decode operation.
func Decode(r *http.Request, v interface{}) error {
	applog.TraceCall(r.Context(), "handler.Decode")
	return respond.Decode(r, v)
}
