package api

import (
	"errors"
	"hire/internal/models"
	"hire/internal/store"
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
	if err := validateRequired(map[string]string{"name": c.Name, "ratings_json": c.RatingsJSON}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateEnum(c.RatingType, "rating_type", models.ValidRatingTypes); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateRatingsJSON(c.RatingsJSON, c.RatingType); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.CreateCompetency(r.Context(), &c); err != nil {
		writeInternalError(w, r, err)
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
	c, err := h.store.GetCompetency(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "competency not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) ListCompetencies(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	list, err := h.store.ListCompetencies(r.Context(), limit, offset)
	if err != nil {
		writeInternalError(w, r, err)
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
	existing, err := h.store.GetCompetency(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "competency not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	var updates models.Competency
	if err := readJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.RatingType != "" {
		if err := validateEnum(updates.RatingType, "rating_type", models.ValidRatingTypes); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		existing.RatingType = updates.RatingType
	}
	if updates.RatingsJSON != "" {
		if err := validateRatingsJSON(updates.RatingsJSON, existing.RatingType); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		existing.RatingsJSON = updates.RatingsJSON
	}
	if err := h.store.UpdateCompetency(r.Context(), existing); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteCompetency(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteCompetency(r.Context(), id); err != nil {
		writeInternalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
