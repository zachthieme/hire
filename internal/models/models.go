package models

import "time"

// User roles.
const (
	RoleAdmin       = "admin"
	RoleScheduler   = "scheduler"
	RoleInterviewer = "interviewer"
)

// ValidRoles is the set of allowed user roles.
var ValidRoles = []string{RoleAdmin, RoleScheduler, RoleInterviewer}

// Feedback recommendations.
const (
	RecommendationStrongHire   = "strong_hire"
	RecommendationHire         = "hire"
	RecommendationNoHire       = "no_hire"
	RecommendationStrongNoHire = "strong_no_hire"
)

// ValidRecommendations is the set of allowed feedback recommendations.
var ValidRecommendations = []string{RecommendationStrongHire, RecommendationHire, RecommendationNoHire, RecommendationStrongNoHire}

// Competency rating types.
const (
	RatingTypeLevels = "levels"
	RatingTypeStars  = "stars"
)

// ValidRatingTypes is the set of allowed rating types.
var ValidRatingTypes = []string{RatingTypeLevels, RatingTypeStars}

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

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Candidate struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	ResumeURL string    `json:"resume_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

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

type Competency struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	RatingType  string    `json:"rating_type"`
	RatingsJSON string    `json:"ratings_json"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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

// JobDetail is a job plus its candidate applications.
type JobDetail struct {
	Job
	Applications []ApplicationSummary `json:"applications"`
}

// ApplicationSummary is an application enriched with the candidate's name/email.
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
	CandidateName string `json:"candidate_name"`
	JobTitle      string `json:"job_title"`
	HasMyFeedback bool   `json:"has_my_feedback"`
}
