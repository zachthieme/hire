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

func (h *Handler) CreateInterview(w http.ResponseWriter, r *http.Request) {
	loopID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid loop id")
		return
	}
	var iv models.Interview
	if err := readJSON(r, &iv); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateRequired(map[string]string{"focus_area": iv.FocusArea}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if iv.InterviewerID == 0 {
		writeError(w, http.StatusBadRequest, "interviewer_id is required")
		return
	}
	iv.LoopID = loopID
	if iv.Status == "" {
		iv.Status = models.InterviewStatusPending
	}
	if err := validateEnum(iv.Status, "status", models.ValidInterviewStatuses); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.CreateInterview(r.Context(), &iv); err != nil {
		writeInternalError(w, r, err)
		return
	}
	notify.InterviewAssigned(r.Context(), h.store, iv.InterviewerID, iv.ID, iv.FocusArea)
	writeJSON(w, http.StatusCreated, iv)
}

func (h *Handler) UpdateInterview(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetInterview(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "interview not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	var updates models.Interview
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if updates.InterviewerID != 0 {
		existing.InterviewerID = updates.InterviewerID
	}
	if updates.FocusArea != "" {
		existing.FocusArea = updates.FocusArea
	}
	existing.ScheduledAt = updates.ScheduledAt
	existing.VideoLink = updates.VideoLink
	existing.NotesForInterviewer = updates.NotesForInterviewer
	if err := h.store.UpdateInterview(r.Context(), existing); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteInterview(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteInterview(r.Context(), id); err != nil {
		writeInternalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListMyInterviews(w http.ResponseWriter, r *http.Request) {
	userID := UserID(r.Context())
	limit, offset := parsePagination(r)
	list, err := h.store.ListInterviewsByUser(r.Context(), userID, limit, offset)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}
