package api

import (
	"hire/internal/models"
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
	if err := h.store.CreateCandidate(&c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
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
	c, err := h.store.GetCandidate(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "candidate not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) ListCandidates(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	list, err := h.store.ListCandidates(limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
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
	var c models.Candidate
	if err := readJSON(r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	c.ID = id
	if err := h.store.UpdateCandidate(&c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) DeleteCandidate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteCandidate(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
