package store

import (
	"context"
	"hire/internal/models"
	"testing"
)

func TestCreateAndGetCompetency(t *testing.T) {
	s := newTestStore(t)
	c := &models.Competency{Name: "Problem Solving", RatingType: "levels", RatingsJSON: `["Learning","Owning","Advising"]`}
	if err := s.CreateCompetency(context.Background(), c); err != nil {
		t.Fatalf("CreateCompetency: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("expected ID")
	}
	got, err := s.GetCompetency(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("GetCompetency: %v", err)
	}
	if got.Name != "Problem Solving" || got.RatingType != "levels" {
		t.Errorf("got %+v", got)
	}
}

func TestListCompetencies(t *testing.T) {
	s := newTestStore(t)
	s.CreateCompetency(context.Background(), &models.Competency{Name: "A", RatingType: "levels", RatingsJSON: `["X"]`})
	s.CreateCompetency(context.Background(), &models.Competency{Name: "B", RatingType: "stars", RatingsJSON: `{"min":1,"max":5}`})
	list, err := s.ListCompetencies(context.Background(), 50, 0)
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
	s.CreateCompetency(context.Background(), c)
	if err := s.DeleteCompetency(context.Background(), c.ID); err != nil {
		t.Fatalf("DeleteCompetency: %v", err)
	}
	_, err := s.GetCompetency(context.Background(), c.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
