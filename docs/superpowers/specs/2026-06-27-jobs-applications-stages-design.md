# Jobs, Applications & Stages — Data Model Remodel

**Date:** 2026-06-27
**Status:** Approved design, pending implementation plan

## Problem

Today the model is `Candidate → Loop → Interview → Feedback`, where a `Loop`
is tied to a single candidate and has no concept of the *job* being hired for.
There is no top-level "open req" entity, a candidate can only belong to one
loop, and an interview supports exactly one interviewer.

We need:

- **Jobs** (open reqs) that schedulers create, with metadata (description,
  hiring manager, etc.).
- A candidate associated with a job, and the same candidate able to apply to
  multiple jobs.
- Per candidate-per-job, a series of **stages** (a phone screen, an interview).
- A stage that can have **more than one interviewer**, each filing **their own
  feedback**.
- A **final decision** + **final interview notes** captured per candidate-job.

## New Model

```
Job ──< Application >── Candidate          (candidate↔job is many-to-many)
            │
            └──< Stage ──< StageInterviewer ──── Feedback ──< CompetencyRating
```

### Job — the open requisition
| field | type | notes |
|-------|------|-------|
| id | int64 PK | |
| title | text | required |
| description | text | required |
| hiring_manager | text | free text (name/email) for v1 — hiring managers are not app users. Can become an FK later. |
| status | enum | `open` \| `closed` \| `filled` |
| created_by | int64 FK users | |
| created_at / updated_at | timestamptz | |

Created/edited by **scheduler** and **admin**.

### Application — a candidate's run at one job (candidate↔job link)
| field | type | notes |
|-------|------|-------|
| id | int64 PK | |
| job_id | int64 FK jobs | |
| candidate_id | int64 FK candidates | |
| status | enum | `active` \| `rejected` \| `hired` \| `withdrawn` |
| final_decision | enum nullable | `strong_hire` \| `hire` \| `no_hire` \| `strong_no_hire` (overall outcome; same 4 levels as per-interview recommendations, set once for the application) |
| final_interview_notes | text nullable | closing summary across all stages |
| created_by | int64 FK users | |
| created_at / updated_at | timestamptz | |

`UNIQUE(job_id, candidate_id)` — a candidate can have many applications across
jobs, but not be duplicated on a single job.

This absorbs the old `loop.final_decision` and `loop.debrief_notes`. The
per-process **status moves here off the candidate**: a candidate can be `hired`
on one job and `active` on another. `Candidate` becomes a pure person record
(name, email, resume) — its `status` column is dropped.

### Stage — one step in an application (was Interview)
| field | type | notes |
|-------|------|-------|
| id | int64 PK | |
| application_id | int64 FK applications | |
| type | enum | `phone_screen` \| `interview` |
| focus_area | text nullable | optional topic label, e.g. "Coding" |
| scheduled_at | timestamptz | |
| video_link | text | |
| notes_for_interviewer | text | |
| status | enum | `pending` \| `complete` \| `canceled` |
| created_at / updated_at | timestamptz | |

### StageInterviewer — join, supports multiple interviewers per stage
| field | type | notes |
|-------|------|-------|
| id | int64 PK | |
| stage_id | int64 FK stages | |
| interviewer_id | int64 FK users | |

`UNIQUE(stage_id, interviewer_id)`.

> Panel mechanics (same-room aggregation) are out of scope. Each interviewer's
> involvement is modeled independently; multiple interviewers each file their
> own feedback.

### Feedback — one per (stage, interviewer)
Repoint the existing `feedback` table from `interview_id` to
`(stage_id, interviewer_id)` with `UNIQUE(stage_id, interviewer_id)`. Fields
otherwise unchanged: `recommendation`, `recommendation_reason`,
`free_form_notes`, timestamps. `competency_ratings` is unchanged (still keyed by
`feedback_id`).

Each interviewer on a stage files their own feedback; the application's debrief
shows them side by side.

## API Surface

**Jobs** — scheduler/admin write, any authenticated user reads
- `GET  /api/jobs` — list (paginated)
- `POST /api/jobs` — create
- `GET  /api/jobs/{id}` — job + its applications (candidate + status summary)
- `PUT  /api/jobs/{id}` — update metadata/status
- `DELETE /api/jobs/{id}`

**Applications**
- `POST   /api/jobs/{id}/applications` — add a candidate to a job `{candidate_id}`
- `GET    /api/applications/{id}` — application + candidate + stages + every
  interviewer's feedback (the debrief view)
- `PUT    /api/applications/{id}` — status, final_decision, final_interview_notes
- `DELETE /api/applications/{id}`

**Stages**
- `POST   /api/applications/{id}/stages` — create stage
- `PUT    /api/stages/{id}` / `DELETE /api/stages/{id}`
- `POST   /api/stages/{id}/interviewers` `{interviewer_id}` — assign interviewer
- `DELETE /api/stages/{id}/interviewers/{interviewerId}` — unassign

**Feedback** — interviewers file their own
- `GET  /api/stages/{id}/feedback` — all interviewers' feedback for the stage
- `POST /api/stages/{id}/feedback` — current user's feedback for the stage
- `PUT  /api/feedback/{id}` — edit own

**Interviewer**
- `GET /api/me/stages` — stages the current user is assigned to (replaces
  `/api/me/interviews`)

**Unchanged**: `candidates` CRUD (minus its `status` field), `users`,
`competencies`, `notifications`, `auth`.

### Authorization
- Job/Application/Stage create-update-delete: **scheduler**, **admin**.
- Stage feedback: the assigned **interviewer** for that stage (and admin).
- Reads: any authenticated user (as today).

### Notifications (new/changed)
- **StageAssigned** — to each interviewer when added to a stage.
- **FeedbackSubmitted** — to the application's creator when an interviewer
  submits.
- **ReadyForDecision** — to the application's creator when every stage on the
  application is `complete` and all assigned interviewers have submitted
  feedback (i.e. the application is ready for a final decision).

## Migration & Seed

Only demo data exists (pre-production), so a single clean migration is
acceptable.

Migration `000006_jobs_applications_stages`:
- Create `jobs`, `applications`, `stages`, `stage_interviewers`.
- Reshape `feedback` to reference `(stage_id, interviewer_id)`; drop its
  `interview_id` column and add the new unique constraint.
- Drop `loops` and `interviews`.
- Drop `candidates.status`.
- `.down.sql` reverses (recreate loops/interviews, restore candidate.status,
  revert feedback FK, drop new tables).

**Existing demo interview/loop/feedback rows are discarded.** Rewrite
`seed/seed.go` to produce: a few Jobs → Applications (candidates on jobs) →
Stages (phone_screen + interview) → assigned interviewers → some submitted
feedback, so every screen has data on first run.

## Frontend Impact

- **Scheduler**
  - **Jobs** list (new) and **Job detail** — metadata + the job's candidate
    applications + "add candidate to job".
  - **Application detail** (new debrief) — stages, add stage, assign multiple
    interviewers per stage, set final decision + final interview notes, view
    each interviewer's feedback.
- **Interviewer**
  - **"My Interviews"** page (label kept, friendly) listing assigned stages
    (phone screen or interview) → Stage detail + the interviewer's own feedback
    form.
- **Nav** gains **Jobs** for schedulers; the Candidates page stays (person
  records).
- **Routes**: `/jobs`, `/jobs/:id`, `/applications/:id`, `/stages/:id`,
  `/my-interviews` (label "My Interviews", backed by `/api/me/stages`).
- API client (`frontend/src/lib/api.ts`): add `jobs`, `applications`, `stages`
  modules; rename `interviews`/`loops` usage; keep the `requestList` null→[]
  coalescing for all new list endpoints.

## Build Phasing (for the implementation plan)

1. **Backend model** — migration `000006`, `models.go` structs (Job,
   Application, Stage, StageInterviewer, reshaped Feedback), store layer CRUD.
2. **Backend API** — routes, authorization, notifications, handler + store
   tests.
3. **Frontend scheduler** — Jobs list/detail, Application detail, stage +
   interviewer assignment.
4. **Frontend interviewer** — My Interviews (stages), Stage detail, feedback.
5. **Seed rewrite + full end-to-end run-through.**

## Out of Scope (v1)

- Panel/same-room aggregated feedback.
- Hiring manager as an app user / role.
- Per-job interview-plan templates (stages are created ad hoc per application).
- Scheduling/calendar integration beyond the existing `scheduled_at` + video
  link fields.
