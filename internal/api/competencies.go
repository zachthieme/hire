package api

import (
	"hire/internal/models"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateCompetency(w http.ResponseWriter, r *http.Request) {
	var c models.Competency
	if err := readJSON(r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.store.CreateCompetency(&c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) GetCompetency(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	c, err := h.store.GetCompetency(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "competency not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) ListCompetencies(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListCompetencies()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) UpdateCompetency(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var c models.Competency
	if err := readJSON(r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	c.ID = id
	if err := h.store.UpdateCompetency(&c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) DeleteCompetency(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteCompetency(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
