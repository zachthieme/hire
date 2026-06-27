package api

import (
	"errors"
	"hire/internal/store"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID := UserID(r.Context())
	limit, offset := parsePagination(r)
	list, err := h.store.ListNotificationsByUser(r.Context(), userID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	userID := UserID(r.Context())
	if err := h.store.MarkNotificationRead(r.Context(), id, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "notification not found")
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
