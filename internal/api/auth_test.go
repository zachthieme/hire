package api

import (
	"bytes"
	"encoding/json"
	"hire/internal/models"
	"hire/internal/store"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newTestHandler(t *testing.T) (*Handler, *store.Store) {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("newTestHandler: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	h := NewHandler(s, "test-secret")
	return h, s
}

func TestLoginSuccess(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("password123")
	s.CreateUser(&models.User{Email: "test@test.com", Name: "Test", PasswordHash: hash, Role: "interviewer"})

	r := chi.NewRouter()
	r.Post("/api/auth/login", h.Login)

	body, _ := json.Marshal(map[string]string{"email": "test@test.com", "password": "password123"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] == nil || resp["token"] == "" {
		t.Fatal("expected token in response")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("correct")
	s.CreateUser(&models.User{Email: "test@test.com", Name: "Test", PasswordHash: hash, Role: "interviewer"})

	r := chi.NewRouter()
	r.Post("/api/auth/login", h.Login)

	body, _ := json.Marshal(map[string]string{"email": "test@test.com", "password": "wrong"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestAuthMiddleware(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "a@a.com", Name: "A", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(u)

	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"user_id": UserID(r.Context()),
			"role":    UserRole(r.Context()),
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["role"] != "scheduler" {
		t.Errorf("role = %v, want scheduler", resp["role"])
	}
}
