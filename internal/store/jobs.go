package store

import (
	"context"
	"database/sql"
	"fmt"

	"hire/internal/models"
)

func (s *Store) CreateJob(ctx context.Context, j *models.Job) error {
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO jobs (title, description, hiring_manager, status, created_by)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`,
		j.Title, j.Description, j.HiringManager, j.Status, j.CreatedBy,
	).Scan(&j.ID, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert job: %w", err)
	}
	return nil
}

func (s *Store) GetJob(ctx context.Context, id int64) (*models.Job, error) {
	var j models.Job
	err := s.db.QueryRowContext(ctx,
		`SELECT id, title, description, hiring_manager, status, created_by, created_at, updated_at
		 FROM jobs WHERE id = $1`, id,
	).Scan(&j.ID, &j.Title, &j.Description, &j.HiringManager, &j.Status, &j.CreatedBy, &j.CreatedAt, &j.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &j, err
}

func (s *Store) ListJobs(ctx context.Context, limit, offset int) ([]*models.Job, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, description, hiring_manager, status, created_by, created_at, updated_at
		 FROM jobs ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Job
	for rows.Next() {
		var j models.Job
		if err := rows.Scan(&j.ID, &j.Title, &j.Description, &j.HiringManager, &j.Status, &j.CreatedBy, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &j)
	}
	return out, rows.Err()
}

func (s *Store) GetJobDetail(ctx context.Context, id int64) (*models.JobDetail, error) {
	job, err := s.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	detail := &models.JobDetail{Job: *job}
	rows, err := s.db.QueryContext(ctx,
		`SELECT a.id, a.job_id, a.candidate_id, a.status, a.final_decision, a.final_interview_notes,
		        a.created_by, a.created_at, a.updated_at, c.name, c.email
		 FROM applications a JOIN candidates c ON c.id = a.candidate_id
		 WHERE a.job_id = $1 ORDER BY a.created_at DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a models.ApplicationSummary
		if err := rows.Scan(&a.ID, &a.JobID, &a.CandidateID, &a.Status, &a.FinalDecision, &a.FinalInterviewNotes,
			&a.CreatedBy, &a.CreatedAt, &a.UpdatedAt, &a.CandidateName, &a.CandidateEmail); err != nil {
			return nil, err
		}
		detail.Applications = append(detail.Applications, a)
	}
	return detail, rows.Err()
}

func (s *Store) UpdateJob(ctx context.Context, j *models.Job) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET title = $1, description = $2, hiring_manager = $3, status = $4, updated_at = NOW()
		 WHERE id = $5`,
		j.Title, j.Description, j.HiringManager, j.Status, j.ID)
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

func (s *Store) DeleteJob(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM jobs WHERE id = $1`, id)
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
