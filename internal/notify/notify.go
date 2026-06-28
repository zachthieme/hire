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
}

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
