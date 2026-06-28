package api

import (
	"errors"
	"net/http"
	"strconv"

	"hire/internal/models"
	"hire/internal/notify"
	"hire/internal/store"

	"github.com/go-chi/chi/v5"
)

// CreateStage. Route: POST /api/applications/{id}/stages
func (h *Handler) CreateStage(w http.ResponseWriter, r *http.Request) {
	appID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid application id")
		return
	}
	var st models.Stage
	if err := readJSON(r, &st); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	st.ApplicationID = appID
	if err := validateEnum(st.Type, "type", models.ValidStageTypes); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if st.Status == "" {
		st.Status = models.StageStatusPending
	}
	if err := validateEnum(st.Status, "status", models.ValidStageStatuses); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if st.ScheduledAt.IsZero() {
		writeError(w, http.StatusBadRequest, "scheduled_at is required")
		return
	}
	if err := h.store.CreateStage(r.Context(), &st); err != nil {
		if code, ok := pgConstraintStatus(err); ok && code == http.StatusBadRequest {
			writeError(w, http.StatusBadRequest, "application not found")
			return
		}
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, st)
}

func (h *Handler) UpdateStage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetStage(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "stage not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	var updates models.Stage
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateEnum(updates.Type, "type", models.ValidStageTypes); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateEnum(updates.Status, "status", models.ValidStageStatuses); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	existing.Type = updates.Type
	existing.FocusArea = updates.FocusArea
	existing.ScheduledAt = updates.ScheduledAt
	existing.VideoLink = updates.VideoLink
	existing.NotesForInterviewer = updates.NotesForInterviewer
	existing.Status = updates.Status
	if err := h.store.UpdateStage(r.Context(), existing); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteStage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteStage(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "stage not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AddStageInterviewer. Route: POST /api/stages/{id}/interviewers
func (h *Handler) AddStageInterviewer(w http.ResponseWriter, r *http.Request) {
	stageID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage id")
		return
	}
	var body struct {
		InterviewerID int64 `json:"interviewer_id"`
	}
	if err := readJSON(r, &body); err != nil || body.InterviewerID == 0 {
		writeError(w, http.StatusBadRequest, "interviewer_id is required")
		return
	}
	if err := h.store.AddStageInterviewer(r.Context(), stageID, body.InterviewerID); err != nil {
		if code, ok := pgConstraintStatus(err); ok && code == http.StatusBadRequest {
			writeError(w, http.StatusBadRequest, "interviewer not found")
			return
		}
		writeInternalError(w, r, err)
		return
	}
	st, err := h.store.GetStage(r.Context(), stageID)
	if err == nil {
		notify.StageAssigned(r.Context(), h.store, body.InterviewerID, stageID, st.Type)
	}
	w.WriteHeader(http.StatusNoContent)
}

// RemoveStageInterviewer. Route: DELETE /api/stages/{id}/interviewers/{uid}
func (h *Handler) RemoveStageInterviewer(w http.ResponseWriter, r *http.Request) {
	stageID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage id")
		return
	}
	uid, err := strconv.ParseInt(chi.URLParam(r, "uid"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid interviewer id")
		return
	}
	if err := h.store.RemoveStageInterviewer(r.Context(), stageID, uid); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not assigned")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListMyStages. Route: GET /api/me/stages
func (h *Handler) ListMyStages(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	stages, err := h.store.ListStagesByUser(r.Context(), UserID(r.Context()), limit, offset)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, stages)
}
