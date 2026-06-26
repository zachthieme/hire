package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID := UserID(r.Context())
	list, err := h.store.ListNotificationsByUser(userID)
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
	if err := h.store.MarkNotificationRead(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
