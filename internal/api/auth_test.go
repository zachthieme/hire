package api

import (
	"bytes"
	"context"
	"encoding/json"
	"hire/internal/models"
	"hire/internal/store"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func testDSN() string {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://hire:devpassword@localhost:5433/hire_test?sslmode=disable"
	}
	return dsn
}

func TestMain(m *testing.M) {
	dsn := testDSN()
	mig, err := migrate.New("file://../../migrations", dsn)
	if err != nil {
		panic("migrate.New: " + err.Error())
	}
	mig.Up()
	os.Exit(m.Run())
}

func newTestHandler(t *testing.T) (*Handler, *store.Store) {
	t.Helper()
	s, err := store.New(testDSN())
	if err != nil {
		t.Fatalf("newTestHandler: %v", err)
	}
	s.DB().Exec("TRUNCATE competency_ratings, notifications, feedback, interviews, interview_loops, competencies, candidates, users RESTART IDENTITY CASCADE")
	t.Cleanup(func() { s.Close() })
	h := NewHandler(s, "test-secret-must-be-at-least-32-chars", []string{"*"})
	return h, s
}

func TestLoginSuccess(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("password123")
	s.CreateUser(context.Background(), &models.User{Email: "test@test.com", Name: "Test", PasswordHash: hash, Role: "interviewer"})

	r := chi.NewRouter()
	r.Post("/api/auth/login", h.Login)

	body, _ := json.Marshal(map[string]string{"email": "test@test.com", "password": "password123"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] == nil || resp["token"] == "" {
		t.Fatal("expected token in response")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("correct")
	s.CreateUser(context.Background(), &models.User{Email: "test@test.com", Name: "Test", PasswordHash: hash, Role: "interviewer"})

	r := chi.NewRouter()
	r.Post("/api/auth/login", h.Login)

	body, _ := json.Marshal(map[string]string{"email": "test@test.com", "password": "wrong"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestRefreshToken(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("password123")
	u := &models.User{Email: "test@test.com", Name: "Test", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(context.Background(), u)

	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Post("/api/auth/refresh", h.RefreshToken)

	req := httptest.NewRequest("POST", "/api/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] == nil || resp["token"] == "" {
		t.Fatal("expected new token in response")
	}
}

func TestAuthMiddleware(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "a@a.com", Name: "A", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(context.Background(), u)

	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"user_id": UserID(r.Context()),
			"role":    UserRole(r.Context()),
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["role"] != "scheduler" {
		t.Errorf("role = %v, want scheduler", resp["role"])
	}
}
