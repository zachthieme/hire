package store

import (
	"hire/internal/models"
	"testing"
)

func TestCreateAndGetUser(t *testing.T) {
	s := newTestStore(t)
	u := &models.User{Email: "alice@example.com", Name: "Alice", PasswordHash: "hash123", Role: "interviewer"}
	if err := s.CreateUser(u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.ID == 0 {
		t.Fatal("expected ID to be set")
	}

	got, err := s.GetUserByID(u.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.Email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", got.Email)
	}
	if got.Role != "interviewer" {
		t.Errorf("role = %q, want interviewer", got.Role)
	}
}

func TestGetUserByEmail(t *testing.T) {
	s := newTestStore(t)
	u := &models.User{Email: "bob@example.com", Name: "Bob", PasswordHash: "hash", Role: "scheduler"}
	s.CreateUser(u)

	got, err := s.GetUserByEmail("bob@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if got.Name != "Bob" {
		t.Errorf("name = %q, want Bob", got.Name)
	}
}

func TestListUsers(t *testing.T) {
	s := newTestStore(t)
	s.CreateUser(&models.User{Email: "a@test.com", Name: "A", PasswordHash: "h", Role: "admin"})
	s.CreateUser(&models.User{Email: "b@test.com", Name: "B", PasswordHash: "h", Role: "interviewer"})

	users, err := s.ListUsers(50, 0)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("got %d users, want 2", len(users))
	}
}

func TestUpdateUser(t *testing.T) {
	s := newTestStore(t)
	u := &models.User{Email: "c@test.com", Name: "C", PasswordHash: "h", Role: "interviewer"}
	s.CreateUser(u)

	u.Name = "Charlie"
	u.Role = "scheduler"
	if err := s.UpdateUser(u); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	got, _ := s.GetUserByID(u.ID)
	if got.Name != "Charlie" || got.Role != "scheduler" {
		t.Errorf("got name=%q role=%q, want Charlie scheduler", got.Name, got.Role)
	}
}

func TestDeleteUser(t *testing.T) {
	s := newTestStore(t)
	u := &models.User{Email: "d@test.com", Name: "D", PasswordHash: "h", Role: "admin"}
	s.CreateUser(u)

	if err := s.DeleteUser(u.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	_, err := s.GetUserByID(u.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
