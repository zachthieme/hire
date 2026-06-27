package store

import (
	"context"
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateCompetency(ctx context.Context, c *models.Competency) error {
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO competencies (name, rating_type, ratings_json) VALUES ($1, $2, $3) RETURNING id`,
		c.Name, c.RatingType, c.RatingsJSON,
	).Scan(&c.ID)
	if err != nil {
		return fmt.Errorf("insert competency: %w", err)
	}
	return nil
}

func (s *Store) GetCompetency(ctx context.Context, id int64) (*models.Competency, error) {
	var c models.Competency
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, rating_type, ratings_json, created_at, updated_at FROM competencies WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.RatingType, &c.RatingsJSON, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &c, err
}

func (s *Store) ListCompetencies(ctx context.Context, limit, offset int) ([]*models.Competency, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, rating_type, ratings_json, created_at, updated_at FROM competencies ORDER BY id LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Competency
	for rows.Next() {
		var c models.Competency
		if err := rows.Scan(&c.ID, &c.Name, &c.RatingType, &c.RatingsJSON, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (s *Store) UpdateCompetency(ctx context.Context, c *models.Competency) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE competencies SET name = $1, rating_type = $2, ratings_json = $3, updated_at = NOW() WHERE id = $4`,
		c.Name, c.RatingType, c.RatingsJSON, c.ID,
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

func (s *Store) DeleteCompetency(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM competencies WHERE id = $1`, id)
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
