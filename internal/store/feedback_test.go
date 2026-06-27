package store

import (
	"context"
	"hire/internal/models"
	"testing"
	"time"
)

func TestCreateAndGetFeedback(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "active", CreatedBy: u.ID}
	s.CreateLoop(context.Background(), loop)

	comp := &models.Competency{Name: "Coding", RatingType: "levels", RatingsJSON: `["Learning","Owning","Advising"]`}
	s.CreateCompetency(context.Background(), comp)

	iv := &models.Interview{LoopID: loop.ID, InterviewerID: u.ID, FocusArea: "coding", ScheduledAt: time.Now(), Status: "pending"}
	s.CreateInterview(context.Background(), iv)

	fb := &models.Feedback{
		InterviewID:          iv.ID,
		Recommendation:       "hire",
		RecommendationReason: "Strong coder",
		FreeFormNotes:        "Good performance",
		CompetencyRatings: []models.CompetencyRating{
			{CompetencyID: comp.ID, RatingValue: "Owning"},
		},
	}
	if _, err := s.CreateFeedback(context.Background(), fb); err != nil {
		t.Fatalf("CreateFeedback: %v", err)
	}
	if fb.ID == 0 {
		t.Fatal("expected ID")
	}

	// Interview should be marked complete
	updatedIV, _ := s.GetInterview(context.Background(), iv.ID)
	if updatedIV.Status != "complete" {
		t.Errorf("interview status = %q, want complete", updatedIV.Status)
	}

	// Get feedback with ratings
	got, err := s.GetFeedbackByInterview(context.Background(), iv.ID)
	if err != nil {
		t.Fatalf("GetFeedbackByInterview: %v", err)
	}
	if got.Recommendation != "hire" {
		t.Errorf("recommendation = %q, want hire", got.Recommendation)
	}
	if len(got.CompetencyRatings) != 1 {
		t.Fatalf("got %d ratings, want 1", len(got.CompetencyRatings))
	}
	if got.CompetencyRatings[0].RatingValue != "Owning" {
		t.Errorf("rating = %q, want Owning", got.CompetencyRatings[0].RatingValue)
	}
}

func TestHasUserSubmittedFeedbackForLoop(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "active", CreatedBy: u.ID}
	s.CreateLoop(context.Background(), loop)

	iv := &models.Interview{LoopID: loop.ID, InterviewerID: u.ID, FocusArea: "coding", ScheduledAt: time.Now(), Status: "pending"}
	s.CreateInterview(context.Background(), iv)

	submitted, err := s.HasUserSubmittedFeedbackForLoop(context.Background(), loop.ID, u.ID)
	if err != nil {
		t.Fatalf("HasUserSubmittedFeedbackForLoop: %v", err)
	}
	if submitted {
		t.Fatal("should not have submitted feedback yet")
	}

	s.CreateFeedback(context.Background(), &models.Feedback{ //nolint:errcheck
		InterviewID:    iv.ID,
		Recommendation: "hire",
	})

	submitted, err = s.HasUserSubmittedFeedbackForLoop(context.Background(), loop.ID, u.ID)
	if err != nil {
		t.Fatalf("HasUserSubmittedFeedbackForLoop: %v", err)
	}
	if !submitted {
		t.Fatal("should have submitted feedback")
	}
}
