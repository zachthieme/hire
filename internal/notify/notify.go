package notify

import (
	"context"
	"fmt"
	"hire/internal/models"
	"hire/internal/store"
)

func InterviewAssigned(ctx context.Context, s *store.Store, interviewerID, interviewID int64, focusArea string) {
	s.CreateNotification(ctx, &models.Notification{
		UserID:  interviewerID,
		Message: fmt.Sprintf("You've been assigned a %s interview", focusArea),
		Link:    fmt.Sprintf("/interviews/%d", interviewID),
	})
}

func FeedbackSubmitted(ctx context.Context, s *store.Store, schedulerID, loopID int64, focusArea string) {
	s.CreateNotification(ctx, &models.Notification{
		UserID:  schedulerID,
		Message: fmt.Sprintf("Feedback submitted for %s interview", focusArea),
		Link:    fmt.Sprintf("/loops/%d/debrief", loopID),
	})
}

func CheckDebriefReady(ctx context.Context, s *store.Store, loop *models.InterviewLoop) {
	interviews, err := s.ListInterviewsByLoop(ctx, loop.ID)
	if err != nil || len(interviews) == 0 {
		return
	}
	for _, iv := range interviews {
		if iv.Status != "complete" {
			return
		}
	}
	s.CreateNotification(ctx, &models.Notification{
		UserID:  loop.CreatedBy,
		Message: "All feedback submitted — ready for debrief",
		Link:    fmt.Sprintf("/loops/%d/debrief", loop.ID),
	})
}
