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

func TestStageCompletesOnlyWhenAllInterviewersSubmit(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	uid := createTestUser(t, s, "sc3@x.com", models.RoleScheduler)
	iv1 := createTestUser(t, s, "iv-a@x.com", models.RoleInterviewer)
	iv2 := createTestUser(t, s, "iv-b@x.com", models.RoleInterviewer)
	iv3 := createTestUser(t, s, "iv-c@x.com", models.RoleInterviewer)

	job := &models.Job{Title: "BE", Status: models.JobStatusOpen, CreatedBy: uid}
	mustCreateJob(t, s, job)
	cand := &models.Candidate{Name: "Mo", Email: "mo@x.com"}
	if err := s.CreateCandidate(ctx, cand); err != nil {
		t.Fatal(err)
	}
	app := &models.Application{JobID: job.ID, CandidateID: cand.ID, Status: models.ApplicationStatusActive, CreatedBy: uid}
	if err := s.CreateApplication(ctx, app); err != nil {
		t.Fatal(err)
	}

	// Stage A: two assigned interviewers.
	stA := &models.Stage{ApplicationID: app.ID, Type: models.StageTypeInterview, ScheduledAt: time.Now(), Status: models.StageStatusPending}
	if err := s.CreateStage(ctx, stA); err != nil {
		t.Fatal(err)
	}
	if err := s.AddStageInterviewer(ctx, stA.ID, iv1); err != nil {
		t.Fatal(err)
	}
	if err := s.AddStageInterviewer(ctx, stA.ID, iv2); err != nil {
		t.Fatal(err)
	}

	// Stage B: single interviewer.
	stB := &models.Stage{ApplicationID: app.ID, Type: models.StageTypeInterview, ScheduledAt: time.Now(), Status: models.StageStatusPending}
	if err := s.CreateStage(ctx, stB); err != nil {
		t.Fatal(err)
	}
	if err := s.AddStageInterviewer(ctx, stB.ID, iv3); err != nil {
		t.Fatal(err)
	}

	// First interviewer on stage A: stage stays pending, app not ready.
	ready, _, err := s.CreateFeedback(ctx, &models.Feedback{StageID: stA.ID, InterviewerID: iv1, Recommendation: models.RecommendationHire})
	if err != nil {
		t.Fatalf("CreateFeedback iv1: %v", err)
	}
	if ready {
		t.Fatal("did not expect app ready after only one of two interviewers on stage A")
	}
	gotA, err := s.GetStage(ctx, stA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotA.Status != models.StageStatusPending {
		t.Fatalf("stage A status = %q, want pending after one submission", gotA.Status)
	}

	// Second interviewer on stage A: stage A now complete, but stage B outstanding.
	ready, _, err = s.CreateFeedback(ctx, &models.Feedback{StageID: stA.ID, InterviewerID: iv2, Recommendation: models.RecommendationHire})
	if err != nil {
		t.Fatalf("CreateFeedback iv2: %v", err)
	}
	if ready {
		t.Fatal("did not expect app ready while stage B is still pending")
	}
	gotA, err = s.GetStage(ctx, stA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotA.Status != models.StageStatusComplete {
		t.Fatalf("stage A status = %q, want complete after both submissions", gotA.Status)
	}

	// Complete stage B: now the whole application is ready.
	ready, appID, err := s.CreateFeedback(ctx, &models.Feedback{StageID: stB.ID, InterviewerID: iv3, Recommendation: models.RecommendationHire})
	if err != nil {
		t.Fatalf("CreateFeedback iv3: %v", err)
	}
	if !ready {
		t.Fatal("expected app ready once all stages complete")
	}
	if appID != app.ID {
		t.Fatalf("expected appID %d, got %d", app.ID, appID)
	}
}

func TestFeedbackCompetencyRatingsRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	uid := createTestUser(t, s, "sc4@x.com", models.RoleScheduler)
	ivID := createTestUser(t, s, "iv-d@x.com", models.RoleInterviewer)

	comp := &models.Competency{Name: "Coding", RatingType: "levels", RatingsJSON: `["Learning","Owning","Advising"]`}
	if err := s.CreateCompetency(ctx, comp); err != nil {
		t.Fatalf("CreateCompetency: %v", err)
	}

	job := &models.Job{Title: "BE", Status: models.JobStatusOpen, CreatedBy: uid}
	mustCreateJob(t, s, job)
	cand := &models.Candidate{Name: "Ravi", Email: "ravi@x.com"}
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

	fb := &models.Feedback{
		StageID:        st.ID,
		InterviewerID:  ivID,
		Recommendation: models.RecommendationHire,
		CompetencyRatings: []models.CompetencyRating{
			{CompetencyID: comp.ID, RatingValue: "Advising"},
		},
	}
	if _, _, err := s.CreateFeedback(ctx, fb); err != nil {
		t.Fatalf("CreateFeedback: %v", err)
	}

	got, err := s.GetFeedbackByStageAndInterviewer(ctx, st.ID, ivID)
	if err != nil {
		t.Fatalf("GetFeedbackByStageAndInterviewer: %v", err)
	}
	if len(got.CompetencyRatings) != 1 {
		t.Fatalf("got %d ratings, want 1", len(got.CompetencyRatings))
	}
	if got.CompetencyRatings[0].CompetencyID != comp.ID {
		t.Errorf("competency id = %d, want %d", got.CompetencyRatings[0].CompetencyID, comp.ID)
	}
	if got.CompetencyRatings[0].RatingValue != "Advising" {
		t.Errorf("rating = %q, want Advising", got.CompetencyRatings[0].RatingValue)
	}
}
