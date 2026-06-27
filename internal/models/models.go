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

// Candidate statuses.
const (
	CandidateStatusActive    = "active"
	CandidateStatusHired     = "hired"
	CandidateStatusRejected  = "rejected"
	CandidateStatusWithdrawn = "withdrawn"
)

// ValidCandidateStatuses is the set of allowed candidate statuses.
var ValidCandidateStatuses = []string{CandidateStatusActive, CandidateStatusHired, CandidateStatusRejected, CandidateStatusWithdrawn}

// Interview loop statuses.
const (
	LoopStatusScheduling = "scheduling"
	LoopStatusActive     = "active"
	LoopStatusComplete   = "complete"
)

// ValidLoopStatuses is the set of allowed loop statuses.
var ValidLoopStatuses = []string{LoopStatusScheduling, LoopStatusActive, LoopStatusComplete}

// Interview statuses.
const (
	InterviewStatusPending  = "pending"
	InterviewStatusComplete = "complete"
)

// ValidInterviewStatuses is the set of allowed interview statuses.
var ValidInterviewStatuses = []string{InterviewStatusPending, InterviewStatusComplete}

// ValidLoopTransitions defines allowed status transitions for interview loops.
var ValidLoopTransitions = map[string][]string{
	LoopStatusScheduling: {LoopStatusActive},
	LoopStatusActive:     {LoopStatusComplete},
	LoopStatusComplete:   {}, // terminal
}

// ValidInterviewTransitions defines allowed status transitions for interviews.
var ValidInterviewTransitions = map[string][]string{
	InterviewStatusPending:  {InterviewStatusComplete},
	InterviewStatusComplete: {}, // terminal — feedback has been submitted
}

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
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type InterviewLoop struct {
	ID            int64     `json:"id"`
	CandidateID   int64     `json:"candidate_id"`
	Status        string    `json:"status"`
	FinalDecision *string   `json:"final_decision"`
	DebriefNotes  *string   `json:"debrief_notes"`
	CreatedBy     int64     `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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
	UpdatedAt           time.Time `json:"updated_at"`
}

type Feedback struct {
	ID                   int64              `json:"id"`
	InterviewID          int64              `json:"interview_id"`
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

// LoopDetail is the expanded view returned by GET /api/loops/:id.
type LoopDetail struct {
	InterviewLoop
	Candidate  Candidate               `json:"candidate"`
	Interviews []InterviewWithFeedback `json:"interviews"`
}

type InterviewWithFeedback struct {
	Interview
	InterviewerName string    `json:"interviewer_name"`
	Feedback        *Feedback `json:"feedback,omitempty"`
}
