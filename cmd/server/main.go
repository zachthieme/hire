package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"hire/internal/api"
	"hire/internal/store"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters")
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
		log.Fatalf("Failed to create migrator: %v", err)
	}
	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	s, err := store.New(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
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
	fmt.Printf("Server listening on %s\n", addr)
	log.Fatal(srv.ListenAndServe())
}
