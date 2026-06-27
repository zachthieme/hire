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

func TestUserCRUD(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	admin := &models.User{Email: "admin@test.com", Name: "Admin", PasswordHash: hash, Role: "admin"}
	s.CreateUser(context.Background(), admin)
	token, _ := h.generateToken(admin.ID, admin.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/users", h.CreateUser)
	r.Get("/api/users", h.ListUsers)
	r.Get("/api/users/{id}", h.GetUser)
	r.Put("/api/users/{id}", h.UpdateUser)
	r.Delete("/api/users/{id}", h.DeleteUser)

	// Create user
	body, _ := json.Marshal(map[string]string{
		"email": "new@test.com", "name": "New User", "password": "password123", "role": "interviewer",
	})
	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201; body: %s", w.Code, w.Body.String())
	}
	var created models.User
	json.Unmarshal(w.Body.Bytes(), &created)
	if created.ID == 0 {
		t.Fatal("expected ID")
	}

	// List users
	req = httptest.NewRequest("GET", "/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list: status = %d", w.Code)
	}
	var users []models.User
	json.Unmarshal(w.Body.Bytes(), &users)
	if len(users) != 2 {
		t.Fatalf("got %d users, want 2", len(users))
	}

	// Get user
	req = httptest.NewRequest("GET", "/api/users/2", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("get: status = %d", w.Code)
	}

	// Update user
	body, _ = json.Marshal(map[string]string{"name": "Updated", "role": "scheduler"})
	req = httptest.NewRequest("PUT", "/api/users/2", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("update: status = %d; body: %s", w.Code, w.Body.String())
	}

	// Delete user
	req = httptest.NewRequest("DELETE", "/api/users/2", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d", w.Code)
	}
}

func TestCreateUserValidation(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	admin := &models.User{Email: "admin@test.com", Name: "Admin", PasswordHash: hash, Role: "admin"}
	s.CreateUser(context.Background(), admin)
	token, _ := h.generateToken(admin.ID, admin.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/users", h.CreateUser)

	tests := []struct {
		name string
		body map[string]string
		want int
	}{
		{"missing email", map[string]string{"name": "A", "password": "password123", "role": "admin"}, 400},
		{"invalid email", map[string]string{"email": "notanemail", "name": "A", "password": "password123", "role": "admin"}, 400},
		{"short password", map[string]string{"email": "a@a.com", "name": "A", "password": "short", "role": "admin"}, 400},
		{"invalid role", map[string]string{"email": "a@a.com", "name": "A", "password": "password123", "role": "superuser"}, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tt.want, w.Body.String())
			}
		})
	}
}
