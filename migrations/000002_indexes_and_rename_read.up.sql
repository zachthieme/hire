CREATE INDEX idx_interviews_loop_id ON interviews(loop_id);
CREATE INDEX idx_interviews_interviewer_id ON interviews(interviewer_id);
CREATE INDEX idx_feedback_interview_id ON feedback(interview_id);
CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_interview_loops_candidate_id ON interview_loops(candidate_id);
CREATE INDEX idx_competency_ratings_feedback_id ON competency_ratings(feedback_id);

ALTER TABLE notifications RENAME COLUMN "read" TO is_read;
