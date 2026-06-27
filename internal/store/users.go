package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateUser(u *models.User) error {
	err := s.db.QueryRow(
		`INSERT INTO users (email, name, password_hash, role) VALUES ($1, $2, $3, $4) RETURNING id`,
		u.Email, u.Name, u.PasswordHash, u.Role,
	).Scan(&u.ID)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (s *Store) GetUserByID(id int64) (*models.User, error) {
	var u models.User
	err := s.db.QueryRow(
		`SELECT id, email, name, role, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return &u, err
}

func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	var u models.User
	err := s.db.QueryRow(
		`SELECT id, email, name, password_hash, role, created_at FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return &u, err
}

func (s *Store) ListUsers(limit, offset int) ([]*models.User, error) {
	rows, err := s.db.Query(`SELECT id, email, name, role, created_at FROM users ORDER BY id LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	var users []*models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

func (s *Store) UpdateUser(u *models.User) error {
	_, err := s.db.Exec(
		`UPDATE users SET email = $1, name = $2, password_hash = $3, role = $4 WHERE id = $5`,
		u.Email, u.Name, u.PasswordHash, u.Role, u.ID,
	)
	return err
}

func (s *Store) DeleteUser(id int64) error {
	_, err := s.db.Exec(`DELETE FROM users WHERE id = $1`, id)
	return err
}
