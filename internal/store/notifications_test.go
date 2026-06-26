package store

import (
	"hire/internal/models"
	"testing"
)

func TestCreateAndListNotifications(t *testing.T) {
	s := newTestStore(t)
	u := &models.User{Email: "a@a.com", Name: "A", PasswordHash: "h", Role: "interviewer"}
	s.CreateUser(u)

	n := &models.Notification{UserID: u.ID, Message: "You have a new interview", Link: "/interviews/1"}
	if err := s.CreateNotification(n); err != nil {
		t.Fatalf("CreateNotification: %v", err)
	}

	list, err := s.ListNotificationsByUser(u.ID)
	if err != nil {
		t.Fatalf("ListNotificationsByUser: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d, want 1", len(list))
	}
	if list[0].Read {
		t.Error("expected unread")
	}
}

func TestMarkNotificationRead(t *testing.T) {
	s := newTestStore(t)
	u := &models.User{Email: "a@a.com", Name: "A", PasswordHash: "h", Role: "interviewer"}
	s.CreateUser(u)
	n := &models.Notification{UserID: u.ID, Message: "Test", Link: "/test"}
	s.CreateNotification(n)

	if err := s.MarkNotificationRead(n.ID, u.ID); err != nil {
		t.Fatalf("MarkNotificationRead: %v", err)
	}

	list, _ := s.ListNotificationsByUser(u.ID)
	if !list[0].Read {
		t.Error("expected read")
	}
}

func TestCountUnreadNotifications(t *testing.T) {
	s := newTestStore(t)
	u := &models.User{Email: "a@a.com", Name: "A", PasswordHash: "h", Role: "interviewer"}
	s.CreateUser(u)
	s.CreateNotification(&models.Notification{UserID: u.ID, Message: "1", Link: "/"})
	s.CreateNotification(&models.Notification{UserID: u.ID, Message: "2", Link: "/"})

	count, err := s.CountUnreadNotifications(u.ID)
	if err != nil {
		t.Fatalf("CountUnread: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}
