package notify

import (
	"context"
	"fmt"
	"hire/internal/models"
	"log/slog"
)

// Notifier defines the store operations needed by the notification helpers.
type Notifier interface {
	CreateNotification(ctx context.Context, n *models.Notification) error
	CountIncompleteInterviews(ctx context.Context, loopID int64) (int, error)
}

func InterviewAssigned(ctx context.Context, s Notifier, interviewerID, interviewID int64, focusArea string) {
	if err := s.CreateNotification(ctx, &models.Notification{
		UserID:  interviewerID,
		Message: fmt.Sprintf("You've been assigned a %s interview", focusArea),
		Link:    fmt.Sprintf("/interviews/%d", interviewID),
	}); err != nil {
		slog.ErrorContext(ctx, "failed to create interview-assigned notification",
			"error", err, "interviewer_id", interviewerID, "interview_id", interviewID)
	}
}

func FeedbackSubmitted(ctx context.Context, s Notifier, schedulerID, loopID int64, focusArea string) {
	if err := s.CreateNotification(ctx, &models.Notification{
		UserID:  schedulerID,
		Message: fmt.Sprintf("Feedback submitted for %s interview", focusArea),
		Link:    fmt.Sprintf("/loops/%d/debrief", loopID),
	}); err != nil {
		slog.ErrorContext(ctx, "failed to create feedback-submitted notification",
			"error", err, "scheduler_id", schedulerID, "loop_id", loopID)
	}
}

func CheckDebriefReady(ctx context.Context, s Notifier, loop *models.InterviewLoop) {
	incomplete, err := s.CountIncompleteInterviews(ctx, loop.ID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to count incomplete interviews",
			"error", err, "loop_id", loop.ID)
		return
	}
	if incomplete > 0 {
		return
	}
	if err := s.CreateNotification(ctx, &models.Notification{
		UserID:  loop.CreatedBy,
		Message: "All feedback submitted — ready for debrief",
		Link:    fmt.Sprintf("/loops/%d/debrief", loop.ID),
	}); err != nil {
		slog.ErrorContext(ctx, "failed to create debrief-ready notification",
			"error", err, "loop_id", loop.ID)
	}
}
