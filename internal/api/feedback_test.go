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

func TestFeedbackCreateAndGet(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	sched := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), sched)
	iv := &models.User{Email: "iv@test.com", Name: "Interviewer", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(context.Background(), iv)

	c := &models.Candidate{Name: "Jane", Email: "jane@test.com", Status: "active"}
	s.CreateCandidate(context.Background(), c)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "active", CreatedBy: sched.ID}
	s.CreateLoop(context.Background(), loop)
	interview := &models.Interview{LoopID: loop.ID, InterviewerID: iv.ID, FocusArea: "coding", ScheduledAt: time.Now(), Status: "pending"}
	s.CreateInterview(context.Background(), interview)

	ivToken, _ := h.generateToken(iv.ID, iv.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/interviews/{id}/feedback", h.CreateFeedback)
	r.Get("/api/interviews/{id}/feedback", h.GetFeedback)

	// Create feedback as interviewer
	body, _ := json.Marshal(map[string]any{
		"recommendation":        "hire",
		"recommendation_reason": "Strong technical skills",
		"free_form_notes":       "Great candidate",
		"competency_ratings":    []any{},
	})
	req := httptest.NewRequest("POST", "/api/interviews/1/feedback", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+ivToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d; body: %s", w.Code, w.Body.String())
	}

	// Get feedback
	req = httptest.NewRequest("GET", "/api/interviews/1/feedback", nil)
	req.Header.Set("Authorization", "Bearer "+ivToken)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("get: status = %d; body: %s", w.Code, w.Body.String())
	}
	var fb models.Feedback
	json.Unmarshal(w.Body.Bytes(), &fb)
	if fb.Recommendation != "hire" {
		t.Errorf("recommendation = %q, want hire", fb.Recommendation)
	}
}

func TestFeedbackNotYourInterview(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	sched := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), sched)
	iv1 := &models.User{Email: "iv1@test.com", Name: "IV1", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(context.Background(), iv1)
	iv2 := &models.User{Email: "iv2@test.com", Name: "IV2", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(context.Background(), iv2)

	c := &models.Candidate{Name: "Jane", Email: "jane@test.com", Status: "active"}
	s.CreateCandidate(context.Background(), c)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "active", CreatedBy: sched.ID}
	s.CreateLoop(context.Background(), loop)
	interview := &models.Interview{LoopID: loop.ID, InterviewerID: iv1.ID, FocusArea: "coding", ScheduledAt: time.Now(), Status: "pending"}
	s.CreateInterview(context.Background(), interview)

	// iv2 tries to submit feedback for iv1's interview
	iv2Token, _ := h.generateToken(iv2.ID, iv2.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/interviews/{id}/feedback", h.CreateFeedback)

	body, _ := json.Marshal(map[string]any{
		"recommendation": "hire", "recommendation_reason": "Good", "free_form_notes": "Notes",
	})
	req := httptest.NewRequest("POST", "/api/interviews/1/feedback", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+iv2Token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestFeedbackValidation(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	iv := &models.User{Email: "iv@test.com", Name: "IV", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(context.Background(), iv)
	sched := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), sched)

	c := &models.Candidate{Name: "Jane", Email: "jane@test.com", Status: "active"}
	s.CreateCandidate(context.Background(), c)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "active", CreatedBy: sched.ID}
	s.CreateLoop(context.Background(), loop)
	interview := &models.Interview{LoopID: loop.ID, InterviewerID: iv.ID, FocusArea: "coding", ScheduledAt: time.Now(), Status: "pending"}
	s.CreateInterview(context.Background(), interview)

	ivToken, _ := h.generateToken(iv.ID, iv.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/interviews/{id}/feedback", h.CreateFeedback)

	// Invalid recommendation
	body, _ := json.Marshal(map[string]any{"recommendation": "maybe"})
	req := httptest.NewRequest("POST", "/api/interviews/1/feedback", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+ivToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}
