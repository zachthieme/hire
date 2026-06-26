# Production Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate from SQLite/embedded SPA to PostgreSQL/Docker Compose/Nginx — three separate containers for db, api, and frontend.

**Architecture:** Replace the SQLite driver with pgx, rewrite all SQL for PostgreSQL syntax (positional params, RETURNING id), remove the embedded frontend, add golang-migrate for schema management, and containerize everything with Docker Compose.

**Tech Stack:** pgx/v5 (PostgreSQL driver), golang-migrate/v4 (schema migrations), Docker Compose, Nginx

---

## File Structure

```
Files to CREATE:
  migrations/000001_initial_schema.up.sql      # PostgreSQL DDL
  migrations/000001_initial_schema.down.sql     # DROP tables
  docker-compose.yml
  Dockerfile                                    # Go API multi-stage
  frontend/Dockerfile                           # React → Nginx multi-stage
  frontend/nginx.conf
  .env.example

Files to REWRITE:
  internal/store/store.go                       # PostgreSQL connection, remove SQLite
  internal/store/users.go                       # ? → $N, RETURNING id
  internal/store/candidates.go                  # same
  internal/store/competencies.go                # same
  internal/store/loops.go                       # same (dynamic WHERE needs param numbering)
  internal/store/interviews.go                  # same
  internal/store/feedback.go                    # same (transactions stay, syntax changes)
  internal/store/notifications.go               # same
  internal/store/store_test.go                  # PostgreSQL test helper
  internal/api/auth_test.go                     # PostgreSQL test helper
  cmd/server/main.go                            # Env vars, golang-migrate, no embed
  internal/api/router.go                        # Configurable CORS
  seed/seed.go                                  # PostgreSQL connection
  Makefile                                      # Docker Compose targets

Files to DELETE:
  embed.go                                      # No longer embedding frontend
  internal/store/migrations/                    # Moved to root migrations/
```

---

### Task 1: PostgreSQL migration files and golang-migrate dependency

**Files:**
- Create: `migrations/000001_initial_schema.up.sql`
- Create: `migrations/000001_initial_schema.down.sql`
- Delete: `internal/store/migrations/001_schema.sql`

- [ ] **Step 1: Install golang-migrate and pgx dependencies**

```bash
cd /home/zach/code/hire
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
go get github.com/jackc/pgx/v5/stdlib
```

- [ ] **Step 2: Create the PostgreSQL up migration**

Create `migrations/000001_initial_schema.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('admin', 'scheduler', 'interviewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS candidates (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    resume_url TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'hired', 'rejected', 'withdrawn')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS interview_loops (
    id SERIAL PRIMARY KEY,
    candidate_id INTEGER NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'scheduling' CHECK(status IN ('scheduling', 'active', 'complete')),
    final_decision TEXT CHECK(final_decision IN ('strong_hire', 'hire', 'no_hire', 'strong_no_hire')),
    debrief_notes TEXT,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS feedback (
    id SERIAL PRIMARY KEY,
    interview_id INTEGER NOT NULL UNIQUE REFERENCES interviews(id) ON DELETE CASCADE,
    recommendation TEXT NOT NULL CHECK(recommendation IN ('strong_hire', 'hire', 'no_hire', 'strong_no_hire')),
    recommendation_reason TEXT NOT NULL DEFAULT '',
    free_form_notes TEXT NOT NULL DEFAULT '',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS competencies (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    rating_type TEXT NOT NULL CHECK(rating_type IN ('levels', 'stars')),
    ratings_json TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS competency_ratings (
    id SERIAL PRIMARY KEY,
    feedback_id INTEGER NOT NULL REFERENCES feedback(id) ON DELETE CASCADE,
    competency_id INTEGER NOT NULL REFERENCES competencies(id),
    rating_value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    link TEXT NOT NULL DEFAULT '',
    read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

- [ ] **Step 3: Create the down migration**

Create `migrations/000001_initial_schema.down.sql`:

```sql
DROP TABLE IF EXISTS competency_ratings;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS feedback;
DROP TABLE IF EXISTS interviews;
DROP TABLE IF EXISTS interview_loops;
DROP TABLE IF EXISTS competencies;
DROP TABLE IF EXISTS candidates;
DROP TABLE IF EXISTS users;
```

- [ ] **Step 4: Remove old SQLite migration directory**

```bash
rm -rf internal/store/migrations/
```

- [ ] **Step 5: Commit**

```bash
git add migrations/ go.mod go.sum
git rm -r internal/store/migrations/
git commit -m "feat: PostgreSQL migration files with golang-migrate"
```

---

### Task 2: Rewrite store.go for PostgreSQL

**Files:**
- Rewrite: `internal/store/store.go`
- Delete: `embed.go`

- [ ] **Step 1: Rewrite store.go**

Replace the entire contents of `internal/store/store.go`:

```go
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Store struct {
	db *sql.DB
}

func New(databaseURL string) (*Store, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
```

- [ ] **Step 2: Delete embed.go**

```bash
rm /home/zach/code/hire/embed.go
```

- [ ] **Step 3: Commit**

```bash
git add internal/store/store.go
git rm embed.go
git commit -m "feat: rewrite store.go for PostgreSQL via pgx"
```

---

### Task 3: Rewrite users.go, candidates.go, competencies.go for PostgreSQL

**Files:**
- Rewrite: `internal/store/users.go`
- Rewrite: `internal/store/candidates.go`
- Rewrite: `internal/store/competencies.go`

- [ ] **Step 1: Rewrite users.go**

Replace `internal/store/users.go`:

```go
package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateUser(u *models.User) error {
	err := s.db.QueryRow(
		`INSERT INTO users (email, name, password_hash, role) VALUES ($1, $2, $3, $4) RETURNING id`,
		u.Email, u.Name, u.PasswordHash, u.Role,
	).Scan(&u.ID)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (s *Store) GetUserByID(id int64) (*models.User, error) {
	var u models.User
	err := s.db.QueryRow(
		`SELECT id, email, name, role, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return &u, err
}

func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	var u models.User
	err := s.db.QueryRow(
		`SELECT id, email, name, password_hash, role, created_at FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return &u, err
}

func (s *Store) ListUsers(limit, offset int) ([]*models.User, error) {
	rows, err := s.db.Query(
		`SELECT id, email, name, role, created_at FROM users ORDER BY id LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	var users []*models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

func (s *Store) UpdateUser(u *models.User) error {
	_, err := s.db.Exec(
		`UPDATE users SET email = $1, name = $2, password_hash = $3, role = $4 WHERE id = $5`,
		u.Email, u.Name, u.PasswordHash, u.Role, u.ID,
	)
	return err
}

func (s *Store) DeleteUser(id int64) error {
	_, err := s.db.Exec(`DELETE FROM users WHERE id = $1`, id)
	return err
}
```

- [ ] **Step 2: Rewrite candidates.go**

Replace `internal/store/candidates.go`:

```go
package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateCandidate(c *models.Candidate) error {
	err := s.db.QueryRow(
		`INSERT INTO candidates (name, email, resume_url, status) VALUES ($1, $2, $3, $4) RETURNING id`,
		c.Name, c.Email, c.ResumeURL, c.Status,
	).Scan(&c.ID)
	if err != nil {
		return fmt.Errorf("insert candidate: %w", err)
	}
	return nil
}

func (s *Store) GetCandidate(id int64) (*models.Candidate, error) {
	var c models.Candidate
	err := s.db.QueryRow(
		`SELECT id, name, email, resume_url, status, created_at FROM candidates WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.Email, &c.ResumeURL, &c.Status, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("candidate not found")
	}
	return &c, err
}

func (s *Store) ListCandidates(limit, offset int) ([]*models.Candidate, error) {
	rows, err := s.db.Query(
		`SELECT id, name, email, resume_url, status, created_at FROM candidates ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Candidate
	for rows.Next() {
		var c models.Candidate
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.ResumeURL, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (s *Store) UpdateCandidate(c *models.Candidate) error {
	_, err := s.db.Exec(
		`UPDATE candidates SET name = $1, email = $2, resume_url = $3, status = $4 WHERE id = $5`,
		c.Name, c.Email, c.ResumeURL, c.Status, c.ID,
	)
	return err
}

func (s *Store) DeleteCandidate(id int64) error {
	_, err := s.db.Exec(`DELETE FROM candidates WHERE id = $1`, id)
	return err
}
```

- [ ] **Step 3: Rewrite competencies.go**

Replace `internal/store/competencies.go`:

```go
package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateCompetency(c *models.Competency) error {
	err := s.db.QueryRow(
		`INSERT INTO competencies (name, rating_type, ratings_json) VALUES ($1, $2, $3) RETURNING id`,
		c.Name, c.RatingType, c.RatingsJSON,
	).Scan(&c.ID)
	if err != nil {
		return fmt.Errorf("insert competency: %w", err)
	}
	return nil
}

func (s *Store) GetCompetency(id int64) (*models.Competency, error) {
	var c models.Competency
	err := s.db.QueryRow(
		`SELECT id, name, rating_type, ratings_json, created_at FROM competencies WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.RatingType, &c.RatingsJSON, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("competency not found")
	}
	return &c, err
}

func (s *Store) ListCompetencies() ([]*models.Competency, error) {
	rows, err := s.db.Query(`SELECT id, name, rating_type, ratings_json, created_at FROM competencies ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Competency
	for rows.Next() {
		var c models.Competency
		if err := rows.Scan(&c.ID, &c.Name, &c.RatingType, &c.RatingsJSON, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (s *Store) UpdateCompetency(c *models.Competency) error {
	_, err := s.db.Exec(
		`UPDATE competencies SET name = $1, rating_type = $2, ratings_json = $3 WHERE id = $4`,
		c.Name, c.RatingType, c.RatingsJSON, c.ID,
	)
	return err
}

func (s *Store) DeleteCompetency(id int64) error {
	_, err := s.db.Exec(`DELETE FROM competencies WHERE id = $1`, id)
	return err
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/store/users.go internal/store/candidates.go internal/store/competencies.go
git commit -m "feat: rewrite users, candidates, competencies store for PostgreSQL"
```

---

### Task 4: Rewrite loops.go and interviews.go for PostgreSQL

**Files:**
- Rewrite: `internal/store/loops.go`
- Rewrite: `internal/store/interviews.go`

- [ ] **Step 1: Rewrite loops.go**

Replace `internal/store/loops.go`. Key changes: `?` → `$N`, `RETURNING id`, dynamic WHERE with numbered params, IN clause with numbered params:

```go
package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateLoop(l *models.InterviewLoop) error {
	err := s.db.QueryRow(
		`INSERT INTO interview_loops (candidate_id, status, created_by) VALUES ($1, $2, $3) RETURNING id`,
		l.CandidateID, l.Status, l.CreatedBy,
	).Scan(&l.ID)
	if err != nil {
		return fmt.Errorf("insert loop: %w", err)
	}
	return nil
}

func (s *Store) GetLoop(id int64) (*models.InterviewLoop, error) {
	var l models.InterviewLoop
	err := s.db.QueryRow(
		`SELECT id, candidate_id, status, final_decision, debrief_notes, created_by, created_at
		 FROM interview_loops WHERE id = $1`, id,
	).Scan(&l.ID, &l.CandidateID, &l.Status, &l.FinalDecision, &l.DebriefNotes, &l.CreatedBy, &l.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("loop not found")
	}
	return &l, err
}

func (s *Store) ListLoops(candidateID *int64, status *string, limit, offset int) ([]*models.InterviewLoop, error) {
	query := `SELECT id, candidate_id, status, final_decision, debrief_notes, created_by, created_at FROM interview_loops WHERE 1=1`
	var args []any
	paramIdx := 1
	if candidateID != nil {
		query += fmt.Sprintf(` AND candidate_id = $%d`, paramIdx)
		args = append(args, *candidateID)
		paramIdx++
	}
	if status != nil {
		query += fmt.Sprintf(` AND status = $%d`, paramIdx)
		args = append(args, *status)
		paramIdx++
	}
	query += fmt.Sprintf(` ORDER BY id DESC LIMIT $%d OFFSET $%d`, paramIdx, paramIdx+1)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.InterviewLoop
	for rows.Next() {
		var l models.InterviewLoop
		if err := rows.Scan(&l.ID, &l.CandidateID, &l.Status, &l.FinalDecision, &l.DebriefNotes, &l.CreatedBy, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &l)
	}
	return out, rows.Err()
}

func (s *Store) UpdateLoop(l *models.InterviewLoop) error {
	_, err := s.db.Exec(
		`UPDATE interview_loops SET status = $1, final_decision = $2, debrief_notes = $3 WHERE id = $4`,
		l.Status, l.FinalDecision, l.DebriefNotes, l.ID,
	)
	return err
}

func (s *Store) DeleteLoop(id int64) error {
	_, err := s.db.Exec(`DELETE FROM interview_loops WHERE id = $1`, id)
	return err
}

// GetLoopDetail returns a loop with its candidate, interviews, and feedback.
func (s *Store) GetLoopDetail(id int64) (*models.LoopDetail, error) {
	loop, err := s.GetLoop(id)
	if err != nil {
		return nil, err
	}
	candidate, err := s.GetCandidate(loop.CandidateID)
	if err != nil {
		return nil, fmt.Errorf("get candidate for loop: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT i.id, i.loop_id, i.interviewer_id, i.focus_area, i.scheduled_at, i.video_link,
		        i.notes_for_interviewer, i.status, i.created_at, u.name
		 FROM interviews i
		 JOIN users u ON i.interviewer_id = u.id
		 WHERE i.loop_id = $1
		 ORDER BY i.scheduled_at`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("list interviews for loop: %w", err)
	}
	defer rows.Close()

	var interviewIDs []int64
	interviewMap := make(map[int64]*models.InterviewWithFeedback)
	detail := &models.LoopDetail{
		InterviewLoop: *loop,
		Candidate:     *candidate,
	}

	for rows.Next() {
		var iwf models.InterviewWithFeedback
		if err := rows.Scan(
			&iwf.ID, &iwf.LoopID, &iwf.InterviewerID, &iwf.FocusArea, &iwf.ScheduledAt,
			&iwf.VideoLink, &iwf.NotesForInterviewer, &iwf.Status, &iwf.CreatedAt,
			&iwf.InterviewerName,
		); err != nil {
			return nil, err
		}
		detail.Interviews = append(detail.Interviews, iwf)
		interviewIDs = append(interviewIDs, iwf.ID)
		interviewMap[iwf.ID] = &detail.Interviews[len(detail.Interviews)-1]
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(interviewIDs) > 0 {
		placeholders := make([]string, len(interviewIDs))
		args := make([]any, len(interviewIDs))
		for i, ivID := range interviewIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = ivID
		}
		fbQuery := `SELECT f.id, f.interview_id, f.recommendation, f.recommendation_reason, f.free_form_notes, f.submitted_at
			FROM feedback f WHERE f.interview_id IN (` + strings.Join(placeholders, ",") + `)`
		fbRows, err := s.db.Query(fbQuery, args...)
		if err != nil {
			return nil, err
		}
		defer fbRows.Close()

		for fbRows.Next() {
			var fb models.Feedback
			if err := fbRows.Scan(&fb.ID, &fb.InterviewID, &fb.Recommendation, &fb.RecommendationReason, &fb.FreeFormNotes, &fb.SubmittedAt); err != nil {
				return nil, err
			}
			fb.CompetencyRatings, _ = s.listCompetencyRatings(fb.ID)
			if iwf, ok := interviewMap[fb.InterviewID]; ok {
				iwf.Feedback = &fb
			}
		}
	}

	return detail, nil
}

```

- [ ] **Step 2: Rewrite interviews.go**

Replace `internal/store/interviews.go`:

```go
package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateInterview(iv *models.Interview) error {
	err := s.db.QueryRow(
		`INSERT INTO interviews (loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		iv.LoopID, iv.InterviewerID, iv.FocusArea, iv.ScheduledAt, iv.VideoLink, iv.NotesForInterviewer, iv.Status,
	).Scan(&iv.ID)
	if err != nil {
		return fmt.Errorf("insert interview: %w", err)
	}
	return nil
}

func (s *Store) GetInterview(id int64) (*models.Interview, error) {
	var iv models.Interview
	err := s.db.QueryRow(
		`SELECT id, loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at
		 FROM interviews WHERE id = $1`, id,
	).Scan(&iv.ID, &iv.LoopID, &iv.InterviewerID, &iv.FocusArea, &iv.ScheduledAt, &iv.VideoLink,
		&iv.NotesForInterviewer, &iv.Status, &iv.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("interview not found")
	}
	return &iv, err
}

func (s *Store) ListInterviewsByLoop(loopID int64) ([]*models.Interview, error) {
	return s.queryInterviews(`SELECT id, loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at
		FROM interviews WHERE loop_id = $1 ORDER BY scheduled_at`, loopID)
}

func (s *Store) ListInterviewsByUser(userID int64) ([]*models.Interview, error) {
	return s.queryInterviews(`SELECT id, loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at
		FROM interviews WHERE interviewer_id = $1 ORDER BY scheduled_at DESC`, userID)
}

func (s *Store) UpdateInterview(iv *models.Interview) error {
	_, err := s.db.Exec(
		`UPDATE interviews SET interviewer_id = $1, focus_area = $2, scheduled_at = $3, video_link = $4, notes_for_interviewer = $5, status = $6
		 WHERE id = $7`,
		iv.InterviewerID, iv.FocusArea, iv.ScheduledAt, iv.VideoLink, iv.NotesForInterviewer, iv.Status, iv.ID,
	)
	return err
}

func (s *Store) DeleteInterview(id int64) error {
	_, err := s.db.Exec(`DELETE FROM interviews WHERE id = $1`, id)
	return err
}

func (s *Store) queryInterviews(query string, args ...any) ([]*models.Interview, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Interview
	for rows.Next() {
		var iv models.Interview
		if err := rows.Scan(&iv.ID, &iv.LoopID, &iv.InterviewerID, &iv.FocusArea, &iv.ScheduledAt,
			&iv.VideoLink, &iv.NotesForInterviewer, &iv.Status, &iv.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &iv)
	}
	return out, rows.Err()
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/store/loops.go internal/store/interviews.go
git commit -m "feat: rewrite loops and interviews store for PostgreSQL"
```

---

### Task 5: Rewrite feedback.go and notifications.go for PostgreSQL

**Files:**
- Rewrite: `internal/store/feedback.go`
- Rewrite: `internal/store/notifications.go`

- [ ] **Step 1: Rewrite feedback.go**

Replace `internal/store/feedback.go`. Key: transactions use `$N` params, `RETURNING id` inside transactions via `tx.QueryRow`:

```go
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
```

- [ ] **Step 2: Rewrite notifications.go**

Replace `internal/store/notifications.go`:

```go
package store

import (
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateNotification(n *models.Notification) error {
	err := s.db.QueryRow(
		`INSERT INTO notifications (user_id, message, link) VALUES ($1, $2, $3) RETURNING id`,
		n.UserID, n.Message, n.Link,
	).Scan(&n.ID)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	return nil
}

func (s *Store) ListNotificationsByUser(userID int64, limit, offset int) ([]*models.Notification, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, message, link, read, created_at FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Message, &n.Link, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (s *Store) MarkNotificationRead(id, userID int64) error {
	res, err := s.db.Exec(`UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("notification not found")
	}
	return nil
}

func (s *Store) CountUnreadNotifications(userID int64) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = false`, userID).Scan(&count)
	return count, err
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/store/feedback.go internal/store/notifications.go
git commit -m "feat: rewrite feedback and notifications store for PostgreSQL"
```

---

### Task 6: Update test infrastructure for PostgreSQL

**Files:**
- Rewrite: `internal/store/store_test.go`
- Rewrite: `internal/api/auth_test.go`

- [ ] **Step 1: Rewrite store test helper**

Replace `internal/store/store_test.go`. Tests require a running PostgreSQL (from Docker Compose). The helper connects, runs migrations via golang-migrate, and truncates all tables between tests:

```go
package store

import (
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func testDSN() string {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://hire:devpassword@localhost:5432/hire_test?sslmode=disable"
	}
	return dsn
}

func TestMain(m *testing.M) {
	dsn := testDSN()
	mig, err := migrate.New("file://../../migrations", dsn)
	if err != nil {
		panic("migrate.New: " + err.Error())
	}
	mig.Up() // ignore "no change" errors
	os.Exit(m.Run())
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(testDSN())
	if err != nil {
		t.Fatalf("newTestStore: %v", err)
	}
	// Truncate all tables (order matters due to FK constraints)
	s.db.Exec("TRUNCATE competency_ratings, notifications, feedback, interviews, interview_loops, competencies, candidates, users RESTART IDENTITY CASCADE")
	t.Cleanup(func() { s.Close() })
	return s
}
```

- [ ] **Step 2: Rewrite API test helper**

Replace the `newTestHandler` function in `internal/api/auth_test.go`. The test helper in the api package also needs to connect to PostgreSQL:

```go
package api

import (
	"bytes"
	"encoding/json"
	"hire/internal/models"
	"hire/internal/store"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func testDSN() string {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://hire:devpassword@localhost:5432/hire_test?sslmode=disable"
	}
	return dsn
}

func TestMain(m *testing.M) {
	dsn := testDSN()
	mig, err := migrate.New("file://../../migrations", dsn)
	if err != nil {
		panic("migrate.New: " + err.Error())
	}
	mig.Up()
	os.Exit(m.Run())
}

func newTestHandler(t *testing.T) (*Handler, *store.Store) {
	t.Helper()
	s, err := store.New(testDSN())
	if err != nil {
		t.Fatalf("newTestHandler: %v", err)
	}
	// Truncate all tables
	db := s.DB()
	db.Exec("TRUNCATE competency_ratings, notifications, feedback, interviews, interview_loops, competencies, candidates, users RESTART IDENTITY CASCADE")
	t.Cleanup(func() { s.Close() })
	h := NewHandler(s, "test-secret")
	return h, s
}

// ... rest of auth_test.go (TestLoginSuccess, TestLoginWrongPassword, TestAuthMiddleware) stays the same
```

**Important:** The test helper calls `s.DB()` to access the underlying `*sql.DB` for truncation. Add this method to `internal/store/store.go`:

```go
// DB returns the underlying database connection for administrative operations (e.g., test cleanup).
func (s *Store) DB() *sql.DB {
	return s.db
}
```

- [ ] **Step 3: Verify tests compile** (they won't pass yet without a running PostgreSQL — that comes in Task 9)

```bash
go build ./internal/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go internal/api/auth_test.go
git commit -m "feat: update test infrastructure for PostgreSQL"
```

---

### Task 7: Rewrite cmd/server/main.go and update router

**Files:**
- Rewrite: `cmd/server/main.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Rewrite main.go**

Replace `cmd/server/main.go`. All config from env vars, runs golang-migrate on startup, no embed/SPA serving:

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"hire/internal/api"
	"hire/internal/store"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Run migrations
	mig, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		log.Fatalf("Failed to create migrator: %v", err)
	}
	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	s, err := store.New(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer s.Close()

	h := api.NewHandler(s, jwtSecret)
	r := h.Router()

	addr := ":" + port
	fmt.Printf("Server listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
```

- [ ] **Step 2: Update router.go CORS**

In `internal/api/router.go`, make CORS allow all origins (nginx handles the real proxying in production, CORS is only needed for local dev):

Replace the CORS config:

```go
r.Use(cors.Handler(cors.Options{
    AllowedOrigins:   []string{"*"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Authorization", "Content-Type"},
    AllowCredentials: false,
}))
```

Note: `AllowCredentials` must be `false` when using `*` origin.

- [ ] **Step 3: Verify build**

```bash
go build ./cmd/server
rm -f server
```

- [ ] **Step 4: Commit**

```bash
git add cmd/server/main.go internal/api/router.go
git commit -m "feat: rewrite main.go for env vars and golang-migrate, update CORS"
```

---

### Task 8: Docker files

**Files:**
- Create: `Dockerfile`
- Create: `frontend/Dockerfile`
- Create: `frontend/nginx.conf`
- Create: `docker-compose.yml`
- Create: `.env.example`

- [ ] **Step 1: Create Go API Dockerfile**

Create `Dockerfile`:

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /app/server /server
COPY migrations/ /migrations/
EXPOSE 8080
CMD ["/server"]
```

- [ ] **Step 2: Create Nginx config**

Create `frontend/nginx.conf`:

```nginx
server {
    listen 80;
    server_name _;

    location / {
        root /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://api:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

- [ ] **Step 3: Create frontend Dockerfile**

Create `frontend/Dockerfile`:

```dockerfile
FROM node:22-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

- [ ] **Step 4: Create docker-compose.yml**

Create `docker-compose.yml`:

```yaml
services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: hire
      POSTGRES_USER: hire
      POSTGRES_PASSWORD: ${DB_PASSWORD:-devpassword}
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U hire"]
      interval: 5s
      timeout: 3s
      retries: 5

  api:
    build: .
    environment:
      DATABASE_URL: postgres://hire:${DB_PASSWORD:-devpassword}@db:5432/hire?sslmode=disable
      JWT_SECRET: ${JWT_SECRET:-dev-secret-change-me}
      PORT: "8080"
    ports:
      - "8080:8080"
    depends_on:
      db:
        condition: service_healthy

  frontend:
    build: ./frontend
    ports:
      - "3000:80"
    depends_on:
      - api

volumes:
  pgdata:
```

- [ ] **Step 5: Create .env.example**

Create `.env.example`:

```
DATABASE_URL=postgres://hire:devpassword@localhost:5432/hire?sslmode=disable
JWT_SECRET=change-me-to-a-real-secret
DB_PASSWORD=devpassword
```

- [ ] **Step 6: Update .gitignore**

Add to `.gitignore`:

```
.env
```

- [ ] **Step 7: Commit**

```bash
git add Dockerfile frontend/Dockerfile frontend/nginx.conf docker-compose.yml .env.example .gitignore
git commit -m "feat: Docker Compose setup with PostgreSQL, Go API, and Nginx frontend"
```

---

### Task 9: Update Makefile, seed.go, and run go mod tidy

**Files:**
- Rewrite: `Makefile`
- Rewrite: `seed/seed.go`

- [ ] **Step 1: Rewrite Makefile**

Replace `Makefile`:

```makefile
.PHONY: up down logs test seed migrate-new clean

# Start all services
up:
	docker compose up --build -d

# Stop all services
down:
	docker compose down

# Follow logs
logs:
	docker compose logs -f

# Run tests (requires running db: docker compose up db -d)
test:
	DATABASE_URL=postgres://hire:devpassword@localhost:5432/hire_test?sslmode=disable go test ./internal/... -v

# Seed demo data (requires running db)
seed:
	DATABASE_URL=postgres://hire:devpassword@localhost:5432/hire?sslmode=disable go run ./seed/seed.go

# Create a new migration
migrate-new:
	migrate create -ext sql -dir migrations -seq $(name)

# Clean
clean:
	docker compose down -v
	rm -f server
```

- [ ] **Step 2: Rewrite seed.go**

Replace `seed/seed.go` to connect via `DATABASE_URL` and use golang-migrate:

```go
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"hire/internal/api"
	"hire/internal/models"
	"hire/internal/store"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	// Run migrations
	mig, err := migrate.New("file://migrations", dsn)
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migrate up: %v", err)
	}

	// Clean existing data
	s, err := store.New(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()
	s.DB().Exec("TRUNCATE competency_ratings, notifications, feedback, interviews, interview_loops, competencies, candidates, users RESTART IDENTITY CASCADE")

	// Admin user
	adminHash, _ := api.HashPassword("admin")
	admin := &models.User{Email: "admin@hire.demo", Name: "Admin User", PasswordHash: adminHash, Role: "admin"}
	s.CreateUser(admin)

	// Scheduler
	schedHash, _ := api.HashPassword("scheduler")
	sched := &models.User{Email: "scheduler@hire.demo", Name: "Sarah Scheduler", PasswordHash: schedHash, Role: "scheduler"}
	s.CreateUser(sched)

	// Interviewers
	ivHash, _ := api.HashPassword("interviewer")
	alice := &models.User{Email: "alice@hire.demo", Name: "Alice Engineer", PasswordHash: ivHash, Role: "interviewer"}
	s.CreateUser(alice)
	bob := &models.User{Email: "bob@hire.demo", Name: "Bob Designer", PasswordHash: ivHash, Role: "interviewer"}
	s.CreateUser(bob)
	carol := &models.User{Email: "carol@hire.demo", Name: "Carol Manager", PasswordHash: ivHash, Role: "interviewer"}
	s.CreateUser(carol)
	dave := &models.User{Email: "dave@hire.demo", Name: "Dave Architect", PasswordHash: ivHash, Role: "interviewer"}
	s.CreateUser(dave)

	// Competencies
	s.CreateCompetency(&models.Competency{Name: "Problem Solving", RatingType: "levels", RatingsJSON: `["Learning","Owning","Advising"]`})
	s.CreateCompetency(&models.Competency{Name: "Communication", RatingType: "levels", RatingsJSON: `["Learning","Owning","Advising"]`})
	s.CreateCompetency(&models.Competency{Name: "Technical Depth", RatingType: "stars", RatingsJSON: `{"min":1,"max":5}`})
	s.CreateCompetency(&models.Competency{Name: "Culture Fit", RatingType: "stars", RatingsJSON: `{"min":1,"max":5}`})

	// Candidates
	jane := &models.Candidate{Name: "Jane Smith", Email: "jane@example.com", ResumeURL: "https://example.com/resume/jane", Status: "active"}
	s.CreateCandidate(jane)
	mike := &models.Candidate{Name: "Mike Johnson", Email: "mike@example.com", ResumeURL: "https://example.com/resume/mike", Status: "active"}
	s.CreateCandidate(mike)

	// Interview loop for Jane
	loop1 := &models.InterviewLoop{CandidateID: jane.ID, Status: "active", CreatedBy: sched.ID}
	s.CreateLoop(loop1)

	tomorrow := time.Now().Add(24 * time.Hour)
	s.CreateInterview(&models.Interview{LoopID: loop1.ID, InterviewerID: alice.ID, FocusArea: "Coding", ScheduledAt: tomorrow, VideoLink: "https://meet.example.com/jane-coding", NotesForInterviewer: "Focus on data structures and algorithms", Status: "pending"})
	s.CreateInterview(&models.Interview{LoopID: loop1.ID, InterviewerID: bob.ID, FocusArea: "System Design", ScheduledAt: tomorrow.Add(time.Hour), VideoLink: "https://meet.example.com/jane-design", NotesForInterviewer: "Distributed systems focus", Status: "pending"})
	s.CreateInterview(&models.Interview{LoopID: loop1.ID, InterviewerID: carol.ID, FocusArea: "Behavioral", ScheduledAt: tomorrow.Add(2 * time.Hour), VideoLink: "https://meet.example.com/jane-behavioral", NotesForInterviewer: "Leadership and teamwork", Status: "pending"})
	s.CreateInterview(&models.Interview{LoopID: loop1.ID, InterviewerID: dave.ID, FocusArea: "Architecture", ScheduledAt: tomorrow.Add(3 * time.Hour), VideoLink: "https://meet.example.com/jane-arch", NotesForInterviewer: "API design and scalability", Status: "pending"})

	// Interview loop for Mike (scheduling phase)
	loop2 := &models.InterviewLoop{CandidateID: mike.ID, Status: "scheduling", CreatedBy: sched.ID}
	s.CreateLoop(loop2)
	s.CreateInterview(&models.Interview{LoopID: loop2.ID, InterviewerID: alice.ID, FocusArea: "Coding", ScheduledAt: tomorrow.Add(48 * time.Hour), VideoLink: "https://meet.example.com/mike-coding", Status: "pending"})

	// Notifications
	s.CreateNotification(&models.Notification{UserID: alice.ID, Message: "You've been assigned a Coding interview", Link: "/interviews/1"})
	s.CreateNotification(&models.Notification{UserID: bob.ID, Message: "You've been assigned a System Design interview", Link: "/interviews/2"})
	s.CreateNotification(&models.Notification{UserID: carol.ID, Message: "You've been assigned a Behavioral interview", Link: "/interviews/3"})
	s.CreateNotification(&models.Notification{UserID: dave.ID, Message: "You've been assigned an Architecture interview", Link: "/interviews/4"})

	fmt.Println("Seed data created successfully!")
	fmt.Println()
	fmt.Println("Demo accounts:")
	fmt.Println("  admin@hire.demo      / admin        (Admin)")
	fmt.Println("  scheduler@hire.demo  / scheduler    (Scheduler)")
	fmt.Println("  alice@hire.demo      / interviewer  (Interviewer)")
	fmt.Println("  bob@hire.demo        / interviewer  (Interviewer)")
	fmt.Println("  carol@hire.demo      / interviewer  (Interviewer)")
	fmt.Println("  dave@hire.demo       / interviewer  (Interviewer)")
}
```

- [ ] **Step 3: Run go mod tidy**

```bash
go mod tidy
```

This will remove the `modernc.org/sqlite` dependency and add the `pgx` and `golang-migrate` dependencies.

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add Makefile seed/seed.go go.mod go.sum
git commit -m "feat: update Makefile and seed for Docker Compose and PostgreSQL"
```

---

### Task 10: End-to-end verification

**Files:** None (verification only)

- [ ] **Step 1: Create the test database**

```bash
docker compose up db -d
sleep 3
docker compose exec db psql -U hire -c "CREATE DATABASE hire_test OWNER hire;"
```

- [ ] **Step 2: Run tests**

```bash
DATABASE_URL=postgres://hire:devpassword@localhost:5432/hire_test?sslmode=disable go test ./internal/... -v
```

Expected: all tests PASS.

- [ ] **Step 3: Start the full stack**

```bash
docker compose up --build -d
```

Expected: all three containers start (db, api, frontend).

- [ ] **Step 4: Seed data**

```bash
DATABASE_URL=postgres://hire:devpassword@localhost:5432/hire?sslmode=disable go run ./seed/seed.go
```

Expected: "Seed data created successfully!" with demo accounts listed.

- [ ] **Step 5: Smoke test**

```bash
# Test API via api container
curl -s -X POST http://localhost:8080/api/auth/login -H 'Content-Type: application/json' -d '{"email":"admin@hire.demo","password":"admin"}'

# Test frontend via nginx
curl -s http://localhost:3000/ | head -3
```

Expected: login returns JWT token, frontend returns HTML.

- [ ] **Step 6: Stop services**

```bash
docker compose down
```
