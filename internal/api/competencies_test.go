package api

import (
	"bytes"
	"encoding/json"
	"hire/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestCompetencyCRUD(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "admin@test.com", Name: "Admin", PasswordHash: hash, Role: "admin"}
	s.CreateUser(u)
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
