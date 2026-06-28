package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hire/internal/models"

	"github.com/go-chi/chi/v5"
)

func TestCreateJobRequiresTitle(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), u)
	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/jobs", h.CreateJob)

	req := httptest.NewRequest("POST", "/api/jobs", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateJobSuccess(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), u)
	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/jobs", h.CreateJob)

	body, _ := json.Marshal(map[string]any{"title": "Backend Engineer", "hiring_manager": "Dana"})
	req := httptest.NewRequest("POST", "/api/jobs", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created models.Job
	json.Unmarshal(w.Body.Bytes(), &created)
	if created.ID == 0 {
		t.Fatal("expected job ID in response")
	}
	if created.Status != models.JobStatusOpen {
		t.Errorf("status = %q, want open", created.Status)
	}
}
