package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()
	r.Use(h.RequestIDMiddleware)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			next.ServeHTTP(w, r)
		})
	})
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   h.corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	// Health checks (unauthenticated)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Public
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(10, time.Minute))
		r.Post("/api/auth/login", h.Login)
	})

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)

		// Any authenticated user
		r.Get("/api/me", h.GetMe)
		r.Post("/api/auth/refresh", h.RefreshToken)
		r.Get("/api/me/interviews", h.ListMyInterviews)
		r.Get("/api/notifications", h.ListNotifications)
		r.Put("/api/notifications/{id}/read", h.MarkNotificationRead)
		r.Get("/api/competencies", h.ListCompetencies)

		// Feedback — interviewer submits, scheduler/interviewer can read
		r.Get("/api/interviews/{id}/feedback", h.GetFeedback)
		r.Post("/api/interviews/{id}/feedback", h.CreateFeedback)
		r.Put("/api/feedback/{id}", h.UpdateFeedback)

		// Loops — readable by scheduler and interviewer
		r.Get("/api/loops", h.ListLoops)
		r.Get("/api/loops/{id}", h.GetLoopDetail)

		// Scheduler and admin can list users (scheduler needs it for interviewer assignment)
		r.Group(func(r chi.Router) {
			r.Use(h.RequireRole("scheduler", "admin"))
			r.Get("/api/users", h.ListUsers)
		})

		// Scheduler-only
		r.Group(func(r chi.Router) {
			r.Use(h.RequireRole("scheduler", "admin"))
			r.Post("/api/candidates", h.CreateCandidate)
			r.Get("/api/candidates", h.ListCandidates)
			r.Get("/api/candidates/{id}", h.GetCandidate)
			r.Put("/api/candidates/{id}", h.UpdateCandidate)
			r.Delete("/api/candidates/{id}", h.DeleteCandidate)

			r.Post("/api/loops", h.CreateLoop)
			r.Put("/api/loops/{id}", h.UpdateLoop)
			r.Delete("/api/loops/{id}", h.DeleteLoop)

			r.Post("/api/loops/{id}/interviews", h.CreateInterview)
			r.Put("/api/interviews/{id}", h.UpdateInterview)
			r.Delete("/api/interviews/{id}", h.DeleteInterview)
		})

		// Admin-only
		r.Group(func(r chi.Router) {
			r.Use(h.RequireRole("admin"))
			r.Post("/api/users", h.CreateUser)
			r.Get("/api/users/{id}", h.GetUser)
			r.Put("/api/users/{id}", h.UpdateUser)
			r.Delete("/api/users/{id}", h.DeleteUser)

			r.Post("/api/competencies", h.CreateCompetency)
			r.Put("/api/competencies/{id}", h.UpdateCompetency)
			r.Delete("/api/competencies/{id}", h.DeleteCompetency)
		})
	})

	return r
}
