package api

import (
	"context"
	"encoding/json"
	"hire/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestNotificationListAndMarkRead(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "iv@test.com", Name: "IV", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(context.Background(), u)
	token, _ := h.generateToken(u.ID, u.Role)

	// Create test notifications
	s.CreateNotification(context.Background(), &models.Notification{
		UserID: u.ID, Message: "Test notification 1", Link: "/test/1",
	})
	s.CreateNotification(context.Background(), &models.Notification{
		UserID: u.ID, Message: "Test notification 2", Link: "/test/2",
	})

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Get("/api/notifications", h.ListNotifications)
	r.Put("/api/notifications/{id}/read", h.MarkNotificationRead)

	// List notifications
	req := httptest.NewRequest("GET", "/api/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list: status = %d", w.Code)
	}
	var notifications []models.Notification
	json.Unmarshal(w.Body.Bytes(), &notifications)
	if len(notifications) != 2 {
		t.Fatalf("got %d, want 2", len(notifications))
	}

	// Mark as read
	req = httptest.NewRequest("PUT", "/api/notifications/1/read", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("mark read: status = %d; body: %s", w.Code, w.Body.String())
	}

	// Verify it's marked read
	req = httptest.NewRequest("GET", "/api/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &notifications)
	readCount := 0
	for _, n := range notifications {
		if n.Read {
			readCount++
		}
	}
	if readCount != 1 {
		t.Errorf("read count = %d, want 1", readCount)
	}
}

func TestMarkNotificationReadNotFound(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "iv@test.com", Name: "IV", PasswordHash: hash, Role: "interviewer"}
	s.CreateUser(context.Background(), u)
	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Put("/api/notifications/{id}/read", h.MarkNotificationRead)

	req := httptest.NewRequest("PUT", "/api/notifications/999/read", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
