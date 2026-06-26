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

	res, err := tx.Exec(
		`INSERT INTO feedback (interview_id, recommendation, recommendation_reason, free_form_notes) VALUES (?, ?, ?, ?)`,
		fb.InterviewID, fb.Recommendation, fb.RecommendationReason, fb.FreeFormNotes,
	)
	if err != nil {
		return fmt.Errorf("insert feedback: %w", err)
	}
	fb.ID, _ = res.LastInsertId()

	for i := range fb.CompetencyRatings {
		cr := &fb.CompetencyRatings[i]
		cr.FeedbackID = fb.ID
		res, err := tx.Exec(
			`INSERT INTO competency_ratings (feedback_id, competency_id, rating_value) VALUES (?, ?, ?)`,
			cr.FeedbackID, cr.CompetencyID, cr.RatingValue,
		)
		if err != nil {
			return fmt.Errorf("insert competency rating: %w", err)
		}
		cr.ID, _ = res.LastInsertId()
	}

	// Mark the interview as complete
	if _, err := tx.Exec(`UPDATE interviews SET status = 'complete' WHERE id = ?`, fb.InterviewID); err != nil {
		return fmt.Errorf("mark interview complete: %w", err)
	}

	return tx.Commit()
}

func (s *Store) GetFeedback(id int64) (*models.Feedback, error) {
	var fb models.Feedback
	err := s.db.QueryRow(
		`SELECT id, interview_id, recommendation, recommendation_reason, free_form_notes, submitted_at
		 FROM feedback WHERE id = ?`, id,
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
		 FROM feedback WHERE interview_id = ?`, interviewID,
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
	_, err := s.db.Exec(
		`UPDATE feedback SET recommendation = ?, recommendation_reason = ?, free_form_notes = ? WHERE id = ?`,
		fb.Recommendation, fb.RecommendationReason, fb.FreeFormNotes, fb.ID,
	)
	return err
}

func (s *Store) HasUserSubmittedFeedbackForLoop(loopID, userID int64) bool {
	var count int
	s.db.QueryRow(
		`SELECT COUNT(*) FROM feedback f
		 JOIN interviews i ON f.interview_id = i.id
		 WHERE i.loop_id = ? AND i.interviewer_id = ?`, loopID, userID,
	).Scan(&count)
	return count > 0
}

func (s *Store) listCompetencyRatings(feedbackID int64) ([]models.CompetencyRating, error) {
	rows, err := s.db.Query(
		`SELECT id, feedback_id, competency_id, rating_value FROM competency_ratings WHERE feedback_id = ?`, feedbackID,
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
