package store

import (
	"context"
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateInterview(ctx context.Context, iv *models.Interview) error {
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO interviews (loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		iv.LoopID, iv.InterviewerID, iv.FocusArea, iv.ScheduledAt, iv.VideoLink, iv.NotesForInterviewer, iv.Status,
	).Scan(&iv.ID)
	if err != nil {
		return fmt.Errorf("insert interview: %w", err)
	}
	return nil
}

func (s *Store) GetInterview(ctx context.Context, id int64) (*models.Interview, error) {
	var iv models.Interview
	err := s.db.QueryRowContext(ctx,
		`SELECT id, loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at
		 FROM interviews WHERE id = $1`, id,
	).Scan(&iv.ID, &iv.LoopID, &iv.InterviewerID, &iv.FocusArea, &iv.ScheduledAt, &iv.VideoLink,
		&iv.NotesForInterviewer, &iv.Status, &iv.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &iv, err
}

func (s *Store) ListInterviewsByLoop(ctx context.Context, loopID int64) ([]*models.Interview, error) {
	return s.queryInterviews(ctx, `SELECT id, loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at
		FROM interviews WHERE loop_id = $1 ORDER BY scheduled_at`, loopID)
}

func (s *Store) ListInterviewsByUser(ctx context.Context, userID int64) ([]*models.Interview, error) {
	return s.queryInterviews(ctx, `SELECT id, loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at
		FROM interviews WHERE interviewer_id = $1 ORDER BY scheduled_at DESC`, userID)
}

func (s *Store) UpdateInterview(ctx context.Context, iv *models.Interview) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE interviews SET interviewer_id = $1, focus_area = $2, scheduled_at = $3, video_link = $4, notes_for_interviewer = $5, status = $6
		 WHERE id = $7`,
		iv.InterviewerID, iv.FocusArea, iv.ScheduledAt, iv.VideoLink, iv.NotesForInterviewer, iv.Status, iv.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteInterview(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM interviews WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) queryInterviews(ctx context.Context, query string, args ...any) ([]*models.Interview, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Interview
	for rows.Next() {
		var iv models.Interview
		if err := rows.Scan(&iv.ID, &iv.LoopID, &iv.InterviewerID, &iv.FocusArea, &iv.ScheduledAt,
			&iv.VideoLink, &iv.NotesForInterviewer, &iv.Status, &iv.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &iv)
	}
	return out, rows.Err()
}
