package store

import (
	"context"
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateUser(ctx context.Context, u *models.User) error {
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (email, name, password_hash, role) VALUES ($1, $2, $3, $4) RETURNING id`,
		u.Email, u.Name, u.PasswordHash, u.Role,
	).Scan(&u.ID)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	var u models.User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, name, role, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &u, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, name, password_hash, role, created_at FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &u, err
}

func (s *Store) ListUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, email, name, role, created_at FROM users ORDER BY id LIMIT $1 OFFSET $2`, limit, offset,
	)
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

func (s *Store) UpdateUser(ctx context.Context, u *models.User) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET email = $1, name = $2, role = $3 WHERE id = $4`,
		u.Email, u.Name, u.Role, u.ID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`,
		passwordHash, id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
