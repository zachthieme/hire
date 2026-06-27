package api

import (
	"context"
	"encoding/json"
	"hire/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRequireRoleAllowed(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "admin@test.com", Name: "Admin", PasswordHash: hash, Role: "admin"}
	s.CreateUser(context.Background(), u)
	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Group(func(r chi.Router) {
		r.Use(h.RequireRole("admin"))
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 200, map[string]string{"ok": "true"})
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestRequireRoleDenied(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "iv@test.com", Name: "IV", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(context.Background(), u)
	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Group(func(r chi.Router) {
		r.Use(h.RequireRole("admin"))
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 200, map[string]string{"ok": "true"})
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestRequireRoleMultipleRolesAllowed(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), u)
	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Group(func(r chi.Router) {
		r.Use(h.RequireRole("scheduler", "admin"))
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 200, map[string]string{"ok": "true"})
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestAuthMiddlewareNoToken(t *testing.T) {
	h, _ := newTestHandler(t)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, nil)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	h, _ := newTestHandler(t)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, nil)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	h, _ := newTestHandler(t)

	r := chi.NewRouter()
	r.Use(h.RequestIDMiddleware)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		rid := RequestID(r.Context())
		writeJSON(w, 200, map[string]string{"request_id": rid})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	rid := w.Header().Get("X-Request-ID")
	if rid == "" {
		t.Fatal("expected X-Request-ID header")
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["request_id"] != rid {
		t.Errorf("context request_id = %q, header = %q", resp["request_id"], rid)
	}
}

func TestSecurityHeaders(t *testing.T) {
	h, _ := newTestHandler(t)
	r := h.Router()

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	expected := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
		"X-XSS-Protection":      "1; mode=block",
	}
	for header, want := range expected {
		got := w.Header().Get(header)
		if got != want {
			t.Errorf("header %s = %q, want %q", header, got, want)
		}
	}
}
