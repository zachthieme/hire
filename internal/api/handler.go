package api

import (
	"encoding/json"
	"hire/internal/store"
	"net/http"
)

type Handler struct {
	store     *store.Store
	jwtSecret []byte
}

func NewHandler(s *store.Store, jwtSecret string) *Handler {
	return &Handler{store: s, jwtSecret: []byte(jwtSecret)}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
