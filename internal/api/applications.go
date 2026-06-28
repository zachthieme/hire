package api

import (
	"errors"
	"net/http"
	"strconv"

	"hire/internal/models"
	"hire/internal/store"

	"github.com/go-chi/chi/v5"
)

// CreateApplication adds a candidate to a job. Route: POST /api/jobs/{id}/applications
func (h *Handler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	jobID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}
	var body struct {
		CandidateID int64 `json:"candidate_id"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.CandidateID == 0 {
		writeError(w, http.StatusBadRequest, "candidate_id is required")
		return
	}
	app := models.Application{
		JobID:       jobID,
		CandidateID: body.CandidateID,
		Status:      models.ApplicationStatusActive,
		CreatedBy:   UserID(r.Context()),
	}
	if err := h.store.CreateApplication(r.Context(), &app); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, app)
}

func (h *Handler) GetApplicationDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	detail, err := h.store.GetApplicationDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "application not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}

	// Interviewer feedback visibility: an interviewer only sees others' feedback
	// on a stage once they've submitted their own for that stage.
	if UserRole(r.Context()) == models.RoleInterviewer {
		userID := UserID(r.Context())
		for si := range detail.Stages {
			submitted := false
			for _, p := range detail.Stages[si].Participants {
				if p.InterviewerID == userID && p.Feedback != nil {
					submitted = true
				}
			}
			if !submitted {
				for pi := range detail.Stages[si].Participants {
					if detail.Stages[si].Participants[pi].InterviewerID != userID {
						detail.Stages[si].Participants[pi].Feedback = nil
					}
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) UpdateApplication(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetApplication(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "application not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	var updates models.Application
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateEnum(updates.Status, "status", models.ValidApplicationStatuses); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if updates.Status != existing.Status {
		if err := validateTransition(existing.Status, updates.Status, "application", models.ValidApplicationTransitions); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if updates.FinalDecision != nil {
		if err := validateEnum(*updates.FinalDecision, "final_decision", models.ValidRecommendations); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	existing.Status = updates.Status
	existing.FinalDecision = updates.FinalDecision
	existing.FinalInterviewNotes = updates.FinalInterviewNotes
	if err := h.store.UpdateApplication(r.Context(), existing); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteApplication(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteApplication(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "application not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
