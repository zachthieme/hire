package store

import (
	"context"
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateFeedback(ctx context.Context, fb *models.Feedback) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx,
		`INSERT INTO feedback (interview_id, recommendation, recommendation_reason, free_form_notes) VALUES ($1, $2, $3, $4) RETURNING id`,
		fb.InterviewID, fb.Recommendation, fb.RecommendationReason, fb.FreeFormNotes,
	).Scan(&fb.ID)
	if err != nil {
		return fmt.Errorf("insert feedback: %w", err)
	}

	for i := range fb.CompetencyRatings {
		cr := &fb.CompetencyRatings[i]
		cr.FeedbackID = fb.ID
		err := tx.QueryRowContext(ctx,
			`INSERT INTO competency_ratings (feedback_id, competency_id, rating_value) VALUES ($1, $2, $3) RETURNING id`,
			cr.FeedbackID, cr.CompetencyID, cr.RatingValue,
		).Scan(&cr.ID)
		if err != nil {
			return fmt.Errorf("insert competency rating: %w", err)
		}
	}

	// Mark the interview as complete
	if _, err := tx.ExecContext(ctx, `UPDATE interviews SET status = 'complete' WHERE id = $1`, fb.InterviewID); err != nil {
		return fmt.Errorf("mark interview complete: %w", err)
	}

	return tx.Commit()
}

func (s *Store) GetFeedback(ctx context.Context, id int64) (*models.Feedback, error) {
	var fb models.Feedback
	err := s.db.QueryRowContext(ctx,
		`SELECT id, interview_id, recommendation, recommendation_reason, free_form_notes, submitted_at
		 FROM feedback WHERE id = $1`, id,
	).Scan(&fb.ID, &fb.InterviewID, &fb.Recommendation, &fb.RecommendationReason, &fb.FreeFormNotes, &fb.SubmittedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	fb.CompetencyRatings, err = s.listCompetencyRatings(ctx, fb.ID)
	return &fb, err
}

func (s *Store) GetFeedbackByInterview(ctx context.Context, interviewID int64) (*models.Feedback, error) {
	var fb models.Feedback
	err := s.db.QueryRowContext(ctx,
		`SELECT id, interview_id, recommendation, recommendation_reason, free_form_notes, submitted_at
		 FROM feedback WHERE interview_id = $1`, interviewID,
	).Scan(&fb.ID, &fb.InterviewID, &fb.Recommendation, &fb.RecommendationReason, &fb.FreeFormNotes, &fb.SubmittedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	fb.CompetencyRatings, err = s.listCompetencyRatings(ctx, fb.ID)
	return &fb, err
}

func (s *Store) UpdateFeedback(ctx context.Context, fb *models.Feedback) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`UPDATE feedback SET recommendation = $1, recommendation_reason = $2, free_form_notes = $3 WHERE id = $4`,
		fb.Recommendation, fb.RecommendationReason, fb.FreeFormNotes, fb.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}

	// Replace competency ratings
	if len(fb.CompetencyRatings) > 0 {
		_, err = tx.ExecContext(ctx, `DELETE FROM competency_ratings WHERE feedback_id = $1`, fb.ID)
		if err != nil {
			return err
		}
		for i := range fb.CompetencyRatings {
			cr := &fb.CompetencyRatings[i]
			cr.FeedbackID = fb.ID
			err := tx.QueryRowContext(ctx,
				`INSERT INTO competency_ratings (feedback_id, competency_id, rating_value) VALUES ($1, $2, $3) RETURNING id`,
				cr.FeedbackID, cr.CompetencyID, cr.RatingValue,
			).Scan(&cr.ID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (s *Store) HasUserSubmittedFeedbackForLoop(ctx context.Context, loopID, userID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM feedback f
		 JOIN interviews i ON f.interview_id = i.id
		 WHERE i.loop_id = $1 AND i.interviewer_id = $2`, loopID, userID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
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
