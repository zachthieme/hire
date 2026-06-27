package store

import (
	"context"
	"hire/internal/models"
	"testing"
	"time"
)

func TestCreateAndGetInterview(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: u.ID}
	s.CreateLoop(context.Background(), loop)

	iv := &models.Interview{
		LoopID:              loop.ID,
		InterviewerID:       u.ID,
		FocusArea:           "coding",
		ScheduledAt:         time.Now().Add(24 * time.Hour),
		VideoLink:           "https://meet.example.com/abc",
		NotesForInterviewer: "Focus on algorithms",
		Status:              "pending",
	}
	if err := s.CreateInterview(context.Background(), iv); err != nil {
		t.Fatalf("CreateInterview: %v", err)
	}
	if iv.ID == 0 {
		t.Fatal("expected ID")
	}

	got, err := s.GetInterview(context.Background(), iv.ID)
	if err != nil {
		t.Fatalf("GetInterview: %v", err)
	}
	if got.FocusArea != "coding" || got.Status != "pending" {
		t.Errorf("got %+v", got)
	}
}

func TestListInterviewsByLoop(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: u.ID}
	s.CreateLoop(context.Background(), loop)

	s.CreateInterview(context.Background(), &models.Interview{LoopID: loop.ID, InterviewerID: u.ID, FocusArea: "coding", ScheduledAt: time.Now(), Status: "pending"})
	s.CreateInterview(context.Background(), &models.Interview{LoopID: loop.ID, InterviewerID: u.ID, FocusArea: "design", ScheduledAt: time.Now(), Status: "pending"})

	list, err := s.ListInterviewsByLoop(context.Background(), loop.ID, 50, 0)
	if err != nil {
		t.Fatalf("ListInterviewsByLoop: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d, want 2", len(list))
	}
}

func TestListInterviewsByUser(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: u.ID}
	s.CreateLoop(context.Background(), loop)

	s.CreateInterview(context.Background(), &models.Interview{LoopID: loop.ID, InterviewerID: u.ID, FocusArea: "coding", ScheduledAt: time.Now(), Status: "pending"})

	list, err := s.ListInterviewsByUser(context.Background(), u.ID, 50, 0)
	if err != nil {
		t.Fatalf("ListInterviewsByUser: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d, want 1", len(list))
	}
}
