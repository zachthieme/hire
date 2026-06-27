package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"hire/internal/api"
	"hire/internal/store"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	// Configure structured JSON logging for production
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		slog.Error("JWT_SECRET is required")
		os.Exit(1)
	}
	if len(jwtSecret) < 32 {
		slog.Error("JWT_SECRET must be at least 32 characters")
		os.Exit(1)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	corsOrigins := os.Getenv("CORS_ORIGINS")
	origins := []string{"*"}
	if corsOrigins != "" {
		origins = strings.Split(corsOrigins, ",")
	}

	// Run migrations
	mig, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		slog.Error("failed to create migrator", "error", err)
		os.Exit(1)
	}
	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	s, err := store.New(databaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer s.Close()

	h := api.NewHandler(s, jwtSecret, origins)
	r := h.Router()

	addr := ":" + port
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	slog.Info("server starting", "addr", addr)

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	fmt.Println() // newline after ^C
	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
