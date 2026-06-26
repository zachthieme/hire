package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateLoop(l *models.InterviewLoop) error {
	res, err := s.db.Exec(
		`INSERT INTO interview_loops (candidate_id, status, created_by) VALUES (?, ?, ?)`,
		l.CandidateID, l.Status, l.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("insert loop: %w", err)
	}
	l.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) GetLoop(id int64) (*models.InterviewLoop, error) {
	var l models.InterviewLoop
	err := s.db.QueryRow(
		`SELECT id, candidate_id, status, final_decision, debrief_notes, created_by, created_at
		 FROM interview_loops WHERE id = ?`, id,
	).Scan(&l.ID, &l.CandidateID, &l.Status, &l.FinalDecision, &l.DebriefNotes, &l.CreatedBy, &l.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("loop not found")
	}
	return &l, err
}

func (s *Store) ListLoops(candidateID *int64, status *string) ([]*models.InterviewLoop, error) {
	query := `SELECT id, candidate_id, status, final_decision, debrief_notes, created_by, created_at FROM interview_loops WHERE 1=1`
	var args []any
	if candidateID != nil {
		query += ` AND candidate_id = ?`
		args = append(args, *candidateID)
	}
	if status != nil {
		query += ` AND status = ?`
		args = append(args, *status)
	}
	query += ` ORDER BY id DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.InterviewLoop
	for rows.Next() {
		var l models.InterviewLoop
		if err := rows.Scan(&l.ID, &l.CandidateID, &l.Status, &l.FinalDecision, &l.DebriefNotes, &l.CreatedBy, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &l)
	}
	return out, rows.Err()
}

func (s *Store) UpdateLoop(l *models.InterviewLoop) error {
	_, err := s.db.Exec(
		`UPDATE interview_loops SET status = ?, final_decision = ?, debrief_notes = ? WHERE id = ?`,
		l.Status, l.FinalDecision, l.DebriefNotes, l.ID,
	)
	return err
}

func (s *Store) DeleteLoop(id int64) error {
	_, err := s.db.Exec(`DELETE FROM interview_loops WHERE id = ?`, id)
	return err
}

// GetLoopDetail returns a loop with its candidate, interviews, and feedback.
func (s *Store) GetLoopDetail(id int64) (*models.LoopDetail, error) {
	loop, err := s.GetLoop(id)
	if err != nil {
		return nil, err
	}
	candidate, err := s.GetCandidate(loop.CandidateID)
	if err != nil {
		return nil, fmt.Errorf("get candidate for loop: %w", err)
	}
	interviews, err := s.ListInterviewsByLoop(id)
	if err != nil {
		return nil, fmt.Errorf("list interviews for loop: %w", err)
	}

	detail := &models.LoopDetail{
		InterviewLoop: *loop,
		Candidate:     *candidate,
	}
	for _, iv := range interviews {
		iwf := models.InterviewWithFeedback{Interview: *iv}
		interviewer, err := s.GetUserByID(iv.InterviewerID)
		if err == nil {
			iwf.InterviewerName = interviewer.Name
		}
		fb, err := s.GetFeedbackByInterview(iv.ID)
		if err == nil {
			iwf.Feedback = fb
		}
		detail.Interviews = append(detail.Interviews, iwf)
	}
	return detail, nil
}
