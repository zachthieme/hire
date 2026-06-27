package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hire/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestLoopCRUD(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), u)
	token, _ := h.generateToken(u.ID, u.Role)

	c := &models.Candidate{Name: "Jane", Email: "jane@test.com", Status: "active"}
	s.CreateCandidate(context.Background(), c)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/loops", h.CreateLoop)
	r.Get("/api/loops", h.ListLoops)
	r.Get("/api/loops/{id}", h.GetLoopDetail)
	r.Put("/api/loops/{id}", h.UpdateLoop)
	r.Delete("/api/loops/{id}", h.DeleteLoop)

	// Create loop
	body, _ := json.Marshal(map[string]any{"candidate_id": c.ID})
	req := httptest.NewRequest("POST", "/api/loops", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d; body: %s", w.Code, w.Body.String())
	}
	var created models.InterviewLoop
	json.Unmarshal(w.Body.Bytes(), &created)
	if created.ID == 0 {
		t.Fatal("expected ID")
	}
	if created.Status != "scheduling" {
		t.Errorf("status = %q, want scheduling", created.Status)
	}

	// List loops
	req = httptest.NewRequest("GET", "/api/loops", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list: status = %d", w.Code)
	}

	// Get loop detail
	req = httptest.NewRequest("GET", "/api/loops/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("get detail: status = %d; body: %s", w.Code, w.Body.String())
	}

	// Update loop
	body, _ = json.Marshal(map[string]string{"status": "active"})
	req = httptest.NewRequest("PUT", "/api/loops/1", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("update: status = %d; body: %s", w.Code, w.Body.String())
	}

	// Delete loop
	req = httptest.NewRequest("DELETE", "/api/loops/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d", w.Code)
	}
}

func TestCreateLoopValidation(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), u)
	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/loops", h.CreateLoop)

	// Missing candidate_id
	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest("POST", "/api/loops", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestUpdateLoopInvalidTransition(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	sched := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), sched)
	schedToken, _ := h.generateToken(sched.ID, sched.Role)

	c := &models.Candidate{Name: "Jane", Email: "jane@test.com", Status: "active"}
	s.CreateCandidate(context.Background(), c)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: sched.ID}
	s.CreateLoop(context.Background(), loop)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Put("/api/loops/{id}", h.UpdateLoop)

	// Try to jump from scheduling directly to complete (should fail)
	body, _ := json.Marshal(map[string]string{"status": "complete"})
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/loops/%d", loop.ID), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+schedToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}
