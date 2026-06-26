# Interview Scheduling & Debrief App — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a single-binary demo web app where schedulers manage interview loops and interviewers submit structured feedback, with admin-configurable competencies and a debrief view.

**Architecture:** Go backend (Chi router) serves a REST API and an embedded React SPA. SQLite database via pure-Go driver. JWT auth. The frontend is built with Vite and bundled into the Go binary via `embed.FS`.

**Tech Stack:** Go 1.22+, Chi v5, modernc.org/sqlite, golang-jwt/jwt v5, bcrypt | React 18, Vite, TypeScript, Tailwind CSS, shadcn/ui, React Router v6, TanStack React Query

---

## File Structure

```
hire/
├── cmd/server/main.go                          # Entry point: flags, starts server
├── embed.go                                    # //go:embed for frontend/dist
├── go.mod
├── go.sum
├── Makefile
├── migrations/
│   └── 001_schema.sql                          # All tables
├── internal/
│   ├── models/
│   │   └── models.go                           # All domain structs
│   ├── store/
│   │   ├── store.go                            # DB init, migration runner
│   │   ├── store_test.go                       # Test helper (newTestStore)
│   │   ├── users.go                            # User CRUD
│   │   ├── users_test.go
│   │   ├── candidates.go
│   │   ├── candidates_test.go
│   │   ├── competencies.go
│   │   ├── competencies_test.go
│   │   ├── loops.go                            # Interview loop CRUD
│   │   ├── loops_test.go
│   │   ├── interviews.go
│   │   ├── interviews_test.go
│   │   ├── feedback.go                         # Feedback + competency ratings
│   │   ├── feedback_test.go
│   │   ├── notifications.go
│   │   └── notifications_test.go
│   ├── api/
│   │   ├── handler.go                          # Handler struct, helpers (JSON, errors)
│   │   ├── router.go                           # Chi router wiring
│   │   ├── middleware.go                        # JWT auth, role check
│   │   ├── auth.go                             # Login handler
│   │   ├── auth_test.go
│   │   ├── users.go
│   │   ├── users_test.go
│   │   ├── candidates.go
│   │   ├── candidates_test.go
│   │   ├── competencies.go
│   │   ├── competencies_test.go
│   │   ├── loops.go
│   │   ├── loops_test.go
│   │   ├── interviews.go
│   │   ├── interviews_test.go
│   │   ├── feedback.go                         # Feedback handlers + visibility rule
│   │   ├── feedback_test.go
│   │   ├── notifications.go
│   │   └── notifications_test.go
│   └── notify/
│       └── notify.go                           # Creates notification records
├── frontend/
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── tsconfig.app.json
│   ├── vite.config.ts
│   ├── tailwind.config.ts
│   ├── postcss.config.js
│   ├── components.json                         # shadcn/ui config
│   └── src/
│       ├── main.tsx
│       ├── App.tsx                              # Routes
│       ├── index.css                            # Tailwind directives
│       ├── lib/
│       │   ├── api.ts                           # Typed fetch wrapper
│       │   ├── auth.tsx                         # AuthContext, ProtectedRoute
│       │   └── utils.ts                         # cn() helper (shadcn)
│       ├── components/
│       │   ├── ui/                              # shadcn/ui primitives (auto-generated)
│       │   ├── Layout.tsx                       # Shell: nav, notification bell, outlet
│       │   └── NotificationBell.tsx
│       └── pages/
│           ├── LoginPage.tsx
│           ├── Dashboard.tsx                    # Role-based redirect
│           ├── admin/
│           │   ├── UserManagement.tsx
│           │   └── CompetencyManagement.tsx
│           ├── scheduler/
│           │   ├── CandidatesList.tsx
│           │   ├── CandidateDetail.tsx
│           │   ├── LoopEditor.tsx
│           │   └── DebriefView.tsx
│           └── interviewer/
│               ├── MyInterviews.tsx
│               ├── InterviewDetail.tsx
│               └── FeedbackForm.tsx
└── seed/
    └── seed.go                                 # Standalone program to populate demo data
```

---

## Phase 1: Backend Foundation

### Task 1: Project scaffolding, schema, and models

**Files:**
- Create: `go.mod`, `migrations/001_schema.sql`, `internal/models/models.go`, `internal/store/store.go`, `internal/store/store_test.go`

- [ ] **Step 1: Initialize Go module**

```bash
cd /home/zach/code/hire
go mod init hire
```

- [ ] **Step 2: Create the database migration**

Create `migrations/001_schema.sql`:

```sql
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('admin', 'scheduler', 'interviewer')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS candidates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    resume_url TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'hired', 'rejected', 'withdrawn')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS interview_loops (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    candidate_id INTEGER NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'scheduling' CHECK(status IN ('scheduling', 'active', 'complete')),
    final_decision TEXT CHECK(final_decision IN ('strong_hire', 'hire', 'no_hire', 'strong_no_hire')),
    debrief_notes TEXT,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS interviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    loop_id INTEGER NOT NULL REFERENCES interview_loops(id) ON DELETE CASCADE,
    interviewer_id INTEGER NOT NULL REFERENCES users(id),
    focus_area TEXT NOT NULL,
    scheduled_at DATETIME NOT NULL,
    video_link TEXT NOT NULL DEFAULT '',
    notes_for_interviewer TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'complete')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS feedback (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    interview_id INTEGER NOT NULL UNIQUE REFERENCES interviews(id) ON DELETE CASCADE,
    recommendation TEXT NOT NULL CHECK(recommendation IN ('strong_hire', 'hire', 'no_hire', 'strong_no_hire')),
    recommendation_reason TEXT NOT NULL DEFAULT '',
    free_form_notes TEXT NOT NULL DEFAULT '',
    submitted_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS competencies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    rating_type TEXT NOT NULL CHECK(rating_type IN ('levels', 'stars')),
    ratings_json TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS competency_ratings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    feedback_id INTEGER NOT NULL REFERENCES feedback(id) ON DELETE CASCADE,
    competency_id INTEGER NOT NULL REFERENCES competencies(id),
    rating_value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    link TEXT NOT NULL DEFAULT '',
    read BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

- [ ] **Step 3: Create domain models**

Create `internal/models/models.go`:

```go
package models

import "time"

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Candidate struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	ResumeURL string    `json:"resume_url"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type InterviewLoop struct {
	ID            int64     `json:"id"`
	CandidateID   int64     `json:"candidate_id"`
	Status        string    `json:"status"`
	FinalDecision *string   `json:"final_decision"`
	DebriefNotes  *string   `json:"debrief_notes"`
	CreatedBy     int64     `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

type Interview struct {
	ID                  int64     `json:"id"`
	LoopID              int64     `json:"loop_id"`
	InterviewerID       int64     `json:"interviewer_id"`
	FocusArea           string    `json:"focus_area"`
	ScheduledAt         time.Time `json:"scheduled_at"`
	VideoLink           string    `json:"video_link"`
	NotesForInterviewer string    `json:"notes_for_interviewer"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
}

type Feedback struct {
	ID                   int64              `json:"id"`
	InterviewID          int64              `json:"interview_id"`
	Recommendation       string             `json:"recommendation"`
	RecommendationReason string             `json:"recommendation_reason"`
	FreeFormNotes        string             `json:"free_form_notes"`
	SubmittedAt          time.Time          `json:"submitted_at"`
	CompetencyRatings    []CompetencyRating `json:"competency_ratings,omitempty"`
}

type Competency struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	RatingType  string    `json:"rating_type"`
	RatingsJSON string    `json:"ratings_json"`
	CreatedAt   time.Time `json:"created_at"`
}

type CompetencyRating struct {
	ID           int64  `json:"id"`
	FeedbackID   int64  `json:"feedback_id"`
	CompetencyID int64  `json:"competency_id"`
	RatingValue  string `json:"rating_value"`
}

type Notification struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Message   string    `json:"message"`
	Link      string    `json:"link"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

// LoopDetail is the expanded view returned by GET /api/loops/:id.
type LoopDetail struct {
	InterviewLoop
	Candidate  Candidate              `json:"candidate"`
	Interviews []InterviewWithFeedback `json:"interviews"`
}

type InterviewWithFeedback struct {
	Interview
	InterviewerName string    `json:"interviewer_name"`
	Feedback        *Feedback `json:"feedback,omitempty"`
}
```

- [ ] **Step 4: Create store initialization and migration runner**

Create `internal/store/store.go`:

```go
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set FK: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	// Find migrations directory relative to this source file so tests work too.
	_, thisFile, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		sql, err := os.ReadFile(filepath.Join(migrationsDir, e.Name()))
		if err != nil {
			return fmt.Errorf("read %s: %w", e.Name(), err)
		}
		if _, err := s.db.Exec(string(sql)); err != nil {
			return fmt.Errorf("exec %s: %w", e.Name(), err)
		}
	}
	return nil
}
```

- [ ] **Step 5: Create test helper**

Create `internal/store/store_test.go`:

```go
package store

import "testing"

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("newTestStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
```

- [ ] **Step 6: Install Go dependencies and verify**

```bash
cd /home/zach/code/hire
go get modernc.org/sqlite
go get github.com/go-chi/chi/v5
go get github.com/go-chi/cors
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto
go build ./...
```

Expected: no errors.

- [ ] **Step 7: Run store test to verify migration works**

```bash
cd /home/zach/code/hire
go test ./internal/store/ -v -run TestNothing 2>&1 | head -5
```

This just verifies the package compiles and the test helper works. There are no test functions yet, so output should show `testing: warning: no tests to run`.

- [ ] **Step 8: Commit**

```bash
git init
git add go.mod go.sum migrations/ internal/models/ internal/store/
git commit -m "feat: project scaffolding with schema, models, and store init"
```

---

### Task 2: User store with tests

**Files:**
- Create: `internal/store/users.go`, `internal/store/users_test.go`

- [ ] **Step 1: Write the user store tests**

Create `internal/store/users_test.go`:

```go
package store

import (
	"hire/internal/models"
	"testing"
)

func TestCreateAndGetUser(t *testing.T) {
	s := newTestStore(t)
	u := &models.User{Email: "alice@example.com", Name: "Alice", PasswordHash: "hash123", Role: "interviewer"}
	if err := s.CreateUser(u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.ID == 0 {
		t.Fatal("expected ID to be set")
	}

	got, err := s.GetUserByID(u.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.Email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", got.Email)
	}
	if got.Role != "interviewer" {
		t.Errorf("role = %q, want interviewer", got.Role)
	}
}

func TestGetUserByEmail(t *testing.T) {
	s := newTestStore(t)
	u := &models.User{Email: "bob@example.com", Name: "Bob", PasswordHash: "hash", Role: "scheduler"}
	s.CreateUser(u)

	got, err := s.GetUserByEmail("bob@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if got.Name != "Bob" {
		t.Errorf("name = %q, want Bob", got.Name)
	}
}

func TestListUsers(t *testing.T) {
	s := newTestStore(t)
	s.CreateUser(&models.User{Email: "a@test.com", Name: "A", PasswordHash: "h", Role: "admin"})
	s.CreateUser(&models.User{Email: "b@test.com", Name: "B", PasswordHash: "h", Role: "interviewer"})

	users, err := s.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("got %d users, want 2", len(users))
	}
}

func TestUpdateUser(t *testing.T) {
	s := newTestStore(t)
	u := &models.User{Email: "c@test.com", Name: "C", PasswordHash: "h", Role: "interviewer"}
	s.CreateUser(u)

	u.Name = "Charlie"
	u.Role = "scheduler"
	if err := s.UpdateUser(u); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	got, _ := s.GetUserByID(u.ID)
	if got.Name != "Charlie" || got.Role != "scheduler" {
		t.Errorf("got name=%q role=%q, want Charlie scheduler", got.Name, got.Role)
	}
}

func TestDeleteUser(t *testing.T) {
	s := newTestStore(t)
	u := &models.User{Email: "d@test.com", Name: "D", PasswordHash: "h", Role: "admin"}
	s.CreateUser(u)

	if err := s.DeleteUser(u.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	_, err := s.GetUserByID(u.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/store/ -v -run TestCreate
```

Expected: compilation error — `s.CreateUser` undefined.

- [ ] **Step 3: Implement the user store**

Create `internal/store/users.go`:

```go
package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateUser(u *models.User) error {
	res, err := s.db.Exec(
		`INSERT INTO users (email, name, password_hash, role) VALUES (?, ?, ?, ?)`,
		u.Email, u.Name, u.PasswordHash, u.Role,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	u.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) GetUserByID(id int64) (*models.User, error) {
	return s.scanUser(s.db.QueryRow(
		`SELECT id, email, name, password_hash, role, created_at FROM users WHERE id = ?`, id,
	))
}

func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	return s.scanUser(s.db.QueryRow(
		`SELECT id, email, name, password_hash, role, created_at FROM users WHERE email = ?`, email,
	))
}

func (s *Store) ListUsers() ([]*models.User, error) {
	rows, err := s.db.Query(`SELECT id, email, name, password_hash, role, created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	var users []*models.User
	for rows.Next() {
		u, err := s.scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) UpdateUser(u *models.User) error {
	_, err := s.db.Exec(
		`UPDATE users SET email = ?, name = ?, password_hash = ?, role = ? WHERE id = ?`,
		u.Email, u.Name, u.PasswordHash, u.Role, u.ID,
	)
	return err
}

func (s *Store) DeleteUser(id int64) error {
	_, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func (s *Store) scanUser(row scanner) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/store/ -v -run TestUser -run TestCreate -run TestGet -run TestList -run TestUpdate -run TestDelete
```

Expected: all 5 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/store/users.go internal/store/users_test.go
git commit -m "feat: user store CRUD with tests"
```

---

### Task 3: Auth system (JWT, password hashing, middleware, login handler)

**Files:**
- Create: `internal/api/handler.go`, `internal/api/middleware.go`, `internal/api/auth.go`, `internal/api/auth_test.go`

- [ ] **Step 1: Create the handler struct and JSON helpers**

Create `internal/api/handler.go`:

```go
package api

import (
	"encoding/json"
	"hire/internal/store"
	"net/http"
)

type Handler struct {
	store     *store.Store
	jwtSecret []byte
}

func NewHandler(s *store.Store, jwtSecret string) *Handler {
	return &Handler{store: s, jwtSecret: []byte(jwtSecret)}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
```

- [ ] **Step 2: Create auth middleware**

Create `internal/api/middleware.go`:

```go
package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "user_id"
const userRoleKey contextKey = "user_role"

func UserID(ctx context.Context) int64 {
	v, _ := ctx.Value(userIDKey).(int64)
	return v
}

func UserRole(ctx context.Context) string {
	v, _ := ctx.Value(userRoleKey).(string)
	return v
}

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing token")
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			return h.jwtSecret, nil
		}, jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !token.Valid {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid claims")
			return
		}

		uid, _ := claims.GetSubject()
		role, _ := claims["role"].(string)

		var userID int64
		for _, c := range uid {
			userID = userID*10 + int64(c-'0')
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, userRoleKey, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !allowed[UserRole(r.Context())] {
				writeError(w, http.StatusForbidden, "insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 3: Create login handler**

Create `internal/api/auth.go`:

```go
package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (h *Handler) generateToken(userID int64, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  fmt.Sprintf("%d", userID),
		"role": role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.store.GetUserByEmail(req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.generateToken(user.ID, user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token generation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  user,
	})
}
```

- [ ] **Step 4: Write auth tests**

Create `internal/api/auth_test.go`:

```go
package api

import (
	"bytes"
	"encoding/json"
	"hire/internal/models"
	"hire/internal/store"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newTestHandler(t *testing.T) (*Handler, *store.Store) {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("newTestHandler: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	h := NewHandler(s, "test-secret")
	return h, s
}

func TestLoginSuccess(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("password123")
	s.CreateUser(&models.User{Email: "test@test.com", Name: "Test", PasswordHash: hash, Role: "interviewer"})

	r := chi.NewRouter()
	r.Post("/api/auth/login", h.Login)

	body, _ := json.Marshal(map[string]string{"email": "test@test.com", "password": "password123"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] == nil || resp["token"] == "" {
		t.Fatal("expected token in response")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("correct")
	s.CreateUser(&models.User{Email: "test@test.com", Name: "Test", PasswordHash: hash, Role: "interviewer"})

	r := chi.NewRouter()
	r.Post("/api/auth/login", h.Login)

	body, _ := json.Marshal(map[string]string{"email": "test@test.com", "password": "wrong"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestAuthMiddleware(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "a@a.com", Name: "A", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(u)

	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"user_id": UserID(r.Context()),
			"role":    UserRole(r.Context()),
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["role"] != "scheduler" {
		t.Errorf("role = %v, want scheduler", resp["role"])
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/api/ -v -run TestLogin -run TestAuth
```

Expected: all 3 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/api/handler.go internal/api/middleware.go internal/api/auth.go internal/api/auth_test.go
git commit -m "feat: auth system with JWT, bcrypt, and middleware"
```

---

### Task 4: Candidate store and API with tests

**Files:**
- Create: `internal/store/candidates.go`, `internal/store/candidates_test.go`, `internal/api/candidates.go`, `internal/api/candidates_test.go`

- [ ] **Step 1: Write candidate store tests**

Create `internal/store/candidates_test.go`:

```go
package store

import (
	"hire/internal/models"
	"testing"
)

func TestCreateAndGetCandidate(t *testing.T) {
	s := newTestStore(t)
	c := &models.Candidate{Name: "Jane Doe", Email: "jane@example.com", ResumeURL: "https://resume.com/jane", Status: "active"}
	if err := s.CreateCandidate(c); err != nil {
		t.Fatalf("CreateCandidate: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("expected ID to be set")
	}

	got, err := s.GetCandidate(c.ID)
	if err != nil {
		t.Fatalf("GetCandidate: %v", err)
	}
	if got.Name != "Jane Doe" || got.Email != "jane@example.com" {
		t.Errorf("got %+v", got)
	}
}

func TestListCandidates(t *testing.T) {
	s := newTestStore(t)
	s.CreateCandidate(&models.Candidate{Name: "A", Email: "a@a.com", Status: "active"})
	s.CreateCandidate(&models.Candidate{Name: "B", Email: "b@b.com", Status: "active"})

	list, err := s.ListCandidates()
	if err != nil {
		t.Fatalf("ListCandidates: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d, want 2", len(list))
	}
}

func TestUpdateCandidate(t *testing.T) {
	s := newTestStore(t)
	c := &models.Candidate{Name: "X", Email: "x@x.com", Status: "active"}
	s.CreateCandidate(c)
	c.Status = "hired"
	if err := s.UpdateCandidate(c); err != nil {
		t.Fatalf("UpdateCandidate: %v", err)
	}
	got, _ := s.GetCandidate(c.ID)
	if got.Status != "hired" {
		t.Errorf("status = %q, want hired", got.Status)
	}
}

func TestDeleteCandidate(t *testing.T) {
	s := newTestStore(t)
	c := &models.Candidate{Name: "Y", Email: "y@y.com", Status: "active"}
	s.CreateCandidate(c)
	if err := s.DeleteCandidate(c.ID); err != nil {
		t.Fatalf("DeleteCandidate: %v", err)
	}
	_, err := s.GetCandidate(c.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
```

- [ ] **Step 2: Implement candidate store**

Create `internal/store/candidates.go`:

```go
package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateCandidate(c *models.Candidate) error {
	res, err := s.db.Exec(
		`INSERT INTO candidates (name, email, resume_url, status) VALUES (?, ?, ?, ?)`,
		c.Name, c.Email, c.ResumeURL, c.Status,
	)
	if err != nil {
		return fmt.Errorf("insert candidate: %w", err)
	}
	c.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) GetCandidate(id int64) (*models.Candidate, error) {
	var c models.Candidate
	err := s.db.QueryRow(
		`SELECT id, name, email, resume_url, status, created_at FROM candidates WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.Email, &c.ResumeURL, &c.Status, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("candidate not found")
	}
	return &c, err
}

func (s *Store) ListCandidates() ([]*models.Candidate, error) {
	rows, err := s.db.Query(`SELECT id, name, email, resume_url, status, created_at FROM candidates ORDER BY id DESC`)
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
		`UPDATE candidates SET name = ?, email = ?, resume_url = ?, status = ? WHERE id = ?`,
		c.Name, c.Email, c.ResumeURL, c.Status, c.ID,
	)
	return err
}

func (s *Store) DeleteCandidate(id int64) error {
	_, err := s.db.Exec(`DELETE FROM candidates WHERE id = ?`, id)
	return err
}
```

- [ ] **Step 3: Run store tests**

```bash
go test ./internal/store/ -v -run TestCandidate -run TestList -run TestUpdate -run TestDelete
```

Expected: all candidate tests PASS.

- [ ] **Step 4: Write candidate API handler tests**

Create `internal/api/candidates_test.go`:

```go
package api

import (
	"bytes"
	"encoding/json"
	"hire/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestCandidateCRUD(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(u)
	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Route("/api/candidates", func(r chi.Router) {
		r.Get("/", h.ListCandidates)
		r.Post("/", h.CreateCandidate)
		r.Get("/{id}", h.GetCandidate)
		r.Put("/{id}", h.UpdateCandidate)
		r.Delete("/{id}", h.DeleteCandidate)
	})

	// Create
	body, _ := json.Marshal(map[string]string{"name": "Jane", "email": "jane@test.com", "resume_url": "", "status": "active"})
	req := httptest.NewRequest("POST", "/api/candidates", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201; body: %s", w.Code, w.Body.String())
	}

	var created models.Candidate
	json.Unmarshal(w.Body.Bytes(), &created)
	if created.ID == 0 {
		t.Fatal("expected ID in response")
	}

	// List
	req = httptest.NewRequest("GET", "/api/candidates", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list: status = %d", w.Code)
	}

	// Get
	req = httptest.NewRequest("GET", "/api/candidates/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("get: status = %d", w.Code)
	}
}
```

- [ ] **Step 5: Implement candidate API handlers**

Create `internal/api/candidates.go`:

```go
package api

import (
	"hire/internal/models"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateCandidate(w http.ResponseWriter, r *http.Request) {
	var c models.Candidate
	if err := readJSON(r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if c.Status == "" {
		c.Status = "active"
	}
	if err := h.store.CreateCandidate(&c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) GetCandidate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	c, err := h.store.GetCandidate(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "candidate not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) ListCandidates(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListCandidates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) UpdateCandidate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var c models.Candidate
	if err := readJSON(r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	c.ID = id
	if err := h.store.UpdateCandidate(&c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) DeleteCandidate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteCandidate(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 6: Run all tests**

```bash
go test ./internal/... -v
```

Expected: all tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/store/candidates.go internal/store/candidates_test.go internal/api/candidates.go internal/api/candidates_test.go
git commit -m "feat: candidate store and API handlers with tests"
```

---

### Task 5: Competency store and API with tests

**Files:**
- Create: `internal/store/competencies.go`, `internal/store/competencies_test.go`, `internal/api/competencies.go`, `internal/api/competencies_test.go`

- [ ] **Step 1: Write competency store tests**

Create `internal/store/competencies_test.go`:

```go
package store

import (
	"hire/internal/models"
	"testing"
)

func TestCreateAndGetCompetency(t *testing.T) {
	s := newTestStore(t)
	c := &models.Competency{Name: "Problem Solving", RatingType: "levels", RatingsJSON: `["Learning","Owning","Advising"]`}
	if err := s.CreateCompetency(c); err != nil {
		t.Fatalf("CreateCompetency: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("expected ID")
	}
	got, err := s.GetCompetency(c.ID)
	if err != nil {
		t.Fatalf("GetCompetency: %v", err)
	}
	if got.Name != "Problem Solving" || got.RatingType != "levels" {
		t.Errorf("got %+v", got)
	}
}

func TestListCompetencies(t *testing.T) {
	s := newTestStore(t)
	s.CreateCompetency(&models.Competency{Name: "A", RatingType: "levels", RatingsJSON: `["X"]`})
	s.CreateCompetency(&models.Competency{Name: "B", RatingType: "stars", RatingsJSON: `{"min":1,"max":5}`})
	list, err := s.ListCompetencies()
	if err != nil {
		t.Fatalf("ListCompetencies: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d, want 2", len(list))
	}
}

func TestDeleteCompetency(t *testing.T) {
	s := newTestStore(t)
	c := &models.Competency{Name: "C", RatingType: "stars", RatingsJSON: `{"min":1,"max":5}`}
	s.CreateCompetency(c)
	if err := s.DeleteCompetency(c.ID); err != nil {
		t.Fatalf("DeleteCompetency: %v", err)
	}
	_, err := s.GetCompetency(c.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
```

- [ ] **Step 2: Implement competency store**

Create `internal/store/competencies.go`:

```go
package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateCompetency(c *models.Competency) error {
	res, err := s.db.Exec(
		`INSERT INTO competencies (name, rating_type, ratings_json) VALUES (?, ?, ?)`,
		c.Name, c.RatingType, c.RatingsJSON,
	)
	if err != nil {
		return fmt.Errorf("insert competency: %w", err)
	}
	c.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) GetCompetency(id int64) (*models.Competency, error) {
	var c models.Competency
	err := s.db.QueryRow(
		`SELECT id, name, rating_type, ratings_json, created_at FROM competencies WHERE id = ?`, id,
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
		`UPDATE competencies SET name = ?, rating_type = ?, ratings_json = ? WHERE id = ?`,
		c.Name, c.RatingType, c.RatingsJSON, c.ID,
	)
	return err
}

func (s *Store) DeleteCompetency(id int64) error {
	_, err := s.db.Exec(`DELETE FROM competencies WHERE id = ?`, id)
	return err
}
```

- [ ] **Step 3: Run store tests**

```bash
go test ./internal/store/ -v -run TestCompetenc
```

Expected: all competency tests PASS.

- [ ] **Step 4: Implement competency API handlers**

Create `internal/api/competencies.go`:

```go
package api

import (
	"hire/internal/models"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateCompetency(w http.ResponseWriter, r *http.Request) {
	var c models.Competency
	if err := readJSON(r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.store.CreateCompetency(&c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) GetCompetency(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	c, err := h.store.GetCompetency(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "competency not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) ListCompetencies(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListCompetencies()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) UpdateCompetency(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var c models.Competency
	if err := readJSON(r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	c.ID = id
	if err := h.store.UpdateCompetency(&c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) DeleteCompetency(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteCompetency(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 5: Write competency API test**

Create `internal/api/competencies_test.go`:

```go
package api

import (
	"bytes"
	"encoding/json"
	"hire/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestCompetencyCRUD(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "admin@test.com", Name: "Admin", PasswordHash: hash, Role: "admin"}
	s.CreateUser(u)
	token, _ := h.generateToken(u.ID, u.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Route("/api/competencies", func(r chi.Router) {
		r.Get("/", h.ListCompetencies)
		r.Post("/", h.CreateCompetency)
		r.Put("/{id}", h.UpdateCompetency)
		r.Delete("/{id}", h.DeleteCompetency)
	})

	// Create
	body, _ := json.Marshal(map[string]string{
		"name": "Problem Solving", "rating_type": "levels",
		"ratings_json": `["Learning","Owning","Advising"]`,
	})
	req := httptest.NewRequest("POST", "/api/competencies", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d; body: %s", w.Code, w.Body.String())
	}

	// List
	req = httptest.NewRequest("GET", "/api/competencies", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list: status = %d", w.Code)
	}
	var list []models.Competency
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Fatalf("got %d competencies, want 1", len(list))
	}
}
```

- [ ] **Step 6: Run all tests**

```bash
go test ./internal/... -v
```

Expected: all tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/store/competencies.go internal/store/competencies_test.go internal/api/competencies.go internal/api/competencies_test.go
git commit -m "feat: competency store and API handlers with tests"
```

---

### Task 6: Interview loop store and API with tests

**Files:**
- Create: `internal/store/loops.go`, `internal/store/loops_test.go`, `internal/api/loops.go`, `internal/api/loops_test.go`

- [ ] **Step 1: Write loop store tests**

Create `internal/store/loops_test.go`:

```go
package store

import (
	"hire/internal/models"
	"testing"
)

func createTestUserAndCandidate(t *testing.T, s *Store) (*models.User, *models.Candidate) {
	t.Helper()
	u := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: "h", Role: "scheduler"}
	s.CreateUser(u)
	c := &models.Candidate{Name: "Candidate", Email: "c@test.com", Status: "active"}
	s.CreateCandidate(c)
	return u, c
}

func TestCreateAndGetLoop(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)

	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: u.ID}
	if err := s.CreateLoop(loop); err != nil {
		t.Fatalf("CreateLoop: %v", err)
	}
	if loop.ID == 0 {
		t.Fatal("expected ID")
	}

	got, err := s.GetLoop(loop.ID)
	if err != nil {
		t.Fatalf("GetLoop: %v", err)
	}
	if got.CandidateID != c.ID || got.Status != "scheduling" {
		t.Errorf("got %+v", got)
	}
}

func TestListLoops(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)
	s.CreateLoop(&models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: u.ID})
	s.CreateLoop(&models.InterviewLoop{CandidateID: c.ID, Status: "active", CreatedBy: u.ID})

	loops, err := s.ListLoops(nil, nil)
	if err != nil {
		t.Fatalf("ListLoops: %v", err)
	}
	if len(loops) != 2 {
		t.Fatalf("got %d, want 2", len(loops))
	}

	// Filter by candidate
	loops, err = s.ListLoops(&c.ID, nil)
	if err != nil {
		t.Fatalf("ListLoops filtered: %v", err)
	}
	if len(loops) != 2 {
		t.Fatalf("got %d, want 2", len(loops))
	}
}

func TestUpdateLoop(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: u.ID}
	s.CreateLoop(loop)

	decision := "hire"
	loop.Status = "complete"
	loop.FinalDecision = &decision
	if err := s.UpdateLoop(loop); err != nil {
		t.Fatalf("UpdateLoop: %v", err)
	}
	got, _ := s.GetLoop(loop.ID)
	if got.Status != "complete" || *got.FinalDecision != "hire" {
		t.Errorf("got status=%q decision=%v", got.Status, got.FinalDecision)
	}
}
```

- [ ] **Step 2: Implement loop store**

Create `internal/store/loops.go`:

```go
package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateLoop(l *models.InterviewLoop) error {
	res, err := s.db.Exec(
		`INSERT INTO interview_loops (candidate_id, status, created_by) VALUES (?, ?, ?)`,
		l.CandidateID, l.Status, l.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("insert loop: %w", err)
	}
	l.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) GetLoop(id int64) (*models.InterviewLoop, error) {
	var l models.InterviewLoop
	err := s.db.QueryRow(
		`SELECT id, candidate_id, status, final_decision, debrief_notes, created_by, created_at
		 FROM interview_loops WHERE id = ?`, id,
	).Scan(&l.ID, &l.CandidateID, &l.Status, &l.FinalDecision, &l.DebriefNotes, &l.CreatedBy, &l.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("loop not found")
	}
	return &l, err
}

func (s *Store) ListLoops(candidateID *int64, status *string) ([]*models.InterviewLoop, error) {
	query := `SELECT id, candidate_id, status, final_decision, debrief_notes, created_by, created_at FROM interview_loops WHERE 1=1`
	var args []any
	if candidateID != nil {
		query += ` AND candidate_id = ?`
		args = append(args, *candidateID)
	}
	if status != nil {
		query += ` AND status = ?`
		args = append(args, *status)
	}
	query += ` ORDER BY id DESC`

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
		`UPDATE interview_loops SET status = ?, final_decision = ?, debrief_notes = ? WHERE id = ?`,
		l.Status, l.FinalDecision, l.DebriefNotes, l.ID,
	)
	return err
}

func (s *Store) DeleteLoop(id int64) error {
	_, err := s.db.Exec(`DELETE FROM interview_loops WHERE id = ?`, id)
	return err
}

// GetLoopDetail returns a loop with its candidate, interviews, and (optionally) feedback.
func (s *Store) GetLoopDetail(id int64) (*models.LoopDetail, error) {
	loop, err := s.GetLoop(id)
	if err != nil {
		return nil, err
	}
	candidate, err := s.GetCandidate(loop.CandidateID)
	if err != nil {
		return nil, fmt.Errorf("get candidate for loop: %w", err)
	}
	interviews, err := s.ListInterviewsByLoop(id)
	if err != nil {
		return nil, fmt.Errorf("list interviews for loop: %w", err)
	}

	detail := &models.LoopDetail{
		InterviewLoop: *loop,
		Candidate:     *candidate,
	}
	for _, iv := range interviews {
		iwf := models.InterviewWithFeedback{Interview: *iv}
		// Get interviewer name
		interviewer, err := s.GetUserByID(iv.InterviewerID)
		if err == nil {
			iwf.InterviewerName = interviewer.Name
		}
		// Get feedback if exists
		fb, err := s.GetFeedbackByInterview(iv.ID)
		if err == nil {
			iwf.Feedback = fb
		}
		detail.Interviews = append(detail.Interviews, iwf)
	}
	return detail, nil
}
```

Note: `ListInterviewsByLoop` and `GetFeedbackByInterview` will be implemented in Tasks 7 and 8. This file will compile after those tasks are done. If building incrementally, stub them first.

- [ ] **Step 3: Implement loop API handlers**

Create `internal/api/loops.go`:

```go
package api

import (
	"hire/internal/models"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateLoop(w http.ResponseWriter, r *http.Request) {
	var l models.InterviewLoop
	if err := readJSON(r, &l); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	l.CreatedBy = UserID(r.Context())
	if l.Status == "" {
		l.Status = "scheduling"
	}
	if err := h.store.CreateLoop(&l); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

func (h *Handler) GetLoopDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	detail, err := h.store.GetLoopDetail(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "loop not found")
		return
	}

	// Enforce feedback visibility rule for interviewers
	role := UserRole(r.Context())
	userID := UserID(r.Context())
	if role == "interviewer" {
		hasSubmitted := h.store.HasUserSubmittedFeedbackForLoop(detail.ID, userID)
		if !hasSubmitted {
			// Strip feedback from all interviews except the user's own
			for i := range detail.Interviews {
				if detail.Interviews[i].InterviewerID != userID {
					detail.Interviews[i].Feedback = nil
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) ListLoops(w http.ResponseWriter, r *http.Request) {
	var candidateID *int64
	var status *string
	if v := r.URL.Query().Get("candidate_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			candidateID = &id
		}
	}
	if v := r.URL.Query().Get("status"); v != "" {
		status = &v
	}

	loops, err := h.store.ListLoops(candidateID, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, loops)
}

func (h *Handler) UpdateLoop(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetLoop(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "loop not found")
		return
	}
	var updates models.InterviewLoop
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	existing.Status = updates.Status
	existing.FinalDecision = updates.FinalDecision
	existing.DebriefNotes = updates.DebriefNotes
	if err := h.store.UpdateLoop(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteLoop(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteLoop(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Write loop API test**

Create `internal/api/loops_test.go`:

```go
package api

import (
	"bytes"
	"encoding/json"
	"hire/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestLoopCRUD(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	u := &models.User{Email: "sched@test.com", Name: "Sched", PasswordHash: hash, Role: "scheduler"}
	s.CreateUser(u)
	token, _ := h.generateToken(u.ID, u.Role)

	c := &models.Candidate{Name: "Jane", Email: "jane@test.com", Status: "active"}
	s.CreateCandidate(c)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Route("/api/loops", func(r chi.Router) {
		r.Get("/", h.ListLoops)
		r.Post("/", h.CreateLoop)
		r.Get("/{id}", h.GetLoopDetail)
		r.Put("/{id}", h.UpdateLoop)
		r.Delete("/{id}", h.DeleteLoop)
	})

	// Create
	body, _ := json.Marshal(map[string]any{"candidate_id": c.ID})
	req := httptest.NewRequest("POST", "/api/loops", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d; body: %s", w.Code, w.Body.String())
	}

	// List
	req = httptest.NewRequest("GET", "/api/loops", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list: status = %d", w.Code)
	}
}
```

- [ ] **Step 5: Commit** (tests may not pass yet — depends on interview/feedback store stubs)

```bash
git add internal/store/loops.go internal/store/loops_test.go internal/api/loops.go internal/api/loops_test.go
git commit -m "feat: interview loop store and API handlers with tests"
```

---

### Task 7: Interview store and API with tests

**Files:**
- Create: `internal/store/interviews.go`, `internal/store/interviews_test.go`, `internal/api/interviews.go`, `internal/api/interviews_test.go`

- [ ] **Step 1: Write interview store tests**

Create `internal/store/interviews_test.go`:

```go
package store

import (
	"hire/internal/models"
	"testing"
	"time"
)

func TestCreateAndGetInterview(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: u.ID}
	s.CreateLoop(loop)

	iv := &models.Interview{
		LoopID:              loop.ID,
		InterviewerID:       u.ID,
		FocusArea:           "coding",
		ScheduledAt:         time.Now().Add(24 * time.Hour),
		VideoLink:           "https://meet.example.com/abc",
		NotesForInterviewer: "Focus on algorithms",
		Status:              "pending",
	}
	if err := s.CreateInterview(iv); err != nil {
		t.Fatalf("CreateInterview: %v", err)
	}
	if iv.ID == 0 {
		t.Fatal("expected ID")
	}

	got, err := s.GetInterview(iv.ID)
	if err != nil {
		t.Fatalf("GetInterview: %v", err)
	}
	if got.FocusArea != "coding" || got.Status != "pending" {
		t.Errorf("got %+v", got)
	}
}

func TestListInterviewsByLoop(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: u.ID}
	s.CreateLoop(loop)

	s.CreateInterview(&models.Interview{LoopID: loop.ID, InterviewerID: u.ID, FocusArea: "coding", ScheduledAt: time.Now(), Status: "pending"})
	s.CreateInterview(&models.Interview{LoopID: loop.ID, InterviewerID: u.ID, FocusArea: "design", ScheduledAt: time.Now(), Status: "pending"})

	list, err := s.ListInterviewsByLoop(loop.ID)
	if err != nil {
		t.Fatalf("ListInterviewsByLoop: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d, want 2", len(list))
	}
}

func TestListInterviewsByUser(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "scheduling", CreatedBy: u.ID}
	s.CreateLoop(loop)

	s.CreateInterview(&models.Interview{LoopID: loop.ID, InterviewerID: u.ID, FocusArea: "coding", ScheduledAt: time.Now(), Status: "pending"})

	list, err := s.ListInterviewsByUser(u.ID)
	if err != nil {
		t.Fatalf("ListInterviewsByUser: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d, want 1", len(list))
	}
}
```

- [ ] **Step 2: Implement interview store**

Create `internal/store/interviews.go`:

```go
package store

import (
	"database/sql"
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateInterview(iv *models.Interview) error {
	res, err := s.db.Exec(
		`INSERT INTO interviews (loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		iv.LoopID, iv.InterviewerID, iv.FocusArea, iv.ScheduledAt, iv.VideoLink, iv.NotesForInterviewer, iv.Status,
	)
	if err != nil {
		return fmt.Errorf("insert interview: %w", err)
	}
	iv.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) GetInterview(id int64) (*models.Interview, error) {
	var iv models.Interview
	err := s.db.QueryRow(
		`SELECT id, loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at
		 FROM interviews WHERE id = ?`, id,
	).Scan(&iv.ID, &iv.LoopID, &iv.InterviewerID, &iv.FocusArea, &iv.ScheduledAt, &iv.VideoLink,
		&iv.NotesForInterviewer, &iv.Status, &iv.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("interview not found")
	}
	return &iv, err
}

func (s *Store) ListInterviewsByLoop(loopID int64) ([]*models.Interview, error) {
	return s.queryInterviews(`SELECT id, loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at
		FROM interviews WHERE loop_id = ? ORDER BY scheduled_at`, loopID)
}

func (s *Store) ListInterviewsByUser(userID int64) ([]*models.Interview, error) {
	return s.queryInterviews(`SELECT id, loop_id, interviewer_id, focus_area, scheduled_at, video_link, notes_for_interviewer, status, created_at
		FROM interviews WHERE interviewer_id = ? ORDER BY scheduled_at DESC`, userID)
}

func (s *Store) UpdateInterview(iv *models.Interview) error {
	_, err := s.db.Exec(
		`UPDATE interviews SET interviewer_id = ?, focus_area = ?, scheduled_at = ?, video_link = ?, notes_for_interviewer = ?, status = ?
		 WHERE id = ?`,
		iv.InterviewerID, iv.FocusArea, iv.ScheduledAt, iv.VideoLink, iv.NotesForInterviewer, iv.Status, iv.ID,
	)
	return err
}

func (s *Store) DeleteInterview(id int64) error {
	_, err := s.db.Exec(`DELETE FROM interviews WHERE id = ?`, id)
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

- [ ] **Step 3: Implement interview API handlers**

Create `internal/api/interviews.go`:

```go
package api

import (
	"hire/internal/models"
	"hire/internal/notify"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateInterview(w http.ResponseWriter, r *http.Request) {
	loopID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid loop id")
		return
	}
	var iv models.Interview
	if err := readJSON(r, &iv); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	iv.LoopID = loopID
	if iv.Status == "" {
		iv.Status = "pending"
	}
	if err := h.store.CreateInterview(&iv); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Notify the assigned interviewer
	notify.InterviewAssigned(h.store, iv.InterviewerID, iv.ID, iv.FocusArea)

	writeJSON(w, http.StatusCreated, iv)
}

func (h *Handler) UpdateInterview(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetInterview(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "interview not found")
		return
	}
	var updates models.Interview
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	existing.InterviewerID = updates.InterviewerID
	existing.FocusArea = updates.FocusArea
	existing.ScheduledAt = updates.ScheduledAt
	existing.VideoLink = updates.VideoLink
	existing.NotesForInterviewer = updates.NotesForInterviewer
	if err := h.store.UpdateInterview(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteInterview(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteInterview(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListMyInterviews(w http.ResponseWriter, r *http.Request) {
	userID := UserID(r.Context())
	list, err := h.store.ListInterviewsByUser(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}
```

- [ ] **Step 4: Run all tests**

```bash
go test ./internal/... -v
```

Expected: all tests PASS (the loop detail test may need the feedback store — if compilation fails, proceed to Task 8 first, then re-run).

- [ ] **Step 5: Commit**

```bash
git add internal/store/interviews.go internal/store/interviews_test.go internal/api/interviews.go internal/api/interviews_test.go
git commit -m "feat: interview store and API handlers with tests"
```

---

### Task 8: Feedback store and API with tests (including visibility rule)

**Files:**
- Create: `internal/store/feedback.go`, `internal/store/feedback_test.go`, `internal/api/feedback.go`, `internal/api/feedback_test.go`

- [ ] **Step 1: Write feedback store tests**

Create `internal/store/feedback_test.go`:

```go
package store

import (
	"hire/internal/models"
	"testing"
	"time"
)

func TestCreateAndGetFeedback(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "active", CreatedBy: u.ID}
	s.CreateLoop(loop)

	comp := &models.Competency{Name: "Coding", RatingType: "levels", RatingsJSON: `["Learning","Owning","Advising"]`}
	s.CreateCompetency(comp)

	iv := &models.Interview{LoopID: loop.ID, InterviewerID: u.ID, FocusArea: "coding", ScheduledAt: time.Now(), Status: "pending"}
	s.CreateInterview(iv)

	fb := &models.Feedback{
		InterviewID:          iv.ID,
		Recommendation:       "hire",
		RecommendationReason: "Strong coder",
		FreeFormNotes:        "Good performance",
		CompetencyRatings: []models.CompetencyRating{
			{CompetencyID: comp.ID, RatingValue: "Owning"},
		},
	}
	if err := s.CreateFeedback(fb); err != nil {
		t.Fatalf("CreateFeedback: %v", err)
	}
	if fb.ID == 0 {
		t.Fatal("expected ID")
	}

	// Interview should be marked complete
	updatedIV, _ := s.GetInterview(iv.ID)
	if updatedIV.Status != "complete" {
		t.Errorf("interview status = %q, want complete", updatedIV.Status)
	}

	// Get feedback with ratings
	got, err := s.GetFeedbackByInterview(iv.ID)
	if err != nil {
		t.Fatalf("GetFeedbackByInterview: %v", err)
	}
	if got.Recommendation != "hire" {
		t.Errorf("recommendation = %q, want hire", got.Recommendation)
	}
	if len(got.CompetencyRatings) != 1 {
		t.Fatalf("got %d ratings, want 1", len(got.CompetencyRatings))
	}
	if got.CompetencyRatings[0].RatingValue != "Owning" {
		t.Errorf("rating = %q, want Owning", got.CompetencyRatings[0].RatingValue)
	}
}

func TestHasUserSubmittedFeedbackForLoop(t *testing.T) {
	s := newTestStore(t)
	u, c := createTestUserAndCandidate(t, s)
	loop := &models.InterviewLoop{CandidateID: c.ID, Status: "active", CreatedBy: u.ID}
	s.CreateLoop(loop)

	iv := &models.Interview{LoopID: loop.ID, InterviewerID: u.ID, FocusArea: "coding", ScheduledAt: time.Now(), Status: "pending"}
	s.CreateInterview(iv)

	if s.HasUserSubmittedFeedbackForLoop(loop.ID, u.ID) {
		t.Fatal("should not have submitted feedback yet")
	}

	s.CreateFeedback(&models.Feedback{
		InterviewID:    iv.ID,
		Recommendation: "hire",
	})

	if !s.HasUserSubmittedFeedbackForLoop(loop.ID, u.ID) {
		t.Fatal("should have submitted feedback")
	}
}
```

- [ ] **Step 2: Implement feedback store**

Create `internal/store/feedback.go`:

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
```

- [ ] **Step 3: Implement feedback API handlers**

Create `internal/api/feedback.go`:

```go
package api

import (
	"hire/internal/models"
	"hire/internal/notify"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetFeedback(w http.ResponseWriter, r *http.Request) {
	interviewID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	fb, err := h.store.GetFeedbackByInterview(interviewID)
	if err != nil {
		writeError(w, http.StatusNotFound, "feedback not found")
		return
	}
	writeJSON(w, http.StatusOK, fb)
}

func (h *Handler) CreateFeedback(w http.ResponseWriter, r *http.Request) {
	interviewID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Verify this interview belongs to the current user
	iv, err := h.store.GetInterview(interviewID)
	if err != nil {
		writeError(w, http.StatusNotFound, "interview not found")
		return
	}
	if iv.InterviewerID != UserID(r.Context()) && UserRole(r.Context()) == "interviewer" {
		writeError(w, http.StatusForbidden, "not your interview")
		return
	}

	var fb models.Feedback
	if err := readJSON(r, &fb); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	fb.InterviewID = interviewID
	if err := h.store.CreateFeedback(&fb); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Notify the scheduler who created the loop
	loop, _ := h.store.GetLoop(iv.LoopID)
	if loop != nil {
		notify.FeedbackSubmitted(h.store, loop.CreatedBy, iv.LoopID, iv.FocusArea)

		// Check if all interviews in the loop have feedback — if so, notify debrief ready
		notify.CheckDebriefReady(h.store, loop)
	}

	writeJSON(w, http.StatusCreated, fb)
}

func (h *Handler) UpdateFeedback(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetFeedback(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "feedback not found")
		return
	}
	var updates models.Feedback
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	existing.Recommendation = updates.Recommendation
	existing.RecommendationReason = updates.RecommendationReason
	existing.FreeFormNotes = updates.FreeFormNotes
	if err := h.store.UpdateFeedback(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, existing)
}
```

- [ ] **Step 4: Run all tests**

```bash
go test ./internal/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/store/feedback.go internal/store/feedback_test.go internal/api/feedback.go internal/api/feedback_test.go
git commit -m "feat: feedback store and API with visibility rule and auto-complete"
```

---

### Task 9: Notification system

**Files:**
- Create: `internal/notify/notify.go`, `internal/store/notifications.go`, `internal/store/notifications_test.go`, `internal/api/notifications.go`

- [ ] **Step 1: Write notification store tests**

Create `internal/store/notifications_test.go`:

```go
package store

import (
	"hire/internal/models"
	"testing"
)

func TestCreateAndListNotifications(t *testing.T) {
	s := newTestStore(t)
	u := &models.User{Email: "a@a.com", Name: "A", PasswordHash: "h", Role: "interviewer"}
	s.CreateUser(u)

	n := &models.Notification{UserID: u.ID, Message: "You have a new interview", Link: "/interviews/1"}
	if err := s.CreateNotification(n); err != nil {
		t.Fatalf("CreateNotification: %v", err)
	}

	list, err := s.ListNotificationsByUser(u.ID)
	if err != nil {
		t.Fatalf("ListNotificationsByUser: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d, want 1", len(list))
	}
	if list[0].Read {
		t.Error("expected unread")
	}
}

func TestMarkNotificationRead(t *testing.T) {
	s := newTestStore(t)
	u := &models.User{Email: "a@a.com", Name: "A", PasswordHash: "h", Role: "interviewer"}
	s.CreateUser(u)
	n := &models.Notification{UserID: u.ID, Message: "Test", Link: "/test"}
	s.CreateNotification(n)

	if err := s.MarkNotificationRead(n.ID); err != nil {
		t.Fatalf("MarkNotificationRead: %v", err)
	}

	list, _ := s.ListNotificationsByUser(u.ID)
	if !list[0].Read {
		t.Error("expected read")
	}
}

func TestCountUnreadNotifications(t *testing.T) {
	s := newTestStore(t)
	u := &models.User{Email: "a@a.com", Name: "A", PasswordHash: "h", Role: "interviewer"}
	s.CreateUser(u)
	s.CreateNotification(&models.Notification{UserID: u.ID, Message: "1", Link: "/"})
	s.CreateNotification(&models.Notification{UserID: u.ID, Message: "2", Link: "/"})

	count, err := s.CountUnreadNotifications(u.ID)
	if err != nil {
		t.Fatalf("CountUnread: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}
```

- [ ] **Step 2: Implement notification store**

Create `internal/store/notifications.go`:

```go
package store

import (
	"fmt"
	"hire/internal/models"
)

func (s *Store) CreateNotification(n *models.Notification) error {
	res, err := s.db.Exec(
		`INSERT INTO notifications (user_id, message, link) VALUES (?, ?, ?)`,
		n.UserID, n.Message, n.Link,
	)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	n.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) ListNotificationsByUser(userID int64) ([]*models.Notification, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, message, link, read, created_at FROM notifications WHERE user_id = ? ORDER BY created_at DESC`,
		userID,
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

func (s *Store) MarkNotificationRead(id int64) error {
	_, err := s.db.Exec(`UPDATE notifications SET read = 1 WHERE id = ?`, id)
	return err
}

func (s *Store) CountUnreadNotifications(userID int64) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read = 0`, userID).Scan(&count)
	return count, err
}
```

- [ ] **Step 3: Implement the notify helper package**

Create `internal/notify/notify.go`:

```go
package notify

import (
	"fmt"
	"hire/internal/models"
	"hire/internal/store"
)

func InterviewAssigned(s *store.Store, interviewerID, interviewID int64, focusArea string) {
	s.CreateNotification(&models.Notification{
		UserID:  interviewerID,
		Message: fmt.Sprintf("You've been assigned a %s interview", focusArea),
		Link:    fmt.Sprintf("/interviews/%d", interviewID),
	})
}

func FeedbackSubmitted(s *store.Store, schedulerID, loopID int64, focusArea string) {
	s.CreateNotification(&models.Notification{
		UserID:  schedulerID,
		Message: fmt.Sprintf("Feedback submitted for %s interview", focusArea),
		Link:    fmt.Sprintf("/loops/%d/debrief", loopID),
	})
}

func CheckDebriefReady(s *store.Store, loop *models.InterviewLoop) {
	interviews, err := s.ListInterviewsByLoop(loop.ID)
	if err != nil || len(interviews) == 0 {
		return
	}
	for _, iv := range interviews {
		if iv.Status != "complete" {
			return
		}
	}
	s.CreateNotification(&models.Notification{
		UserID:  loop.CreatedBy,
		Message: "All feedback submitted — ready for debrief",
		Link:    fmt.Sprintf("/loops/%d/debrief", loop.ID),
	})
}
```

- [ ] **Step 4: Implement notification API handlers**

Create `internal/api/notifications.go`:

```go
package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID := UserID(r.Context())
	list, err := h.store.ListNotificationsByUser(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.MarkNotificationRead(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 5: Run all tests**

```bash
go test ./internal/... -v
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/store/notifications.go internal/store/notifications_test.go internal/notify/ internal/api/notifications.go
git commit -m "feat: notification system with store, notify helpers, and API"
```

---

### Task 10: User API handlers

**Files:**
- Create: `internal/api/users.go`, `internal/api/users_test.go`

- [ ] **Step 1: Implement user API handlers**

Create `internal/api/users.go`:

```go
package api

import (
	"hire/internal/models"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type CreateUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "password hash failed")
		return
	}
	u := &models.User{Email: req.Email, Name: req.Name, PasswordHash: hash, Role: req.Role}
	if err := h.store.CreateUser(u); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, err := h.store.GetUserByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetUserByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	var req CreateUserRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	existing.Email = req.Email
	existing.Name = req.Name
	existing.Role = req.Role
	if req.Password != "" {
		hash, err := HashPassword(req.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "password hash failed")
			return
		}
		existing.PasswordHash = hash
	}
	if err := h.store.UpdateUser(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteUser(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 2: Write user API tests**

Create `internal/api/users_test.go`:

```go
package api

import (
	"bytes"
	"encoding/json"
	"hire/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestUserCRUD(t *testing.T) {
	h, s := newTestHandler(t)
	hash, _ := HashPassword("pass")
	admin := &models.User{Email: "admin@test.com", Name: "Admin", PasswordHash: hash, Role: "admin"}
	s.CreateUser(admin)
	token, _ := h.generateToken(admin.ID, admin.Role)

	r := chi.NewRouter()
	r.Use(h.AuthMiddleware)
	r.Route("/api/users", func(r chi.Router) {
		r.Get("/", h.ListUsers)
		r.Post("/", h.CreateUser)
		r.Put("/{id}", h.UpdateUser)
		r.Delete("/{id}", h.DeleteUser)
	})

	// Create
	body, _ := json.Marshal(map[string]string{
		"email": "new@test.com", "name": "New User", "password": "secret", "role": "interviewer",
	})
	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d; body: %s", w.Code, w.Body.String())
	}

	// List — should have admin + new user
	req = httptest.NewRequest("GET", "/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var users []models.User
	json.Unmarshal(w.Body.Bytes(), &users)
	if len(users) != 2 {
		t.Fatalf("list: got %d, want 2", len(users))
	}
}
```

- [ ] **Step 3: Run all tests**

```bash
go test ./internal/... -v
```

Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/api/users.go internal/api/users_test.go
git commit -m "feat: user API handlers with tests"
```

---

### Task 11: Router and server entry point

**Files:**
- Create: `internal/api/router.go`, `embed.go`, `cmd/server/main.go`, `Makefile`

- [ ] **Step 1: Create the router wiring**

Create `internal/api/router.go`:

```go
package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Public
	r.Post("/api/auth/login", h.Login)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)

		// Any authenticated user
		r.Get("/api/me/interviews", h.ListMyInterviews)
		r.Get("/api/notifications", h.ListNotifications)
		r.Put("/api/notifications/{id}/read", h.MarkNotificationRead)
		r.Get("/api/competencies", h.ListCompetencies)

		// Feedback — interviewer submits, scheduler/interviewer can read
		r.Get("/api/interviews/{id}/feedback", h.GetFeedback)
		r.Post("/api/interviews/{id}/feedback", h.CreateFeedback)
		r.Put("/api/feedback/{id}", h.UpdateFeedback)

		// Loops — readable by scheduler and interviewer
		r.Get("/api/loops", h.ListLoops)
		r.Get("/api/loops/{id}", h.GetLoopDetail)

		// Scheduler and admin can list users (scheduler needs it for interviewer assignment)
		r.Group(func(r chi.Router) {
			r.Use(h.RequireRole("scheduler", "admin"))
			r.Get("/api/users", h.ListUsers)
		})

		// Scheduler-only
		r.Group(func(r chi.Router) {
			r.Use(h.RequireRole("scheduler", "admin"))
			r.Post("/api/candidates", h.CreateCandidate)
			r.Get("/api/candidates", h.ListCandidates)
			r.Get("/api/candidates/{id}", h.GetCandidate)
			r.Put("/api/candidates/{id}", h.UpdateCandidate)
			r.Delete("/api/candidates/{id}", h.DeleteCandidate)

			r.Post("/api/loops", h.CreateLoop)
			r.Put("/api/loops/{id}", h.UpdateLoop)
			r.Delete("/api/loops/{id}", h.DeleteLoop)

			r.Post("/api/loops/{id}/interviews", h.CreateInterview)
			r.Put("/api/interviews/{id}", h.UpdateInterview)
			r.Delete("/api/interviews/{id}", h.DeleteInterview)
		})

		// Admin-only
		r.Group(func(r chi.Router) {
			r.Use(h.RequireRole("admin"))
			r.Post("/api/users", h.CreateUser)
			r.Get("/api/users/{id}", h.GetUser)
			r.Put("/api/users/{id}", h.UpdateUser)
			r.Delete("/api/users/{id}", h.DeleteUser)

			r.Post("/api/competencies", h.CreateCompetency)
			r.Put("/api/competencies/{id}", h.UpdateCompetency)
			r.Delete("/api/competencies/{id}", h.DeleteCompetency)
		})
	})

	return r
}
```

- [ ] **Step 2: Create the embed file**

Create `embed.go` at the project root:

```go
package hire

import "embed"

//go:embed all:frontend/dist
var FrontendFS embed.FS
```

Note: This will fail to compile until the frontend is built (Task 14). For now, create a placeholder so the file exists:

```bash
mkdir -p frontend/dist
echo '<!DOCTYPE html><html><body>placeholder</body></html>' > frontend/dist/index.html
```

- [ ] **Step 3: Create the server entry point**

Create `cmd/server/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	root "hire"
	"hire/internal/api"
	"hire/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "hire.db", "SQLite database path")
	jwtSecret := flag.String("jwt-secret", "change-me-in-production", "JWT signing secret")
	flag.Parse()

	s, err := store.New(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer s.Close()

	h := api.NewHandler(s, *jwtSecret)
	r := h.Router()

	// Serve embedded frontend
	frontendDist, err := fs.Sub(root.FrontendFS, "frontend/dist")
	if err != nil {
		log.Fatalf("Failed to load frontend: %v", err)
	}
	fileServer := http.FileServer(http.FS(frontendDist))
	r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Try to serve the file; if it doesn't exist, serve index.html (SPA routing)
		f, err := frontendDist.Open(req.URL.Path[1:])
		if err != nil {
			req.URL.Path = "/"
		} else {
			f.Close()
		}
		fileServer.ServeHTTP(w, req)
	}))

	fmt.Printf("Server listening on %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, r))
}
```

- [ ] **Step 4: Create the Makefile**

Create `Makefile`:

```makefile
.PHONY: dev dev-backend dev-frontend build clean test seed

# Development: run backend and frontend concurrently
dev:
	@echo "Starting backend and frontend..."
	@$(MAKE) dev-backend &
	@$(MAKE) dev-frontend
	@wait

dev-backend:
	go run ./cmd/server -addr :8080 -db hire.db

dev-frontend:
	cd frontend && npm run dev

# Build: compile frontend then embed into Go binary
build: frontend/dist
	go build -o hire-server ./cmd/server

frontend/dist: frontend/node_modules frontend/src/**
	cd frontend && npm run build

frontend/node_modules: frontend/package.json
	cd frontend && npm install

# Test
test:
	go test ./internal/... -v

# Seed demo data
seed:
	go run ./seed/seed.go

# Clean
clean:
	rm -f hire-server hire.db
	rm -rf frontend/dist
```

- [ ] **Step 5: Verify Go compilation**

```bash
go build ./...
```

Expected: compiles without error (the embed requires `frontend/dist/` to exist with at least one file).

- [ ] **Step 6: Commit**

```bash
git add internal/api/router.go embed.go cmd/ Makefile frontend/dist/index.html
git commit -m "feat: router, server entry point, embed, and Makefile"
```

---

## Phase 2: Frontend

### Task 12: Frontend scaffolding (Vite, React, Tailwind, shadcn/ui)

**Files:**
- Create: `frontend/` directory with all config files and initial source

- [ ] **Step 1: Scaffold Vite React TypeScript project**

```bash
cd /home/zach/code/hire
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install
```

- [ ] **Step 2: Install Tailwind CSS**

```bash
cd /home/zach/code/hire/frontend
npm install -D tailwindcss @tailwindcss/vite
```

Update `frontend/vite.config.ts`:

```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
```

Replace `frontend/src/index.css` with:

```css
@import "tailwindcss";
```

- [ ] **Step 3: Initialize shadcn/ui**

```bash
cd /home/zach/code/hire/frontend
npx shadcn@latest init -d
```

This creates `components.json` and sets up the `@/lib/utils.ts` file with the `cn()` helper.

- [ ] **Step 4: Add required shadcn/ui components**

```bash
cd /home/zach/code/hire/frontend
npx shadcn@latest add button input label card table dialog select textarea badge dropdown-menu radio-group alert tabs separator
```

- [ ] **Step 5: Install additional dependencies**

```bash
cd /home/zach/code/hire/frontend
npm install react-router-dom @tanstack/react-query lucide-react
```

- [ ] **Step 6: Clean up Vite boilerplate**

Delete `frontend/src/App.css` and `frontend/src/assets/` if they exist. Replace `frontend/src/main.tsx`:

```tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthProvider } from '@/lib/auth'
import App from './App'
import './index.css'

const queryClient = new QueryClient()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthProvider>
          <App />
        </AuthProvider>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
)
```

Replace `frontend/src/App.tsx` with a placeholder:

```tsx
export default function App() {
  return <div className="p-8 text-xl">Hire App — scaffolding complete</div>
}
```

- [ ] **Step 7: Verify the dev server starts**

```bash
cd /home/zach/code/hire/frontend
npm run dev &
sleep 3
curl -s http://localhost:5173 | head -5
kill %1
```

Expected: HTML output containing the React app root div.

- [ ] **Step 8: Commit**

```bash
cd /home/zach/code/hire
rm frontend/dist/index.html
git add frontend/
git commit -m "feat: frontend scaffolding with Vite, React, Tailwind, shadcn/ui"
```

---

### Task 13: API client and auth context

**Files:**
- Create: `frontend/src/lib/api.ts`, `frontend/src/lib/auth.tsx`

- [ ] **Step 1: Create the typed API client**

Create `frontend/src/lib/api.ts`:

```ts
const API_BASE = '/api'

function getToken(): string | null {
  return localStorage.getItem('token')
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })
  if (res.status === 204) return undefined as T
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || res.statusText)
  }
  return res.json()
}

// Auth
export const auth = {
  login: (email: string, password: string) =>
    request<{ token: string; user: User }>('POST', '/auth/login', { email, password }),
}

// Users
export const users = {
  list: () => request<User[]>('GET', '/users'),
  create: (data: CreateUserReq) => request<User>('POST', '/users', data),
  update: (id: number, data: CreateUserReq) => request<User>('PUT', `/users/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/users/${id}`),
}

// Candidates
export const candidates = {
  list: () => request<Candidate[]>('GET', '/candidates'),
  get: (id: number) => request<Candidate>('GET', `/candidates/${id}`),
  create: (data: Partial<Candidate>) => request<Candidate>('POST', '/candidates', data),
  update: (id: number, data: Partial<Candidate>) => request<Candidate>('PUT', `/candidates/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/candidates/${id}`),
}

// Loops
export const loops = {
  list: (params?: { candidate_id?: number; status?: string }) => {
    const q = new URLSearchParams()
    if (params?.candidate_id) q.set('candidate_id', String(params.candidate_id))
    if (params?.status) q.set('status', params.status)
    const qs = q.toString()
    return request<InterviewLoop[]>('GET', `/loops${qs ? '?' + qs : ''}`)
  },
  get: (id: number) => request<LoopDetail>('GET', `/loops/${id}`),
  create: (data: { candidate_id: number }) => request<InterviewLoop>('POST', '/loops', data),
  update: (id: number, data: Partial<InterviewLoop>) => request<InterviewLoop>('PUT', `/loops/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/loops/${id}`),
}

// Interviews
export const interviews = {
  createInLoop: (loopId: number, data: Partial<Interview>) =>
    request<Interview>('POST', `/loops/${loopId}/interviews`, data),
  update: (id: number, data: Partial<Interview>) => request<Interview>('PUT', `/interviews/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/interviews/${id}`),
  listMine: () => request<Interview[]>('GET', '/me/interviews'),
}

// Feedback
export const feedback = {
  get: (interviewId: number) => request<Feedback>('GET', `/interviews/${interviewId}/feedback`),
  create: (interviewId: number, data: FeedbackCreate) => request<Feedback>('POST', `/interviews/${interviewId}/feedback`, data),
  update: (id: number, data: Partial<Feedback>) => request<Feedback>('PUT', `/feedback/${id}`, data),
}

// Competencies
export const competencies = {
  list: () => request<Competency[]>('GET', '/competencies'),
  create: (data: Partial<Competency>) => request<Competency>('POST', '/competencies', data),
  update: (id: number, data: Partial<Competency>) => request<Competency>('PUT', `/competencies/${id}`, data),
  delete: (id: number) => request<void>('DELETE', `/competencies/${id}`),
}

// Notifications
export const notifications = {
  list: () => request<Notification[]>('GET', '/notifications'),
  markRead: (id: number) => request<void>('PUT', `/notifications/${id}/read`),
}

// Types
export interface User {
  id: number
  email: string
  name: string
  role: 'admin' | 'scheduler' | 'interviewer'
  created_at: string
}

export interface CreateUserReq {
  email: string
  name: string
  password: string
  role: string
}

export interface Candidate {
  id: number
  name: string
  email: string
  resume_url: string
  status: 'active' | 'hired' | 'rejected' | 'withdrawn'
  created_at: string
}

export interface InterviewLoop {
  id: number
  candidate_id: number
  status: 'scheduling' | 'active' | 'complete'
  final_decision: string | null
  debrief_notes: string | null
  created_by: number
  created_at: string
}

export interface LoopDetail extends InterviewLoop {
  candidate: Candidate
  interviews: InterviewWithFeedback[]
}

export interface Interview {
  id: number
  loop_id: number
  interviewer_id: number
  focus_area: string
  scheduled_at: string
  video_link: string
  notes_for_interviewer: string
  status: 'pending' | 'complete'
  created_at: string
}

export interface InterviewWithFeedback extends Interview {
  interviewer_name: string
  feedback: Feedback | null
}

export interface Feedback {
  id: number
  interview_id: number
  recommendation: 'strong_hire' | 'hire' | 'no_hire' | 'strong_no_hire'
  recommendation_reason: string
  free_form_notes: string
  submitted_at: string
  competency_ratings: CompetencyRating[]
}

export interface FeedbackCreate {
  recommendation: string
  recommendation_reason: string
  free_form_notes: string
  competency_ratings: { competency_id: number; rating_value: string }[]
}

export interface Competency {
  id: number
  name: string
  rating_type: 'levels' | 'stars'
  ratings_json: string
  created_at: string
}

export interface CompetencyRating {
  id: number
  feedback_id: number
  competency_id: number
  rating_value: string
}

export interface Notification {
  id: number
  user_id: number
  message: string
  link: string
  read: boolean
  created_at: string
}
```

- [ ] **Step 2: Create the auth context**

Create `frontend/src/lib/auth.tsx`:

```tsx
import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'
import { auth as authApi, type User } from './api'

interface AuthContextType {
  user: User | null
  token: string | null
  login: (email: string, password: string) => Promise<void>
  logout: () => void
  isAuthenticated: boolean
}

const AuthContext = createContext<AuthContextType | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(() => {
    const stored = localStorage.getItem('user')
    return stored ? JSON.parse(stored) : null
  })
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('token'))

  const login = useCallback(async (email: string, password: string) => {
    const res = await authApi.login(email, password)
    localStorage.setItem('token', res.token)
    localStorage.setItem('user', JSON.stringify(res.user))
    setToken(res.token)
    setUser(res.user)
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    setToken(null)
    setUser(null)
  }, [])

  return (
    <AuthContext.Provider value={{ user, token, login, logout, isAuthenticated: !!token }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd /home/zach/code/hire/frontend
npx tsc --noEmit
```

Expected: no type errors (may need to adjust tsconfig paths).

- [ ] **Step 4: Commit**

```bash
cd /home/zach/code/hire
git add frontend/src/lib/api.ts frontend/src/lib/auth.tsx frontend/src/main.tsx
git commit -m "feat: API client with types and auth context"
```

---

### Task 14: Login page and app layout

**Files:**
- Create: `frontend/src/pages/LoginPage.tsx`, `frontend/src/pages/Dashboard.tsx`, `frontend/src/components/Layout.tsx`, `frontend/src/components/NotificationBell.tsx`
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Create the login page**

Create `frontend/src/pages/LoginPage.tsx`:

```tsx
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/lib/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export default function LoginPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const { login } = useAuth()
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      await login(email, password)
      navigate('/')
    } catch (err: any) {
      setError(err.message || 'Login failed')
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle className="text-2xl text-center">Hire</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input id="email" type="email" value={email} onChange={e => setEmail(e.target.value)} required />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input id="password" type="password" value={password} onChange={e => setPassword(e.target.value)} required />
            </div>
            {error && <p className="text-sm text-red-600">{error}</p>}
            <Button type="submit" className="w-full">Sign in</Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
```

- [ ] **Step 2: Create the notification bell**

Create `frontend/src/components/NotificationBell.tsx`:

```tsx
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { notifications as notifApi, type Notification } from '@/lib/api'
import { Bell } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useNavigate } from 'react-router-dom'

export default function NotificationBell() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { data: notifs = [] } = useQuery({ queryKey: ['notifications'], queryFn: notifApi.list, refetchInterval: 15000 })
  const markRead = useMutation({
    mutationFn: notifApi.markRead,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  })

  const unreadCount = notifs.filter((n: Notification) => !n.read).length

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="relative p-2">
        <Bell className="h-5 w-5" />
        {unreadCount > 0 && (
          <Badge variant="destructive" className="absolute -top-1 -right-1 h-5 w-5 flex items-center justify-center p-0 text-xs">
            {unreadCount}
          </Badge>
        )}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-80">
        {notifs.length === 0 && (
          <DropdownMenuItem disabled>No notifications</DropdownMenuItem>
        )}
        {notifs.map((n: Notification) => (
          <DropdownMenuItem
            key={n.id}
            className={n.read ? 'opacity-60' : 'font-medium'}
            onClick={() => {
              if (!n.read) markRead.mutate(n.id)
              if (n.link) navigate(n.link)
            }}
          >
            {n.message}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
```

- [ ] **Step 3: Create the app layout**

Create `frontend/src/components/Layout.tsx`:

```tsx
import { Outlet, Link, useNavigate } from 'react-router-dom'
import { useAuth } from '@/lib/auth'
import { Button } from '@/components/ui/button'
import NotificationBell from './NotificationBell'

export default function Layout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const navLinks = () => {
    switch (user?.role) {
      case 'admin':
        return (
          <>
            <Link to="/admin/users" className="hover:underline">Users</Link>
            <Link to="/admin/competencies" className="hover:underline">Competencies</Link>
          </>
        )
      case 'scheduler':
        return (
          <>
            <Link to="/candidates" className="hover:underline">Candidates</Link>
          </>
        )
      case 'interviewer':
        return (
          <>
            <Link to="/" className="hover:underline">My Interviews</Link>
          </>
        )
      default:
        return null
    }
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b px-6 py-3 flex items-center justify-between">
        <div className="flex items-center gap-6">
          <Link to="/" className="text-xl font-bold">Hire</Link>
          <div className="flex items-center gap-4 text-sm">{navLinks()}</div>
        </div>
        <div className="flex items-center gap-4">
          <NotificationBell />
          <span className="text-sm text-gray-600">{user?.name} ({user?.role})</span>
          <Button variant="ghost" size="sm" onClick={handleLogout}>Logout</Button>
        </div>
      </nav>
      <main className="max-w-6xl mx-auto p-6">
        <Outlet />
      </main>
    </div>
  )
}
```

- [ ] **Step 4: Create the dashboard (role-based redirect)**

Create `frontend/src/pages/Dashboard.tsx`:

```tsx
import { Navigate } from 'react-router-dom'
import { useAuth } from '@/lib/auth'

export default function Dashboard() {
  const { user } = useAuth()

  switch (user?.role) {
    case 'admin':
      return <Navigate to="/admin/users" replace />
    case 'scheduler':
      return <Navigate to="/candidates" replace />
    case 'interviewer':
    default:
      return <Navigate to="/my-interviews" replace />
  }
}
```

- [ ] **Step 5: Wire up routing in App.tsx**

Replace `frontend/src/App.tsx`:

```tsx
import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from '@/lib/auth'
import Layout from '@/components/Layout'
import LoginPage from '@/pages/LoginPage'
import Dashboard from '@/pages/Dashboard'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth()
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        element={
          <ProtectedRoute>
            <Layout />
          </ProtectedRoute>
        }
      >
        <Route path="/" element={<Dashboard />} />
        {/* Placeholder routes — implemented in Tasks 15-17 */}
        <Route path="/my-interviews" element={<div>My Interviews (coming soon)</div>} />
        <Route path="/interviews/:id" element={<div>Interview Detail (coming soon)</div>} />
        <Route path="/candidates" element={<div>Candidates (coming soon)</div>} />
        <Route path="/candidates/:id" element={<div>Candidate Detail (coming soon)</div>} />
        <Route path="/loops/:id/edit" element={<div>Loop Editor (coming soon)</div>} />
        <Route path="/loops/:id/debrief" element={<div>Debrief (coming soon)</div>} />
        <Route path="/admin/users" element={<div>User Management (coming soon)</div>} />
        <Route path="/admin/competencies" element={<div>Competency Management (coming soon)</div>} />
      </Route>
    </Routes>
  )
}
```

- [ ] **Step 6: Verify in browser**

Start both backend and frontend:
```bash
cd /home/zach/code/hire
go run ./cmd/server &
cd frontend && npm run dev &
```

Open `http://localhost:5173` — should redirect to `/login`. The login page should render with email/password fields and a "Sign in" button.

- [ ] **Step 7: Commit**

```bash
cd /home/zach/code/hire
git add frontend/src/
git commit -m "feat: login page, layout with nav, notification bell, and routing"
```

---

### Task 15: Admin pages (user management, competency management)

**Files:**
- Create: `frontend/src/pages/admin/UserManagement.tsx`, `frontend/src/pages/admin/CompetencyManagement.tsx`
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Create user management page**

Create `frontend/src/pages/admin/UserManagement.tsx`:

```tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { users as usersApi, type User, type CreateUserReq } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Trash2, Plus } from 'lucide-react'

export default function UserManagement() {
  const queryClient = useQueryClient()
  const { data: userList = [] } = useQuery({ queryKey: ['users'], queryFn: usersApi.list })
  const createUser = useMutation({
    mutationFn: (data: CreateUserReq) => usersApi.create(data),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['users'] }); setOpen(false); resetForm() },
  })
  const deleteUser = useMutation({
    mutationFn: usersApi.delete,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['users'] }),
  })

  const [open, setOpen] = useState(false)
  const [form, setForm] = useState<CreateUserReq>({ email: '', name: '', password: '', role: 'interviewer' })
  const resetForm = () => setForm({ email: '', name: '', password: '', role: 'interviewer' })

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">User Management</h1>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button><Plus className="h-4 w-4 mr-2" />Add User</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader><DialogTitle>Create User</DialogTitle></DialogHeader>
            <form onSubmit={e => { e.preventDefault(); createUser.mutate(form) }} className="space-y-4">
              <div className="space-y-2">
                <Label>Name</Label>
                <Input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required />
              </div>
              <div className="space-y-2">
                <Label>Email</Label>
                <Input type="email" value={form.email} onChange={e => setForm({ ...form, email: e.target.value })} required />
              </div>
              <div className="space-y-2">
                <Label>Password</Label>
                <Input type="password" value={form.password} onChange={e => setForm({ ...form, password: e.target.value })} required />
              </div>
              <div className="space-y-2">
                <Label>Role</Label>
                <Select value={form.role} onValueChange={v => setForm({ ...form, role: v })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="admin">Admin</SelectItem>
                    <SelectItem value="scheduler">Scheduler</SelectItem>
                    <SelectItem value="interviewer">Interviewer</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <Button type="submit" className="w-full">Create</Button>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Email</TableHead>
            <TableHead>Role</TableHead>
            <TableHead className="w-16"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {userList.map((u: User) => (
            <TableRow key={u.id}>
              <TableCell>{u.name}</TableCell>
              <TableCell>{u.email}</TableCell>
              <TableCell><Badge role={u.role} /></TableCell>
              <TableCell>
                <Button variant="ghost" size="sm" onClick={() => deleteUser.mutate(u.id)}>
                  <Trash2 className="h-4 w-4 text-red-500" />
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function Badge({ role }: { role: string }) {
  const colors: Record<string, string> = {
    admin: 'bg-purple-100 text-purple-800',
    scheduler: 'bg-blue-100 text-blue-800',
    interviewer: 'bg-green-100 text-green-800',
  }
  return <span className={`px-2 py-1 rounded text-xs font-medium ${colors[role] || ''}`}>{role}</span>
}
```

- [ ] **Step 2: Create competency management page**

Create `frontend/src/pages/admin/CompetencyManagement.tsx`:

```tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { competencies as compApi, type Competency } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Trash2, Plus } from 'lucide-react'

export default function CompetencyManagement() {
  const queryClient = useQueryClient()
  const { data: comps = [] } = useQuery({ queryKey: ['competencies'], queryFn: compApi.list })
  const createComp = useMutation({
    mutationFn: (data: Partial<Competency>) => compApi.create(data),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['competencies'] }); setOpen(false) },
  })
  const deleteComp = useMutation({
    mutationFn: compApi.delete,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['competencies'] }),
  })

  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [ratingType, setRatingType] = useState<'levels' | 'stars'>('levels')
  const [levelsInput, setLevelsInput] = useState('Learning, Owning, Advising')
  const [starsMax, setStarsMax] = useState('5')

  const handleCreate = () => {
    const ratingsJson = ratingType === 'levels'
      ? JSON.stringify(levelsInput.split(',').map(s => s.trim()).filter(Boolean))
      : JSON.stringify({ min: 1, max: parseInt(starsMax) })
    createComp.mutate({ name, rating_type: ratingType, ratings_json: ratingsJson })
  }

  const parseRatings = (c: Competency) => {
    try {
      const parsed = JSON.parse(c.ratings_json)
      if (c.rating_type === 'levels') return (parsed as string[]).join(', ')
      return `1-${parsed.max} stars`
    } catch { return c.ratings_json }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Competency Management</h1>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button><Plus className="h-4 w-4 mr-2" />Add Competency</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader><DialogTitle>Create Competency</DialogTitle></DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Name</Label>
                <Input value={name} onChange={e => setName(e.target.value)} placeholder="e.g. Problem Solving" />
              </div>
              <div className="space-y-2">
                <Label>Rating Type</Label>
                <Select value={ratingType} onValueChange={v => setRatingType(v as 'levels' | 'stars')}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="levels">Levels (custom labels)</SelectItem>
                    <SelectItem value="stars">Stars (numeric scale)</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {ratingType === 'levels' ? (
                <div className="space-y-2">
                  <Label>Levels (comma-separated)</Label>
                  <Input value={levelsInput} onChange={e => setLevelsInput(e.target.value)} placeholder="Learning, Owning, Advising" />
                </div>
              ) : (
                <div className="space-y-2">
                  <Label>Max Stars</Label>
                  <Input type="number" value={starsMax} onChange={e => setStarsMax(e.target.value)} min="2" max="10" />
                </div>
              )}
              <Button onClick={handleCreate} className="w-full">Create</Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Type</TableHead>
            <TableHead>Ratings</TableHead>
            <TableHead className="w-16"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {comps.map((c: Competency) => (
            <TableRow key={c.id}>
              <TableCell className="font-medium">{c.name}</TableCell>
              <TableCell>{c.rating_type}</TableCell>
              <TableCell>{parseRatings(c)}</TableCell>
              <TableCell>
                <Button variant="ghost" size="sm" onClick={() => deleteComp.mutate(c.id)}>
                  <Trash2 className="h-4 w-4 text-red-500" />
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
```

- [ ] **Step 3: Update App.tsx routes**

In `frontend/src/App.tsx`, replace the admin placeholder routes:

```tsx
import UserManagement from '@/pages/admin/UserManagement'
import CompetencyManagement from '@/pages/admin/CompetencyManagement'

// Inside Routes, replace:
//   <Route path="/admin/users" element={<div>User Management (coming soon)</div>} />
//   <Route path="/admin/competencies" element={<div>Competency Management (coming soon)</div>} />
// With:
//   <Route path="/admin/users" element={<UserManagement />} />
//   <Route path="/admin/competencies" element={<CompetencyManagement />} />
```

- [ ] **Step 4: Verify in browser**

Log in as an admin user. Navigate to `/admin/users` and `/admin/competencies`. Verify you can create and delete entries.

- [ ] **Step 5: Commit**

```bash
cd /home/zach/code/hire
git add frontend/src/pages/admin/ frontend/src/App.tsx
git commit -m "feat: admin pages for user and competency management"
```

---

### Task 16: Scheduler pages (candidates, loop editor, debrief)

**Files:**
- Create: `frontend/src/pages/scheduler/CandidatesList.tsx`, `frontend/src/pages/scheduler/CandidateDetail.tsx`, `frontend/src/pages/scheduler/LoopEditor.tsx`, `frontend/src/pages/scheduler/DebriefView.tsx`
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Create candidates list page**

Create `frontend/src/pages/scheduler/CandidatesList.tsx`:

```tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { candidates as candApi, type Candidate } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { Plus } from 'lucide-react'

export default function CandidatesList() {
  const queryClient = useQueryClient()
  const { data: cands = [] } = useQuery({ queryKey: ['candidates'], queryFn: candApi.list })
  const createCand = useMutation({
    mutationFn: (data: Partial<Candidate>) => candApi.create(data),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['candidates'] }); setOpen(false); resetForm() },
  })

  const [open, setOpen] = useState(false)
  const [form, setForm] = useState({ name: '', email: '', resume_url: '' })
  const resetForm = () => setForm({ name: '', email: '', resume_url: '' })

  const statusColor: Record<string, string> = {
    active: 'bg-blue-100 text-blue-800',
    hired: 'bg-green-100 text-green-800',
    rejected: 'bg-red-100 text-red-800',
    withdrawn: 'bg-gray-100 text-gray-800',
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Candidates</h1>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button><Plus className="h-4 w-4 mr-2" />Add Candidate</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader><DialogTitle>Add Candidate</DialogTitle></DialogHeader>
            <form onSubmit={e => { e.preventDefault(); createCand.mutate({ ...form, status: 'active' }) }} className="space-y-4">
              <div className="space-y-2">
                <Label>Name</Label>
                <Input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required />
              </div>
              <div className="space-y-2">
                <Label>Email</Label>
                <Input type="email" value={form.email} onChange={e => setForm({ ...form, email: e.target.value })} required />
              </div>
              <div className="space-y-2">
                <Label>Resume URL</Label>
                <Input value={form.resume_url} onChange={e => setForm({ ...form, resume_url: e.target.value })} />
              </div>
              <Button type="submit" className="w-full">Create</Button>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Email</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Added</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {cands.map((c: Candidate) => (
            <TableRow key={c.id}>
              <TableCell>
                <Link to={`/candidates/${c.id}`} className="font-medium text-blue-600 hover:underline">{c.name}</Link>
              </TableCell>
              <TableCell>{c.email}</TableCell>
              <TableCell>
                <span className={`px-2 py-1 rounded text-xs font-medium ${statusColor[c.status] || ''}`}>{c.status}</span>
              </TableCell>
              <TableCell className="text-sm text-gray-500">{new Date(c.created_at).toLocaleDateString()}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
```

- [ ] **Step 2: Create candidate detail page**

Create `frontend/src/pages/scheduler/CandidateDetail.tsx`:

```tsx
import { useParams, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { candidates as candApi, loops as loopsApi, type LoopDetail } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Plus, ExternalLink } from 'lucide-react'

export default function CandidateDetail() {
  const { id } = useParams<{ id: string }>()
  const candidateId = parseInt(id!)
  const queryClient = useQueryClient()

  const { data: candidate } = useQuery({ queryKey: ['candidates', candidateId], queryFn: () => candApi.get(candidateId) })
  const { data: candidateLoops = [] } = useQuery({
    queryKey: ['loops', { candidate_id: candidateId }],
    queryFn: () => loopsApi.list({ candidate_id: candidateId }),
  })

  const createLoop = useMutation({
    mutationFn: () => loopsApi.create({ candidate_id: candidateId }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['loops'] }),
  })

  if (!candidate) return <div>Loading...</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{candidate.name}</h1>
          <p className="text-gray-500">{candidate.email}</p>
          {candidate.resume_url && (
            <a href={candidate.resume_url} target="_blank" rel="noopener" className="text-blue-600 text-sm flex items-center gap-1">
              Resume <ExternalLink className="h-3 w-3" />
            </a>
          )}
        </div>
        <Button onClick={() => createLoop.mutate()}>
          <Plus className="h-4 w-4 mr-2" />New Interview Loop
        </Button>
      </div>

      {candidateLoops.length === 0 && (
        <p className="text-gray-500">No interview loops yet. Create one to get started.</p>
      )}

      {candidateLoops.map(loop => (
        <Card key={loop.id}>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-lg">Loop #{loop.id}</CardTitle>
            <div className="flex items-center gap-2">
              <Badge variant={loop.status === 'complete' ? 'default' : 'secondary'}>{loop.status}</Badge>
              <Link to={`/loops/${loop.id}/edit`}>
                <Button variant="outline" size="sm">Edit Loop</Button>
              </Link>
              <Link to={`/loops/${loop.id}/debrief`}>
                <Button variant="outline" size="sm">Debrief</Button>
              </Link>
            </div>
          </CardHeader>
          <CardContent>
            {loop.final_decision && (
              <p className="text-sm">Final Decision: <strong>{loop.final_decision.replace('_', ' ')}</strong></p>
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
```

- [ ] **Step 3: Create loop editor page**

Create `frontend/src/pages/scheduler/LoopEditor.tsx`:

```tsx
import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { loops as loopsApi, interviews as ivApi, users as usersApi, type InterviewWithFeedback } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Plus, Trash2 } from 'lucide-react'

export default function LoopEditor() {
  const { id } = useParams<{ id: string }>()
  const loopId = parseInt(id!)
  const queryClient = useQueryClient()

  const { data: loop } = useQuery({ queryKey: ['loops', loopId], queryFn: () => loopsApi.get(loopId) })
  const { data: userList = [] } = useQuery({ queryKey: ['users'], queryFn: usersApi.list })
  const interviewers = userList.filter(u => u.role === 'interviewer')

  const createInterview = useMutation({
    mutationFn: (data: any) => ivApi.createInLoop(loopId, data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['loops', loopId] }),
  })
  const deleteInterview = useMutation({
    mutationFn: ivApi.delete,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['loops', loopId] }),
  })

  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({
    interviewer_id: 0,
    focus_area: '',
    scheduled_at: '',
    video_link: '',
    notes_for_interviewer: '',
  })

  const handleAdd = () => {
    createInterview.mutate({
      ...form,
      interviewer_id: Number(form.interviewer_id),
      scheduled_at: new Date(form.scheduled_at).toISOString(),
      status: 'pending',
    })
    setForm({ interviewer_id: 0, focus_area: '', scheduled_at: '', video_link: '', notes_for_interviewer: '' })
    setShowForm(false)
  }

  if (!loop) return <div>Loading...</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Edit Loop — {loop.candidate.name}</h1>
          <p className="text-gray-500">{loop.candidate.email}</p>
        </div>
        <Badge variant={loop.status === 'complete' ? 'default' : 'secondary'}>{loop.status}</Badge>
      </div>

      {/* Existing interviews */}
      <div className="space-y-3">
        {loop.interviews?.map((iv: InterviewWithFeedback) => (
          <Card key={iv.id}>
            <CardContent className="flex items-center justify-between py-4">
              <div>
                <p className="font-medium">{iv.focus_area}</p>
                <p className="text-sm text-gray-500">
                  {iv.interviewer_name} &middot; {new Date(iv.scheduled_at).toLocaleString()}
                </p>
                {iv.video_link && <p className="text-sm text-blue-600">{iv.video_link}</p>}
                {iv.notes_for_interviewer && <p className="text-sm text-gray-400 mt-1">{iv.notes_for_interviewer}</p>}
              </div>
              <div className="flex items-center gap-2">
                <Badge variant={iv.status === 'complete' ? 'default' : 'outline'}>{iv.status}</Badge>
                <Button variant="ghost" size="sm" onClick={() => deleteInterview.mutate(iv.id)}>
                  <Trash2 className="h-4 w-4 text-red-500" />
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Add interview form */}
      {showForm ? (
        <Card>
          <CardHeader><CardTitle>Add Interview</CardTitle></CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Interviewer</Label>
                <Select value={String(form.interviewer_id)} onValueChange={v => setForm({ ...form, interviewer_id: parseInt(v) })}>
                  <SelectTrigger><SelectValue placeholder="Select interviewer" /></SelectTrigger>
                  <SelectContent>
                    {interviewers.map(u => (
                      <SelectItem key={u.id} value={String(u.id)}>{u.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>Focus Area</Label>
                <Input value={form.focus_area} onChange={e => setForm({ ...form, focus_area: e.target.value })} placeholder="e.g. Coding, System Design" />
              </div>
              <div className="space-y-2">
                <Label>Scheduled At</Label>
                <Input type="datetime-local" value={form.scheduled_at} onChange={e => setForm({ ...form, scheduled_at: e.target.value })} />
              </div>
              <div className="space-y-2">
                <Label>Video Link</Label>
                <Input value={form.video_link} onChange={e => setForm({ ...form, video_link: e.target.value })} placeholder="https://meet.example.com/..." />
              </div>
            </div>
            <div className="space-y-2">
              <Label>Notes for Interviewer</Label>
              <Textarea value={form.notes_for_interviewer} onChange={e => setForm({ ...form, notes_for_interviewer: e.target.value })} />
            </div>
            <div className="flex gap-2">
              <Button onClick={handleAdd}>Add Interview</Button>
              <Button variant="outline" onClick={() => setShowForm(false)}>Cancel</Button>
            </div>
          </CardContent>
        </Card>
      ) : (
        <Button variant="outline" onClick={() => setShowForm(true)}>
          <Plus className="h-4 w-4 mr-2" />Add Interview
        </Button>
      )}
    </div>
  )
}
```

- [ ] **Step 4: Create debrief view page**

Create `frontend/src/pages/scheduler/DebriefView.tsx`:

```tsx
import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { loops as loopsApi, competencies as compApi, type InterviewWithFeedback, type Competency } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Separator } from '@/components/ui/separator'
import { AlertTriangle } from 'lucide-react'

const DECISIONS = ['strong_hire', 'hire', 'no_hire', 'strong_no_hire'] as const

export default function DebriefView() {
  const { id } = useParams<{ id: string }>()
  const loopId = parseInt(id!)
  const queryClient = useQueryClient()

  const { data: loop } = useQuery({ queryKey: ['loops', loopId], queryFn: () => loopsApi.get(loopId) })
  const { data: comps = [] } = useQuery({ queryKey: ['competencies'], queryFn: compApi.list })

  const [decision, setDecision] = useState('')
  const [notes, setNotes] = useState('')

  const updateLoop = useMutation({
    mutationFn: () => loopsApi.update(loopId, {
      status: 'complete',
      final_decision: decision,
      debrief_notes: notes,
    }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['loops', loopId] }),
  })

  if (!loop) return <div>Loading...</div>

  const allComplete = loop.interviews?.every((iv: InterviewWithFeedback) => iv.feedback != null)
  const pendingCount = loop.interviews?.filter((iv: InterviewWithFeedback) => !iv.feedback).length ?? 0

  const compMap = Object.fromEntries(comps.map((c: Competency) => [c.id, c]))

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Debrief — {loop.candidate.name}</h1>

      {!allComplete && (
        <Alert>
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>{pendingCount} interview(s) still awaiting feedback.</AlertDescription>
        </Alert>
      )}

      {/* Feedback cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {loop.interviews?.map((iv: InterviewWithFeedback) => (
          <Card key={iv.id}>
            <CardHeader>
              <CardTitle className="text-base">{iv.focus_area} — {iv.interviewer_name}</CardTitle>
            </CardHeader>
            <CardContent>
              {iv.feedback ? (
                <div className="space-y-3">
                  <div>
                    <span className="text-sm font-medium">Recommendation: </span>
                    <span className="font-bold">{iv.feedback.recommendation.replace(/_/g, ' ')}</span>
                  </div>
                  {iv.feedback.recommendation_reason && (
                    <div>
                      <span className="text-sm font-medium">Reason: </span>
                      <span className="text-sm">{iv.feedback.recommendation_reason}</span>
                    </div>
                  )}
                  {iv.feedback.competency_ratings?.map(cr => {
                    const comp = compMap[cr.competency_id]
                    return (
                      <div key={cr.id} className="flex justify-between text-sm">
                        <span className="text-gray-600">{comp?.name || `Competency ${cr.competency_id}`}</span>
                        <span className="font-medium">{cr.rating_value}</span>
                      </div>
                    )
                  })}
                  <Separator />
                  {iv.feedback.free_form_notes && (
                    <p className="text-sm text-gray-700 whitespace-pre-wrap">{iv.feedback.free_form_notes}</p>
                  )}
                </div>
              ) : (
                <p className="text-gray-400 text-sm">Awaiting feedback</p>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Final decision */}
      <Card>
        <CardHeader><CardTitle>Final Decision</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Decision</Label>
            <Select value={decision || loop.final_decision || ''} onValueChange={setDecision}>
              <SelectTrigger><SelectValue placeholder="Select decision" /></SelectTrigger>
              <SelectContent>
                {DECISIONS.map(d => (
                  <SelectItem key={d} value={d}>{d.replace(/_/g, ' ')}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Debrief Notes</Label>
            <Textarea
              value={notes || loop.debrief_notes || ''}
              onChange={e => setNotes(e.target.value)}
              rows={4}
              placeholder="Summary of debrief discussion..."
            />
          </div>
          <Button onClick={() => updateLoop.mutate()}>Save Decision</Button>
        </CardContent>
      </Card>
    </div>
  )
}
```

- [ ] **Step 5: Update App.tsx routes**

In `frontend/src/App.tsx`, import and wire up the scheduler pages:

```tsx
import CandidatesList from '@/pages/scheduler/CandidatesList'
import CandidateDetail from '@/pages/scheduler/CandidateDetail'
import LoopEditor from '@/pages/scheduler/LoopEditor'
import DebriefView from '@/pages/scheduler/DebriefView'

// Replace the scheduler placeholder routes with:
//   <Route path="/candidates" element={<CandidatesList />} />
//   <Route path="/candidates/:id" element={<CandidateDetail />} />
//   <Route path="/loops/:id/edit" element={<LoopEditor />} />
//   <Route path="/loops/:id/debrief" element={<DebriefView />} />
```

- [ ] **Step 6: Verify in browser**

Log in as a scheduler. Navigate through candidates list, create a candidate, create a loop, add interviews, and view the debrief page.

- [ ] **Step 7: Commit**

```bash
cd /home/zach/code/hire
git add frontend/src/pages/scheduler/ frontend/src/App.tsx
git commit -m "feat: scheduler pages — candidates, loop editor, debrief view"
```

---

### Task 17: Interviewer pages (my interviews, interview detail, feedback form)

**Files:**
- Create: `frontend/src/pages/interviewer/MyInterviews.tsx`, `frontend/src/pages/interviewer/InterviewDetail.tsx`, `frontend/src/pages/interviewer/FeedbackForm.tsx`
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Create my interviews page**

Create `frontend/src/pages/interviewer/MyInterviews.tsx`:

```tsx
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { interviews as ivApi, type Interview } from '@/lib/api'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'

export default function MyInterviews() {
  const { data: myInterviews = [] } = useQuery({ queryKey: ['my-interviews'], queryFn: ivApi.listMine })

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">My Interviews</h1>

      {myInterviews.length === 0 && <p className="text-gray-500">No interviews assigned yet.</p>}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Focus Area</TableHead>
            <TableHead>Scheduled</TableHead>
            <TableHead>Status</TableHead>
            <TableHead></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {myInterviews.map((iv: Interview) => (
            <TableRow key={iv.id}>
              <TableCell className="font-medium">{iv.focus_area}</TableCell>
              <TableCell>{new Date(iv.scheduled_at).toLocaleString()}</TableCell>
              <TableCell>
                <Badge variant={iv.status === 'complete' ? 'default' : 'outline'}>
                  {iv.status === 'complete' ? 'Feedback Submitted' : 'Pending'}
                </Badge>
              </TableCell>
              <TableCell>
                <Link to={`/interviews/${iv.id}`} className="text-blue-600 hover:underline text-sm">
                  {iv.status === 'complete' ? 'View' : 'Submit Feedback'}
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

- [ ] **Step 2: Create feedback form component**

Create `frontend/src/pages/interviewer/FeedbackForm.tsx`:

```tsx
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { feedback as fbApi, type Competency, type FeedbackCreate } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Star } from 'lucide-react'

const RECOMMENDATIONS = [
  { value: 'strong_hire', label: 'Strong Hire' },
  { value: 'hire', label: 'Hire' },
  { value: 'no_hire', label: 'No Hire' },
  { value: 'strong_no_hire', label: 'Strong No Hire' },
]

interface Props {
  interviewId: number
  competencies: Competency[]
  onSubmitted: () => void
}

export default function FeedbackForm({ interviewId, competencies, onSubmitted }: Props) {
  const queryClient = useQueryClient()
  const [recommendation, setRecommendation] = useState('')
  const [reason, setReason] = useState('')
  const [notes, setNotes] = useState('')
  const [ratings, setRatings] = useState<Record<number, string>>({})
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async () => {
    setSubmitting(true)
    try {
      const data: FeedbackCreate = {
        recommendation,
        recommendation_reason: reason,
        free_form_notes: notes,
        competency_ratings: Object.entries(ratings).map(([compId, value]) => ({
          competency_id: parseInt(compId),
          rating_value: value,
        })),
      }
      await fbApi.create(interviewId, data)
      queryClient.invalidateQueries({ queryKey: ['my-interviews'] })
      onSubmitted()
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Card>
      <CardHeader><CardTitle>Submit Feedback</CardTitle></CardHeader>
      <CardContent className="space-y-6">
        {/* Recommendation */}
        <div className="space-y-3">
          <Label className="text-base font-semibold">Hiring Recommendation</Label>
          <RadioGroup value={recommendation} onValueChange={setRecommendation}>
            {RECOMMENDATIONS.map(r => (
              <div key={r.value} className="flex items-center space-x-2">
                <RadioGroupItem value={r.value} id={r.value} />
                <Label htmlFor={r.value}>{r.label}</Label>
              </div>
            ))}
          </RadioGroup>
        </div>

        {/* Reason */}
        <div className="space-y-2">
          <Label>Reason for Recommendation</Label>
          <Textarea value={reason} onChange={e => setReason(e.target.value)} rows={3} placeholder="Why are you making this recommendation?" />
        </div>

        {/* Competency Ratings */}
        {competencies.map(comp => {
          const options = JSON.parse(comp.ratings_json)
          return (
            <div key={comp.id} className="space-y-2">
              <Label className="font-semibold">{comp.name}</Label>
              {comp.rating_type === 'levels' ? (
                <Select value={ratings[comp.id] || ''} onValueChange={v => setRatings({ ...ratings, [comp.id]: v })}>
                  <SelectTrigger><SelectValue placeholder="Select level" /></SelectTrigger>
                  <SelectContent>
                    {(options as string[]).map((level: string) => (
                      <SelectItem key={level} value={level}>{level}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <div className="flex gap-1">
                  {Array.from({ length: options.max }, (_, i) => i + 1).map(n => (
                    <button
                      key={n}
                      type="button"
                      onClick={() => setRatings({ ...ratings, [comp.id]: String(n) })}
                      className="p-1"
                    >
                      <Star
                        className={`h-6 w-6 ${parseInt(ratings[comp.id] || '0') >= n ? 'fill-yellow-400 text-yellow-400' : 'text-gray-300'}`}
                      />
                    </button>
                  ))}
                </div>
              )}
            </div>
          )
        })}

        {/* Free-form notes */}
        <div className="space-y-2">
          <Label>Additional Notes</Label>
          <Textarea value={notes} onChange={e => setNotes(e.target.value)} rows={4} placeholder="Any other observations..." />
        </div>

        <Button onClick={handleSubmit} disabled={!recommendation || submitting} className="w-full">
          {submitting ? 'Submitting...' : 'Submit Feedback'}
        </Button>
      </CardContent>
    </Card>
  )
}
```

- [ ] **Step 3: Create interview detail page**

Create `frontend/src/pages/interviewer/InterviewDetail.tsx`:

```tsx
import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { loops as loopsApi, competencies as compApi, type Interview, type InterviewWithFeedback, type Competency } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { ExternalLink } from 'lucide-react'
import { useAuth } from '@/lib/auth'
import { interviews as ivApi } from '@/lib/api'
import FeedbackForm from './FeedbackForm'

export default function InterviewDetail() {
  const { id } = useParams<{ id: string }>()
  const interviewId = parseInt(id!)
  const { user } = useAuth()
  const [feedbackSubmitted, setFeedbackSubmitted] = useState(false)

  // First get the interview to find its loop
  const { data: myInterviews = [] } = useQuery({ queryKey: ['my-interviews'], queryFn: ivApi.listMine })
  const interview = myInterviews.find((iv: Interview) => iv.id === interviewId)

  // Then load the full loop detail (visibility is enforced server-side)
  const { data: loop, refetch: refetchLoop } = useQuery({
    queryKey: ['loops', interview?.loop_id],
    queryFn: () => loopsApi.get(interview!.loop_id),
    enabled: !!interview,
  })

  const { data: competenciesList = [] } = useQuery({ queryKey: ['competencies'], queryFn: compApi.list })

  if (!interview) return <div>Loading...</div>

  const myInterview = loop?.interviews?.find((iv: InterviewWithFeedback) => iv.id === interviewId)
  const hasFeedback = myInterview?.feedback != null || feedbackSubmitted

  const compMap = Object.fromEntries(competenciesList.map((c: Competency) => [c.id, c]))

  return (
    <div className="space-y-6">
      {/* Interview info */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>{interview.focus_area} Interview</span>
            <Badge variant={hasFeedback ? 'default' : 'outline'}>
              {hasFeedback ? 'Feedback Submitted' : 'Pending'}
            </Badge>
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {loop && <p><strong>Candidate:</strong> {loop.candidate.name} ({loop.candidate.email})</p>}
          <p><strong>Scheduled:</strong> {new Date(interview.scheduled_at).toLocaleString()}</p>
          {interview.video_link && (
            <p>
              <strong>Video: </strong>
              <a href={interview.video_link} target="_blank" rel="noopener" className="text-blue-600 inline-flex items-center gap-1">
                Join <ExternalLink className="h-3 w-3" />
              </a>
            </p>
          )}
          {interview.notes_for_interviewer && (
            <div>
              <strong>Notes from scheduler:</strong>
              <p className="text-sm text-gray-600 mt-1">{interview.notes_for_interviewer}</p>
            </div>
          )}
          {loop?.candidate.resume_url && (
            <p>
              <strong>Resume: </strong>
              <a href={loop.candidate.resume_url} target="_blank" rel="noopener" className="text-blue-600 inline-flex items-center gap-1">
                View <ExternalLink className="h-3 w-3" />
              </a>
            </p>
          )}
        </CardContent>
      </Card>

      {/* Feedback form or submitted feedback */}
      {!hasFeedback ? (
        <FeedbackForm
          interviewId={interviewId}
          competencies={competenciesList}
          onSubmitted={() => { setFeedbackSubmitted(true); refetchLoop() }}
        />
      ) : (
        <>
          {/* Show own feedback */}
          {myInterview?.feedback && (
            <Card>
              <CardHeader><CardTitle>Your Feedback</CardTitle></CardHeader>
              <CardContent className="space-y-2">
                <p><strong>Recommendation:</strong> {myInterview.feedback.recommendation.replace(/_/g, ' ')}</p>
                {myInterview.feedback.recommendation_reason && <p><strong>Reason:</strong> {myInterview.feedback.recommendation_reason}</p>}
                {myInterview.feedback.competency_ratings?.map(cr => (
                  <div key={cr.id} className="flex justify-between text-sm">
                    <span>{compMap[cr.competency_id]?.name || `Competency ${cr.competency_id}`}</span>
                    <span className="font-medium">{cr.rating_value}</span>
                  </div>
                ))}
                {myInterview.feedback.free_form_notes && (
                  <>
                    <Separator />
                    <p className="text-sm whitespace-pre-wrap">{myInterview.feedback.free_form_notes}</p>
                  </>
                )}
              </CardContent>
            </Card>
          )}

          {/* Show others' feedback (if visibility rule allows — server strips it otherwise) */}
          {loop?.interviews?.filter((iv: InterviewWithFeedback) => iv.id !== interviewId && iv.feedback).map((iv: InterviewWithFeedback) => (
            <Card key={iv.id}>
              <CardHeader><CardTitle className="text-base">{iv.focus_area} — {iv.interviewer_name}</CardTitle></CardHeader>
              <CardContent className="space-y-2">
                <p><strong>Recommendation:</strong> {iv.feedback!.recommendation.replace(/_/g, ' ')}</p>
                {iv.feedback!.recommendation_reason && <p className="text-sm">{iv.feedback!.recommendation_reason}</p>}
                {iv.feedback!.competency_ratings?.map(cr => (
                  <div key={cr.id} className="flex justify-between text-sm">
                    <span>{compMap[cr.competency_id]?.name || `Competency ${cr.competency_id}`}</span>
                    <span className="font-medium">{cr.rating_value}</span>
                  </div>
                ))}
                {iv.feedback!.free_form_notes && (
                  <>
                    <Separator />
                    <p className="text-sm whitespace-pre-wrap">{iv.feedback!.free_form_notes}</p>
                  </>
                )}
              </CardContent>
            </Card>
          ))}
        </>
      )}
    </div>
  )
}
```

- [ ] **Step 4: Update App.tsx routes**

In `frontend/src/App.tsx`, import and wire up the interviewer pages:

```tsx
import MyInterviews from '@/pages/interviewer/MyInterviews'
import InterviewDetail from '@/pages/interviewer/InterviewDetail'

// Replace the interviewer placeholder routes with:
//   <Route path="/my-interviews" element={<MyInterviews />} />
//   <Route path="/interviews/:id" element={<InterviewDetail />} />
```

- [ ] **Step 5: Verify in browser**

Log in as an interviewer. View my interviews, click into an interview detail, and submit feedback with competency ratings.

- [ ] **Step 6: Commit**

```bash
cd /home/zach/code/hire
git add frontend/src/pages/interviewer/ frontend/src/App.tsx
git commit -m "feat: interviewer pages — my interviews, detail, and feedback form"
```

---

## Phase 3: Integration

### Task 18: Seed data, build integration, and final verification

**Files:**
- Create: `seed/seed.go`
- Modify: `embed.go` (remove placeholder comment)

- [ ] **Step 1: Create the seed data program**

Create `seed/seed.go`:

```go
package main

import (
	"fmt"
	"hire/internal/api"
	"hire/internal/models"
	"hire/internal/store"
	"log"
	"os"
	"time"
)

func main() {
	dbPath := "hire.db"
	os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

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

- [ ] **Step 2: Run the seed**

```bash
cd /home/zach/code/hire
go run ./seed/seed.go
```

Expected: prints the demo accounts and "Seed data created successfully!"

- [ ] **Step 3: Run all Go tests**

```bash
go test ./internal/... -v
```

Expected: all tests PASS.

- [ ] **Step 4: Build the frontend**

```bash
cd /home/zach/code/hire/frontend
npm run build
```

Expected: builds to `frontend/dist/` without errors.

- [ ] **Step 5: Build the full binary**

```bash
cd /home/zach/code/hire
go build -o hire-server ./cmd/server
```

Expected: produces a `hire-server` binary.

- [ ] **Step 6: Test the full binary**

```bash
cd /home/zach/code/hire
./hire-server -db hire.db &
sleep 1
# Test login
curl -s -X POST http://localhost:8080/api/auth/login -H 'Content-Type: application/json' -d '{"email":"admin@hire.demo","password":"admin"}'
# Test frontend serving
curl -s http://localhost:8080/ | head -3
kill %1
```

Expected: login returns a JSON response with a token. Frontend serves HTML.

- [ ] **Step 7: Commit**

```bash
cd /home/zach/code/hire
echo "hire-server" >> .gitignore
echo "hire.db" >> .gitignore
echo "node_modules" >> .gitignore
git add seed/ .gitignore Makefile
git commit -m "feat: seed data, gitignore, and build integration"
```

- [ ] **Step 8: Final verification in browser**

```bash
cd /home/zach/code/hire
go run ./seed/seed.go
go run ./cmd/server &
cd frontend && npm run dev &
```

Open `http://localhost:5173` and verify the full flow:
1. Log in as `scheduler@hire.demo` / `scheduler` — see candidates, create a loop, add interviews
2. Log in as `alice@hire.demo` / `interviewer` — see assigned interviews, submit feedback with competency ratings
3. Log in as `admin@hire.demo` / `admin` — manage users and competencies
4. Log back in as scheduler — view debrief with all submitted feedback, record final decision
