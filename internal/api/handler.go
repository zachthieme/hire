package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type Handler struct {
	store       Store
	jwtSecret   []byte
	corsOrigins []string
}

func NewHandler(s Store, jwtSecret string, corsOrigins []string) *Handler {
	return &Handler{store: s, jwtSecret: []byte(jwtSecret), corsOrigins: corsOrigins}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1MB limit
	return json.NewDecoder(r.Body).Decode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeInternalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "internal error",
		"error", err,
		"method", r.Method,
		"path", r.URL.Path,
		"request_id", RequestID(r.Context()),
	)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}

func validateRequired(fields map[string]string) error {
	for name, val := range fields {
		if strings.TrimSpace(val) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	return nil
}

func validateEnum(value, name string, allowed []string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("%s must be one of: %s", name, strings.Join(allowed, ", "))
}

var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func validateEmail(email string) error {
	if !emailRegexp.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func validateRatingsJSON(ratingsJSON, ratingType string) error {
	switch ratingType {
	case "levels":
		var levels []string
		if err := json.Unmarshal([]byte(ratingsJSON), &levels); err != nil {
			return fmt.Errorf("ratings_json must be a JSON array of strings for levels")
		}
		if len(levels) == 0 {
			return fmt.Errorf("ratings_json must be non-empty for levels")
		}
		for _, l := range levels {
			if strings.TrimSpace(l) == "" {
				return fmt.Errorf("ratings_json must not contain empty strings")
			}
		}
	case "stars":
		var stars struct {
			Min int `json:"min"`
			Max int `json:"max"`
		}
		if err := json.Unmarshal([]byte(ratingsJSON), &stars); err != nil {
			return fmt.Errorf("ratings_json must be a JSON object with min and max for stars")
		}
		if stars.Max < 2 {
			return fmt.Errorf("ratings_json max must be >= 2 for stars")
		}
	}
	return nil
}

func validateURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}
	switch strings.ToLower(u.Scheme) {
	case "", "http", "https":
		return nil
	default:
		return fmt.Errorf("invalid URL scheme: only http and https are allowed")
	}
}

func parsePagination(r *http.Request) (limit, offset int) {
	limit = 50
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}

func generateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func validateTransition(current, proposed, entity string, transitions map[string][]string) error {
	allowed, ok := transitions[current]
	if !ok {
		return fmt.Errorf("unknown current %s status: %s", entity, current)
	}
	for _, a := range allowed {
		if proposed == a {
			return nil
		}
	}
	return fmt.Errorf("cannot transition %s from %s to %s", entity, current, proposed)
}
