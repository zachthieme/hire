package api

import (
	"hire/internal/models"
	"hire/internal/notify"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetFeedback(w http.ResponseWriter, r *http.Request) {
	interviewID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Check visibility for interviewers
	role := UserRole(r.Context())
	userID := UserID(r.Context())
	if role == "interviewer" {
		iv, err := h.store.GetInterview(interviewID)
		if err != nil {
			writeError(w, http.StatusNotFound, "interview not found")
			return
		}
		// If not their own interview, check if they've submitted feedback for this loop
		if iv.InterviewerID != userID {
			if !h.store.HasUserSubmittedFeedbackForLoop(iv.LoopID, userID) {
				writeError(w, http.StatusForbidden, "submit your feedback first")
				return
			}
		}
	}

	fb, err := h.store.GetFeedbackByInterview(interviewID)
	if err != nil {
		writeError(w, http.StatusNotFound, "feedback not found")
		return
	}
	writeJSON(w, http.StatusOK, fb)
}

func (h *Handler) CreateFeedback(w http.ResponseWriter, r *http.Request) {
	interviewID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Verify this interview belongs to the current user
	iv, err := h.store.GetInterview(interviewID)
	if err != nil {
		writeError(w, http.StatusNotFound, "interview not found")
		return
	}
	if iv.InterviewerID != UserID(r.Context()) && UserRole(r.Context()) == "interviewer" {
		writeError(w, http.StatusForbidden, "not your interview")
		return
	}

	var fb models.Feedback
	if err := readJSON(r, &fb); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateEnum(fb.Recommendation, "recommendation", []string{"strong_hire", "hire", "no_hire", "strong_no_hire"}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	fb.InterviewID = interviewID
	if err := h.store.CreateFeedback(&fb); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	loop, _ := h.store.GetLoop(iv.LoopID)
	if loop != nil {
		notify.FeedbackSubmitted(h.store, loop.CreatedBy, iv.LoopID, iv.FocusArea)
		notify.CheckDebriefReady(h.store, loop)
	}

	writeJSON(w, http.StatusCreated, fb)
}

func (h *Handler) UpdateFeedback(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetFeedback(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "feedback not found")
		return
	}

	// Authorization: verify the caller owns this feedback
	iv, err := h.store.GetInterview(existing.InterviewID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify ownership")
		return
	}
	if iv.InterviewerID != UserID(r.Context()) && UserRole(r.Context()) != "admin" {
		writeError(w, http.StatusForbidden, "not your feedback")
		return
	}

	var updates models.Feedback
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	existing.Recommendation = updates.Recommendation
	existing.RecommendationReason = updates.RecommendationReason
	existing.FreeFormNotes = updates.FreeFormNotes
	if err := h.store.UpdateFeedback(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, existing)
}
