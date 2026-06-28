package store

import (
	"context"
	"hire/internal/models"
	"testing"
)

func TestCreateAndGetCandidate(t *testing.T) {
	s := newTestStore(t)
	c := &models.Candidate{Name: "Jane Doe", Email: "jane@example.com", ResumeURL: "https://resume.com/jane"}
	if err := s.CreateCandidate(context.Background(), c); err != nil {
		t.Fatalf("CreateCandidate: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("expected ID to be set")
	}
	got, err := s.GetCandidate(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("GetCandidate: %v", err)
	}
	if got.Name != "Jane Doe" || got.Email != "jane@example.com" {
		t.Errorf("got %+v", got)
	}
}

func TestListCandidates(t *testing.T) {
	s := newTestStore(t)
	s.CreateCandidate(context.Background(), &models.Candidate{Name: "A", Email: "a@a.com"})
	s.CreateCandidate(context.Background(), &models.Candidate{Name: "B", Email: "b@b.com"})
	list, err := s.ListCandidates(context.Background(), 50, 0)
	if err != nil {
		t.Fatalf("ListCandidates: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d, want 2", len(list))
	}
}

func TestUpdateCandidate(t *testing.T) {
	s := newTestStore(t)
	c := &models.Candidate{Name: "X", Email: "x@x.com"}
	s.CreateCandidate(context.Background(), c)
	c.Name = "Xavier"
	if err := s.UpdateCandidate(context.Background(), c); err != nil {
		t.Fatalf("UpdateCandidate: %v", err)
	}
	got, _ := s.GetCandidate(context.Background(), c.ID)
	if got.Name != "Xavier" {
		t.Errorf("name = %q, want Xavier", got.Name)
	}
}

func TestDeleteCandidate(t *testing.T) {
	s := newTestStore(t)
	c := &models.Candidate{Name: "Y", Email: "y@y.com"}
	s.CreateCandidate(context.Background(), c)
	if err := s.DeleteCandidate(context.Background(), c.ID); err != nil {
		t.Fatalf("DeleteCandidate: %v", err)
	}
	_, err := s.GetCandidate(context.Background(), c.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
