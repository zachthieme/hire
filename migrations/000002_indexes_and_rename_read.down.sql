ALTER TABLE notifications RENAME COLUMN is_read TO "read";
DROP INDEX IF EXISTS idx_competency_ratings_feedback_id;
DROP INDEX IF EXISTS idx_interview_loops_candidate_id;
DROP INDEX IF EXISTS idx_notifications_user_id;
DROP INDEX IF EXISTS idx_feedback_interview_id;
DROP INDEX IF EXISTS idx_interviews_interviewer_id;
DROP INDEX IF EXISTS idx_interviews_loop_id;
