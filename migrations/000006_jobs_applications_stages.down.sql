-- Restore candidate status
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active'
    CHECK(status IN ('active', 'hired', 'rejected', 'withdrawn'));

-- Recreate old loop + interview tables
CREATE TABLE IF NOT EXISTS interview_loops (
    id SERIAL PRIMARY KEY,
    candidate_id INTEGER NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'scheduling' CHECK(status IN ('scheduling', 'active', 'complete')),
    final_decision TEXT CHECK(final_decision IN ('strong_hire', 'hire', 'no_hire', 'strong_no_hire')),
    debrief_notes TEXT,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS interviews (
    id SERIAL PRIMARY KEY,
    loop_id INTEGER NOT NULL REFERENCES interview_loops(id) ON DELETE CASCADE,
    interviewer_id INTEGER NOT NULL REFERENCES users(id),
    focus_area TEXT NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    video_link TEXT NOT NULL DEFAULT '',
    notes_for_interviewer TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'complete')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Restore feedback against interviews
-- Drop competency_ratings first: CASCADE on feedback only drops the FK constraint, not the table
DROP TABLE IF EXISTS competency_ratings CASCADE;
DROP TABLE IF EXISTS feedback CASCADE;
CREATE TABLE IF NOT EXISTS feedback (
    id SERIAL PRIMARY KEY,
    interview_id INTEGER NOT NULL UNIQUE REFERENCES interviews(id) ON DELETE CASCADE,
    recommendation TEXT NOT NULL CHECK(recommendation IN ('strong_hire', 'hire', 'no_hire', 'strong_no_hire')),
    recommendation_reason TEXT NOT NULL DEFAULT '',
    free_form_notes TEXT NOT NULL DEFAULT '',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS competency_ratings (
    id SERIAL PRIMARY KEY,
    feedback_id INTEGER NOT NULL REFERENCES feedback(id) ON DELETE CASCADE,
    competency_id INTEGER NOT NULL REFERENCES competencies(id),
    rating_value TEXT NOT NULL,
    CONSTRAINT competency_ratings_feedback_competency_unique UNIQUE (feedback_id, competency_id)
);

-- Restore indexes from 000002 and 000004 on the recreated old tables
CREATE INDEX IF NOT EXISTS idx_interviews_loop_id ON interviews(loop_id);
CREATE INDEX IF NOT EXISTS idx_interviews_interviewer_id ON interviews(interviewer_id);
CREATE INDEX IF NOT EXISTS idx_feedback_interview_id ON feedback(interview_id);
CREATE INDEX IF NOT EXISTS idx_interview_loops_candidate_id ON interview_loops(candidate_id);
CREATE INDEX IF NOT EXISTS idx_competency_ratings_feedback_id ON competency_ratings(feedback_id);
CREATE INDEX IF NOT EXISTS idx_interviews_status ON interviews (status);
CREATE INDEX IF NOT EXISTS idx_interview_loops_created_by ON interview_loops (created_by);
CREATE INDEX IF NOT EXISTS idx_interviews_loop_id_status ON interviews (loop_id, status);

DROP TABLE IF EXISTS stage_interviewers CASCADE;
DROP TABLE IF EXISTS stages CASCADE;
DROP TABLE IF EXISTS applications CASCADE;
DROP TABLE IF EXISTS jobs CASCADE;
