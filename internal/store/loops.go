package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
	"strings"
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

func (s *Store) ListLoops(candidateID *int64, status *string, limit, offset int) ([]*models.InterviewLoop, error) {
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
	query += ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

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

	// Fetch interviews with interviewer names in one query
	rows, err := s.db.Query(
		`SELECT i.id, i.loop_id, i.interviewer_id, i.focus_area, i.scheduled_at, i.video_link,
		        i.notes_for_interviewer, i.status, i.created_at, u.name
		 FROM interviews i
		 JOIN users u ON i.interviewer_id = u.id
		 WHERE i.loop_id = ?
		 ORDER BY i.scheduled_at`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("list interviews for loop: %w", err)
	}
	defer rows.Close()

	var interviewIDs []int64
	interviewMap := make(map[int64]*models.InterviewWithFeedback)
	detail := &models.LoopDetail{
		InterviewLoop: *loop,
		Candidate:     *candidate,
	}

	for rows.Next() {
		var iwf models.InterviewWithFeedback
		if err := rows.Scan(
			&iwf.ID, &iwf.LoopID, &iwf.InterviewerID, &iwf.FocusArea, &iwf.ScheduledAt,
			&iwf.VideoLink, &iwf.NotesForInterviewer, &iwf.Status, &iwf.CreatedAt,
			&iwf.InterviewerName,
		); err != nil {
			return nil, err
		}
		detail.Interviews = append(detail.Interviews, iwf)
		interviewIDs = append(interviewIDs, iwf.ID)
		interviewMap[iwf.ID] = &detail.Interviews[len(detail.Interviews)-1]
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch all feedback for these interviews in one query
	if len(interviewIDs) > 0 {
		placeholders := make([]string, len(interviewIDs))
		args := make([]any, len(interviewIDs))
		for i, id := range interviewIDs {
			placeholders[i] = "?"
			args[i] = id
		}
		fbRows, err := s.db.Query(
			`SELECT f.id, f.interview_id, f.recommendation, f.recommendation_reason, f.free_form_notes, f.submitted_at
			 FROM feedback f WHERE f.interview_id IN (`+strings.Join(placeholders, ",")+`)`, args...,
		)
		if err != nil {
			return nil, err
		}
		defer fbRows.Close()

		for fbRows.Next() {
			var fb models.Feedback
			if err := fbRows.Scan(&fb.ID, &fb.InterviewID, &fb.Recommendation, &fb.RecommendationReason, &fb.FreeFormNotes, &fb.SubmittedAt); err != nil {
				return nil, err
			}
			// Load competency ratings for this feedback
			fb.CompetencyRatings, _ = s.listCompetencyRatings(fb.ID)
			if iwf, ok := interviewMap[fb.InterviewID]; ok {
				iwf.Feedback = &fb
			}
		}
	}

	return detail, nil
}
