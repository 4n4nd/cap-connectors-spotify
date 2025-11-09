package http

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// WriteJSON renders the payload as JSON with the provided status code.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
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
