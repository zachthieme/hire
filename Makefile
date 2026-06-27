.PHONY: up down logs test seed migrate-new clean

DB_PORT ?= 5433

# Start all services
up:
	docker compose up --build -d

# Stop all services
down:
	docker compose down

# Follow logs
logs:
	docker compose logs -f

# Run tests (requires running db: docker compose up db -d)
test:
	DATABASE_URL=postgres://hire:devpassword@localhost:$(DB_PORT)/hire_test?sslmode=disable go test ./internal/... -v

# Seed demo data (requires running db)
seed:
	DATABASE_URL=postgres://hire:devpassword@localhost:$(DB_PORT)/hire?sslmode=disable go run ./seed/seed.go

# Create a new migration
migrate-new:
	migrate create -ext sql -dir migrations -seq $(name)

# Clean
clean:
	docker compose down -v
	rm -f server
