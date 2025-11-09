package svc

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/4n4nd/cap-connectors-spotify/internal/spotify"
)

// Service coordinates HTTP handlers with the Spotify client.
type Service struct {
	client spotify.Client
}

// New constructs a Service.
func New(client spotify.Client) *Service {
	return &Service{client: client}
}

// HandleListDevices returns the handler for POST /v1/spotify/devices:list.
func (s *Service) HandleListDevices() http.HandlerFunc {
	type request struct {
		UserID string `json:"user_id"`
	}
	type response struct {
		Devices []spotify.Device `json:"devices"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.UserID) == "" {
			writeClientError(w, http.StatusBadRequest, "invalid user_id")
			return
		}

		devs, err := s.client.ListDevices(r.Context(), body.UserID)
		if err != nil {
			writeSpotifyError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, response{Devices: devs})
	}
}

// HandleTransferPlayback manages POST /v1/spotify/playback:transfer.
func (s *Service) HandleTransferPlayback() http.HandlerFunc {
	type request struct {
		UserID   string `json:"user_id"`
		DeviceID string `json:"device_id"`
		PlayNow  bool   `json:"play_now"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
			strings.TrimSpace(body.UserID) == "" ||
			strings.TrimSpace(body.DeviceID) == "" {
			writeClientError(w, http.StatusBadRequest, "invalid request")
			return
		}

		if err := s.client.TransferPlayback(r.Context(), body.UserID, body.DeviceID, body.PlayNow); err != nil {
			writeSpotifyError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// HandleAddToQueue manages POST /v1/spotify/queue:append.
func (s *Service) HandleAddToQueue() http.HandlerFunc {
	type request struct {
		UserID   string `json:"user_id"`
		DeviceID string `json:"device_id"`
		URI      string `json:"uri"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
			strings.TrimSpace(body.UserID) == "" ||
			strings.TrimSpace(body.DeviceID) == "" ||
			!strings.HasPrefix(body.URI, "spotify:track:") {
			writeClientError(w, http.StatusBadRequest, "invalid request")
			return
		}

		if err := s.client.AddToQueue(r.Context(), body.UserID, body.DeviceID, body.URI); err != nil {
			writeSpotifyError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// HandleSearchTracks manages GET /v1/spotify/tracks:search.
func (s *Service) HandleSearchTracks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" {
			writeClientError(w, http.StatusBadRequest, "missing q")
			return
		}

		limit := 20
		if raw := r.URL.Query().Get("limit"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 || n > 50 {
				writeClientError(w, http.StatusBadRequest, "invalid limit")
				return
			}
			limit = n
		}

		market := strings.TrimSpace(r.URL.Query().Get("market"))
		items, err := s.client.SearchTracks(r.Context(), q, limit, market)
		if err != nil {
			writeSpotifyError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

// HandleResolveByISRC manages GET /v1/spotify/tracks:by-isrc.
func (s *Service) HandleResolveByISRC() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isrc := strings.TrimSpace(r.URL.Query().Get("isrc"))
		if isrc == "" {
			writeClientError(w, http.StatusBadRequest, "missing isrc")
			return
		}

		track, err := s.client.ResolveByISRC(r.Context(), isrc)
		if err != nil {
			writeSpotifyError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, track)
	}
}

func writeClientError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeSpotifyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, spotify.ErrUnauthorized):
		writeClientError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, spotify.ErrNotFound):
		writeClientError(w, http.StatusNotFound, "not found")
	case errors.Is(err, spotify.ErrRateLimited):
		writeClientError(w, http.StatusTooManyRequests, "rate limited")
	default:
		writeClientError(w, http.StatusBadGateway, "upstream failure")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}
