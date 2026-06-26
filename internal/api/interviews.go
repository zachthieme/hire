package api

import (
	"hire/internal/models"
	"hire/internal/notify"
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
		iv.Status = "pending"
	}
	if err := h.store.CreateInterview(&iv); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	notify.InterviewAssigned(h.store, iv.InterviewerID, iv.ID, iv.FocusArea)
	writeJSON(w, http.StatusCreated, iv)
}

func (h *Handler) UpdateInterview(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetInterview(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "interview not found")
		return
	}
	var updates models.Interview
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	existing.InterviewerID = updates.InterviewerID
	existing.FocusArea = updates.FocusArea
	existing.ScheduledAt = updates.ScheduledAt
	existing.VideoLink = updates.VideoLink
	existing.NotesForInterviewer = updates.NotesForInterviewer
	if err := h.store.UpdateInterview(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
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
	if err := h.store.DeleteInterview(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListMyInterviews(w http.ResponseWriter, r *http.Request) {
	userID := UserID(r.Context())
	list, err := h.store.ListInterviewsByUser(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}
