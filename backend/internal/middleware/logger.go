package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	applog "github.com/knowledgeos/backend/internal/logger"
)

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

// WriteHeader executes the middleware.responseWriter.WriteHeader operation.
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write executes the middleware.responseWriter.Write operation.
func (rw *responseWriter) Write(p []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(p)
	rw.size += n
	return n, err
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Logger executes the middleware.Logger operation.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", requestID)

		reqLogger := applog.From(r.Context()).With().
			Str("request_id", requestID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Logger()
		ctx := reqLogger.WithContext(SetRequestID(r.Context(), requestID))

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		reqLogger.Debug().Msg("http request started")
		next.ServeHTTP(rw, r.WithContext(ctx))

		event := reqLogger.Info()
		switch {
		case rw.status >= http.StatusInternalServerError:
			event = reqLogger.Error()
		case rw.status >= http.StatusBadRequest:
			event = reqLogger.Warn()
		}
		event.
			Int("status", rw.status).
			Int("bytes", rw.size).
			Dur("duration", time.Since(start)).
			Msg("http request completed")
	})
}
