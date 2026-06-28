package store

import (
	"context"
	"testing"

	"hire/internal/models"
)

func TestCreateAndGetApplication(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	uid := createTestUser(t, s, "boss2@x.com", models.RoleScheduler)

	job := &models.Job{Title: "BE", Status: models.JobStatusOpen, CreatedBy: uid}
	mustCreateJob(t, s, job)
	cand := &models.Candidate{Name: "Pat", Email: "pat@x.com"}
	if err := s.CreateCandidate(ctx, cand); err != nil {
		t.Fatalf("CreateCandidate: %v", err)
	}

	app := &models.Application{JobID: job.ID, CandidateID: cand.ID, Status: models.ApplicationStatusActive, CreatedBy: uid}
	if err := s.CreateApplication(ctx, app); err != nil {
		t.Fatalf("CreateApplication: %v", err)
	}
	if app.ID == 0 {
		t.Fatal("expected application ID")
	}

	got, err := s.GetApplication(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetApplication: %v", err)
	}
	if got.JobID != job.ID || got.CandidateID != cand.ID {
		t.Fatalf("unexpected application: %+v", got)
	}

	// Duplicate candidate on same job rejected by unique constraint
	dup := &models.Application{JobID: job.ID, CandidateID: cand.ID, Status: models.ApplicationStatusActive, CreatedBy: uid}
	if err := s.CreateApplication(ctx, dup); err == nil {
		t.Fatal("expected unique violation for duplicate application")
	}
}

func mustCreateJob(t *testing.T, s *Store, j *models.Job) {
	t.Helper()
	if err := s.CreateJob(context.Background(), j); err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
}
