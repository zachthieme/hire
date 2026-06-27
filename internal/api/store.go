package api

import (
	"context"
	"hire/internal/models"
)

// Store defines the data access methods required by the API handlers.
type Store interface {
	// Users
	CreateUser(ctx context.Context, u *models.User) error
	GetUserByID(ctx context.Context, id int64) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	ListUsers(ctx context.Context, limit, offset int) ([]*models.User, error)
	UpdateUser(ctx context.Context, u *models.User) error
	UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error
	DeleteUser(ctx context.Context, id int64) error

	// Candidates
	CreateCandidate(ctx context.Context, c *models.Candidate) error
	GetCandidate(ctx context.Context, id int64) (*models.Candidate, error)
	ListCandidates(ctx context.Context, limit, offset int) ([]*models.Candidate, error)
	UpdateCandidate(ctx context.Context, c *models.Candidate) error
	DeleteCandidate(ctx context.Context, id int64) error

	// Interview Loops
	CreateLoop(ctx context.Context, l *models.InterviewLoop) error
	GetLoop(ctx context.Context, id int64) (*models.InterviewLoop, error)
	GetLoopDetail(ctx context.Context, id int64) (*models.LoopDetail, error)
	ListLoops(ctx context.Context, candidateID *int64, status *string, limit, offset int) ([]*models.InterviewLoop, error)
	UpdateLoop(ctx context.Context, l *models.InterviewLoop) error
	DeleteLoop(ctx context.Context, id int64) error

	// Interviews
	CreateInterview(ctx context.Context, iv *models.Interview) error
	GetInterview(ctx context.Context, id int64) (*models.Interview, error)
	ListInterviewsByLoop(ctx context.Context, loopID int64, limit, offset int) ([]*models.Interview, error)
	ListInterviewsByUser(ctx context.Context, userID int64, limit, offset int) ([]*models.Interview, error)
	UpdateInterview(ctx context.Context, iv *models.Interview) error
	DeleteInterview(ctx context.Context, id int64) error

	// Feedback
	CreateFeedback(ctx context.Context, fb *models.Feedback) error
	GetFeedback(ctx context.Context, id int64) (*models.Feedback, error)
	GetFeedbackByInterview(ctx context.Context, interviewID int64) (*models.Feedback, error)
	UpdateFeedback(ctx context.Context, fb *models.Feedback) error
	HasUserSubmittedFeedbackForLoop(ctx context.Context, loopID, userID int64) (bool, error)

	// Competencies
	CreateCompetency(ctx context.Context, c *models.Competency) error
	GetCompetency(ctx context.Context, id int64) (*models.Competency, error)
	ListCompetencies(ctx context.Context, limit, offset int) ([]*models.Competency, error)
	UpdateCompetency(ctx context.Context, c *models.Competency) error
	DeleteCompetency(ctx context.Context, id int64) error

	// Notifications
	CreateNotification(ctx context.Context, n *models.Notification) error
	ListNotificationsByUser(ctx context.Context, userID int64, limit, offset int) ([]*models.Notification, error)
	MarkNotificationRead(ctx context.Context, id, userID int64) error
	CountUnreadNotifications(ctx context.Context, userID int64) (int, error)

	// Aggregate queries
	CountIncompleteInterviews(ctx context.Context, loopID int64) (int, error)
}
