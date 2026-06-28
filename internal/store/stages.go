package store

import (
	"context"
	"database/sql"
	"fmt"

	"hire/internal/models"
)

func (s *Store) CreateStage(ctx context.Context, st *models.Stage) error {
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO stages (application_id, type, focus_area, scheduled_at, video_link, notes_for_interviewer, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at, updated_at`,
		st.ApplicationID, st.Type, st.FocusArea, st.ScheduledAt, st.VideoLink, st.NotesForInterviewer, st.Status,
	).Scan(&st.ID, &st.CreatedAt, &st.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert stage: %w", err)
	}
	return nil
}

func (s *Store) GetStage(ctx context.Context, id int64) (*models.Stage, error) {
	var st models.Stage
	err := s.db.QueryRowContext(ctx,
		`SELECT id, application_id, type, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at, updated_at
		 FROM stages WHERE id = $1`, id,
	).Scan(&st.ID, &st.ApplicationID, &st.Type, &st.FocusArea, &st.ScheduledAt, &st.VideoLink,
		&st.NotesForInterviewer, &st.Status, &st.CreatedAt, &st.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &st, err
}

func (s *Store) ListStagesByApplication(ctx context.Context, appID int64) ([]*models.Stage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, application_id, type, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at, updated_at
		 FROM stages WHERE application_id = $1 ORDER BY scheduled_at`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Stage
	for rows.Next() {
		var st models.Stage
		if err := rows.Scan(&st.ID, &st.ApplicationID, &st.Type, &st.FocusArea, &st.ScheduledAt, &st.VideoLink,
			&st.NotesForInterviewer, &st.Status, &st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &st)
	}
	return out, rows.Err()
}

func (s *Store) UpdateStage(ctx context.Context, st *models.Stage) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE stages SET type = $1, focus_area = $2, scheduled_at = $3, video_link = $4,
		        notes_for_interviewer = $5, status = $6, updated_at = NOW()
		 WHERE id = $7`,
		st.Type, st.FocusArea, st.ScheduledAt, st.VideoLink, st.NotesForInterviewer, st.Status, st.ID)
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

func (s *Store) DeleteStage(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM stages WHERE id = $1`, id)
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

func (s *Store) AddStageInterviewer(ctx context.Context, stageID, interviewerID int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO stage_interviewers (stage_id, interviewer_id) VALUES ($1, $2)
		 ON CONFLICT (stage_id, interviewer_id) DO NOTHING`, stageID, interviewerID)
	if err != nil {
		return fmt.Errorf("add stage interviewer: %w", err)
	}
	return nil
}

func (s *Store) RemoveStageInterviewer(ctx context.Context, stageID, interviewerID int64) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM stage_interviewers WHERE stage_id = $1 AND interviewer_id = $2`, stageID, interviewerID)
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

func (s *Store) IsStageInterviewer(ctx context.Context, stageID, interviewerID int64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM stage_interviewers WHERE stage_id = $1 AND interviewer_id = $2`,
		stageID, interviewerID).Scan(&n)
	return n > 0, err
}

// ListStagesByUser returns stages the user is assigned to, enriched for the
// "My Interviews" list.
func (s *Store) ListStagesByUser(ctx context.Context, userID int64, limit, offset int) ([]*models.MyStage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT st.id, st.application_id, st.type, st.focus_area, st.scheduled_at, st.video_link,
		        st.notes_for_interviewer, st.status, st.created_at, st.updated_at,
		        c.name, j.title,
		        EXISTS(SELECT 1 FROM feedback f WHERE f.stage_id = st.id AND f.interviewer_id = $1) AS has_my_feedback
		 FROM stage_interviewers si
		 JOIN stages st ON st.id = si.stage_id
		 JOIN applications a ON a.id = st.application_id
		 JOIN candidates c ON c.id = a.candidate_id
		 JOIN jobs j ON j.id = a.job_id
		 WHERE si.interviewer_id = $1
		 ORDER BY st.scheduled_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.MyStage
	for rows.Next() {
		var m models.MyStage
		if err := rows.Scan(&m.ID, &m.ApplicationID, &m.Type, &m.FocusArea, &m.ScheduledAt, &m.VideoLink,
			&m.NotesForInterviewer, &m.Status, &m.CreatedAt, &m.UpdatedAt,
			&m.CandidateName, &m.JobTitle, &m.HasMyFeedback); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// CountIncompleteStages counts stages on an application not yet complete.
func (s *Store) CountIncompleteStages(ctx context.Context, appID int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM stages WHERE application_id = $1 AND status != $2`,
		appID, models.StageStatusComplete).Scan(&n)
	return n, err
}
