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
