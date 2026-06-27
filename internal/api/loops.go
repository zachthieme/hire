package api

import (
	"errors"
	"hire/internal/models"
	"hire/internal/store"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateLoop(w http.ResponseWriter, r *http.Request) {
	var l models.InterviewLoop
	if err := readJSON(r, &l); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if l.CandidateID == 0 {
		writeError(w, http.StatusBadRequest, "candidate_id is required")
		return
	}
	l.CreatedBy = UserID(r.Context())
	if l.Status == "" {
		l.Status = models.LoopStatusScheduling
	}
	if err := validateEnum(l.Status, "status", models.ValidLoopStatuses); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.CreateLoop(r.Context(), &l); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

func (h *Handler) GetLoopDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	detail, err := h.store.GetLoopDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "loop not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}

	// Enforce feedback visibility rule for interviewers
	role := UserRole(r.Context())
	userID := UserID(r.Context())
	if role == models.RoleInterviewer {
		hasSubmitted, err := h.store.HasUserSubmittedFeedbackForLoop(r.Context(), detail.ID, userID)
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		if !hasSubmitted {
			for i := range detail.Interviews {
				if detail.Interviews[i].InterviewerID != userID {
					detail.Interviews[i].Feedback = nil
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) ListLoops(w http.ResponseWriter, r *http.Request) {
	var candidateID *int64
	var status *string
	if v := r.URL.Query().Get("candidate_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			candidateID = &id
		}
	}
	if v := r.URL.Query().Get("status"); v != "" {
		status = &v
	}

	limit, offset := parsePagination(r)
	loops, err := h.store.ListLoops(r.Context(), candidateID, status, limit, offset)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, loops)
}

func (h *Handler) UpdateLoop(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetLoop(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "loop not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	var updates models.InterviewLoop
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateEnum(updates.Status, "status", models.ValidLoopStatuses); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if updates.Status != existing.Status {
		if err := validateTransition(existing.Status, updates.Status, "loop", models.ValidLoopTransitions); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	existing.Status = updates.Status
	existing.FinalDecision = updates.FinalDecision
	existing.DebriefNotes = updates.DebriefNotes
	if err := h.store.UpdateLoop(r.Context(), existing); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteLoop(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteLoop(r.Context(), id); err != nil {
		writeInternalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
