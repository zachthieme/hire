package api

import (
	"bytes"
	"context"
	"encoding/json"
	"hire/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestCreateCompetencyInvalidRatingsJSON(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	admin := &models.User{Email: "admin@test.com", Name: "Admin", PasswordHash: hash, Role: "admin"}
	s.CreateUser(context.Background(), admin)
	adminToken, _ := h.generateToken(admin.ID, admin.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Use(h.RequireRole("admin"))
	r.Post("/api/competencies", h.CreateCompetency)

	tests := []struct {
		name        string
		ratingsJSON string
		ratingType  string
	}{
		{"not json", "not-json", "levels"},
		{"object instead of array for levels", `{"foo":"bar"}`, "levels"},
		{"empty array for levels", `[]`, "levels"},
		{"object missing max for stars", `{"min":1}`, "stars"},
		{"string instead of object for stars", `"hello"`, "stars"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{
				"name": "Test", "rating_type": tt.ratingType, "ratings_json": tt.ratingsJSON,
			})
			req := httptest.NewRequest("POST", "/api/competencies", bytes.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+adminToken)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("%s: status = %d, want 400; body: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

func TestCompetencyCRUD(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "admin@test.com", Name: "Admin", PasswordHash: hash, Role: "admin"}
	s.CreateUser(context.Background(), u)
	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Route("/api/competencies", func(r chi.Router) {
		r.Get("/", h.ListCompetencies)
		r.Post("/", h.CreateCompetency)
		r.Put("/{id}", h.UpdateCompetency)
		r.Delete("/{id}", h.DeleteCompetency)
	})

	// Create
	body, _ := json.Marshal(map[string]string{
		"name": "Problem Solving", "rating_type": "levels",
		"ratings_json": `["Learning","Owning","Advising"]`,
	})
	req := httptest.NewRequest("POST", "/api/competencies", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d; body: %s", w.Code, w.Body.String())
	}

	// List
	req = httptest.NewRequest("GET", "/api/competencies", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list: status = %d", w.Code)
	}
	var list []models.Competency
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Fatalf("got %d competencies, want 1", len(list))
	}
}
