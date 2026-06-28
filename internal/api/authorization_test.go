package api

import (
	"bytes"
	"context"
	"encoding/json"
	"hire/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorizationBoundaries(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")

	admin := &models.User{Email: "admin@test.com", Name: "Admin", PasswordHash: hash, Role: "admin"}
	s.CreateUser(context.Background(), admin)
	sched := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), sched)
	iv := &models.User{Email: "iv@test.com", Name: "IV", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(context.Background(), iv)

	adminToken, _ := h.generateToken(admin.ID, admin.Role)
	schedToken, _ := h.generateToken(sched.ID, sched.Role)
	ivToken, _ := h.generateToken(iv.ID, iv.Role)

	router := h.Router()

	jsonBody := func(v any) *bytes.Reader {
		b, _ := json.Marshal(v)
		return bytes.NewReader(b)
	}

	tests := []struct {
		name       string
		method     string
		path       string
		token      string
		body       any
		wantStatus int
	}{
		// Interviewer cannot access admin endpoints
		{"interviewer cannot create user", "POST", "/api/users", ivToken, map[string]string{"email": "x@x.com", "name": "X", "password": "12345678", "role": "interviewer"}, http.StatusForbidden},
		{"interviewer cannot list users", "GET", "/api/users", ivToken, nil, http.StatusForbidden},
		{"interviewer cannot delete user", "DELETE", "/api/users/1", ivToken, nil, http.StatusForbidden},
		{"interviewer cannot create competency", "POST", "/api/competencies", ivToken, map[string]string{"name": "Test", "rating_type": "levels", "ratings_json": `["a","b"]`}, http.StatusForbidden},
		{"interviewer cannot delete competency", "DELETE", "/api/competencies/1", ivToken, nil, http.StatusForbidden},

		// Interviewer cannot access scheduler endpoints
		{"interviewer cannot create candidate", "POST", "/api/candidates", ivToken, map[string]string{"name": "J", "email": "j@j.com"}, http.StatusForbidden},
		{"interviewer cannot list candidates", "GET", "/api/candidates", ivToken, nil, http.StatusForbidden},
		{"interviewer cannot create job", "POST", "/api/jobs", ivToken, map[string]any{"title": "BE"}, http.StatusForbidden},
		{"interviewer cannot create application", "POST", "/api/jobs/1/applications", ivToken, map[string]any{"candidate_id": 1}, http.StatusForbidden},
		{"interviewer cannot create stage", "POST", "/api/applications/1/stages", ivToken, map[string]any{"type": "interview"}, http.StatusForbidden},
		{"interviewer cannot add stage interviewer", "POST", "/api/stages/1/interviewers", ivToken, map[string]any{"interviewer_id": 1}, http.StatusForbidden},

		// Scheduler cannot access admin-only endpoints
		{"scheduler cannot create user", "POST", "/api/users", schedToken, map[string]string{"email": "x@x.com", "name": "X", "password": "12345678", "role": "interviewer"}, http.StatusForbidden},
		{"scheduler cannot get user by id", "GET", "/api/users/1", schedToken, nil, http.StatusForbidden},
		{"scheduler cannot delete user", "DELETE", "/api/users/1", schedToken, nil, http.StatusForbidden},
		{"scheduler cannot create competency", "POST", "/api/competencies", schedToken, map[string]string{"name": "Test", "rating_type": "levels", "ratings_json": `["a","b"]`}, http.StatusForbidden},
		{"scheduler cannot delete competency", "DELETE", "/api/competencies/1", schedToken, nil, http.StatusForbidden},

		// Scheduler CAN access scheduler endpoints
		{"scheduler can list candidates", "GET", "/api/candidates", schedToken, nil, http.StatusOK},
		{"scheduler can list users", "GET", "/api/users", schedToken, nil, http.StatusOK},
		{"scheduler can create job", "POST", "/api/jobs", schedToken, map[string]any{"title": "BE"}, http.StatusCreated},

		// Unauthenticated access denied
		{"no token on protected route", "GET", "/api/me", "", nil, http.StatusUnauthorized},

		// All authenticated users can access common endpoints
		{"interviewer can list competencies", "GET", "/api/competencies", ivToken, nil, http.StatusOK},
		{"interviewer can get own stages", "GET", "/api/me/stages", ivToken, nil, http.StatusOK},
		{"interviewer can list jobs", "GET", "/api/jobs", ivToken, nil, http.StatusOK},
		{"admin can list jobs", "GET", "/api/jobs", adminToken, nil, http.StatusOK},
		{"admin can list competencies", "GET", "/api/competencies", adminToken, nil, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				req = httptest.NewRequest(tt.method, tt.path, jsonBody(tt.body))
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}
