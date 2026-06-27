package api

import (
	"errors"
	"hire/internal/models"
	"hire/internal/notify"
	"hire/internal/store"
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
		iv, err := h.store.GetInterview(r.Context(), interviewID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "interview not found")
			} else {
				writeInternalError(w, err)
			}
			return
		}
		// If not their own interview, check if they've submitted feedback for this loop
		if iv.InterviewerID != userID {
			submitted, err := h.store.HasUserSubmittedFeedbackForLoop(r.Context(), iv.LoopID, userID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "internal error")
				return
			}
			if !submitted {
				writeError(w, http.StatusForbidden, "submit your feedback first")
				return
			}
		}
	}

	fb, err := h.store.GetFeedbackByInterview(r.Context(), interviewID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "feedback not found")
		} else {
			writeInternalError(w, err)
		}
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
	iv, err := h.store.GetInterview(r.Context(), interviewID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "interview not found")
		} else {
			writeInternalError(w, err)
		}
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
	if err := h.store.CreateFeedback(r.Context(), &fb); err != nil {
		writeInternalError(w, err)
		return
	}

	loop, _ := h.store.GetLoop(r.Context(), iv.LoopID)
	if loop != nil {
		notify.FeedbackSubmitted(r.Context(), h.store, loop.CreatedBy, iv.LoopID, iv.FocusArea)
		notify.CheckDebriefReady(r.Context(), h.store, loop)
	}

	writeJSON(w, http.StatusCreated, fb)
}

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
			writeInternalError(w, err)
		}
		return
	}

	// Authorization: verify the caller owns this feedback
	iv, err := h.store.GetInterview(r.Context(), existing.InterviewID)
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
	if updates.Recommendation != "" {
		if err := validateEnum(updates.Recommendation, "recommendation", []string{"strong_hire", "hire", "no_hire", "strong_no_hire"}); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	existing.Recommendation = updates.Recommendation
	existing.RecommendationReason = updates.RecommendationReason
	existing.FreeFormNotes = updates.FreeFormNotes
	if err := h.store.UpdateFeedback(r.Context(), existing); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
		} else {
			writeInternalError(w, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, existing)
}
