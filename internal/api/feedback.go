package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"hire/internal/models"
	"hire/internal/notify"
	"hire/internal/store"

	"github.com/go-chi/chi/v5"
)

// GetStageFeedback. Route: GET /api/stages/{id}/feedback — all interviewers'.
func (h *Handler) GetStageFeedback(w http.ResponseWriter, r *http.Request) {
	stageID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	list, err := h.store.ListFeedbackByStage(r.Context(), stageID)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	// Bias prevention: an interviewer who has not yet submitted their own
	// feedback for this stage sees only their own (an empty list until they submit).
	if UserRole(r.Context()) == models.RoleInterviewer {
		uid := UserID(r.Context())
		submitted := false
		for _, f := range list {
			if f.InterviewerID == uid {
				submitted = true
				break
			}
		}
		if !submitted {
			own := make([]*models.Feedback, 0)
			for _, f := range list {
				if f.InterviewerID == uid {
					own = append(own, f)
				}
			}
			list = own
		}
	}
	writeJSON(w, http.StatusOK, list)
}

// CreateFeedback. Route: POST /api/stages/{id}/feedback — current user's.
func (h *Handler) CreateFeedback(w http.ResponseWriter, r *http.Request) {
	stageID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	userID := UserID(r.Context())

	// Only an assigned interviewer (or admin) may submit.
	if UserRole(r.Context()) == models.RoleInterviewer {
		ok, err := h.store.IsStageInterviewer(r.Context(), stageID, userID)
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		if !ok {
			writeError(w, http.StatusForbidden, "not your stage")
			return
		}
	}

	var fb models.Feedback
	if err := readJSON(r, &fb); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateEnum(fb.Recommendation, "recommendation", models.ValidRecommendations); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	fb.StageID = stageID
	fb.InterviewerID = userID

	ready, applicationID, err := h.store.CreateFeedback(r.Context(), &fb)
	if err != nil {
		if st, ok := pgConstraintStatus(err); ok {
			msg := "feedback already submitted; use update"
			if st == http.StatusBadRequest {
				msg = "stage not found"
			}
			writeError(w, st, msg)
			return
		}
		writeInternalError(w, r, err)
		return
	}

	app, err := h.store.GetApplication(r.Context(), applicationID)
	if err != nil {
		slog.ErrorContext(r.Context(), "load application for notification", "error", err, "application_id", applicationID)
	} else {
		st, _ := h.store.GetStage(r.Context(), stageID)
		stageType := models.StageTypeInterview
		if st != nil {
			stageType = st.Type
		}
		notify.FeedbackSubmitted(r.Context(), h.store, app.CreatedBy, applicationID, stageType)
		if ready {
			notify.ReadyForDecision(r.Context(), h.store, app.CreatedBy, applicationID)
		}
	}
	writeJSON(w, http.StatusCreated, fb)
}

// UpdateFeedback. Route: PUT /api/feedback/{id} — edit own.
func (h *Handler) UpdateFeedback(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetFeedback(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "feedback not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	if existing.InterviewerID != UserID(r.Context()) && UserRole(r.Context()) == models.RoleInterviewer {
		writeError(w, http.StatusForbidden, "not your feedback")
		return
	}
	var updates models.Feedback
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateEnum(updates.Recommendation, "recommendation", models.ValidRecommendations); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	existing.Recommendation = updates.Recommendation
	existing.RecommendationReason = updates.RecommendationReason
	existing.FreeFormNotes = updates.FreeFormNotes
	existing.CompetencyRatings = updates.CompetencyRatings
	if err := h.store.UpdateFeedback(r.Context(), existing); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, existing)
}
