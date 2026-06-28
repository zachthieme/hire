package api

import (
	"bytes"
	"context"
	"encoding/json"
	"hire/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestCreateCandidateInvalidResumeURL(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	sched := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), sched)
	schedToken, _ := h.generateToken(sched.ID, sched.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Use(h.RequireRole("scheduler", "admin"))
	r.Post("/api/candidates", h.CreateCandidate)

	body, _ := json.Marshal(map[string]string{
		"name": "Jane", "email": "jane@test.com", "resume_url": "javascript:alert(1)",
	})
	req := httptest.NewRequest("POST", "/api/candidates", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+schedToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}

func TestCandidateCRUD(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), u)
	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Route("/api/candidates", func(r chi.Router) {
		r.Get("/", h.ListCandidates)
		r.Post("/", h.CreateCandidate)
		r.Get("/{id}", h.GetCandidate)
		r.Put("/{id}", h.UpdateCandidate)
		r.Delete("/{id}", h.DeleteCandidate)
	})

	// Create
	body, _ := json.Marshal(map[string]string{"name": "Jane", "email": "jane@test.com", "resume_url": ""})
	req := httptest.NewRequest("POST", "/api/candidates", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201; body: %s", w.Code, w.Body.String())
	}

	var created models.Candidate
	json.Unmarshal(w.Body.Bytes(), &created)
	if created.ID == 0 {
		t.Fatal("expected ID in response")
	}

	// List
	req = httptest.NewRequest("GET", "/api/candidates", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list: status = %d", w.Code)
	}

	// Get
	req = httptest.NewRequest("GET", "/api/candidates/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("get: status = %d", w.Code)
	}
}
