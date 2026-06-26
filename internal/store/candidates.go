package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateCandidate(c *models.Candidate) error {
	res, err := s.db.Exec(
		`INSERT INTO candidates (name, email, resume_url, status) VALUES (?, ?, ?, ?)`,
		c.Name, c.Email, c.ResumeURL, c.Status,
	)
	if err != nil {
		return fmt.Errorf("insert candidate: %w", err)
	}
	c.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) GetCandidate(id int64) (*models.Candidate, error) {
	var c models.Candidate
	err := s.db.QueryRow(
		`SELECT id, name, email, resume_url, status, created_at FROM candidates WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.Email, &c.ResumeURL, &c.Status, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("candidate not found")
	}
	return &c, err
}

func (s *Store) ListCandidates() ([]*models.Candidate, error) {
	rows, err := s.db.Query(`SELECT id, name, email, resume_url, status, created_at FROM candidates ORDER BY id DESC`)
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

func (s *Store) UpdateCandidate(c *models.Candidate) error {
	_, err := s.db.Exec(
		`UPDATE candidates SET name = ?, email = ?, resume_url = ?, status = ? WHERE id = ?`,
		c.Name, c.Email, c.ResumeURL, c.Status, c.ID,
	)
	return err
}

func (s *Store) DeleteCandidate(id int64) error {
	_, err := s.db.Exec(`DELETE FROM candidates WHERE id = ?`, id)
	return err
}
