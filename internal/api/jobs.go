package api

import (
	"errors"
	"net/http"
	"strconv"

	"hire/internal/models"
	"hire/internal/store"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var j models.Job
	if err := readJSON(r, &j); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateRequired(map[string]string{"title": j.Title}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if j.Status == "" {
		j.Status = models.JobStatusOpen
	}
	if err := validateEnum(j.Status, "status", models.ValidJobStatuses); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	j.CreatedBy = UserID(r.Context())
	if err := h.store.CreateJob(r.Context(), &j); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, j)
}

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	jobs, err := h.store.ListJobs(r.Context(), limit, offset)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (h *Handler) GetJobDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	detail, err := h.store.GetJobDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "job not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetJob(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "job not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	var updates models.Job
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateRequired(map[string]string{"title": updates.Title}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateEnum(updates.Status, "status", models.ValidJobStatuses); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if updates.Status != existing.Status {
		if err := validateTransition(existing.Status, updates.Status, "job", models.ValidJobTransitions); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	existing.Title = updates.Title
	existing.Description = updates.Description
	existing.HiringManager = updates.HiringManager
	existing.Status = updates.Status
	if err := h.store.UpdateJob(r.Context(), existing); err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteJob(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "job not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
