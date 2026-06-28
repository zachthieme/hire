package store

import (
	"context"
	"testing"

	"hire/internal/models"
)

func TestCreateAndGetJob(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	uid := createTestUser(t, s, "boss@x.com", models.RoleScheduler)

	job := &models.Job{Title: "Backend Engineer", Description: "Build APIs", HiringManager: "Dana", Status: models.JobStatusOpen, CreatedBy: uid}
	if err := s.CreateJob(ctx, job); err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	if job.ID == 0 {
		t.Fatal("expected job ID to be set")
	}

	got, err := s.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if got.Title != "Backend Engineer" || got.HiringManager != "Dana" {
		t.Fatalf("unexpected job: %+v", got)
	}

	jobs, err := s.ListJobs(ctx, 50, 0)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
}
