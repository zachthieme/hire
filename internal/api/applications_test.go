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

// submitFeedback POSTs feedback for a stage as the given token holder.
func submitFeedback(t *testing.T, r chi.Router, token string, stageID int64) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"recommendation": "hire", "recommendation_reason": "ok"})
	req := httptest.NewRequest("POST", "/api/stages/"+strconv.FormatInt(stageID, 10)+"/feedback", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("submit feedback: status = %d; body: %s", w.Code, w.Body.String())
	}
}

func findStage(t *testing.T, detail models.ApplicationDetail, stageID int64) models.StageWithFeedback {
	t.Helper()
	for _, s := range detail.Stages {
		if s.ID == stageID {
			return s
		}
	}
	t.Fatalf("stage %d not found in application detail", stageID)
	return models.StageWithFeedback{}
}

func participantFeedback(sw models.StageWithFeedback, interviewerID int64) *models.Feedback {
	for _, p := range sw.Participants {
		if p.InterviewerID == interviewerID {
			return p.Feedback
		}
	}
	return nil
}

func TestPeerFeedbackVisibility(t *testing.T) {
	h, s := newTestHandler(t)
	ctx := context.Background()
	hash, _ := HashPassword("pass")
	sched := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(ctx, sched)
	iv1 := &models.User{Email: "iv1@test.com", Name: "IV1", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(ctx, iv1)
	iv2 := &models.User{Email: "iv2@test.com", Name: "IV2", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(ctx, iv2)

	job := &models.Job{Title: "BE", Status: models.JobStatusOpen, CreatedBy: sched.ID}
	if err := s.CreateJob(ctx, job); err != nil {
		t.Fatal(err)
	}
	cand := &models.Candidate{Name: "Jane", Email: "jane@test.com"}
	if err := s.CreateCandidate(ctx, cand); err != nil {
		t.Fatal(err)
	}
	app := &models.Application{JobID: job.ID, CandidateID: cand.ID, Status: models.ApplicationStatusActive, CreatedBy: sched.ID}
	if err := s.CreateApplication(ctx, app); err != nil {
		t.Fatal(err)
	}
	st := &models.Stage{ApplicationID: app.ID, Type: models.StageTypeInterview, FocusArea: "Coding", ScheduledAt: time.Now(), Status: models.StageStatusPending}
	if err := s.CreateStage(ctx, st); err != nil {
		t.Fatal(err)
	}
	if err := s.AddStageInterviewer(ctx, st.ID, iv1.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.AddStageInterviewer(ctx, st.ID, iv2.ID); err != nil {
		t.Fatal(err)
	}

	iv1Token, _ := h.generateToken(iv1.ID, iv1.Role)
	iv2Token, _ := h.generateToken(iv2.ID, iv2.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/stages/{id}/feedback", h.CreateFeedback)
	r.Get("/api/stages/{id}/feedback", h.GetStageFeedback)
	r.Get("/api/applications/{id}", h.GetApplicationDetail)

	getDetail := func(token string) models.ApplicationDetail {
		req := httptest.NewRequest("GET", "/api/applications/"+strconv.FormatInt(app.ID, 10), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("get application detail: status = %d; body: %s", w.Code, w.Body.String())
		}
		var d models.ApplicationDetail
		if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil {
			t.Fatalf("decode detail: %v", err)
		}
		return d
	}

	getStageFeedback := func(token string) []*models.Feedback {
		req := httptest.NewRequest("GET", "/api/stages/"+strconv.FormatInt(st.ID, 10)+"/feedback", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("get stage feedback: status = %d; body: %s", w.Code, w.Body.String())
		}
		var list []*models.Feedback
		if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
			t.Fatalf("decode list: %v", err)
		}
		return list
	}

	// iv1 submits feedback.
	submitFeedback(t, r, iv1Token, st.ID)

	// iv2 has NOT submitted: peer (iv1) feedback must be hidden in application detail.
	d := getDetail(iv2Token)
	sw := findStage(t, d, st.ID)
	if fb := participantFeedback(sw, iv1.ID); fb != nil {
		t.Errorf("iv1 feedback should be hidden from iv2 pre-submit, got %+v", fb)
	}
	if fb := participantFeedback(sw, iv2.ID); fb != nil {
		t.Errorf("iv2 own feedback should be nil (not submitted), got %+v", fb)
	}

	// iv2 stage-feedback list must be empty pre-submit (peers hidden).
	if list := getStageFeedback(iv2Token); len(list) != 0 {
		t.Errorf("expected empty stage feedback list pre-submit, got %d entries", len(list))
	}

	// iv2 submits; now iv1's feedback becomes visible to iv2.
	submitFeedback(t, r, iv2Token, st.ID)
	d = getDetail(iv2Token)
	sw = findStage(t, d, st.ID)
	if fb := participantFeedback(sw, iv1.ID); fb == nil {
		t.Error("iv1 feedback should be visible to iv2 after iv2 submits")
	}
	if fb := participantFeedback(sw, iv2.ID); fb == nil {
		t.Error("iv2 own feedback should be visible after submit")
	}
}

func TestDuplicateApplicationConflict(t *testing.T) {
	h, s := newTestHandler(t)
	ctx := context.Background()
	hash, _ := HashPassword("pass")
	sched := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(ctx, sched)
	token, _ := h.generateToken(sched.ID, sched.Role)

	job := &models.Job{Title: "BE", Status: models.JobStatusOpen, CreatedBy: sched.ID}
	if err := s.CreateJob(ctx, job); err != nil {
		t.Fatal(err)
	}
	cand := &models.Candidate{Name: "Jane", Email: "jane@test.com"}
	if err := s.CreateCandidate(ctx, cand); err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/jobs/{id}/applications", h.CreateApplication)

	post := func() int {
		body, _ := json.Marshal(map[string]any{"candidate_id": cand.ID})
		req := httptest.NewRequest("POST", "/api/jobs/"+strconv.FormatInt(job.ID, 10)+"/applications", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	if code := post(); code != http.StatusCreated {
		t.Fatalf("first create: status = %d, want 201", code)
	}
	if code := post(); code != http.StatusConflict {
		t.Fatalf("duplicate create: status = %d, want 409", code)
	}
}
