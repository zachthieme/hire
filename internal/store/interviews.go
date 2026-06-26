package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateInterview(iv *models.Interview) error {
	res, err := s.db.Exec(
		`INSERT INTO interviews (loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		iv.LoopID, iv.InterviewerID, iv.FocusArea, iv.ScheduledAt, iv.VideoLink, iv.NotesForInterviewer, iv.Status,
	)
	if err != nil {
		return fmt.Errorf("insert interview: %w", err)
	}
	iv.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) GetInterview(id int64) (*models.Interview, error) {
	var iv models.Interview
	err := s.db.QueryRow(
		`SELECT id, loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at
		 FROM interviews WHERE id = ?`, id,
	).Scan(&iv.ID, &iv.LoopID, &iv.InterviewerID, &iv.FocusArea, &iv.ScheduledAt, &iv.VideoLink,
		&iv.NotesForInterviewer, &iv.Status, &iv.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("interview not found")
	}
	return &iv, err
}

func (s *Store) ListInterviewsByLoop(loopID int64) ([]*models.Interview, error) {
	return s.queryInterviews(`SELECT id, loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at
		FROM interviews WHERE loop_id = ? ORDER BY scheduled_at`, loopID)
}

func (s *Store) ListInterviewsByUser(userID int64) ([]*models.Interview, error) {
	return s.queryInterviews(`SELECT id, loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at
		FROM interviews WHERE interviewer_id = ? ORDER BY scheduled_at DESC`, userID)
}

func (s *Store) UpdateInterview(iv *models.Interview) error {
	_, err := s.db.Exec(
		`UPDATE interviews SET interviewer_id = ?, focus_area = ?, scheduled_at = ?, video_link = ?, notes_for_interviewer = ?, status = ?
		 WHERE id = ?`,
		iv.InterviewerID, iv.FocusArea, iv.ScheduledAt, iv.VideoLink, iv.NotesForInterviewer, iv.Status, iv.ID,
	)
	return err
}

func (s *Store) DeleteInterview(id int64) error {
	_, err := s.db.Exec(`DELETE FROM interviews WHERE id = ?`, id)
	return err
}

func (s *Store) queryInterviews(query string, args ...any) ([]*models.Interview, error) {
	rows, err := s.db.Query(query, args...)
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
