package store

import (
	"context"
	"hire/internal/models"
	"testing"
)

func createTestUserAndCandidate(t *testing.T, s *Store) (*models.User, *models.Candidate) {
	t.Helper()
	u := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: "h", Role: "scheduler"}
	s.CreateUser(context.Background(), u)
	c := &models.Candidate{Name: "Candidate", Email: "c@test.com", Status: "active"}
	s.CreateCandidate(context.Background(), c)
	return u, c
}

func TestCreateAndGetLoop(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)

	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: u.ID}
	if err := s.CreateLoop(context.Background(), loop); err != nil {
		t.Fatalf("CreateLoop: %v", err)
	}
	if loop.ID == 0 {
		t.Fatal("expected ID")
	}

	got, err := s.GetLoop(context.Background(), loop.ID)
	if err != nil {
		t.Fatalf("GetLoop: %v", err)
	}
	if got.CandidateID != c.ID || got.Status != "scheduling" {
		t.Errorf("got %+v", got)
	}
}

func TestListLoops(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)
	s.CreateLoop(context.Background(), &models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: u.ID})
	s.CreateLoop(context.Background(), &models.InterviewLoop{CandidateID: c.ID, Status: "active", CreatedBy: u.ID})

	loops, err := s.ListLoops(context.Background(), nil, nil, 50, 0)
	if err != nil {
		t.Fatalf("ListLoops: %v", err)
	}
	if len(loops) != 2 {
		t.Fatalf("got %d, want 2", len(loops))
	}
}

func TestUpdateLoop(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: u.ID}
	s.CreateLoop(context.Background(), loop)

	decision := "hire"
	loop.Status = "complete"
	loop.FinalDecision = &decision
	if err := s.UpdateLoop(context.Background(), loop); err != nil {
		t.Fatalf("UpdateLoop: %v", err)
	}
	got, _ := s.GetLoop(context.Background(), loop.ID)
	if got.Status != "complete" || *got.FinalDecision != "hire" {
		t.Errorf("got status=%q decision=%v", got.Status, got.FinalDecision)
	}
}
