package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateCompetency(c *models.Competency) error {
	err := s.db.QueryRow(
		`INSERT INTO competencies (name, rating_type, ratings_json) VALUES ($1, $2, $3) RETURNING id`,
		c.Name, c.RatingType, c.RatingsJSON,
	).Scan(&c.ID)
	if err != nil {
		return fmt.Errorf("insert competency: %w", err)
	}
	return nil
}

func (s *Store) GetCompetency(id int64) (*models.Competency, error) {
	var c models.Competency
	err := s.db.QueryRow(
		`SELECT id, name, rating_type, ratings_json, created_at FROM competencies WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.RatingType, &c.RatingsJSON, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("competency not found")
	}
	return &c, err
}

func (s *Store) ListCompetencies() ([]*models.Competency, error) {
	rows, err := s.db.Query(`SELECT id, name, rating_type, ratings_json, created_at FROM competencies ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Competency
	for rows.Next() {
		var c models.Competency
		if err := rows.Scan(&c.ID, &c.Name, &c.RatingType, &c.RatingsJSON, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (s *Store) UpdateCompetency(c *models.Competency) error {
	_, err := s.db.Exec(
		`UPDATE competencies SET name = $1, rating_type = $2, ratings_json = $3 WHERE id = $4`,
		c.Name, c.RatingType, c.RatingsJSON, c.ID,
	)
	return err
}

func (s *Store) DeleteCompetency(id int64) error {
	_, err := s.db.Exec(`DELETE FROM competencies WHERE id = $1`, id)
	return err
}
