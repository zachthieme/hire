package api

import (
	"errors"
	"hire/internal/models"
	"hire/internal/store"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateCandidate(w http.ResponseWriter, r *http.Request) {
	var c models.Candidate
	if err := readJSON(r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateRequired(map[string]string{"name": c.Name, "email": c.Email}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if c.Status == "" {
		c.Status = "active"
	}
	if err := h.store.CreateCandidate(r.Context(), &c); err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) GetCandidate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	c, err := h.store.GetCandidate(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "candidate not found")
		} else {
			writeInternalError(w, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) ListCandidates(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	list, err := h.store.ListCandidates(r.Context(), limit, offset)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) UpdateCandidate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetCandidate(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "candidate not found")
		} else {
			writeInternalError(w, err)
		}
		return
	}
	var updates models.Candidate
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if updates.Status != "" {
		if err := validateEnum(updates.Status, "status", []string{"active", "hired", "rejected", "withdrawn"}); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		existing.Status = updates.Status
	}
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.Email != "" {
		existing.Email = updates.Email
	}
	if updates.ResumeURL != "" {
		existing.ResumeURL = updates.ResumeURL
	}
	if err := h.store.UpdateCandidate(r.Context(), existing); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
		} else {
			writeInternalError(w, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteCandidate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteCandidate(r.Context(), id); err != nil {
		writeInternalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
