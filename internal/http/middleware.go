package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type ctxKey string

const (
	requestIDKey    ctxKey = "request_id"
	requestIDHeader        = "X-Request-Id"
)

// RequestID injects a UUIDv4 into the request context and response headers.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Recoverer converts panics into 500 responses with a stable body.
func Recoverer(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error().
						Interface("panic", rec).
						Str("request_id", RequestIDFromContext(r.Context())).
						Msg("panic recovered")
					WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// AccessLog records structured access logs for each request.
func AccessLog(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)

			logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.Status()).
				Dur("latency", time.Since(start)).
				Str("request_id", RequestIDFromContext(r.Context())).
				Msg("http request")
		})
	}
}

// RequestIDFromContext fetches the request ID populated by the middleware.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}
