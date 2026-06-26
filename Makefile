.PHONY: dev dev-backend dev-frontend build clean test seed

# Development: run backend and frontend concurrently
dev:
	@echo "Starting backend and frontend..."
	@$(MAKE) dev-backend &
	@$(MAKE) dev-frontend
	@wait

dev-backend:
	go run ./cmd/server -addr :8080 -db hire.db

dev-frontend:
	cd frontend && npm run dev

# Build: compile frontend then embed into Go binary
build: frontend/dist
	go build -o hire-server ./cmd/server

frontend/dist: frontend/node_modules frontend/src/**
	cd frontend && npm run build

frontend/node_modules: frontend/package.json
	cd frontend && npm install

# Test
test:
	go test ./internal/... -v

# Seed demo data
seed:
	go run ./seed/seed.go

# Clean
clean:
	rm -f hire-server hire.db
	rm -rf frontend/dist
