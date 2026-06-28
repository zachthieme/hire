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

	// Feedback
	CreateFeedback(ctx context.Context, fb *models.Feedback) (appReady bool, applicationID int64, err error)
	GetFeedback(ctx context.Context, id int64) (*models.Feedback, error)
	GetFeedbackByStageAndInterviewer(ctx context.Context, stageID, interviewerID int64) (*models.Feedback, error)
	ListFeedbackByStage(ctx context.Context, stageID int64) ([]*models.Feedback, error)
	UpdateFeedback(ctx context.Context, fb *models.Feedback) error

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
}
