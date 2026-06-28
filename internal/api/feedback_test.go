package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"hire/internal/models"

	"github.com/go-chi/chi/v5"
)

// buildStageWithInterviewer creates a job → application → stage and assigns the
// given interviewer to the stage. Returns the stage ID.
func buildStageWithInterviewer(t *testing.T, s storeForTest, schedID, ivID int64) int64 {
	t.Helper()
	ctx := context.Background()
	job := &models.Job{Title: "BE", Status: models.JobStatusOpen, CreatedBy: schedID}
	if err := s.CreateJob(ctx, job); err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	cand := &models.Candidate{Name: "Jane", Email: "jane@test.com"}
	if err := s.CreateCandidate(ctx, cand); err != nil {
		t.Fatalf("CreateCandidate: %v", err)
	}
	app := &models.Application{JobID: job.ID, CandidateID: cand.ID, Status: models.ApplicationStatusActive, CreatedBy: schedID}
	if err := s.CreateApplication(ctx, app); err != nil {
		t.Fatalf("CreateApplication: %v", err)
	}
	st := &models.Stage{ApplicationID: app.ID, Type: models.StageTypeInterview, FocusArea: "Coding", ScheduledAt: time.Now(), Status: models.StageStatusPending}
	if err := s.CreateStage(ctx, st); err != nil {
		t.Fatalf("CreateStage: %v", err)
	}
	if ivID != 0 {
		if err := s.AddStageInterviewer(ctx, st.ID, ivID); err != nil {
			t.Fatalf("AddStageInterviewer: %v", err)
		}
	}
	return st.ID
}

// storeForTest is the subset of the real store used in these tests.
type storeForTest interface {
	CreateJob(ctx context.Context, j *models.Job) error
	CreateCandidate(ctx context.Context, c *models.Candidate) error
	CreateApplication(ctx context.Context, a *models.Application) error
	CreateStage(ctx context.Context, st *models.Stage) error
	AddStageInterviewer(ctx context.Context, stageID, interviewerID int64) error
}

func TestCreateStageFeedbackAssignedInterviewer(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	sched := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), sched)
	iv := &models.User{Email: "iv@test.com", Name: "IV", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(context.Background(), iv)

	stageID := buildStageWithInterviewer(t, s, sched.ID, iv.ID)
	ivToken, _ := h.generateToken(iv.ID, iv.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/stages/{id}/feedback", h.CreateFeedback)

	body, _ := json.Marshal(map[string]any{
		"recommendation":        "hire",
		"recommendation_reason": "Strong",
		"free_form_notes":       "Great",
	})
	req := httptest.NewRequest("POST", "/api/stages/"+strconv.FormatInt(stageID, 10)+"/feedback", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+ivToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStageFeedbackNotAssigned(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	sched := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), sched)
	iv1 := &models.User{Email: "iv1@test.com", Name: "IV1", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(context.Background(), iv1)
	iv2 := &models.User{Email: "iv2@test.com", Name: "IV2", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(context.Background(), iv2)

	stageID := buildStageWithInterviewer(t, s, sched.ID, iv1.ID)
	iv2Token, _ := h.generateToken(iv2.ID, iv2.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/stages/{id}/feedback", h.CreateFeedback)

	body, _ := json.Marshal(map[string]any{"recommendation": "hire"})
	req := httptest.NewRequest("POST", "/api/stages/"+strconv.FormatInt(stageID, 10)+"/feedback", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+iv2Token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
