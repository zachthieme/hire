CREATE INDEX IF NOT EXISTS idx_interviews_status ON interviews (status);
CREATE INDEX IF NOT EXISTS idx_interview_loops_created_by ON interview_loops (created_by);
CREATE INDEX IF NOT EXISTS idx_interviews_loop_id_status ON interviews (loop_id, status);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id_is_read ON notifications (user_id, is_read);
