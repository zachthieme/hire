package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	root "hire"
	"hire/internal/api"
	"hire/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "hire.db", "SQLite database path")
	jwtSecret := flag.String("jwt-secret", "change-me-in-production", "JWT signing secret")
	flag.Parse()

	s, err := store.New(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer s.Close()

	h := api.NewHandler(s, *jwtSecret)
	r := h.Router()

	// Serve embedded frontend
	frontendDist, err := fs.Sub(root.FrontendFS, "frontend/dist")
	if err != nil {
		log.Fatalf("Failed to load frontend: %v", err)
	}
	fileServer := http.FileServer(http.FS(frontendDist))
	r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Try to serve the file; if it doesn't exist, serve index.html (SPA routing)
		f, err := frontendDist.Open(req.URL.Path[1:])
		if err != nil {
			req.URL.Path = "/"
		} else {
			f.Close()
		}
		fileServer.ServeHTTP(w, req)
	}))

	fmt.Printf("Server listening on %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, r))
}
