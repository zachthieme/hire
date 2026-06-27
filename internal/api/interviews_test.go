package api

import (
	"bytes"
	"context"
	"encoding/json"
	"hire/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestInterviewCRUD(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	sched := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), sched)
	iv := &models.User{Email: "iv@test.com", Name: "Interviewer", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(context.Background(), iv)
	token, _ := h.generateToken(sched.ID, sched.Role)

	c := &models.Candidate{Name: "Jane", Email: "jane@test.com", Status: "active"}
	s.CreateCandidate(context.Background(), c)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: sched.ID}
	s.CreateLoop(context.Background(), loop)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/loops/{id}/interviews", h.CreateInterview)
	r.Put("/api/interviews/{id}", h.UpdateInterview)
	r.Delete("/api/interviews/{id}", h.DeleteInterview)
	r.Get("/api/me/interviews", h.ListMyInterviews)

	// Create interview
	body, _ := json.Marshal(map[string]any{
		"interviewer_id": iv.ID,
		"focus_area":     "coding",
		"scheduled_at":   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
	req := httptest.NewRequest("POST", "/api/loops/1/interviews", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d; body: %s", w.Code, w.Body.String())
	}

	// Update interview
	body, _ = json.Marshal(map[string]any{"focus_area": "system design", "video_link": "https://meet.example.com"})
	req = httptest.NewRequest("PUT", "/api/interviews/1", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("update: status = %d; body: %s", w.Code, w.Body.String())
	}

	// List my interviews (as interviewer)
	ivToken, _ := h.generateToken(iv.ID, iv.Role)
	req = httptest.NewRequest("GET", "/api/me/interviews", nil)
	req.Header.Set("Authorization", "Bearer "+ivToken)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list my: status = %d", w.Code)
	}
	var interviews []models.Interview
	json.Unmarshal(w.Body.Bytes(), &interviews)
	if len(interviews) != 1 {
		t.Fatalf("got %d interviews, want 1", len(interviews))
	}

	// Delete interview
	req = httptest.NewRequest("DELETE", "/api/interviews/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d", w.Code)
	}
}

func TestCreateInterviewValidation(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), u)
	token, _ := h.generateToken(u.ID, u.Role)

	c := &models.Candidate{Name: "Jane", Email: "jane@test.com", Status: "active"}
	s.CreateCandidate(context.Background(), c)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: u.ID}
	s.CreateLoop(context.Background(), loop)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/loops/{id}/interviews", h.CreateInterview)

	// Missing focus_area
	body, _ := json.Marshal(map[string]any{"interviewer_id": 1})
	req := httptest.NewRequest("POST", "/api/loops/1/interviews", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}

	// Missing interviewer_id
	body, _ = json.Marshal(map[string]any{"focus_area": "coding"})
	req = httptest.NewRequest("POST", "/api/loops/1/interviews", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
