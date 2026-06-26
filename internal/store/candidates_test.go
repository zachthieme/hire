package store

import (
	"hire/internal/models"
	"testing"
)

func TestCreateAndGetCandidate(t *testing.T) {
	s := newTestStore(t)
	c := &models.Candidate{Name: "Jane Doe", Email: "jane@example.com", ResumeURL: "https://resume.com/jane", Status: "active"}
	if err := s.CreateCandidate(c); err != nil {
		t.Fatalf("CreateCandidate: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("expected ID to be set")
	}
	got, err := s.GetCandidate(c.ID)
	if err != nil {
		t.Fatalf("GetCandidate: %v", err)
	}
	if got.Name != "Jane Doe" || got.Email != "jane@example.com" {
		t.Errorf("got %+v", got)
	}
}

func TestListCandidates(t *testing.T) {
	s := newTestStore(t)
	s.CreateCandidate(&models.Candidate{Name: "A", Email: "a@a.com", Status: "active"})
	s.CreateCandidate(&models.Candidate{Name: "B", Email: "b@b.com", Status: "active"})
	list, err := s.ListCandidates(50, 0)
	if err != nil {
		t.Fatalf("ListCandidates: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d, want 2", len(list))
	}
}

func TestUpdateCandidate(t *testing.T) {
	s := newTestStore(t)
	c := &models.Candidate{Name: "X", Email: "x@x.com", Status: "active"}
	s.CreateCandidate(c)
	c.Status = "hired"
	if err := s.UpdateCandidate(c); err != nil {
		t.Fatalf("UpdateCandidate: %v", err)
	}
	got, _ := s.GetCandidate(c.ID)
	if got.Status != "hired" {
		t.Errorf("status = %q, want hired", got.Status)
	}
}

func TestDeleteCandidate(t *testing.T) {
	s := newTestStore(t)
	c := &models.Candidate{Name: "Y", Email: "y@y.com", Status: "active"}
	s.CreateCandidate(c)
	if err := s.DeleteCandidate(c.ID); err != nil {
		t.Fatalf("DeleteCandidate: %v", err)
	}
	_, err := s.GetCandidate(c.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
