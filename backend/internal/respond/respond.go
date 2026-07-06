package respond

import (
	"encoding/json"
	applog "github.com/knowledgeos/backend/internal/logger"
	"net/http"
)

type envelope struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
	Total *int64      `json:"total,omitempty"`
}

// JSON executes the respond.JSON operation.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Data: data})
}

// JSONList executes the respond.JSONList operation.
func JSONList(w http.ResponseWriter, status int, data interface{}, total int64) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Data: data, Total: &total})
}

// Error executes the respond.Error operation.
func Error(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Error: msg})
}

// Decode executes the respond.Decode operation.
func Decode(r *http.Request, v interface{}) error {
	applog.TraceCall(r.Context(), "respond.Decode")
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
