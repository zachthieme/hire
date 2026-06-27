package api

import (
	"errors"
	"hire/internal/models"
	"hire/internal/store"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type CreateUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateRequired(map[string]string{"email": req.Email, "name": req.Name, "password": req.Password}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateEmail(req.Email); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if err := validateEnum(req.Role, "role", models.ValidRoles); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "password hash failed")
		return
	}
	u := &models.User{Email: req.Email, Name: req.Name, PasswordHash: hash, Role: req.Role}
	if err := h.store.CreateUser(r.Context(), u); err != nil {
		writeInternalError(w, r, err)
		return
	}
	u.PasswordHash = ""
	writeJSON(w, http.StatusCreated, u)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	users, err := h.store.ListUsers(r.Context(), limit, offset)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, err := h.store.GetUserByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.store.GetUserByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	var req CreateUserRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Role != "" {
		if err := validateEnum(req.Role, "role", models.ValidRoles); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		existing.Role = req.Role
	}
	if req.Email != "" {
		if err := validateEmail(req.Email); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		existing.Email = req.Email
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if err := h.store.UpdateUser(r.Context(), existing); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	if req.Password != "" {
		if len(req.Password) < 8 {
			writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}
		hash, err := HashPassword(req.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "password hash failed")
			return
		}
		if err := h.store.UpdateUserPassword(r.Context(), id, hash); err != nil {
			writeInternalError(w, r, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteUser(r.Context(), id); err != nil {
		writeInternalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := UserID(r.Context())
	u, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
		} else {
			writeInternalError(w, r, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, u)
}
