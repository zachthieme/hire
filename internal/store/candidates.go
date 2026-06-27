package store

import (
	"context"
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateCandidate(ctx context.Context, c *models.Candidate) error {
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO candidates (name, email, resume_url, status) VALUES ($1, $2, $3, $4) RETURNING id`,
		c.Name, c.Email, c.ResumeURL, c.Status,
	).Scan(&c.ID)
	if err != nil {
		return fmt.Errorf("insert candidate: %w", err)
	}
	return nil
}

func (s *Store) GetCandidate(ctx context.Context, id int64) (*models.Candidate, error) {
	var c models.Candidate
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, email, resume_url, status, created_at FROM candidates WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.Email, &c.ResumeURL, &c.Status, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &c, err
}

func (s *Store) ListCandidates(ctx context.Context, limit, offset int) ([]*models.Candidate, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, email, resume_url, status, created_at FROM candidates ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Candidate
	for rows.Next() {
		var c models.Candidate
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.ResumeURL, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (s *Store) UpdateCandidate(ctx context.Context, c *models.Candidate) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE candidates SET name = $1, email = $2, resume_url = $3, status = $4 WHERE id = $5`,
		c.Name, c.Email, c.ResumeURL, c.Status, c.ID,
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

func (s *Store) DeleteCandidate(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM candidates WHERE id = $1`, id)
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
