# Interview Scheduling & Debrief App — Design Spec

## Purpose

A demo/pitch web application for managing the interview process. Replaces spreadsheets and email threads with a single tool where schedulers set up interview loops, interviewers view assignments and submit feedback, and everyone can see consolidated results at debrief.

## Users & Roles

| Role | Capabilities |
|------|-------------|
| **Admin** | Configure competencies and rating scales, manage users and roles |
| **Scheduler** | Create candidates, build interview loops, assign interviewers, view all feedback, record final hiring decisions |
| **Interviewer** | View assigned interviews, submit structured feedback, view others' feedback only after submitting their own |

A user has a single role. Auth is email/password with JWT tokens.

## Tech Stack

- **Backend:** Go with Chi router, SQLite
- **Frontend:** React (Vite), Tailwind CSS, shadcn/ui
- **Bundling:** Go `embed.FS` bundles the built React SPA into a single binary
- **Auth:** JWT (Bearer token in header)
- **Dev workflow:** Vite dev server proxies API requests to the Go backend. `make build` produces a single binary for demo.

## Data Model

### users

| Column | Type | Notes |
|--------|------|-------|
| id | INTEGER PK | |
| email | TEXT UNIQUE | |
| name | TEXT | |
| password_hash | TEXT | |
| role | TEXT | admin, scheduler, interviewer |
| created_at | DATETIME | |

### candidates

| Column | Type | Notes |
|--------|------|-------|
| id | INTEGER PK | |
| name | TEXT | |
| email | TEXT | |
| resume_url | TEXT | Link to resume |
| status | TEXT | active, hired, rejected, withdrawn |
| created_at | DATETIME | |

### interview_loops

| Column | Type | Notes |
|--------|------|-------|
| id | INTEGER PK | |
| candidate_id | INTEGER FK | References candidates |
| status | TEXT | scheduling, active, complete |
| final_decision | TEXT | Nullable. strong_hire, hire, no_hire, strong_no_hire |
| debrief_notes | TEXT | Nullable. Free-form notes from debrief |
| created_by | INTEGER FK | References users (scheduler) |
| created_at | DATETIME | |

### interviews

| Column | Type | Notes |
|--------|------|-------|
| id | INTEGER PK | |
| loop_id | INTEGER FK | References interview_loops |
| interviewer_id | INTEGER FK | References users |
| focus_area | TEXT | e.g., coding, system design, culture |
| scheduled_at | DATETIME | |
| video_link | TEXT | |
| notes_for_interviewer | TEXT | Notes from scheduler to interviewer |
| status | TEXT | pending, complete (set automatically when feedback is submitted) |
| created_at | DATETIME | |

### feedback

| Column | Type | Notes |
|--------|------|-------|
| id | INTEGER PK | |
| interview_id | INTEGER FK | References interviews (one-to-one) |
| recommendation | TEXT | strong_hire, hire, no_hire, strong_no_hire |
| recommendation_reason | TEXT | Why they chose this recommendation |
| free_form_notes | TEXT | |
| submitted_at | DATETIME | |

### competencies

| Column | Type | Notes |
|--------|------|-------|
| id | INTEGER PK | |
| name | TEXT | e.g., "Problem Solving", "Communication" |
| rating_type | TEXT | e.g., "levels", "stars" |
| ratings_json | TEXT | JSON config. Levels: `["Learning", "Owning", "Advising"]`. Stars: `{"min":1, "max":5}` |
| created_at | DATETIME | |

### competency_ratings

| Column | Type | Notes |
|--------|------|-------|
| id | INTEGER PK | |
| feedback_id | INTEGER FK | References feedback |
| competency_id | INTEGER FK | References competencies |
| rating_value | TEXT | The selected rating (e.g., "Owning" or "4") |

### notifications

| Column | Type | Notes |
|--------|------|-------|
| id | INTEGER PK | |
| user_id | INTEGER FK | References users |
| message | TEXT | |
| link | TEXT | In-app URL to navigate to |
| read | BOOLEAN | Default false |
| created_at | DATETIME | |

## API Endpoints

### Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/auth/login` | Email/password login, returns JWT |
| POST | `/api/auth/logout` | Invalidate session |

### Users (admin only)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/users` | List all users |
| POST | `/api/users` | Create user |
| PUT | `/api/users/:id` | Update user/role |
| DELETE | `/api/users/:id` | Delete user |

### Candidates (scheduler)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/candidates` | List candidates |
| POST | `/api/candidates` | Create candidate |
| GET | `/api/candidates/:id` | Get candidate detail |
| PUT | `/api/candidates/:id` | Update candidate |
| DELETE | `/api/candidates/:id` | Delete candidate |

### Interview Loops (scheduler)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/loops` | List loops (filterable by candidate, status) |
| POST | `/api/loops` | Create loop for a candidate |
| GET | `/api/loops/:id` | Get loop with its interviews and feedback (enforces feedback visibility rule for interviewers) |
| PUT | `/api/loops/:id` | Update loop (status, final decision, debrief notes) |
| DELETE | `/api/loops/:id` | Delete loop |

### Interviews (scheduler)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/loops/:id/interviews` | Add interview to a loop |
| PUT | `/api/interviews/:id` | Update interview details |
| DELETE | `/api/interviews/:id` | Remove interview from loop |

### Feedback (interviewer)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/interviews/:id/feedback` | Get feedback for an interview |
| POST | `/api/interviews/:id/feedback` | Submit feedback |
| PUT | `/api/feedback/:id` | Update feedback |

### Competencies (admin)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/competencies` | List all competencies |
| POST | `/api/competencies` | Create competency with rating config |
| PUT | `/api/competencies/:id` | Update competency |
| DELETE | `/api/competencies/:id` | Delete competency |

### Notifications

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/notifications` | List notifications for current user |
| PUT | `/api/notifications/:id/read` | Mark notification as read |

### My Interviews (interviewer)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/me/interviews` | List current user's assigned interviews |

## Frontend Pages

### Shared Components

- **Login page** — email/password form
- **Top nav** — app name, notification bell with unread count badge, user menu with role indicator
- **Notification dropdown** — list of notifications, click to navigate, mark as read

### Interviewer Views

- **My Interviews (dashboard)** — list of upcoming and past interviews showing candidate name, focus area, time, status (pending / feedback submitted). Click to open detail.
- **Interview Detail** — candidate info (name, resume link), focus area, video link, scheduler notes. "Submit Feedback" button.
  - **Before feedback submitted:** Only shows own interview info and the feedback form. No visibility into other interviewers' feedback.
  - **After feedback submitted:** Unlocks read-only view of all submitted feedback from the loop for debrief preparation.
- **Feedback Form** — hire recommendation (radio: strong hire / hire / no hire / strong no hire) with reason text field, competency ratings (dynamically rendered from admin config), free-form notes textarea.

### Scheduler Views

- **Candidates (dashboard)** — list of all candidates with status. Click to open detail.
- **Candidate Detail** — candidate info, interview loop with all interviews listed (interviewer, focus area, time, feedback status). Button to create/edit the loop.
- **Loop Editor** — add/remove interviews to a loop. Per interview: assign interviewer (dropdown), focus area, scheduled time, video link, notes for interviewer. Saving triggers notifications to assigned interviewers.
- **Debrief View** — all submitted feedback displayed side by side (recommendation, competency ratings, notes). Final Decision selector (strong hire / hire / no hire / strong no hire) and Debrief Notes text field. Accessible by scheduler at any time; shows warning if not all feedback is submitted yet. Saving updates the loop record.

### Admin Views

- **Competency Management** — list competencies with rating type. Add/edit/delete. When editing, configure rating type (levels or stars) and the specific options.
- **User Management** — list users, create/edit/delete, assign roles.

## Feedback Visibility Rule

An interviewer cannot see other interviewers' feedback for the same candidate/loop until they have submitted their own feedback (including the hire recommendation). This prevents bias. After submission, they gain read-only access to all submitted feedback for debrief preparation.

## Notifications

Notifications are in-app only. They are created when:

- A scheduler assigns an interviewer to an interview (notifies the interviewer)
- An interviewer submits feedback (notifies the scheduler who created the loop)
- All feedback for a loop is submitted (notifies the scheduler that debrief is ready)

## Project Structure

```
hire/
├── cmd/
│   └── server/
│       └── main.go            # Entry point, starts HTTP server, serves API + SPA
├── internal/
│   ├── api/                   # HTTP handlers, routes, middleware, auth
│   ├── models/                # Data types (Candidate, Interview, Feedback, etc.)
│   ├── store/                 # SQLite data access layer
│   └── notify/                # In-app notification logic
├── frontend/
│   ├── src/
│   │   ├── components/        # Reusable UI components (shadcn/ui based)
│   │   ├── pages/             # Route-level page components
│   │   ├── hooks/             # Custom React hooks
│   │   ├── lib/               # API client, utilities
│   │   └── App.tsx
│   ├── index.html
│   ├── vite.config.ts
│   ├── tailwind.config.ts
│   └── package.json
├── migrations/                # SQLite schema migrations (SQL files)
├── Makefile                   # Build frontend, embed in Go binary, compile
└── go.mod
```
