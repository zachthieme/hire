package store

import (
	"context"
	"testing"
	"time"

	"hire/internal/models"
)

func TestStageLifecycle(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	uid := createTestUser(t, s, "sch@x.com", models.RoleScheduler)
	ivID := createTestUser(t, s, "iv@x.com", models.RoleInterviewer)

	job := &models.Job{Title: "BE", Status: models.JobStatusOpen, CreatedBy: uid}
	mustCreateJob(t, s, job)
	cand := &models.Candidate{Name: "Sam", Email: "sam@x.com"}
	if err := s.CreateCandidate(ctx, cand); err != nil {
		t.Fatal(err)
	}
	app := &models.Application{JobID: job.ID, CandidateID: cand.ID, Status: models.ApplicationStatusActive, CreatedBy: uid}
	if err := s.CreateApplication(ctx, app); err != nil {
		t.Fatal(err)
	}

	st := &models.Stage{ApplicationID: app.ID, Type: models.StageTypeInterview, FocusArea: "Coding", ScheduledAt: time.Now(), Status: models.StageStatusPending}
	if err := s.CreateStage(ctx, st); err != nil {
		t.Fatalf("CreateStage: %v", err)
	}
	if st.ID == 0 {
		t.Fatal("expected stage ID")
	}

	if err := s.AddStageInterviewer(ctx, st.ID, ivID); err != nil {
		t.Fatalf("AddStageInterviewer: %v", err)
	}
	mine, err := s.ListStagesByUser(ctx, ivID, 50, 0)
	if err != nil {
		t.Fatalf("ListStagesByUser: %v", err)
	}
	if len(mine) != 1 || mine[0].JobTitle != "BE" || mine[0].CandidateName != "Sam" {
		t.Fatalf("unexpected my stages: %+v", mine)
	}

	if err := s.RemoveStageInterviewer(ctx, st.ID, ivID); err != nil {
		t.Fatalf("RemoveStageInterviewer: %v", err)
	}
	mine, _ = s.ListStagesByUser(ctx, ivID, 50, 0)
	if len(mine) != 0 {
		t.Fatalf("expected 0 stages after removal, got %d", len(mine))
	}
}
