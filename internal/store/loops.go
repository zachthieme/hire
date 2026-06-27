package store

import (
	"context"
	"database/sql"
	"fmt"
	"hire/internal/models"
	"strings"
)

func (s *Store) CreateLoop(ctx context.Context, l *models.InterviewLoop) error {
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO interview_loops (candidate_id, status, created_by) VALUES ($1, $2, $3) RETURNING id`,
		l.CandidateID, l.Status, l.CreatedBy,
	).Scan(&l.ID)
	if err != nil {
		return fmt.Errorf("insert loop: %w", err)
	}
	return nil
}

func (s *Store) GetLoop(ctx context.Context, id int64) (*models.InterviewLoop, error) {
	var l models.InterviewLoop
	err := s.db.QueryRowContext(ctx,
		`SELECT id, candidate_id, status, final_decision, debrief_notes, created_by, created_at, updated_at
		 FROM interview_loops WHERE id = $1`, id,
	).Scan(&l.ID, &l.CandidateID, &l.Status, &l.FinalDecision, &l.DebriefNotes, &l.CreatedBy, &l.CreatedAt, &l.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &l, err
}

func (s *Store) ListLoops(ctx context.Context, candidateID *int64, status *string, limit, offset int) ([]*models.InterviewLoop, error) {
	query := `SELECT id, candidate_id, status, final_decision, debrief_notes, created_by, created_at, updated_at FROM interview_loops WHERE 1=1`
	var args []any
	paramIdx := 1
	if candidateID != nil {
		query += fmt.Sprintf(` AND candidate_id = $%d`, paramIdx)
		args = append(args, *candidateID)
		paramIdx++
	}
	if status != nil {
		query += fmt.Sprintf(` AND status = $%d`, paramIdx)
		args = append(args, *status)
		paramIdx++
	}
	query += fmt.Sprintf(` ORDER BY id DESC LIMIT $%d OFFSET $%d`, paramIdx, paramIdx+1)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.InterviewLoop
	for rows.Next() {
		var l models.InterviewLoop
		if err := rows.Scan(&l.ID, &l.CandidateID, &l.Status, &l.FinalDecision, &l.DebriefNotes, &l.CreatedBy, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &l)
	}
	return out, rows.Err()
}

func (s *Store) UpdateLoop(ctx context.Context, l *models.InterviewLoop) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE interview_loops SET status = $1, final_decision = $2, debrief_notes = $3, updated_at = NOW() WHERE id = $4`,
		l.Status, l.FinalDecision, l.DebriefNotes, l.ID,
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

func (s *Store) DeleteLoop(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM interview_loops WHERE id = $1`, id)
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

// GetLoopDetail returns a loop with its candidate, interviews, and feedback.
func (s *Store) GetLoopDetail(ctx context.Context, id int64) (*models.LoopDetail, error) {
	loop, err := s.GetLoop(ctx, id)
	if err != nil {
		return nil, err
	}
	candidate, err := s.GetCandidate(ctx, loop.CandidateID)
	if err != nil {
		return nil, fmt.Errorf("get candidate for loop: %w", err)
	}

	// Fetch interviews with interviewer names in one query
	rows, err := s.db.QueryContext(ctx,
		`SELECT i.id, i.loop_id, i.interviewer_id, i.focus_area, i.scheduled_at, i.video_link,
		        i.notes_for_interviewer, i.status, i.created_at, i.updated_at, u.name
		 FROM interviews i
		 JOIN users u ON i.interviewer_id = u.id
		 WHERE i.loop_id = $1
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
			&iwf.VideoLink, &iwf.NotesForInterviewer, &iwf.Status, &iwf.CreatedAt, &iwf.UpdatedAt,
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
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = id
		}
		fbRows, err := s.db.QueryContext(ctx,
			`SELECT f.id, f.interview_id, f.recommendation, f.recommendation_reason, f.free_form_notes, f.submitted_at, f.updated_at
			 FROM feedback f WHERE f.interview_id IN (`+strings.Join(placeholders, ",")+`)`, args...,
		)
		if err != nil {
			return nil, err
		}
		defer fbRows.Close()

		for fbRows.Next() {
			var fb models.Feedback
			if err := fbRows.Scan(&fb.ID, &fb.InterviewID, &fb.Recommendation, &fb.RecommendationReason, &fb.FreeFormNotes, &fb.SubmittedAt, &fb.UpdatedAt); err != nil {
				return nil, err
			}
			if iwf, ok := interviewMap[fb.InterviewID]; ok {
				iwf.Feedback = &fb
			}
		}

		// Batch-fetch all competency ratings
		var feedbackIDs []int64
		for _, iwf := range detail.Interviews {
			if iwf.Feedback != nil {
				feedbackIDs = append(feedbackIDs, iwf.Feedback.ID)
			}
		}
		if len(feedbackIDs) > 0 {
			crPlaceholders := make([]string, len(feedbackIDs))
			crArgs := make([]any, len(feedbackIDs))
			for i, fid := range feedbackIDs {
				crPlaceholders[i] = fmt.Sprintf("$%d", i+1)
				crArgs[i] = fid
			}
			crRows, err := s.db.QueryContext(ctx,
				`SELECT id, feedback_id, competency_id, rating_value FROM competency_ratings WHERE feedback_id IN (`+strings.Join(crPlaceholders, ",")+`)`,
				crArgs...,
			)
			if err != nil {
				return nil, fmt.Errorf("list competency ratings: %w", err)
			}
			defer crRows.Close()
			for crRows.Next() {
				var cr models.CompetencyRating
				if err := crRows.Scan(&cr.ID, &cr.FeedbackID, &cr.CompetencyID, &cr.RatingValue); err != nil {
					return nil, err
				}
				for i := range detail.Interviews {
					if detail.Interviews[i].Feedback != nil && detail.Interviews[i].Feedback.ID == cr.FeedbackID {
						detail.Interviews[i].Feedback.CompetencyRatings = append(detail.Interviews[i].Feedback.CompetencyRatings, cr)
						break
					}
				}
			}
		}
	}

	return detail, nil
}
