package store

import (
	"hire/internal/models"
	"testing"
)

func TestCreateAndGetCompetency(t *testing.T) {
	s := newTestStore(t)
	c := &models.Competency{Name: "Problem Solving", RatingType: "levels", RatingsJSON: `["Learning","Owning","Advising"]`}
	if err := s.CreateCompetency(c); err != nil {
		t.Fatalf("CreateCompetency: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("expected ID")
	}
	got, err := s.GetCompetency(c.ID)
	if err != nil {
		t.Fatalf("GetCompetency: %v", err)
	}
	if got.Name != "Problem Solving" || got.RatingType != "levels" {
		t.Errorf("got %+v", got)
	}
}

func TestListCompetencies(t *testing.T) {
	s := newTestStore(t)
	s.CreateCompetency(&models.Competency{Name: "A", RatingType: "levels", RatingsJSON: `["X"]`})
	s.CreateCompetency(&models.Competency{Name: "B", RatingType: "stars", RatingsJSON: `{"min":1,"max":5}`})
	list, err := s.ListCompetencies()
	if err != nil {
		t.Fatalf("ListCompetencies: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d, want 2", len(list))
	}
}

func TestDeleteCompetency(t *testing.T) {
	s := newTestStore(t)
	c := &models.Competency{Name: "C", RatingType: "stars", RatingsJSON: `{"min":1,"max":5}`}
	s.CreateCompetency(c)
	if err := s.DeleteCompetency(c.ID); err != nil {
		t.Fatalf("DeleteCompetency: %v", err)
	}
	_, err := s.GetCompetency(c.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
