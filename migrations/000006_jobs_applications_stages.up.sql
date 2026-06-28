-- Jobs: top-level open reqs
CREATE TABLE IF NOT EXISTS jobs (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    hiring_manager TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'open' CHECK(status IN ('open', 'closed', 'filled')),
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Applications: a candidate's run at one job (candidate <-> job many-to-many)
CREATE TABLE IF NOT EXISTS applications (
    id SERIAL PRIMARY KEY,
    job_id INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    candidate_id INTEGER NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'rejected', 'hired', 'withdrawn')),
    final_decision TEXT CHECK(final_decision IN ('strong_hire', 'hire', 'no_hire', 'strong_no_hire')),
    final_interview_notes TEXT,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (job_id, candidate_id)
);

-- Stages: a step within an application (phone screen or interview)
CREATE TABLE IF NOT EXISTS stages (
    id SERIAL PRIMARY KEY,
    application_id INTEGER NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK(type IN ('phone_screen', 'interview')),
    focus_area TEXT NOT NULL DEFAULT '',
    scheduled_at TIMESTAMPTZ NOT NULL,
    video_link TEXT NOT NULL DEFAULT '',
    notes_for_interviewer TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'complete', 'canceled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Stage interviewers: many interviewers per stage
CREATE TABLE IF NOT EXISTS stage_interviewers (
    id SERIAL PRIMARY KEY,
    stage_id INTEGER NOT NULL REFERENCES stages(id) ON DELETE CASCADE,
    interviewer_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (stage_id, interviewer_id)
);

-- Reshape feedback: one per (stage, interviewer)
-- Drop competency_ratings first: CASCADE on feedback only drops the FK constraint, not the table
DROP TABLE IF EXISTS competency_ratings CASCADE;
DROP TABLE IF EXISTS feedback CASCADE;
CREATE TABLE IF NOT EXISTS feedback (
    id SERIAL PRIMARY KEY,
    stage_id INTEGER NOT NULL REFERENCES stages(id) ON DELETE CASCADE,
    interviewer_id INTEGER NOT NULL REFERENCES users(id),
    recommendation TEXT NOT NULL CHECK(recommendation IN ('strong_hire', 'hire', 'no_hire', 'strong_no_hire')),
    recommendation_reason TEXT NOT NULL DEFAULT '',
    free_form_notes TEXT NOT NULL DEFAULT '',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (stage_id, interviewer_id)
);

-- Recreate competency_ratings against the new feedback table
CREATE TABLE IF NOT EXISTS competency_ratings (
    id SERIAL PRIMARY KEY,
    feedback_id INTEGER NOT NULL REFERENCES feedback(id) ON DELETE CASCADE,
    competency_id INTEGER NOT NULL REFERENCES competencies(id),
    rating_value TEXT NOT NULL,
    CONSTRAINT competency_ratings_feedback_competency_unique UNIQUE (feedback_id, competency_id)
);

-- Drop the old model
DROP TABLE IF EXISTS interviews CASCADE;
DROP TABLE IF EXISTS interview_loops CASCADE;

-- Candidate status moves onto applications
ALTER TABLE candidates DROP COLUMN IF EXISTS status;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_applications_job ON applications(job_id);
CREATE INDEX IF NOT EXISTS idx_applications_candidate ON applications(candidate_id);
CREATE INDEX IF NOT EXISTS idx_stages_application ON stages(application_id);
CREATE INDEX IF NOT EXISTS idx_stage_interviewers_stage ON stage_interviewers(stage_id);
CREATE INDEX IF NOT EXISTS idx_stage_interviewers_interviewer ON stage_interviewers(interviewer_id);
CREATE INDEX IF NOT EXISTS idx_feedback_stage ON feedback(stage_id);
