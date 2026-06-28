# Jobs / Applications / Stages Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `Candidate → Loop → Interview → Feedback` model with `Job → Application → Stage → StageInterviewer → Feedback`, giving schedulers top-level Jobs, many-to-many candidate↔job applications, and stages that hold multiple interviewers who each file their own feedback.

**Architecture:** Go (`net/http` + chi) REST API over Postgres via `database/sql`, layered as `migrations → models → store → api(handlers) → router`. React + Vite + Tailwind v4 + shadcn frontend calling `/api` through `frontend/src/lib/api.ts`. The remodel is destructive (demo data only) — one migration drops the old tables and creates the new ones; `seed/seed.go` is rewritten.

**Tech Stack:** Go 1.23, chi v5, golang-migrate, Postgres 16, React 19, TanStack Query, Radix/shadcn.

**Spec:** `docs/superpowers/specs/2026-06-27-jobs-applications-stages-design.md`

---

## Reference patterns (read before starting)

- **Migrations:** `migrations/000001_initial_schema.up.sql` (table style: `SERIAL PRIMARY KEY`, `REFERENCES ... ON DELETE CASCADE`, `CHECK(... IN (...))`), `migrations/000005_*.up.sql` (ALTER style). Tables are run by `golang-migrate`; the loop table is named `interview_loops`.
- **Models:** `internal/models/models.go` — structs with `json:"..."` tags, status constants, `ValidX` slices, `ValidXTransitions` maps.
- **Store:** `internal/store/interviews.go`, `internal/store/feedback.go` — methods on `*Store`, raw SQL via `s.db.QueryRowContext/QueryContext/ExecContext`, `sql.ErrNoRows → ErrNotFound`, `RowsAffected()==0 → ErrNotFound`, multi-write inside `BeginTx`/`tx.Commit`.
- **Store interface (consumed by handlers):** `internal/api/store.go` — every store method the handlers call is listed here. Add new methods, remove deleted ones.
- **Handlers:** `internal/api/loops.go`, `internal/api/interviews.go`, `internal/api/feedback.go` — `readJSON`, `writeJSON`, `writeError`, `writeInternalError`, `validateRequired`, `validateEnum`, `validateTransition`, `parsePagination`; auth helpers `UserID(ctx)`, `UserRole(ctx)`.
- **Router:** `internal/api/router.go` — route groups by `h.RequireRole(...)`.
- **Notifications:** `internal/notify/notify.go` — fire-and-forget helpers taking a `Notifier`.
- **Tests:** `internal/store/*_test.go` and `internal/api/*_test.go`. Run store/api tests with a live test DB: `make test` (needs `docker compose up db -d`).
- **Seed:** `seed/seed.go`.
- **Frontend API client:** `frontend/src/lib/api.ts` (has `request`, `requestList` null→[] helper, typed modules + interfaces).
- **Frontend pages to mirror:** `frontend/src/pages/scheduler/CandidatesList.tsx` (list + create dialog), `frontend/src/pages/scheduler/CandidateDetail.tsx` (detail + nested create), `frontend/src/pages/scheduler/DebriefView.tsx` (read aggregate + decision form), `frontend/src/pages/interviewer/MyInterviews.tsx`, `frontend/src/pages/interviewer/FeedbackForm.tsx`. Routes in `frontend/src/App.tsx`, nav in `frontend/src/components/Layout.tsx`.

## File structure (created / modified)

**Backend — create**
- `migrations/000006_jobs_applications_stages.up.sql` / `.down.sql`
- `internal/store/jobs.go`, `internal/store/applications.go`, `internal/store/stages.go`
- `internal/store/jobs_test.go`, `internal/store/applications_test.go`, `internal/store/stages_test.go`
- `internal/api/jobs.go`, `internal/api/applications.go`, `internal/api/stages.go`
- `internal/api/jobs_test.go`, `internal/api/applications_test.go`, `internal/api/stages_test.go`

**Backend — modify**
- `internal/models/models.go` (add Job/Application/Stage/StageInterviewer + details; reshape Feedback; new constants; drop Loop/Interview-only detail types)
- `internal/store/feedback.go` (repoint to stage_id+interviewer_id), `internal/store/store.go` if present
- `internal/api/store.go` (interface), `internal/api/router.go`, `internal/api/feedback.go`, `internal/api/interviews.go` (becomes stage feedback helpers or is removed)
- `internal/notify/notify.go`
- delete `internal/store/loops.go`, `internal/store/interviews.go`, `internal/api/loops.go`, and their `_test.go`; replace interview/loop tests with stage/application tests.
- `seed/seed.go`

**Frontend — modify**
- `frontend/src/lib/api.ts`, `frontend/src/App.tsx`, `frontend/src/components/Layout.tsx`
- create `frontend/src/pages/scheduler/JobsList.tsx`, `JobDetail.tsx`, `ApplicationDetail.tsx`
- rewrite `frontend/src/pages/interviewer/MyInterviews.tsx`, `InterviewDetail.tsx` (→ stage), `FeedbackForm.tsx`
- update/remove `frontend/src/pages/scheduler/CandidatesList.tsx`, `CandidateDetail.tsx`, `LoopEditor.tsx`, `DebriefView.tsx`

---

# Phase 1 — Backend model, migration, store

## Task 1: Migration — new schema

**Files:**
- Create: `migrations/000006_jobs_applications_stages.up.sql`
- Create: `migrations/000006_jobs_applications_stages.down.sql`

- [ ] **Step 1: Write the up migration**

`migrations/000006_jobs_applications_stages.up.sql`:
```sql
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
DROP TABLE IF EXISTS feedback CASCADE;
CREATE TABLE feedback (
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

-- competency_ratings was dropped via CASCADE above; recreate it
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
```

> Note: `competency_ratings` is dropped by `DROP TABLE feedback CASCADE` (FK dependency) and recreated, so its definition lives here intact.

- [ ] **Step 2: Write the down migration**

`migrations/000006_jobs_applications_stages.down.sql`:
```sql
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
DROP TABLE IF EXISTS feedback CASCADE;
CREATE TABLE feedback (
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

DROP TABLE IF EXISTS stage_interviewers CASCADE;
DROP TABLE IF EXISTS stages CASCADE;
DROP TABLE IF EXISTS applications CASCADE;
DROP TABLE IF EXISTS jobs CASCADE;
```

- [ ] **Step 3: Apply migration against the dev DB to verify it runs**

Run (DB must be up — `docker compose up db -d`):
```bash
docker compose up db -d
DATABASE_URL=postgres://hire:devpassword@localhost:5433/hire?sslmode=disable \
  go run ./cmd/server >/tmp/srv.log 2>&1 &
sleep 2 && grep -i "migrat" /tmp/srv.log; kill %1 2>/dev/null
```
Expected: server logs show migrations applied with no error (the app runs migrations on boot — confirm by checking `cmd/server`; if it does not, apply with the `migrate` CLI: `migrate -path migrations -database "$DATABASE_URL" up`).

- [ ] **Step 4: Commit**
```bash
git add migrations/000006_jobs_applications_stages.up.sql migrations/000006_jobs_applications_stages.down.sql
git commit -m "feat(db): migration for jobs/applications/stages model"
```

---

## Task 2: Models — structs, constants, detail types

**Files:**
- Modify: `internal/models/models.go`

- [ ] **Step 1: Add status constants and transition maps**

Append to the constants section of `internal/models/models.go` (after `ValidRatingTypes`):
```go
// Job statuses.
const (
	JobStatusOpen   = "open"
	JobStatusClosed = "closed"
	JobStatusFilled = "filled"
)

var ValidJobStatuses = []string{JobStatusOpen, JobStatusClosed, JobStatusFilled}

var ValidJobTransitions = map[string][]string{
	JobStatusOpen:   {JobStatusClosed, JobStatusFilled},
	JobStatusClosed: {JobStatusOpen},
	JobStatusFilled: {JobStatusOpen},
}

// Application statuses.
const (
	ApplicationStatusActive    = "active"
	ApplicationStatusRejected  = "rejected"
	ApplicationStatusHired     = "hired"
	ApplicationStatusWithdrawn = "withdrawn"
)

var ValidApplicationStatuses = []string{
	ApplicationStatusActive, ApplicationStatusRejected, ApplicationStatusHired, ApplicationStatusWithdrawn,
}

var ValidApplicationTransitions = map[string][]string{
	ApplicationStatusActive:    {ApplicationStatusRejected, ApplicationStatusHired, ApplicationStatusWithdrawn},
	ApplicationStatusRejected:  {ApplicationStatusActive},
	ApplicationStatusHired:     {ApplicationStatusActive},
	ApplicationStatusWithdrawn: {ApplicationStatusActive},
}

// Stage types.
const (
	StageTypePhoneScreen = "phone_screen"
	StageTypeInterview   = "interview"
)

var ValidStageTypes = []string{StageTypePhoneScreen, StageTypeInterview}

// Stage statuses.
const (
	StageStatusPending  = "pending"
	StageStatusComplete = "complete"
	StageStatusCanceled = "canceled"
)

var ValidStageStatuses = []string{StageStatusPending, StageStatusComplete, StageStatusCanceled}
```

- [ ] **Step 2: Add the new structs and reshape Feedback**

In `internal/models/models.go`: **delete** the `InterviewLoop`, `Interview`, `LoopDetail`, and `InterviewWithFeedback` structs and the loop/interview status constants (`LoopStatus*`, `ValidLoopStatuses`, `ValidLoopTransitions`, `InterviewStatus*`, `ValidInterviewStatuses`, `ValidInterviewTransitions`). Change `Feedback.InterviewID` to two fields, and add the new structs:

```go
type Job struct {
	ID            int64     `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	HiringManager string    `json:"hiring_manager"`
	Status        string    `json:"status"`
	CreatedBy     int64     `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Application struct {
	ID                  int64     `json:"id"`
	JobID               int64     `json:"job_id"`
	CandidateID         int64     `json:"candidate_id"`
	Status              string    `json:"status"`
	FinalDecision       *string   `json:"final_decision"`
	FinalInterviewNotes *string   `json:"final_interview_notes"`
	CreatedBy           int64     `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Stage struct {
	ID                  int64     `json:"id"`
	ApplicationID       int64     `json:"application_id"`
	Type                string    `json:"type"`
	FocusArea           string    `json:"focus_area"`
	ScheduledAt         time.Time `json:"scheduled_at"`
	VideoLink           string    `json:"video_link"`
	NotesForInterviewer string    `json:"notes_for_interviewer"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type StageInterviewer struct {
	ID              int64  `json:"id"`
	StageID         int64  `json:"stage_id"`
	InterviewerID   int64  `json:"interviewer_id"`
	InterviewerName string `json:"interviewer_name,omitempty"`
}

// Feedback — one per (stage, interviewer).
type Feedback struct {
	ID                   int64              `json:"id"`
	StageID              int64              `json:"stage_id"`
	InterviewerID        int64              `json:"interviewer_id"`
	Recommendation       string             `json:"recommendation"`
	RecommendationReason string             `json:"recommendation_reason"`
	FreeFormNotes        string             `json:"free_form_notes"`
	SubmittedAt          time.Time          `json:"submitted_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
	CompetencyRatings    []CompetencyRating `json:"competency_ratings,omitempty"`
}

// --- Read/aggregate types ---

// JobDetail is a job plus its candidate applications.
type JobDetail struct {
	Job
	Applications []ApplicationSummary `json:"applications"`
}

// ApplicationSummary is an application row enriched with the candidate's name,
// for listing under a job.
type ApplicationSummary struct {
	Application
	CandidateName  string `json:"candidate_name"`
	CandidateEmail string `json:"candidate_email"`
}

// ApplicationDetail is the debrief view: application + job + candidate + stages.
type ApplicationDetail struct {
	Application
	Job       Job                 `json:"job"`
	Candidate Candidate           `json:"candidate"`
	Stages    []StageWithFeedback `json:"stages"`
}

// StageWithFeedback is a stage plus each assigned interviewer and their feedback.
type StageWithFeedback struct {
	Stage
	Participants []StageParticipant `json:"participants"`
}

// StageParticipant is one interviewer on a stage plus their feedback (if filed).
type StageParticipant struct {
	InterviewerID   int64     `json:"interviewer_id"`
	InterviewerName string    `json:"interviewer_name"`
	Feedback        *Feedback `json:"feedback,omitempty"`
}

// MyStage is a stage assigned to the current interviewer, enriched for the
// "My Interviews" list (candidate + job titles).
type MyStage struct {
	Stage
	CandidateName  string `json:"candidate_name"`
	JobTitle       string `json:"job_title"`
	HasMyFeedback  bool   `json:"has_my_feedback"`
}
```

- [ ] **Step 3: Update Candidate struct (drop Status)**

In `internal/models/models.go`, remove the `Status` field from the `Candidate` struct. Leave `CandidateStatus*`/`ValidCandidateStatuses` constants deleted (they are no longer referenced; remove them).

- [ ] **Step 4: Compile-check models package**

Run: `go build ./internal/models/`
Expected: builds clean. (The rest of the repo will not compile yet — that is fixed in later tasks.)

- [ ] **Step 5: Commit**
```bash
git add internal/models/models.go
git commit -m "feat(models): job/application/stage structs, reshape feedback"
```

---

## Task 3: Store — jobs

**Files:**
- Create: `internal/store/jobs.go`
- Create: `internal/store/jobs_test.go`

- [ ] **Step 1: Write the failing test**

`internal/store/jobs_test.go` (mirror `internal/store/candidates_test.go` for setup — it uses the shared `testStore(t)` helper from `store_test.go`):
```go
package store

import (
	"context"
	"testing"

	"hire/internal/models"
)

func TestCreateAndGetJob(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	uid := createTestUser(t, s, "boss@x.com", models.RoleScheduler)

	job := &models.Job{Title: "Backend Engineer", Description: "Build APIs", HiringManager: "Dana", Status: models.JobStatusOpen, CreatedBy: uid}
	if err := s.CreateJob(ctx, job); err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	if job.ID == 0 {
		t.Fatal("expected job ID to be set")
	}

	got, err := s.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if got.Title != "Backend Engineer" || got.HiringManager != "Dana" {
		t.Fatalf("unexpected job: %+v", got)
	}

	jobs, err := s.ListJobs(ctx, 50, 0)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
}
```

> If `createTestUser` / `testStore` helpers do not exist with these exact names, open `internal/store/store_test.go` and `internal/store/users_test.go` and use whatever helper the existing tests use to get a `*Store` and a user id. Match the existing helper names exactly.

- [ ] **Step 2: Run the test to verify it fails to compile**

Run: `docker compose up db -d && make test 2>&1 | grep -i job | head`
Expected: FAIL — `s.CreateJob undefined`.

- [ ] **Step 3: Implement the jobs store**

`internal/store/jobs.go`:
```go
package store

import (
	"context"
	"database/sql"
	"fmt"

	"hire/internal/models"
)

func (s *Store) CreateJob(ctx context.Context, j *models.Job) error {
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO jobs (title, description, hiring_manager, status, created_by)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`,
		j.Title, j.Description, j.HiringManager, j.Status, j.CreatedBy,
	).Scan(&j.ID, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert job: %w", err)
	}
	return nil
}

func (s *Store) GetJob(ctx context.Context, id int64) (*models.Job, error) {
	var j models.Job
	err := s.db.QueryRowContext(ctx,
		`SELECT id, title, description, hiring_manager, status, created_by, created_at, updated_at
		 FROM jobs WHERE id = $1`, id,
	).Scan(&j.ID, &j.Title, &j.Description, &j.HiringManager, &j.Status, &j.CreatedBy, &j.CreatedAt, &j.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &j, err
}

func (s *Store) ListJobs(ctx context.Context, limit, offset int) ([]*models.Job, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, description, hiring_manager, status, created_by, created_at, updated_at
		 FROM jobs ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Job
	for rows.Next() {
		var j models.Job
		if err := rows.Scan(&j.ID, &j.Title, &j.Description, &j.HiringManager, &j.Status, &j.CreatedBy, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &j)
	}
	return out, rows.Err()
}

func (s *Store) GetJobDetail(ctx context.Context, id int64) (*models.JobDetail, error) {
	job, err := s.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	detail := &models.JobDetail{Job: *job}
	rows, err := s.db.QueryContext(ctx,
		`SELECT a.id, a.job_id, a.candidate_id, a.status, a.final_decision, a.final_interview_notes,
		        a.created_by, a.created_at, a.updated_at, c.name, c.email
		 FROM applications a JOIN candidates c ON c.id = a.candidate_id
		 WHERE a.job_id = $1 ORDER BY a.created_at DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a models.ApplicationSummary
		if err := rows.Scan(&a.ID, &a.JobID, &a.CandidateID, &a.Status, &a.FinalDecision, &a.FinalInterviewNotes,
			&a.CreatedBy, &a.CreatedAt, &a.UpdatedAt, &a.CandidateName, &a.CandidateEmail); err != nil {
			return nil, err
		}
		detail.Applications = append(detail.Applications, a)
	}
	return detail, rows.Err()
}

func (s *Store) UpdateJob(ctx context.Context, j *models.Job) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET title = $1, description = $2, hiring_manager = $3, status = $4, updated_at = NOW()
		 WHERE id = $5`,
		j.Title, j.Description, j.HiringManager, j.Status, j.ID)
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

func (s *Store) DeleteJob(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM jobs WHERE id = $1`, id)
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
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `make test 2>&1 | grep -iE "job|FAIL|ok" | head`
Expected: `TestCreateAndGetJob` passes (other packages may still fail to build — that is fixed in Phase 2; if `make test` aborts on build errors elsewhere, run only this package: `DATABASE_URL=postgres://hire:devpassword@localhost:5433/hire_test?sslmode=disable go test ./internal/store/ -run TestCreateAndGetJob -v`).

- [ ] **Step 5: Commit**
```bash
git add internal/store/jobs.go internal/store/jobs_test.go
git commit -m "feat(store): jobs CRUD + detail"
```

---

## Task 4: Store — applications

**Files:**
- Create: `internal/store/applications.go`
- Create: `internal/store/applications_test.go`

- [ ] **Step 1: Write the failing test**

`internal/store/applications_test.go`:
```go
package store

import (
	"context"
	"testing"

	"hire/internal/models"
)

func TestCreateAndGetApplication(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	uid := createTestUser(t, s, "boss2@x.com", models.RoleScheduler)

	job := &models.Job{Title: "BE", Status: models.JobStatusOpen, CreatedBy: uid}
	mustCreateJob(t, s, job)
	cand := &models.Candidate{Name: "Pat", Email: "pat@x.com"}
	if err := s.CreateCandidate(ctx, cand); err != nil {
		t.Fatalf("CreateCandidate: %v", err)
	}

	app := &models.Application{JobID: job.ID, CandidateID: cand.ID, Status: models.ApplicationStatusActive, CreatedBy: uid}
	if err := s.CreateApplication(ctx, app); err != nil {
		t.Fatalf("CreateApplication: %v", err)
	}
	if app.ID == 0 {
		t.Fatal("expected application ID")
	}

	got, err := s.GetApplication(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetApplication: %v", err)
	}
	if got.JobID != job.ID || got.CandidateID != cand.ID {
		t.Fatalf("unexpected application: %+v", got)
	}

	// Duplicate candidate on same job rejected by unique constraint
	dup := &models.Application{JobID: job.ID, CandidateID: cand.ID, Status: models.ApplicationStatusActive, CreatedBy: uid}
	if err := s.CreateApplication(ctx, dup); err == nil {
		t.Fatal("expected unique violation for duplicate application")
	}
}

func mustCreateJob(t *testing.T, s *Store, j *models.Job) {
	t.Helper()
	if err := s.CreateJob(context.Background(), j); err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `make test 2>&1 | grep -iE "application|undefined" | head`
Expected: FAIL — `s.CreateApplication undefined`.

- [ ] **Step 3: Implement the applications store**

`internal/store/applications.go`:
```go
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
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `DATABASE_URL=postgres://hire:devpassword@localhost:5433/hire_test?sslmode=disable go test ./internal/store/ -run TestCreateAndGetApplication -v`
Expected: PASS. (`GetApplicationDetail` references `ListStagesByApplication` and `GetFeedbackByStageAndInterviewer` — defined in Tasks 5 and 6. If running the whole package now fails to build, finish Tasks 5–6 then run the full suite.)

- [ ] **Step 5: Commit**
```bash
git add internal/store/applications.go internal/store/applications_test.go
git commit -m "feat(store): applications CRUD + detail aggregate"
```

---

## Task 5: Store — stages and stage interviewers

**Files:**
- Create: `internal/store/stages.go`
- Create: `internal/store/stages_test.go`

- [ ] **Step 1: Write the failing test**

`internal/store/stages_test.go`:
```go
package store

import (
	"context"
	"testing"
	"time"

	"hire/internal/models"
)

func TestStageLifecycle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	uid := createTestUser(t, s, "sch@x.com", models.RoleScheduler)
	ivID := createTestUser(t, s, "iv@x.com", models.RoleInterviewer)

	job := &models.Job{Title: "BE", Status: models.JobStatusOpen, CreatedBy: uid}
	mustCreateJob(t, s, job)
	cand := &models.Candidate{Name: "Sam", Email: "sam@x.com"}
	if err := s.CreateCandidate(ctx, cand); err != nil {
		t.Fatal(err)
	}
	app := &models.Application{JobID: job.ID, CandidateID: cand.ID, Status: models.ApplicationStatusActive, CreatedBy: uid}
	if err := s.CreateApplication(ctx, app); err != nil {
		t.Fatal(err)
	}

	st := &models.Stage{ApplicationID: app.ID, Type: models.StageTypeInterview, FocusArea: "Coding", ScheduledAt: time.Now(), Status: models.StageStatusPending}
	if err := s.CreateStage(ctx, st); err != nil {
		t.Fatalf("CreateStage: %v", err)
	}
	if st.ID == 0 {
		t.Fatal("expected stage ID")
	}

	if err := s.AddStageInterviewer(ctx, st.ID, ivID); err != nil {
		t.Fatalf("AddStageInterviewer: %v", err)
	}
	mine, err := s.ListStagesByUser(ctx, ivID, 50, 0)
	if err != nil {
		t.Fatalf("ListStagesByUser: %v", err)
	}
	if len(mine) != 1 || mine[0].JobTitle != "BE" || mine[0].CandidateName != "Sam" {
		t.Fatalf("unexpected my stages: %+v", mine)
	}

	if err := s.RemoveStageInterviewer(ctx, st.ID, ivID); err != nil {
		t.Fatalf("RemoveStageInterviewer: %v", err)
	}
	mine, _ = s.ListStagesByUser(ctx, ivID, 50, 0)
	if len(mine) != 0 {
		t.Fatalf("expected 0 stages after removal, got %d", len(mine))
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `DATABASE_URL=postgres://hire:devpassword@localhost:5433/hire_test?sslmode=disable go test ./internal/store/ -run TestStageLifecycle -v`
Expected: FAIL — `s.CreateStage undefined`.

- [ ] **Step 3: Implement the stages store**

`internal/store/stages.go`:
```go
package store

import (
	"context"
	"database/sql"
	"fmt"

	"hire/internal/models"
)

func (s *Store) CreateStage(ctx context.Context, st *models.Stage) error {
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO stages (application_id, type, focus_area, scheduled_at, video_link, notes_for_interviewer, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at, updated_at`,
		st.ApplicationID, st.Type, st.FocusArea, st.ScheduledAt, st.VideoLink, st.NotesForInterviewer, st.Status,
	).Scan(&st.ID, &st.CreatedAt, &st.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert stage: %w", err)
	}
	return nil
}

func (s *Store) GetStage(ctx context.Context, id int64) (*models.Stage, error) {
	var st models.Stage
	err := s.db.QueryRowContext(ctx,
		`SELECT id, application_id, type, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at, updated_at
		 FROM stages WHERE id = $1`, id,
	).Scan(&st.ID, &st.ApplicationID, &st.Type, &st.FocusArea, &st.ScheduledAt, &st.VideoLink,
		&st.NotesForInterviewer, &st.Status, &st.CreatedAt, &st.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &st, err
}

func (s *Store) ListStagesByApplication(ctx context.Context, appID int64) ([]*models.Stage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, application_id, type, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at, updated_at
		 FROM stages WHERE application_id = $1 ORDER BY scheduled_at`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Stage
	for rows.Next() {
		var st models.Stage
		if err := rows.Scan(&st.ID, &st.ApplicationID, &st.Type, &st.FocusArea, &st.ScheduledAt, &st.VideoLink,
			&st.NotesForInterviewer, &st.Status, &st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &st)
	}
	return out, rows.Err()
}

func (s *Store) UpdateStage(ctx context.Context, st *models.Stage) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE stages SET type = $1, focus_area = $2, scheduled_at = $3, video_link = $4,
		        notes_for_interviewer = $5, status = $6, updated_at = NOW()
		 WHERE id = $7`,
		st.Type, st.FocusArea, st.ScheduledAt, st.VideoLink, st.NotesForInterviewer, st.Status, st.ID)
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

func (s *Store) DeleteStage(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM stages WHERE id = $1`, id)
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

func (s *Store) AddStageInterviewer(ctx context.Context, stageID, interviewerID int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO stage_interviewers (stage_id, interviewer_id) VALUES ($1, $2)
		 ON CONFLICT (stage_id, interviewer_id) DO NOTHING`, stageID, interviewerID)
	if err != nil {
		return fmt.Errorf("add stage interviewer: %w", err)
	}
	return nil
}

func (s *Store) RemoveStageInterviewer(ctx context.Context, stageID, interviewerID int64) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM stage_interviewers WHERE stage_id = $1 AND interviewer_id = $2`, stageID, interviewerID)
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

func (s *Store) IsStageInterviewer(ctx context.Context, stageID, interviewerID int64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM stage_interviewers WHERE stage_id = $1 AND interviewer_id = $2`,
		stageID, interviewerID).Scan(&n)
	return n > 0, err
}

// ListStagesByUser returns stages the user is assigned to, enriched for the
// "My Interviews" list.
func (s *Store) ListStagesByUser(ctx context.Context, userID int64, limit, offset int) ([]*models.MyStage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT st.id, st.application_id, st.type, st.focus_area, st.scheduled_at, st.video_link,
		        st.notes_for_interviewer, st.status, st.created_at, st.updated_at,
		        c.name, j.title,
		        EXISTS(SELECT 1 FROM feedback f WHERE f.stage_id = st.id AND f.interviewer_id = $1) AS has_my_feedback
		 FROM stage_interviewers si
		 JOIN stages st ON st.id = si.stage_id
		 JOIN applications a ON a.id = st.application_id
		 JOIN candidates c ON c.id = a.candidate_id
		 JOIN jobs j ON j.id = a.job_id
		 WHERE si.interviewer_id = $1
		 ORDER BY st.scheduled_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.MyStage
	for rows.Next() {
		var m models.MyStage
		if err := rows.Scan(&m.ID, &m.ApplicationID, &m.Type, &m.FocusArea, &m.ScheduledAt, &m.VideoLink,
			&m.NotesForInterviewer, &m.Status, &m.CreatedAt, &m.UpdatedAt,
			&m.CandidateName, &m.JobTitle, &m.HasMyFeedback); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// CountIncompleteStages counts stages on an application not yet complete.
func (s *Store) CountIncompleteStages(ctx context.Context, appID int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM stages WHERE application_id = $1 AND status != $2`,
		appID, models.StageStatusComplete).Scan(&n)
	return n, err
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `DATABASE_URL=postgres://hire:devpassword@localhost:5433/hire_test?sslmode=disable go test ./internal/store/ -run TestStageLifecycle -v`
Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/store/stages.go internal/store/stages_test.go
git commit -m "feat(store): stages + stage interviewers"
```

---

## Task 6: Store — reshape feedback

**Files:**
- Modify: `internal/store/feedback.go`
- Delete: `internal/store/loops.go`, `internal/store/interviews.go`, `internal/store/loops_test.go`, `internal/store/interviews_test.go`
- Modify: `internal/store/feedback_test.go`

- [ ] **Step 1: Delete the obsolete loop/interview store files**
```bash
git rm internal/store/loops.go internal/store/interviews.go internal/store/loops_test.go internal/store/interviews_test.go
```

- [ ] **Step 2: Rewrite `internal/store/feedback.go`**

Replace the whole file with:
```go
package store

import (
	"context"
	"database/sql"
	"fmt"

	"hire/internal/models"
)

// CreateFeedback inserts feedback for (stage, interviewer), records competency
// ratings, marks the stage complete, and reports whether the whole application
// is now ready for a decision (all stages complete).
func (s *Store) CreateFeedback(ctx context.Context, fb *models.Feedback) (appReady bool, applicationID int64, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx,
		`INSERT INTO feedback (stage_id, interviewer_id, recommendation, recommendation_reason, free_form_notes)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		fb.StageID, fb.InterviewerID, fb.Recommendation, fb.RecommendationReason, fb.FreeFormNotes,
	).Scan(&fb.ID)
	if err != nil {
		return false, 0, fmt.Errorf("insert feedback: %w", err)
	}

	for i := range fb.CompetencyRatings {
		cr := &fb.CompetencyRatings[i]
		cr.FeedbackID = fb.ID
		if err := tx.QueryRowContext(ctx,
			`INSERT INTO competency_ratings (feedback_id, competency_id, rating_value) VALUES ($1, $2, $3) RETURNING id`,
			cr.FeedbackID, cr.CompetencyID, cr.RatingValue,
		).Scan(&cr.ID); err != nil {
			return false, 0, fmt.Errorf("insert competency rating: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE stages SET status = $1, updated_at = NOW() WHERE id = $2`,
		models.StageStatusComplete, fb.StageID); err != nil {
		return false, 0, fmt.Errorf("mark stage complete: %w", err)
	}

	if err := tx.QueryRowContext(ctx,
		`SELECT application_id FROM stages WHERE id = $1`, fb.StageID).Scan(&applicationID); err != nil {
		return false, 0, fmt.Errorf("get application_id: %w", err)
	}
	var incomplete int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM stages WHERE application_id = $1 AND status != $2`,
		applicationID, models.StageStatusComplete).Scan(&incomplete); err != nil {
		return false, 0, fmt.Errorf("count incomplete: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, 0, fmt.Errorf("commit: %w", err)
	}
	return incomplete == 0, applicationID, nil
}

func (s *Store) GetFeedback(ctx context.Context, id int64) (*models.Feedback, error) {
	var fb models.Feedback
	err := s.db.QueryRowContext(ctx,
		`SELECT id, stage_id, interviewer_id, recommendation, recommendation_reason, free_form_notes, submitted_at, updated_at
		 FROM feedback WHERE id = $1`, id,
	).Scan(&fb.ID, &fb.StageID, &fb.InterviewerID, &fb.Recommendation, &fb.RecommendationReason,
		&fb.FreeFormNotes, &fb.SubmittedAt, &fb.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	fb.CompetencyRatings, err = s.listCompetencyRatings(ctx, fb.ID)
	return &fb, err
}

func (s *Store) GetFeedbackByStageAndInterviewer(ctx context.Context, stageID, interviewerID int64) (*models.Feedback, error) {
	var fb models.Feedback
	err := s.db.QueryRowContext(ctx,
		`SELECT id, stage_id, interviewer_id, recommendation, recommendation_reason, free_form_notes, submitted_at, updated_at
		 FROM feedback WHERE stage_id = $1 AND interviewer_id = $2`, stageID, interviewerID,
	).Scan(&fb.ID, &fb.StageID, &fb.InterviewerID, &fb.Recommendation, &fb.RecommendationReason,
		&fb.FreeFormNotes, &fb.SubmittedAt, &fb.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	fb.CompetencyRatings, err = s.listCompetencyRatings(ctx, fb.ID)
	return &fb, err
}

func (s *Store) ListFeedbackByStage(ctx context.Context, stageID int64) ([]*models.Feedback, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, stage_id, interviewer_id, recommendation, recommendation_reason, free_form_notes, submitted_at, updated_at
		 FROM feedback WHERE stage_id = $1 ORDER BY submitted_at`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Feedback
	for rows.Next() {
		var fb models.Feedback
		if err := rows.Scan(&fb.ID, &fb.StageID, &fb.InterviewerID, &fb.Recommendation, &fb.RecommendationReason,
			&fb.FreeFormNotes, &fb.SubmittedAt, &fb.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &fb)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, fb := range out {
		fb.CompetencyRatings, err = s.listCompetencyRatings(ctx, fb.ID)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Store) UpdateFeedback(ctx context.Context, fb *models.Feedback) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`UPDATE feedback SET recommendation = $1, recommendation_reason = $2, free_form_notes = $3, updated_at = NOW() WHERE id = $4`,
		fb.Recommendation, fb.RecommendationReason, fb.FreeFormNotes, fb.ID,
	)
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

	if len(fb.CompetencyRatings) > 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM competency_ratings WHERE feedback_id = $1`, fb.ID); err != nil {
			return err
		}
		for i := range fb.CompetencyRatings {
			cr := &fb.CompetencyRatings[i]
			cr.FeedbackID = fb.ID
			if err := tx.QueryRowContext(ctx,
				`INSERT INTO competency_ratings (feedback_id, competency_id, rating_value) VALUES ($1, $2, $3) RETURNING id`,
				cr.FeedbackID, cr.CompetencyID, cr.RatingValue,
			).Scan(&cr.ID); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
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
```

- [ ] **Step 3: Rewrite `internal/store/feedback_test.go`**

Replace any loop/interview references. Minimal test:
```go
package store

import (
	"context"
	"testing"
	"time"

	"hire/internal/models"
)

func TestCreateFeedbackMarksStageCompleteAndReady(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	uid := createTestUser(t, s, "sc@x.com", models.RoleScheduler)
	ivID := createTestUser(t, s, "iv2@x.com", models.RoleInterviewer)

	job := &models.Job{Title: "BE", Status: models.JobStatusOpen, CreatedBy: uid}
	mustCreateJob(t, s, job)
	cand := &models.Candidate{Name: "Lee", Email: "lee@x.com"}
	if err := s.CreateCandidate(ctx, cand); err != nil {
		t.Fatal(err)
	}
	app := &models.Application{JobID: job.ID, CandidateID: cand.ID, Status: models.ApplicationStatusActive, CreatedBy: uid}
	if err := s.CreateApplication(ctx, app); err != nil {
		t.Fatal(err)
	}
	st := &models.Stage{ApplicationID: app.ID, Type: models.StageTypeInterview, ScheduledAt: time.Now(), Status: models.StageStatusPending}
	if err := s.CreateStage(ctx, st); err != nil {
		t.Fatal(err)
	}
	if err := s.AddStageInterviewer(ctx, st.ID, ivID); err != nil {
		t.Fatal(err)
	}

	fb := &models.Feedback{StageID: st.ID, InterviewerID: ivID, Recommendation: models.RecommendationHire}
	ready, appID, err := s.CreateFeedback(ctx, fb)
	if err != nil {
		t.Fatalf("CreateFeedback: %v", err)
	}
	if !ready {
		t.Fatal("expected application ready (only stage now complete)")
	}
	if appID != app.ID {
		t.Fatalf("expected appID %d, got %d", app.ID, appID)
	}

	got, err := s.GetStage(ctx, st.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.StageStatusComplete {
		t.Fatalf("stage status = %q, want complete", got.Status)
	}
}
```

- [ ] **Step 4: Run the store tests**

Run: `DATABASE_URL=postgres://hire:devpassword@localhost:5433/hire_test?sslmode=disable go test ./internal/store/ -v 2>&1 | tail -30`
Expected: all store tests PASS. (Other store tests that referenced candidate `Status` or loops must be updated/removed — search `internal/store` for `Status:` on candidates and `Loop`/`Interview` references and fix. `grep -rn "Loop\|Interview\|CandidateStatus" internal/store`.)

- [ ] **Step 5: Commit**
```bash
git add -A internal/store/
git commit -m "feat(store): feedback per (stage, interviewer); drop loop/interview store"
```

---

# Phase 2 — Backend API, authz, notifications

## Task 7: Update the Store interface + notifications

**Files:**
- Modify: `internal/api/store.go`
- Modify: `internal/notify/notify.go`

- [ ] **Step 1: Replace the Loop/Interview/Feedback sections of the `Store` interface**

In `internal/api/store.go`, remove the `Interview Loops`, `Interviews`, the old `Feedback` block, and `CountIncompleteInterviews`. Add:
```go
	// Jobs
	CreateJob(ctx context.Context, j *models.Job) error
	GetJob(ctx context.Context, id int64) (*models.Job, error)
	GetJobDetail(ctx context.Context, id int64) (*models.JobDetail, error)
	ListJobs(ctx context.Context, limit, offset int) ([]*models.Job, error)
	UpdateJob(ctx context.Context, j *models.Job) error
	DeleteJob(ctx context.Context, id int64) error

	// Applications
	CreateApplication(ctx context.Context, a *models.Application) error
	GetApplication(ctx context.Context, id int64) (*models.Application, error)
	GetApplicationDetail(ctx context.Context, id int64) (*models.ApplicationDetail, error)
	UpdateApplication(ctx context.Context, a *models.Application) error
	DeleteApplication(ctx context.Context, id int64) error

	// Stages
	CreateStage(ctx context.Context, st *models.Stage) error
	GetStage(ctx context.Context, id int64) (*models.Stage, error)
	ListStagesByApplication(ctx context.Context, appID int64) ([]*models.Stage, error)
	ListStagesByUser(ctx context.Context, userID int64, limit, offset int) ([]*models.MyStage, error)
	UpdateStage(ctx context.Context, st *models.Stage) error
	DeleteStage(ctx context.Context, id int64) error
	AddStageInterviewer(ctx context.Context, stageID, interviewerID int64) error
	RemoveStageInterviewer(ctx context.Context, stageID, interviewerID int64) error
	IsStageInterviewer(ctx context.Context, stageID, interviewerID int64) (bool, error)
	CountIncompleteStages(ctx context.Context, appID int64) (int, error)

	// Feedback
	CreateFeedback(ctx context.Context, fb *models.Feedback) (appReady bool, applicationID int64, err error)
	GetFeedback(ctx context.Context, id int64) (*models.Feedback, error)
	GetFeedbackByStageAndInterviewer(ctx context.Context, stageID, interviewerID int64) (*models.Feedback, error)
	ListFeedbackByStage(ctx context.Context, stageID int64) ([]*models.Feedback, error)
	UpdateFeedback(ctx context.Context, fb *models.Feedback) error
```
Keep the candidate `Update`/`Get`/etc. methods, but the `Candidate` struct no longer has `Status` — no interface change needed there.

- [ ] **Step 2: Rewrite `internal/notify/notify.go` helpers**

Replace the three helper functions with:
```go
func StageAssigned(ctx context.Context, s Notifier, interviewerID, stageID int64, stageType string) {
	if err := s.CreateNotification(ctx, &models.Notification{
		UserID:  interviewerID,
		Message: fmt.Sprintf("You've been assigned a %s", humanStageType(stageType)),
		Link:    fmt.Sprintf("/interviews/%d", stageID),
	}); err != nil {
		slog.ErrorContext(ctx, "failed to create stage-assigned notification",
			"error", err, "interviewer_id", interviewerID, "stage_id", stageID)
	}
}

func FeedbackSubmitted(ctx context.Context, s Notifier, schedulerID, applicationID int64, stageType string) {
	if err := s.CreateNotification(ctx, &models.Notification{
		UserID:  schedulerID,
		Message: fmt.Sprintf("Feedback submitted for a %s", humanStageType(stageType)),
		Link:    fmt.Sprintf("/applications/%d", applicationID),
	}); err != nil {
		slog.ErrorContext(ctx, "failed to create feedback-submitted notification",
			"error", err, "scheduler_id", schedulerID, "application_id", applicationID)
	}
}

func ReadyForDecision(ctx context.Context, s Notifier, schedulerID, applicationID int64) {
	if err := s.CreateNotification(ctx, &models.Notification{
		UserID:  schedulerID,
		Message: "All feedback submitted — ready for a decision",
		Link:    fmt.Sprintf("/applications/%d", applicationID),
	}); err != nil {
		slog.ErrorContext(ctx, "failed to create ready-for-decision notification",
			"error", err, "application_id", applicationID)
	}
}

func humanStageType(t string) string {
	if t == models.StageTypePhoneScreen {
		return "phone screen"
	}
	return "interview"
}
```

- [ ] **Step 3: Commit (no test; compiles after Task 10)**
```bash
git add internal/api/store.go internal/notify/notify.go
git commit -m "feat(api): store interface + notifications for new model"
```

---

## Task 8: Handlers — jobs

**Files:**
- Create: `internal/api/jobs.go`
- Create: `internal/api/jobs_test.go`

- [ ] **Step 1: Implement the jobs handlers**

`internal/api/jobs.go`:
```go
package api

import (
	"errors"
	"net/http"
	"strconv"

	"hire/internal/models"
	"hire/internal/store"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var j models.Job
	if err := readJSON(r, &j); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateRequired(map[string]string{"title": j.Title}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if j.Status == "" {
		j.Status = models.JobStatusOpen
	}
	if err := validateEnum(j.Status, "status", models.ValidJobStatuses); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	j.CreatedBy = UserID(r.Context())
	if err := h.store.CreateJob(r.Context(), &j); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, j)
}

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	jobs, err := h.store.ListJobs(r.Context(), limit, offset)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (h *Handler) GetJobDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	detail, err := h.store.GetJobDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "job not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetJob(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "job not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	var updates models.Job
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateRequired(map[string]string{"title": updates.Title}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateEnum(updates.Status, "status", models.ValidJobStatuses); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if updates.Status != existing.Status {
		if err := validateTransition(existing.Status, updates.Status, "job", models.ValidJobTransitions); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	existing.Title = updates.Title
	existing.Description = updates.Description
	existing.HiringManager = updates.HiringManager
	existing.Status = updates.Status
	if err := h.store.UpdateJob(r.Context(), existing); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteJob(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "job not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 2: Write a handler test**

`internal/api/jobs_test.go` (mirror `internal/api/loops_test.go` for the in-memory/mock-store + auth-context test harness it uses; reuse the same helper that builds a `Handler` with a fake store and an authed request). Minimal:
```go
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateJobRequiresTitle(t *testing.T) {
	h, _ := newTestHandler(t) // see loops_test.go for the existing helper
	r := chi.NewRouter()
	r.Post("/api/jobs", h.CreateJob)

	req := authedRequest(t, "POST", "/api/jobs", strings.NewReader(`{}`), "scheduler")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
```
> Open `internal/api/loops_test.go` and copy the exact harness helpers it uses (`newTestHandler`, how it builds an authed request, and any fake store). Match those names/signatures rather than the placeholders above. Add the `chi` import.

- [ ] **Step 3: Run the test**

Run: `make test 2>&1 | grep -iE "job|FAIL" | head` (after Task 10 the full build is green; if needed run `go vet ./internal/api/` first to confirm signatures).
Expected: job tests PASS.

- [ ] **Step 4: Commit**
```bash
git add internal/api/jobs.go internal/api/jobs_test.go
git commit -m "feat(api): jobs handlers"
```

---

## Task 9: Handlers — applications and stages (+ feedback rewrite)

**Files:**
- Create: `internal/api/applications.go`
- Create: `internal/api/stages.go`
- Modify: `internal/api/feedback.go`
- Delete: `internal/api/loops.go`, `internal/api/interviews.go`, `internal/api/loops_test.go`, `internal/api/interviews_test.go`
- Modify: `internal/api/interviews.go` users → `ListMyInterviews` moves to stages

- [ ] **Step 1: Delete obsolete handlers**
```bash
git rm internal/api/loops.go internal/api/loops_test.go internal/api/interviews_test.go
```
Keep `internal/api/interviews.go` open — you will move `ListMyInterviews` out of it (Step 3) then delete it.

- [ ] **Step 2: Implement application handlers**

`internal/api/applications.go`:
```go
package api

import (
	"errors"
	"net/http"
	"strconv"

	"hire/internal/models"
	"hire/internal/store"

	"github.com/go-chi/chi/v5"
)

// CreateApplication adds a candidate to a job. Route: POST /api/jobs/{id}/applications
func (h *Handler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	jobID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}
	var body struct {
		CandidateID int64 `json:"candidate_id"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.CandidateID == 0 {
		writeError(w, http.StatusBadRequest, "candidate_id is required")
		return
	}
	app := models.Application{
		JobID:       jobID,
		CandidateID: body.CandidateID,
		Status:      models.ApplicationStatusActive,
		CreatedBy:   UserID(r.Context()),
	}
	if err := h.store.CreateApplication(r.Context(), &app); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, app)
}

func (h *Handler) GetApplicationDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	detail, err := h.store.GetApplicationDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "application not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}

	// Interviewer feedback visibility: an interviewer only sees others' feedback
	// on a stage once they've submitted their own for that stage.
	if UserRole(r.Context()) == models.RoleInterviewer {
		userID := UserID(r.Context())
		for si := range detail.Stages {
			submitted := false
			for _, p := range detail.Stages[si].Participants {
				if p.InterviewerID == userID && p.Feedback != nil {
					submitted = true
				}
			}
			if !submitted {
				for pi := range detail.Stages[si].Participants {
					if detail.Stages[si].Participants[pi].InterviewerID != userID {
						detail.Stages[si].Participants[pi].Feedback = nil
					}
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) UpdateApplication(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetApplication(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "application not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	var updates models.Application
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateEnum(updates.Status, "status", models.ValidApplicationStatuses); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if updates.Status != existing.Status {
		if err := validateTransition(existing.Status, updates.Status, "application", models.ValidApplicationTransitions); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if updates.FinalDecision != nil {
		if err := validateEnum(*updates.FinalDecision, "final_decision", models.ValidRecommendations); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	existing.Status = updates.Status
	existing.FinalDecision = updates.FinalDecision
	existing.FinalInterviewNotes = updates.FinalInterviewNotes
	if err := h.store.UpdateApplication(r.Context(), existing); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteApplication(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteApplication(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "application not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 3: Implement stage handlers (incl. ListMyStages)**

`internal/api/stages.go`:
```go
package api

import (
	"errors"
	"net/http"
	"strconv"

	"hire/internal/models"
	"hire/internal/notify"
	"hire/internal/store"

	"github.com/go-chi/chi/v5"
)

// CreateStage. Route: POST /api/applications/{id}/stages
func (h *Handler) CreateStage(w http.ResponseWriter, r *http.Request) {
	appID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid application id")
		return
	}
	var st models.Stage
	if err := readJSON(r, &st); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	st.ApplicationID = appID
	if err := validateEnum(st.Type, "type", models.ValidStageTypes); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if st.Status == "" {
		st.Status = models.StageStatusPending
	}
	if err := validateEnum(st.Status, "status", models.ValidStageStatuses); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.CreateStage(r.Context(), &st); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, st)
}

func (h *Handler) UpdateStage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetStage(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "stage not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	var updates models.Stage
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateEnum(updates.Type, "type", models.ValidStageTypes); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateEnum(updates.Status, "status", models.ValidStageStatuses); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	existing.Type = updates.Type
	existing.FocusArea = updates.FocusArea
	existing.ScheduledAt = updates.ScheduledAt
	existing.VideoLink = updates.VideoLink
	existing.NotesForInterviewer = updates.NotesForInterviewer
	existing.Status = updates.Status
	if err := h.store.UpdateStage(r.Context(), existing); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteStage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteStage(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "stage not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AddStageInterviewer. Route: POST /api/stages/{id}/interviewers
func (h *Handler) AddStageInterviewer(w http.ResponseWriter, r *http.Request) {
	stageID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage id")
		return
	}
	var body struct {
		InterviewerID int64 `json:"interviewer_id"`
	}
	if err := readJSON(r, &body); err != nil || body.InterviewerID == 0 {
		writeError(w, http.StatusBadRequest, "interviewer_id is required")
		return
	}
	if err := h.store.AddStageInterviewer(r.Context(), stageID, body.InterviewerID); err != nil {
		writeInternalError(w, r, err)
		return
	}
	st, err := h.store.GetStage(r.Context(), stageID)
	if err == nil {
		notify.StageAssigned(r.Context(), h.store, body.InterviewerID, stageID, st.Type)
	}
	w.WriteHeader(http.StatusNoContent)
}

// RemoveStageInterviewer. Route: DELETE /api/stages/{id}/interviewers/{uid}
func (h *Handler) RemoveStageInterviewer(w http.ResponseWriter, r *http.Request) {
	stageID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage id")
		return
	}
	uid, err := strconv.ParseInt(chi.URLParam(r, "uid"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid interviewer id")
		return
	}
	if err := h.store.RemoveStageInterviewer(r.Context(), stageID, uid); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not assigned")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListMyStages. Route: GET /api/me/stages
func (h *Handler) ListMyStages(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	stages, err := h.store.ListStagesByUser(r.Context(), UserID(r.Context()), limit, offset)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, stages)
}
```

- [ ] **Step 4: Rewrite `internal/api/feedback.go` for stages**

Replace the file's handlers with stage-based versions:
```go
package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"hire/internal/models"
	"hire/internal/notify"
	"hire/internal/store"

	"github.com/go-chi/chi/v5"
)

// GetStageFeedback. Route: GET /api/stages/{id}/feedback — all interviewers'.
func (h *Handler) GetStageFeedback(w http.ResponseWriter, r *http.Request) {
	stageID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	list, err := h.store.ListFeedbackByStage(r.Context(), stageID)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// CreateFeedback. Route: POST /api/stages/{id}/feedback — current user's.
func (h *Handler) CreateFeedback(w http.ResponseWriter, r *http.Request) {
	stageID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	userID := UserID(r.Context())

	// Only an assigned interviewer (or admin) may submit.
	if UserRole(r.Context()) == models.RoleInterviewer {
		ok, err := h.store.IsStageInterviewer(r.Context(), stageID, userID)
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		if !ok {
			writeError(w, http.StatusForbidden, "not your stage")
			return
		}
	}

	var fb models.Feedback
	if err := readJSON(r, &fb); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateEnum(fb.Recommendation, "recommendation", models.ValidRecommendations); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	fb.StageID = stageID
	fb.InterviewerID = userID

	ready, applicationID, err := h.store.CreateFeedback(r.Context(), &fb)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}

	app, err := h.store.GetApplication(r.Context(), applicationID)
	if err != nil {
		slog.ErrorContext(r.Context(), "load application for notification", "error", err, "application_id", applicationID)
	} else {
		st, _ := h.store.GetStage(r.Context(), stageID)
		stageType := models.StageTypeInterview
		if st != nil {
			stageType = st.Type
		}
		notify.FeedbackSubmitted(r.Context(), h.store, app.CreatedBy, applicationID, stageType)
		if ready {
			notify.ReadyForDecision(r.Context(), h.store, app.CreatedBy, applicationID)
		}
	}
	writeJSON(w, http.StatusCreated, fb)
}

// UpdateFeedback. Route: PUT /api/feedback/{id} — edit own.
func (h *Handler) UpdateFeedback(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetFeedback(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "feedback not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	if existing.InterviewerID != UserID(r.Context()) && UserRole(r.Context()) == models.RoleInterviewer {
		writeError(w, http.StatusForbidden, "not your feedback")
		return
	}
	var updates models.Feedback
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateEnum(updates.Recommendation, "recommendation", models.ValidRecommendations); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	existing.Recommendation = updates.Recommendation
	existing.RecommendationReason = updates.RecommendationReason
	existing.FreeFormNotes = updates.FreeFormNotes
	existing.CompetencyRatings = updates.CompetencyRatings
	if err := h.store.UpdateFeedback(r.Context(), existing); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, existing)
}
```

- [ ] **Step 5: Remove `internal/api/interviews.go`**

The only handler worth keeping was `ListMyInterviews`; it is replaced by `ListMyStages`. Delete the file:
```bash
git rm internal/api/interviews.go
```
If `interviews.go` contained other helpers still referenced, move them into `stages.go` first.

- [ ] **Step 6: Build the api package**

Run: `go build ./internal/api/`
Expected: builds clean (router still references removed handlers until Task 10 — if so, do Task 10 before building).

- [ ] **Step 7: Commit**
```bash
git add -A internal/api/
git commit -m "feat(api): application, stage, and stage-feedback handlers"
```

---

## Task 10: Router wiring

**Files:**
- Modify: `internal/api/router.go`

- [ ] **Step 1: Replace the loop/interview/feedback routes**

In `internal/api/router.go`, inside the authenticated group:

Replace the "Any authenticated user" feedback + loop lines and the scheduler-only loop/interview lines with:
```go
		// Any authenticated user
		r.Get("/api/me", h.GetMe)
		r.Post("/api/auth/refresh", h.RefreshToken)
		r.Get("/api/me/stages", h.ListMyStages)
		r.Get("/api/notifications", h.ListNotifications)
		r.Put("/api/notifications/{id}/read", h.MarkNotificationRead)
		r.Get("/api/competencies", h.ListCompetencies)

		// Jobs & applications — readable by any authenticated user
		r.Get("/api/jobs", h.ListJobs)
		r.Get("/api/jobs/{id}", h.GetJobDetail)
		r.Get("/api/applications/{id}", h.GetApplicationDetail)

		// Stage feedback
		r.Get("/api/stages/{id}/feedback", h.GetStageFeedback)
		r.Post("/api/stages/{id}/feedback", h.CreateFeedback)
		r.Put("/api/feedback/{id}", h.UpdateFeedback)
```
And replace the scheduler-only block's loop/interview routes with:
```go
		// Scheduler and admin
		r.Group(func(r chi.Router) {
			r.Use(h.RequireRole("scheduler", "admin"))
			r.Get("/api/users", h.ListUsers)

			r.Post("/api/candidates", h.CreateCandidate)
			r.Get("/api/candidates", h.ListCandidates)
			r.Get("/api/candidates/{id}", h.GetCandidate)
			r.Put("/api/candidates/{id}", h.UpdateCandidate)
			r.Delete("/api/candidates/{id}", h.DeleteCandidate)

			r.Post("/api/jobs", h.CreateJob)
			r.Put("/api/jobs/{id}", h.UpdateJob)
			r.Delete("/api/jobs/{id}", h.DeleteJob)

			r.Post("/api/jobs/{id}/applications", h.CreateApplication)
			r.Put("/api/applications/{id}", h.UpdateApplication)
			r.Delete("/api/applications/{id}", h.DeleteApplication)

			r.Post("/api/applications/{id}/stages", h.CreateStage)
			r.Put("/api/stages/{id}", h.UpdateStage)
			r.Delete("/api/stages/{id}", h.DeleteStage)
			r.Post("/api/stages/{id}/interviewers", h.AddStageInterviewer)
			r.Delete("/api/stages/{id}/interviewers/{uid}", h.RemoveStageInterviewer)
		})
```
> Note: the `/api/users` route was previously in its own scheduler/admin sub-group; folding it into this block is fine. Keep the admin-only group (users CRUD, competencies CRUD) unchanged.

- [ ] **Step 2: Build the whole backend**

Run: `go build ./...`
Expected: builds clean.

- [ ] **Step 3: Run the full backend test suite**

Run: `docker compose up db -d && make test 2>&1 | tail -30`
Expected: all packages PASS. Fix any remaining references to removed symbols (grep: `grep -rn "Loop\|ListMyInterviews\|InterviewID\|CandidateStatus" internal/`).

- [ ] **Step 4: Commit**
```bash
git add internal/api/router.go
git commit -m "feat(api): wire jobs/applications/stages routes"
```

---

## Task 11: Authorization tests

**Files:**
- Modify: `internal/api/authorization_test.go`

- [ ] **Step 1: Update the authorization matrix test**

Open `internal/api/authorization_test.go`. It enumerates routes and expected status per role. Replace loop/interview rows with the new routes:
- `POST /api/jobs` → scheduler/admin allowed, interviewer 403
- `POST /api/jobs/{id}/applications` → scheduler/admin allowed, interviewer 403
- `POST /api/applications/{id}/stages` → scheduler/admin allowed, interviewer 403
- `POST /api/stages/{id}/interviewers` → scheduler/admin allowed, interviewer 403
- `GET /api/jobs` → all roles allowed
- `GET /api/me/stages` → all roles allowed
- `POST /api/stages/{id}/feedback` → interviewer allowed (when assigned), tested in feedback test

Match the existing table's exact structure/field names.

- [ ] **Step 2: Run the authorization test**

Run: `make test 2>&1 | grep -iE "authoriz|FAIL|ok" | head`
Expected: PASS.

- [ ] **Step 3: Commit**
```bash
git add internal/api/authorization_test.go
git commit -m "test(api): authorization matrix for new routes"
```

---

## Task 12: Rewrite the seed

**Files:**
- Modify: `seed/seed.go`

- [ ] **Step 1: Rewrite seed to produce jobs → applications → stages → interviewers → feedback**

Open `seed/seed.go`. Keep the user + competency seeding. Replace the loop/interview/feedback seeding with:
1. Create 2 Jobs (e.g. "Backend Engineer" / open, "Product Designer" / open) via `INSERT INTO jobs ...` using the scheduler's id as `created_by`.
2. Create candidates (existing) — drop any `status` column from the candidate INSERT.
3. For a couple of candidates, `INSERT INTO applications (job_id, candidate_id, status, created_by)`.
4. For one application, `INSERT INTO stages (...)` for a `phone_screen` and an `interview`, then `INSERT INTO stage_interviewers (stage_id, interviewer_id)` assigning alice/bob, and `INSERT INTO feedback (stage_id, interviewer_id, recommendation, ...)` for one of them so the debrief has data.

Use the existing seed's raw-SQL style (it already opens a `*sql.DB`). Mirror the column lists from the migration in Task 1. Print the same demo-accounts summary at the end.

- [ ] **Step 2: Run the seed against a fresh DB**

Run:
```bash
docker compose down -v && docker compose up db -d && sleep 3
# start server once to run migrations, then seed:
DATABASE_URL=postgres://hire:devpassword@localhost:5433/hire?sslmode=disable go run ./cmd/server >/tmp/srv.log 2>&1 &
sleep 3
DATABASE_URL=postgres://hire:devpassword@localhost:5433/hire?sslmode=disable go run ./seed/seed.go
kill %1 2>/dev/null
```
Expected: "Seed data created successfully!" and demo accounts printed, no SQL errors.

- [ ] **Step 3: Smoke-test the API**

Run:
```bash
T=$(curl -s -X POST localhost:8081/api/auth/login -H 'Content-Type: application/json' -d '{"email":"scheduler@hire.demo","password":"scheduler"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")
curl -s localhost:8081/api/jobs -H "Authorization: Bearer $T" | head -c 300
```
Expected: a JSON array of jobs (or `[]` — never `null`).
> If the server isn't running on 8081, start the full stack: `make up` (note the port remap in `docker-compose.override.yml`, frontend on 3002).

- [ ] **Step 4: Commit**
```bash
git add seed/seed.go
git commit -m "feat(seed): jobs/applications/stages demo data"
```

---

# Phase 3 — Frontend: scheduler (jobs → applications → stages)

## Task 13: API client — jobs/applications/stages modules + types

**Files:**
- Modify: `frontend/src/lib/api.ts`

- [ ] **Step 1: Replace the `loops`/`interviews` modules and add types**

In `frontend/src/lib/api.ts`:

Remove the `loops` and `interviews` modules and the `InterviewLoop`/`LoopDetail`/`Interview`/`InterviewWithFeedback` interfaces. Remove `status` from the `Candidate` interface. Add:
```ts
// Jobs
export const jobs = {
  list: (params?: { limit?: number; offset?: number }) => {
    const q = new URLSearchParams()
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.offset) q.set('offset', String(params.offset))
    const qs = q.toString()
    return requestList<Job>('GET', `/jobs${qs ? '?' + qs : ''}`)
  },
  get: (id: number) => request<JobDetail>('GET', `/jobs/${id}`),
  create: (data: Partial<Job>) => request<Job>('POST', '/jobs', data),
  update: (id: number, data: Partial<Job>) => request<Job>('PUT', `/jobs/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/jobs/${id}`),
}

// Applications
export const applications = {
  get: (id: number) => request<ApplicationDetail>('GET', `/applications/${id}`),
  create: (jobId: number, candidateId: number) =>
    request<Application>('POST', `/jobs/${jobId}/applications`, { candidate_id: candidateId }),
  update: (id: number, data: Partial<Application>) => request<Application>('PUT', `/applications/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/applications/${id}`),
}

// Stages
export const stages = {
  create: (applicationId: number, data: Partial<Stage>) =>
    request<Stage>('POST', `/applications/${applicationId}/stages`, data),
  update: (id: number, data: Partial<Stage>) => request<Stage>('PUT', `/stages/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/stages/${id}`),
  addInterviewer: (stageId: number, interviewerId: number) =>
    request<void>('POST', `/stages/${stageId}/interviewers`, { interviewer_id: interviewerId }),
  removeInterviewer: (stageId: number, interviewerId: number) =>
    request<void>('DELETE', `/stages/${stageId}/interviewers/${interviewerId}`),
  feedback: (stageId: number) => requestList<Feedback>('GET', `/stages/${stageId}/feedback`),
  submitFeedback: (stageId: number, data: FeedbackCreate) =>
    request<Feedback>('POST', `/stages/${stageId}/feedback`, data),
}

// Interviewer
export const myStages = {
  list: () => requestList<MyStage>('GET', '/me/stages'),
}
```
Then add the interfaces (place near the other `export interface` blocks):
```ts
export interface Job {
  id: number
  title: string
  description: string
  hiring_manager: string
  status: 'open' | 'closed' | 'filled'
  created_by: number
  created_at: string
}

export interface Application {
  id: number
  job_id: number
  candidate_id: number
  status: 'active' | 'rejected' | 'hired' | 'withdrawn'
  final_decision: 'strong_hire' | 'hire' | 'no_hire' | 'strong_no_hire' | null
  final_interview_notes: string | null
  created_by: number
  created_at: string
}

export interface ApplicationSummary extends Application {
  candidate_name: string
  candidate_email: string
}

export interface JobDetail extends Job {
  applications: ApplicationSummary[]
}

export interface Stage {
  id: number
  application_id: number
  type: 'phone_screen' | 'interview'
  focus_area: string
  scheduled_at: string
  video_link: string
  notes_for_interviewer: string
  status: 'pending' | 'complete' | 'canceled'
  created_at: string
}

export interface StageParticipant {
  interviewer_id: number
  interviewer_name: string
  feedback?: Feedback | null
}

export interface StageWithFeedback extends Stage {
  participants: StageParticipant[]
}

export interface ApplicationDetail extends Application {
  job: Job
  candidate: Candidate
  stages: StageWithFeedback[]
}

export interface MyStage extends Stage {
  candidate_name: string
  job_title: string
  has_my_feedback: boolean
}
```
Keep the existing `Feedback`, `FeedbackCreate`, `Competency`, `CompetencyRating` interfaces but update `Feedback` to the new shape:
```ts
export interface Feedback {
  id: number
  stage_id: number
  interviewer_id: number
  recommendation: 'strong_hire' | 'hire' | 'no_hire' | 'strong_no_hire'
  recommendation_reason: string
  free_form_notes: string
  competency_ratings?: CompetencyRating[]
}
```

- [ ] **Step 2: Type-check**

Run: `cd frontend && npx tsc --noEmit 2>&1 | head -30`
Expected: errors ONLY in the page files that still import the removed modules (fixed in Tasks 14–18). `api.ts` itself should have no errors.

- [ ] **Step 3: Commit**
```bash
git add frontend/src/lib/api.ts
git commit -m "feat(web): api client for jobs/applications/stages"
```

---

## Task 14: Jobs list + create

**Files:**
- Create: `frontend/src/pages/scheduler/JobsList.tsx`

- [ ] **Step 1: Implement JobsList**

Mirror `CandidatesList.tsx` (React Query list + shadcn create Dialog). `frontend/src/pages/scheduler/JobsList.tsx`:
```tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { jobs as jobsApi, type Job } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Plus } from 'lucide-react'

const statusColor: Record<string, string> = {
  open: 'bg-secondary text-primary',
  closed: 'bg-muted text-muted-foreground',
  filled: 'bg-green-100 text-green-800',
}

export default function JobsList() {
  const queryClient = useQueryClient()
  const { data: jobs = [] } = useQuery({ queryKey: ['jobs'], queryFn: () => jobsApi.list() })
  const [open, setOpen] = useState(false)
  const [error, setError] = useState('')
  const [form, setForm] = useState({ title: '', description: '', hiring_manager: '' })
  const reset = () => setForm({ title: '', description: '', hiring_manager: '' })

  const create = useMutation({
    mutationFn: (data: Partial<Job>) => jobsApi.create(data),
    onSuccess: () => { setError(''); queryClient.invalidateQueries({ queryKey: ['jobs'] }); setOpen(false); reset() },
    onError: (e: Error) => setError(e.message),
  })

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Jobs</h1>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button><Plus className="h-4 w-4 mr-2" />New Job</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader><DialogTitle>New Job</DialogTitle></DialogHeader>
            <form onSubmit={e => { e.preventDefault(); create.mutate({ ...form, status: 'open' }) }} className="space-y-4">
              <div className="space-y-2"><Label>Title</Label>
                <Input value={form.title} onChange={e => setForm({ ...form, title: e.target.value })} required /></div>
              <div className="space-y-2"><Label>Description</Label>
                <Textarea value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} /></div>
              <div className="space-y-2"><Label>Hiring Manager</Label>
                <Input value={form.hiring_manager} onChange={e => setForm({ ...form, hiring_manager: e.target.value })} /></div>
              {error && <p className="text-sm text-red-600">{error}</p>}
              <Button type="submit" className="w-full" disabled={create.isPending}>{create.isPending ? 'Creating…' : 'Create'}</Button>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Table>
        <TableHeader><TableRow>
          <TableHead>Title</TableHead><TableHead>Hiring Manager</TableHead><TableHead>Status</TableHead>
        </TableRow></TableHeader>
        <TableBody>
          {jobs.map((j: Job) => (
            <TableRow key={j.id}>
              <TableCell><Link to={`/jobs/${j.id}`} className="font-medium text-primary hover:underline">{j.title}</Link></TableCell>
              <TableCell>{j.hiring_manager || '—'}</TableCell>
              <TableCell><span className={`px-2 py-1 rounded text-xs font-medium ${statusColor[j.status] || ''}`}>{j.status}</span></TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
```

- [ ] **Step 2: Commit** (wired to routes in Task 17)
```bash
git add frontend/src/pages/scheduler/JobsList.tsx
git commit -m "feat(web): jobs list + create"
```

---

## Task 15: Job detail — metadata + applications + add candidate

**Files:**
- Create: `frontend/src/pages/scheduler/JobDetail.tsx`

- [ ] **Step 1: Implement JobDetail**

`frontend/src/pages/scheduler/JobDetail.tsx`:
```tsx
import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { jobs as jobsApi, applications as appsApi, candidates as candApi, type Candidate } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Plus } from 'lucide-react'

export default function JobDetail() {
  const { id } = useParams()
  const jobId = Number(id)
  const queryClient = useQueryClient()
  const { data: job } = useQuery({ queryKey: ['jobs', jobId], queryFn: () => jobsApi.get(jobId) })
  const { data: allCandidates = [] } = useQuery({ queryKey: ['candidates'], queryFn: () => candApi.list() })
  const [open, setOpen] = useState(false)
  const [selected, setSelected] = useState('')
  const [error, setError] = useState('')

  const addCandidate = useMutation({
    mutationFn: (candidateId: number) => appsApi.create(jobId, candidateId),
    onSuccess: () => { setError(''); queryClient.invalidateQueries({ queryKey: ['jobs', jobId] }); setOpen(false); setSelected('') },
    onError: (e: Error) => setError(e.message),
  })

  if (!job) return <div>Loading…</div>

  const existingCandidateIds = new Set(job.applications.map(a => a.candidate_id))
  const available = allCandidates.filter((c: Candidate) => !existingCandidateIds.has(c.id))

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{job.title}</h1>
        <p className="text-muted-foreground">{job.hiring_manager && `Hiring manager: ${job.hiring_manager}`}</p>
      </div>
      <Card>
        <CardHeader><CardTitle>Description</CardTitle></CardHeader>
        <CardContent><p className="whitespace-pre-wrap text-sm">{job.description || 'No description.'}</p></CardContent>
      </Card>

      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">Candidates</h2>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild><Button><Plus className="h-4 w-4 mr-2" />Add Candidate</Button></DialogTrigger>
          <DialogContent>
            <DialogHeader><DialogTitle>Add Candidate to Job</DialogTitle></DialogHeader>
            <form onSubmit={e => { e.preventDefault(); if (selected) addCandidate.mutate(Number(selected)) }} className="space-y-4">
              <Select value={selected} onValueChange={setSelected}>
                <SelectTrigger><SelectValue placeholder="Select candidate" /></SelectTrigger>
                <SelectContent>
                  {available.map((c: Candidate) => <SelectItem key={c.id} value={String(c.id)}>{c.name}</SelectItem>)}
                </SelectContent>
              </Select>
              {error && <p className="text-sm text-red-600">{error}</p>}
              <Button type="submit" className="w-full" disabled={!selected || addCandidate.isPending}>Add</Button>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Table>
        <TableHeader><TableRow>
          <TableHead>Candidate</TableHead><TableHead>Status</TableHead><TableHead>Decision</TableHead>
        </TableRow></TableHeader>
        <TableBody>
          {job.applications.map(a => (
            <TableRow key={a.id}>
              <TableCell><Link to={`/applications/${a.id}`} className="font-medium text-primary hover:underline">{a.candidate_name}</Link></TableCell>
              <TableCell><span className="px-2 py-1 rounded text-xs font-medium bg-secondary text-primary">{a.status}</span></TableCell>
              <TableCell>{a.final_decision ?? '—'}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
```

- [ ] **Step 2: Commit**
```bash
git add frontend/src/pages/scheduler/JobDetail.tsx
git commit -m "feat(web): job detail + add candidate (application)"
```

---

## Task 16: Application detail — stages, interviewers, decision (debrief)

**Files:**
- Create: `frontend/src/pages/scheduler/ApplicationDetail.tsx`

- [ ] **Step 1: Implement ApplicationDetail**

This is the new debrief. It lists stages with each participant's feedback, lets the scheduler add a stage, assign interviewers, and set the final decision + notes. `frontend/src/pages/scheduler/ApplicationDetail.tsx`:
```tsx
import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { applications as appsApi, stages as stagesApi, users as usersApi, type Stage } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

const DECISIONS = ['strong_hire', 'hire', 'no_hire', 'strong_no_hire'] as const

export default function ApplicationDetail() {
  const { id } = useParams()
  const appId = Number(id)
  const queryClient = useQueryClient()
  const invalidate = () => queryClient.invalidateQueries({ queryKey: ['application', appId] })
  const { data: app } = useQuery({ queryKey: ['application', appId], queryFn: () => appsApi.get(appId) })
  const { data: interviewers = [] } = useQuery({ queryKey: ['users'], queryFn: () => usersApi.list() })

  const addStage = useMutation({
    mutationFn: (data: Partial<Stage>) => stagesApi.create(appId, data),
    onSuccess: invalidate,
  })
  const assign = useMutation({
    mutationFn: ({ stageId, interviewerId }: { stageId: number; interviewerId: number }) =>
      stagesApi.addInterviewer(stageId, interviewerId),
    onSuccess: invalidate,
  })
  const saveDecision = useMutation({
    mutationFn: (data: { status: string; final_decision: string | null; final_interview_notes: string | null }) =>
      appsApi.update(appId, data as never),
    onSuccess: invalidate,
  })

  const [notes, setNotes] = useState<string | null>(null)
  const [decision, setDecision] = useState<string>('')

  if (!app) return <div>Loading…</div>
  const onlyInterviewers = interviewers.filter(u => u.role === 'interviewer')

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{app.candidate.name}</h1>
        <p className="text-muted-foreground">{app.job.title}</p>
      </div>

      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">Stages</h2>
        <AddStageButton onAdd={(data) => addStage.mutate(data)} />
      </div>

      {app.stages.map(st => (
        <Card key={st.id}>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              <span>{st.type === 'phone_screen' ? 'Phone Screen' : 'Interview'}{st.focus_area && ` — ${st.focus_area}`}</span>
              <span className="text-xs font-normal text-muted-foreground">{st.status}</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {st.participants.map(p => (
              <div key={p.interviewer_id} className="border-b pb-2 last:border-b-0">
                <div className="flex items-center justify-between">
                  <span className="font-medium">{p.interviewer_name}</span>
                  {p.feedback
                    ? <span className="text-sm font-semibold text-primary">{p.feedback.recommendation}</span>
                    : <span className="text-sm text-muted-foreground">awaiting feedback</span>}
                </div>
                {p.feedback?.recommendation_reason && <p className="text-sm text-muted-foreground">{p.feedback.recommendation_reason}</p>}
              </div>
            ))}
            <div className="flex items-center gap-2 pt-2">
              <Select onValueChange={(v) => assign.mutate({ stageId: st.id, interviewerId: Number(v) })}>
                <SelectTrigger className="w-56"><SelectValue placeholder="Assign interviewer" /></SelectTrigger>
                <SelectContent>
                  {onlyInterviewers.map(u => <SelectItem key={u.id} value={String(u.id)}>{u.name}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>
      ))}

      <Card>
        <CardHeader><CardTitle>Final Decision</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Decision</Label>
            <Select value={decision || app.final_decision || ''} onValueChange={setDecision}>
              <SelectTrigger className="w-56"><SelectValue placeholder="Select decision" /></SelectTrigger>
              <SelectContent>
                {DECISIONS.map(d => <SelectItem key={d} value={d}>{d}</SelectItem>)}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Final Interview Notes</Label>
            <Textarea value={notes ?? app.final_interview_notes ?? ''} onChange={e => setNotes(e.target.value)} placeholder="Summary of the debrief…" />
          </div>
          <Button onClick={() => saveDecision.mutate({
            status: app.status,
            final_decision: (decision || app.final_decision) || null,
            final_interview_notes: (notes ?? app.final_interview_notes) || null,
          })} disabled={saveDecision.isPending}>Save Decision</Button>
        </CardContent>
      </Card>
    </div>
  )
}

function AddStageButton({ onAdd }: { onAdd: (data: Partial<Stage>) => void }) {
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState({ type: 'interview', focus_area: '', scheduled_at: '', video_link: '' })
  // Lazy import to keep this file cohesive — reuse Dialog from ui.
  const { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } = require('@/components/ui/dialog')
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild><Button>Add Stage</Button></DialogTrigger>
      <DialogContent>
        <DialogHeader><DialogTitle>Add Stage</DialogTitle></DialogHeader>
        <form onSubmit={(e: React.FormEvent) => {
          e.preventDefault()
          onAdd({
            type: form.type as Stage['type'],
            focus_area: form.focus_area,
            scheduled_at: form.scheduled_at ? new Date(form.scheduled_at).toISOString() : new Date().toISOString(),
            video_link: form.video_link,
            status: 'pending',
          })
          setOpen(false)
        }} className="space-y-4">
          <div className="space-y-2"><Label>Type</Label>
            <Select value={form.type} onValueChange={v => setForm({ ...form, type: v })}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="phone_screen">Phone Screen</SelectItem>
                <SelectItem value="interview">Interview</SelectItem>
              </SelectContent>
            </Select></div>
          <div className="space-y-2"><Label>Focus Area</Label>
            <Input value={form.focus_area} onChange={e => setForm({ ...form, focus_area: e.target.value })} /></div>
          <div className="space-y-2"><Label>Scheduled At</Label>
            <Input type="datetime-local" value={form.scheduled_at} onChange={e => setForm({ ...form, scheduled_at: e.target.value })} /></div>
          <div className="space-y-2"><Label>Video Link</Label>
            <Input value={form.video_link} onChange={e => setForm({ ...form, video_link: e.target.value })} /></div>
          <Button type="submit" className="w-full">Add Stage</Button>
        </form>
      </DialogContent>
    </Dialog>
  )
}
```
> Implementation note: the `require(...)` inside `AddStageButton` will not work with Vite/ESM. Replace it with top-of-file imports: add `Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger` to the import from `@/components/ui/dialog` and delete the `require` line. (Left explicit so the executor consolidates imports at the top.)

- [ ] **Step 2: Type-check this file**

Run: `cd frontend && npx tsc --noEmit 2>&1 | grep ApplicationDetail | head`
Expected: no errors after fixing the imports per the note.

- [ ] **Step 3: Commit**
```bash
git add frontend/src/pages/scheduler/ApplicationDetail.tsx
git commit -m "feat(web): application detail — stages, assignment, decision"
```

---

## Task 17: Routes + nav + remove dead scheduler pages

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/Layout.tsx`
- Delete: `frontend/src/pages/scheduler/LoopEditor.tsx`, `frontend/src/pages/scheduler/DebriefView.tsx`, `frontend/src/pages/scheduler/CandidateDetail.tsx`

- [ ] **Step 1: Delete the obsolete scheduler pages**
```bash
git rm frontend/src/pages/scheduler/LoopEditor.tsx frontend/src/pages/scheduler/DebriefView.tsx frontend/src/pages/scheduler/CandidateDetail.tsx
```
(The old CandidateDetail listed loops; candidate management stays as the list page only. If you want a person-level candidate page later, it's out of scope here.)

- [ ] **Step 2: Update routes in `App.tsx`**

Open `frontend/src/App.tsx`. Remove imports/routes for `LoopEditor`, `DebriefView`, `CandidateDetail`, and any `/loops/...` routes. Add:
```tsx
import JobsList from '@/pages/scheduler/JobsList'
import JobDetail from '@/pages/scheduler/JobDetail'
import ApplicationDetail from '@/pages/scheduler/ApplicationDetail'
```
And within the authenticated `<Route element={<Layout/>}>` group (mirror how existing scheduler routes are declared, including any role guard wrapper the file already uses):
```tsx
<Route path="/jobs" element={<JobsList />} />
<Route path="/jobs/:id" element={<JobDetail />} />
<Route path="/applications/:id" element={<ApplicationDetail />} />
```
Keep `/candidates` (CandidatesList). Remove the `/candidates/:id` route. Ensure the interviewer routes point at `/my-interviews` and `/interviews/:id` (Task 18).

- [ ] **Step 3: Update nav in `Layout.tsx`**

In `frontend/src/components/Layout.tsx`, change the scheduler nav links to include Jobs:
```tsx
      case 'scheduler':
        return (
          <>
            <Link to="/jobs" className={linkClass}>Jobs</Link>
            <Link to="/candidates" className={linkClass}>Candidates</Link>
          </>
        )
```
Leave interviewer (`My Interviews`) and admin links unchanged.

- [ ] **Step 4: Build the frontend**

Run: `cd frontend && npm run build 2>&1 | tail -20`
Expected: build succeeds (errors here will be interviewer pages — fixed in Task 18; if so, finish Task 18 then build).

- [ ] **Step 5: Commit**
```bash
git add -A frontend/src/App.tsx frontend/src/components/Layout.tsx frontend/src/pages/scheduler/
git commit -m "feat(web): routes + nav for jobs/applications; drop loop pages"
```

---

# Phase 4 — Frontend: interviewer (stages → feedback)

## Task 18: My Interviews + Stage detail + feedback form

**Files:**
- Rewrite: `frontend/src/pages/interviewer/MyInterviews.tsx`
- Rewrite: `frontend/src/pages/interviewer/InterviewDetail.tsx`
- Modify: `frontend/src/pages/interviewer/FeedbackForm.tsx`

- [ ] **Step 1: Rewrite `MyInterviews.tsx` to use `myStages`**

```tsx
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { myStages as myStagesApi, type MyStage } from '@/lib/api'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'

export default function MyInterviews() {
  const { data: stages = [] } = useQuery({ queryKey: ['my-stages'], queryFn: () => myStagesApi.list() })
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">My Interviews</h1>
      <Table>
        <TableHeader><TableRow>
          <TableHead>Type</TableHead><TableHead>Candidate</TableHead><TableHead>Job</TableHead>
          <TableHead>Scheduled</TableHead><TableHead>Status</TableHead><TableHead></TableHead>
        </TableRow></TableHeader>
        <TableBody>
          {stages.map((s: MyStage) => (
            <TableRow key={s.id}>
              <TableCell>{s.type === 'phone_screen' ? 'Phone Screen' : 'Interview'}{s.focus_area && ` — ${s.focus_area}`}</TableCell>
              <TableCell>{s.candidate_name}</TableCell>
              <TableCell>{s.job_title}</TableCell>
              <TableCell>{new Date(s.scheduled_at).toLocaleString()}</TableCell>
              <TableCell>
                {s.has_my_feedback
                  ? <Badge>Feedback Submitted</Badge>
                  : <Badge variant="outline">Pending</Badge>}
              </TableCell>
              <TableCell>
                <Link to={`/interviews/${s.id}`} className="text-primary hover:underline text-sm">
                  {s.has_my_feedback ? 'View' : 'Submit Feedback'}
                </Link>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
```

- [ ] **Step 2: Rewrite `InterviewDetail.tsx` for a stage**

The page loads the stage via the application detail is overkill; instead fetch the stage's feedback list and render the feedback form. Since there is no single-stage GET endpoint, drive this page from `myStages` + the feedback form. Implementation:
```tsx
import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { myStages as myStagesApi, stages as stagesApi } from '@/lib/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import FeedbackForm from './FeedbackForm'

export default function InterviewDetail() {
  const { id } = useParams()
  const stageId = Number(id)
  const { data: myStages = [] } = useQuery({ queryKey: ['my-stages'], queryFn: () => myStagesApi.list() })
  const { data: existing = [] } = useQuery({ queryKey: ['stage-feedback', stageId], queryFn: () => stagesApi.feedback(stageId) })
  const stage = myStages.find(s => s.id === stageId)
  if (!stage) return <div>Loading…</div>

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader><CardTitle className="flex items-center justify-between">
          <span>{stage.type === 'phone_screen' ? 'Phone Screen' : 'Interview'}{stage.focus_area && ` — ${stage.focus_area}`}</span>
          <span className="text-xs font-normal text-muted-foreground">{stage.status}</span>
        </CardTitle></CardHeader>
        <CardContent className="space-y-1 text-sm">
          <p><span className="font-semibold">Candidate:</span> {stage.candidate_name}</p>
          <p><span className="font-semibold">Job:</span> {stage.job_title}</p>
          <p><span className="font-semibold">Scheduled:</span> {new Date(stage.scheduled_at).toLocaleString()}</p>
          {stage.video_link && <p><span className="font-semibold">Video:</span> <a href={stage.video_link} target="_blank" rel="noopener noreferrer" className="text-primary">Join</a></p>}
        </CardContent>
      </Card>
      <FeedbackForm stageId={stageId} alreadySubmitted={stage.has_my_feedback} existingCount={existing.length} />
    </div>
  )
}
```

- [ ] **Step 3: Update `FeedbackForm.tsx` to submit per-stage**

Open `frontend/src/pages/interviewer/FeedbackForm.tsx`. Change its props to `{ stageId: number; alreadySubmitted: boolean; existingCount?: number }` and its submit mutation to `stages.submitFeedback(stageId, data)`. Keep the existing recommendation radios + competency selects + reason/notes. The `FeedbackCreate` payload shape is unchanged except it no longer carries an interview id (the URL carries the stage id). On success, `invalidateQueries({ queryKey: ['my-stages'] })` and `['stage-feedback', stageId]`. If `alreadySubmitted`, render the read-only summary instead of the form (mirror current behavior).

> Match the existing component's competency-loading (`competencies.list()`) and payload (`recommendation`, `recommendation_reason`, `free_form_notes`, `competency_ratings`) exactly — only the submit target and props change.

- [ ] **Step 4: Build the frontend**

Run: `cd frontend && npm run build 2>&1 | tail -20`
Expected: build succeeds with no type errors.

- [ ] **Step 5: Commit**
```bash
git add frontend/src/pages/interviewer/
git commit -m "feat(web): interviewer my-interviews/stage/feedback for new model"
```

---

# Phase 5 — Seed + full run-through

## Task 19: End-to-end verification

**Files:** none (verification only)

- [ ] **Step 1: Rebuild and start the full stack**

Run:
```bash
docker compose down -v
docker compose up --build -d
# frontend is on 3002 per docker-compose.override.yml; api on 8081
sleep 5
DATABASE_URL=postgres://hire:devpassword@localhost:5433/hire?sslmode=disable go run ./seed/seed.go
```
Expected: services healthy, seed succeeds.

- [ ] **Step 2: API smoke — scheduler flow**

Run:
```bash
B=localhost:8081/api
tok(){ curl -s -X POST $B/auth/login -H 'Content-Type: application/json' -d "{\"email\":\"$1\",\"password\":\"$2\"}" | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])"; }
ST=$(tok scheduler@hire.demo scheduler)
JOB=$(curl -s -X POST $B/jobs -H "Authorization: Bearer $ST" -H 'Content-Type: application/json' -d '{"title":"SRE","description":"oncall","hiring_manager":"Dana"}')
echo "$JOB"
JID=$(echo "$JOB" | python3 -c "import sys,json;print(json.load(sys.stdin)['id'])")
curl -s $B/jobs/$JID -H "Authorization: Bearer $ST"   # expect applications: [] (never null)
```
Expected: job created; `GET /jobs/{id}` returns `"applications":[]`.

- [ ] **Step 3: Browser run-through**

Using the running app at `http://localhost:3002`, verify (log in as `scheduler@hire.demo`/`scheduler`):
- Jobs list shows seeded jobs; create a new job.
- Open a job → see candidate applications; add a candidate.
- Open an application → add a stage, assign an interviewer, see it appear.
- Log in as `alice@hire.demo`/`interviewer` → My Interviews lists assigned stages → open one → submit feedback.
- Back as scheduler → application detail shows the interviewer's feedback and a "ready for decision" notification; set final decision + notes → Save.

> Browser-driving caveat (from prior session): synthetic clicks may not trigger React handlers in the automation harness — drive forms via the page's JS (`requestSubmit`) and navigate by href if needed. A human using the app is unaffected.

- [ ] **Step 4: Full backend test suite green**

Run: `make test 2>&1 | tail -15`
Expected: all packages PASS.

- [ ] **Step 5: Final commit (if any verification fixes were needed)**
```bash
git add -A && git commit -m "test: e2e verification fixes for jobs/applications/stages"
```

---

## Notes for the executor

- **DB ports:** dev DB is on host `5433` (`docker-compose.yml`), test DB uses `hire_test` on the same server. `make test` already sets `DATABASE_URL` to the test DB.
- **Migrations run on server boot** (confirm in `cmd/server`). If not, apply with the `migrate` CLI shown in Task 1.
- **The repo is mid-branch:** uncommitted Starbucks-theme changes exist in the working tree from an earlier session. Do not revert them; they are unrelated. This plan's commits stack on top.
- **`requestList` null→[]:** every new list endpoint already goes through `requestList` in `api.ts`, preserving the null-safety fix.
- **Out of scope (do not build):** panel feedback aggregation, hiring-manager-as-user, per-job stage templates, calendar integration.
