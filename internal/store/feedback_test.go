package store

import (
	"context"
	"testing"
	"time"

	"hire/internal/models"
)

func TestCreateFeedbackMarksStageCompleteAndReady(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	uid := createTestUser(t, s, "sc@x.com", models.RoleScheduler)
	ivID := createTestUser(t, s, "iv2@x.com", models.RoleInterviewer)

	job := &models.Job{Title: "BE", Status: models.JobStatusOpen, CreatedBy: uid}
	mustCreateJob(t, s, job)
	cand := &models.Candidate{Name: "Lee", Email: "lee@x.com"}
	if err := s.CreateCandidate(ctx, cand); err != nil {
		t.Fatal(err)
	}
	app := &models.Application{JobID: job.ID, CandidateID: cand.ID, Status: models.ApplicationStatusActive, CreatedBy: uid}
	if err := s.CreateApplication(ctx, app); err != nil {
		t.Fatal(err)
	}
	st := &models.Stage{ApplicationID: app.ID, Type: models.StageTypeInterview, ScheduledAt: time.Now(), Status: models.StageStatusPending}
	if err := s.CreateStage(ctx, st); err != nil {
		t.Fatal(err)
	}
	if err := s.AddStageInterviewer(ctx, st.ID, ivID); err != nil {
		t.Fatal(err)
	}

	fb := &models.Feedback{StageID: st.ID, InterviewerID: ivID, Recommendation: models.RecommendationHire}
	ready, appID, err := s.CreateFeedback(ctx, fb)
	if err != nil {
		t.Fatalf("CreateFeedback: %v", err)
	}
	if !ready {
		t.Fatal("expected application ready (only stage now complete)")
	}
	if appID != app.ID {
		t.Fatalf("expected appID %d, got %d", app.ID, appID)
	}

	got, err := s.GetStage(ctx, st.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.StageStatusComplete {
		t.Fatalf("stage status = %q, want complete", got.Status)
	}
}
