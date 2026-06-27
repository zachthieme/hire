ALTER TABLE competency_ratings DROP CONSTRAINT competency_ratings_feedback_competency_unique;

ALTER TABLE competencies DROP COLUMN updated_at;
ALTER TABLE feedback DROP COLUMN updated_at;
ALTER TABLE interviews DROP COLUMN updated_at;
ALTER TABLE interview_loops DROP COLUMN updated_at;
ALTER TABLE candidates DROP COLUMN updated_at;
ALTER TABLE users DROP COLUMN updated_at;
