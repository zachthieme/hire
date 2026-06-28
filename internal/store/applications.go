package store

import (
	"context"
	"database/sql"
	"fmt"

	"hire/internal/models"
)

func (s *Store) CreateApplication(ctx context.Context, a *models.Application) error {
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO applications (job_id, candidate_id, status, created_by)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		a.JobID, a.CandidateID, a.Status, a.CreatedBy,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert application: %w", err)
	}
	return nil
}

func (s *Store) GetApplication(ctx context.Context, id int64) (*models.Application, error) {
	var a models.Application
	err := s.db.QueryRowContext(ctx,
		`SELECT id, job_id, candidate_id, status, final_decision, final_interview_notes, created_by, created_at, updated_at
		 FROM applications WHERE id = $1`, id,
	).Scan(&a.ID, &a.JobID, &a.CandidateID, &a.Status, &a.FinalDecision, &a.FinalInterviewNotes,
		&a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &a, err
}

func (s *Store) UpdateApplication(ctx context.Context, a *models.Application) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE applications SET status = $1, final_decision = $2, final_interview_notes = $3, updated_at = NOW()
		 WHERE id = $4`,
		a.Status, a.FinalDecision, a.FinalInterviewNotes, a.ID)
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

func (s *Store) DeleteApplication(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM applications WHERE id = $1`, id)
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

// GetApplicationDetail returns the application with job, candidate, stages, and
// per-interviewer feedback (the debrief view).
func (s *Store) GetApplicationDetail(ctx context.Context, id int64) (*models.ApplicationDetail, error) {
	app, err := s.GetApplication(ctx, id)
	if err != nil {
		return nil, err
	}
	detail := &models.ApplicationDetail{Application: *app}

	if err := s.db.QueryRowContext(ctx,
		`SELECT id, title, description, hiring_manager, status, created_by, created_at, updated_at
		 FROM jobs WHERE id = $1`, app.JobID,
	).Scan(&detail.Job.ID, &detail.Job.Title, &detail.Job.Description, &detail.Job.HiringManager,
		&detail.Job.Status, &detail.Job.CreatedBy, &detail.Job.CreatedAt, &detail.Job.UpdatedAt); err != nil {
		return nil, fmt.Errorf("load job: %w", err)
	}
	if err := s.db.QueryRowContext(ctx,
		`SELECT id, name, email, resume_url, created_at, updated_at FROM candidates WHERE id = $1`, app.CandidateID,
	).Scan(&detail.Candidate.ID, &detail.Candidate.Name, &detail.Candidate.Email, &detail.Candidate.ResumeURL,
		&detail.Candidate.CreatedAt, &detail.Candidate.UpdatedAt); err != nil {
		return nil, fmt.Errorf("load candidate: %w", err)
	}

	stages, err := s.ListStagesByApplication(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, st := range stages {
		sw := models.StageWithFeedback{Stage: *st}
		participants, err := s.listStageParticipantsWithFeedback(ctx, st.ID)
		if err != nil {
			return nil, err
		}
		sw.Participants = participants
		detail.Stages = append(detail.Stages, sw)
	}
	return detail, nil
}

// listStageParticipantsWithFeedback returns each assigned interviewer on a stage
// plus their feedback (nil if not yet filed).
func (s *Store) listStageParticipantsWithFeedback(ctx context.Context, stageID int64) ([]models.StageParticipant, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT si.interviewer_id, u.name FROM stage_interviewers si
		 JOIN users u ON u.id = si.interviewer_id
		 WHERE si.stage_id = $1 ORDER BY u.name`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.StageParticipant
	for rows.Next() {
		var p models.StageParticipant
		if err := rows.Scan(&p.InterviewerID, &p.InterviewerName); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		fb, err := s.GetFeedbackByStageAndInterviewer(ctx, stageID, out[i].InterviewerID)
		if err != nil && err != ErrNotFound {
			return nil, err
		}
		if err == nil {
			out[i].Feedback = fb
		}
	}
	return out, nil
}
