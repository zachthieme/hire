package store

import (
	"context"
	"database/sql"
	"fmt"

	"hire/internal/models"
)

// CreateFeedback inserts feedback for (stage, interviewer), records competency
// ratings, marks the stage complete, and reports whether the whole application
// is now ready for a decision (all stages complete).
func (s *Store) CreateFeedback(ctx context.Context, fb *models.Feedback) (appReady bool, applicationID int64, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx,
		`INSERT INTO feedback (stage_id, interviewer_id, recommendation, recommendation_reason, free_form_notes)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		fb.StageID, fb.InterviewerID, fb.Recommendation, fb.RecommendationReason, fb.FreeFormNotes,
	).Scan(&fb.ID)
	if err != nil {
		return false, 0, fmt.Errorf("insert feedback: %w", err)
	}

	for i := range fb.CompetencyRatings {
		cr := &fb.CompetencyRatings[i]
		cr.FeedbackID = fb.ID
		if err := tx.QueryRowContext(ctx,
			`INSERT INTO competency_ratings (feedback_id, competency_id, rating_value) VALUES ($1, $2, $3) RETURNING id`,
			cr.FeedbackID, cr.CompetencyID, cr.RatingValue,
		).Scan(&cr.ID); err != nil {
			return false, 0, fmt.Errorf("insert competency rating: %w", err)
		}
	}

	// Mark the stage complete only once every assigned interviewer has submitted.
	var remaining int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM stage_interviewers si
		 WHERE si.stage_id = $1
		   AND NOT EXISTS (
		     SELECT 1 FROM feedback f
		     WHERE f.stage_id = si.stage_id AND f.interviewer_id = si.interviewer_id
		   )`, fb.StageID).Scan(&remaining); err != nil {
		return false, 0, fmt.Errorf("count remaining interviewers: %w", err)
	}
	if remaining == 0 {
		if _, err := tx.ExecContext(ctx,
			`UPDATE stages SET status = $1, updated_at = NOW() WHERE id = $2`,
			models.StageStatusComplete, fb.StageID); err != nil {
			return false, 0, fmt.Errorf("mark stage complete: %w", err)
		}
	}

	if err := tx.QueryRowContext(ctx,
		`SELECT application_id FROM stages WHERE id = $1`, fb.StageID).Scan(&applicationID); err != nil {
		return false, 0, fmt.Errorf("get application_id: %w", err)
	}
	// A stage counts as done when it is complete or canceled; a canceled stage
	// must not block application readiness.
	var incomplete int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM stages WHERE application_id = $1 AND status NOT IN ('complete','canceled')`,
		applicationID).Scan(&incomplete); err != nil {
		return false, 0, fmt.Errorf("count incomplete: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, 0, fmt.Errorf("commit: %w", err)
	}
	return incomplete == 0, applicationID, nil
}

func (s *Store) GetFeedback(ctx context.Context, id int64) (*models.Feedback, error) {
	var fb models.Feedback
	err := s.db.QueryRowContext(ctx,
		`SELECT id, stage_id, interviewer_id, recommendation, recommendation_reason, free_form_notes, submitted_at, updated_at
		 FROM feedback WHERE id = $1`, id,
	).Scan(&fb.ID, &fb.StageID, &fb.InterviewerID, &fb.Recommendation, &fb.RecommendationReason,
		&fb.FreeFormNotes, &fb.SubmittedAt, &fb.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	fb.CompetencyRatings, err = s.listCompetencyRatings(ctx, fb.ID)
	return &fb, err
}

func (s *Store) GetFeedbackByStageAndInterviewer(ctx context.Context, stageID, interviewerID int64) (*models.Feedback, error) {
	var fb models.Feedback
	err := s.db.QueryRowContext(ctx,
		`SELECT id, stage_id, interviewer_id, recommendation, recommendation_reason, free_form_notes, submitted_at, updated_at
		 FROM feedback WHERE stage_id = $1 AND interviewer_id = $2`, stageID, interviewerID,
	).Scan(&fb.ID, &fb.StageID, &fb.InterviewerID, &fb.Recommendation, &fb.RecommendationReason,
		&fb.FreeFormNotes, &fb.SubmittedAt, &fb.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	fb.CompetencyRatings, err = s.listCompetencyRatings(ctx, fb.ID)
	return &fb, err
}

func (s *Store) ListFeedbackByStage(ctx context.Context, stageID int64) ([]*models.Feedback, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, stage_id, interviewer_id, recommendation, recommendation_reason, free_form_notes, submitted_at, updated_at
		 FROM feedback WHERE stage_id = $1 ORDER BY submitted_at`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Feedback
	for rows.Next() {
		var fb models.Feedback
		if err := rows.Scan(&fb.ID, &fb.StageID, &fb.InterviewerID, &fb.Recommendation, &fb.RecommendationReason,
			&fb.FreeFormNotes, &fb.SubmittedAt, &fb.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &fb)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, fb := range out {
		fb.CompetencyRatings, err = s.listCompetencyRatings(ctx, fb.ID)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Store) UpdateFeedback(ctx context.Context, fb *models.Feedback) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`UPDATE feedback SET recommendation = $1, recommendation_reason = $2, free_form_notes = $3, updated_at = NOW() WHERE id = $4`,
		fb.Recommendation, fb.RecommendationReason, fb.FreeFormNotes, fb.ID,
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

	if len(fb.CompetencyRatings) > 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM competency_ratings WHERE feedback_id = $1`, fb.ID); err != nil {
			return err
		}
		for i := range fb.CompetencyRatings {
			cr := &fb.CompetencyRatings[i]
			cr.FeedbackID = fb.ID
			if err := tx.QueryRowContext(ctx,
				`INSERT INTO competency_ratings (feedback_id, competency_id, rating_value) VALUES ($1, $2, $3) RETURNING id`,
				cr.FeedbackID, cr.CompetencyID, cr.RatingValue,
			).Scan(&cr.ID); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) listCompetencyRatings(ctx context.Context, feedbackID int64) ([]models.CompetencyRating, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, feedback_id, competency_id, rating_value FROM competency_ratings WHERE feedback_id = $1`, feedbackID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.CompetencyRating
	for rows.Next() {
		var cr models.CompetencyRating
		if err := rows.Scan(&cr.ID, &cr.FeedbackID, &cr.CompetencyID, &cr.RatingValue); err != nil {
			return nil, err
		}
		out = append(out, cr)
	}
	return out, rows.Err()
}
