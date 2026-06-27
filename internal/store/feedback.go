package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateFeedback(fb *models.Feedback) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = tx.QueryRow(
		`INSERT INTO feedback (interview_id, recommendation, recommendation_reason, free_form_notes) VALUES ($1, $2, $3, $4) RETURNING id`,
		fb.InterviewID, fb.Recommendation, fb.RecommendationReason, fb.FreeFormNotes,
	).Scan(&fb.ID)
	if err != nil {
		return fmt.Errorf("insert feedback: %w", err)
	}

	for i := range fb.CompetencyRatings {
		cr := &fb.CompetencyRatings[i]
		cr.FeedbackID = fb.ID
		err := tx.QueryRow(
			`INSERT INTO competency_ratings (feedback_id, competency_id, rating_value) VALUES ($1, $2, $3) RETURNING id`,
			cr.FeedbackID, cr.CompetencyID, cr.RatingValue,
		).Scan(&cr.ID)
		if err != nil {
			return fmt.Errorf("insert competency rating: %w", err)
		}
	}

	// Mark the interview as complete
	if _, err := tx.Exec(`UPDATE interviews SET status = 'complete' WHERE id = $1`, fb.InterviewID); err != nil {
		return fmt.Errorf("mark interview complete: %w", err)
	}

	return tx.Commit()
}

func (s *Store) GetFeedback(id int64) (*models.Feedback, error) {
	var fb models.Feedback
	err := s.db.QueryRow(
		`SELECT id, interview_id, recommendation, recommendation_reason, free_form_notes, submitted_at
		 FROM feedback WHERE id = $1`, id,
	).Scan(&fb.ID, &fb.InterviewID, &fb.Recommendation, &fb.RecommendationReason, &fb.FreeFormNotes, &fb.SubmittedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("feedback not found")
	}
	if err != nil {
		return nil, err
	}
	fb.CompetencyRatings, err = s.listCompetencyRatings(fb.ID)
	return &fb, err
}

func (s *Store) GetFeedbackByInterview(interviewID int64) (*models.Feedback, error) {
	var fb models.Feedback
	err := s.db.QueryRow(
		`SELECT id, interview_id, recommendation, recommendation_reason, free_form_notes, submitted_at
		 FROM feedback WHERE interview_id = $1`, interviewID,
	).Scan(&fb.ID, &fb.InterviewID, &fb.Recommendation, &fb.RecommendationReason, &fb.FreeFormNotes, &fb.SubmittedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("feedback not found")
	}
	if err != nil {
		return nil, err
	}
	fb.CompetencyRatings, err = s.listCompetencyRatings(fb.ID)
	return &fb, err
}

func (s *Store) UpdateFeedback(fb *models.Feedback) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`UPDATE feedback SET recommendation = $1, recommendation_reason = $2, free_form_notes = $3 WHERE id = $4`,
		fb.Recommendation, fb.RecommendationReason, fb.FreeFormNotes, fb.ID,
	)
	if err != nil {
		return err
	}

	// Replace competency ratings
	if len(fb.CompetencyRatings) > 0 {
		_, err = tx.Exec(`DELETE FROM competency_ratings WHERE feedback_id = $1`, fb.ID)
		if err != nil {
			return err
		}
		for i := range fb.CompetencyRatings {
			cr := &fb.CompetencyRatings[i]
			cr.FeedbackID = fb.ID
			err := tx.QueryRow(
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

func (s *Store) HasUserSubmittedFeedbackForLoop(loopID, userID int64) bool {
	var count int
	s.db.QueryRow(
		`SELECT COUNT(*) FROM feedback f
		 JOIN interviews i ON f.interview_id = i.id
		 WHERE i.loop_id = $1 AND i.interviewer_id = $2`, loopID, userID,
	).Scan(&count)
	return count > 0
}

func (s *Store) listCompetencyRatings(feedbackID int64) ([]models.CompetencyRating, error) {
	rows, err := s.db.Query(
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
