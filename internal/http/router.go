package http

import (
	"net/http"

	"github.com/4n4nd/cap-connectors-spotify/internal/svc"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

// NewRouter wires routes and middlewares.
func NewRouter(logger zerolog.Logger, service *svc.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(RequestID)
	r.Use(AccessLog(logger))
	r.Use(Recoverer(logger))
	r.Use(middleware.Compress(5))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	r.Route("/v1/spotify", func(r chi.Router) {
		r.Post("/devices:list", service.HandleListDevices())
		r.Post("/playback:transfer", service.HandleTransferPlayback())
		r.Post("/queue:append", service.HandleAddToQueue())
		r.Get("/tracks:search", service.HandleSearchTracks())
		r.Get("/tracks:by-isrc", service.HandleResolveByISRC())
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	})

	return r
}
