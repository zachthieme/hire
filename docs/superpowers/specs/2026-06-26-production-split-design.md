# Production Split — Design Spec

## Purpose

Split the interview scheduling app from a single Go binary (embedded SPA + SQLite) into a production-ready containerized architecture: standalone React frontend served by Nginx, Go API backend, and PostgreSQL database. All three run as Docker Compose services.

## What Changes

| Component | Current | Target |
|-----------|---------|--------|
| Database | SQLite (modernc.org/sqlite, embedded) | PostgreSQL 16 (pgx driver, container) |
| Backend | Single binary serving API + embedded SPA | API-only container, env var config |
| Frontend | Embedded in Go binary via embed.FS | Nginx container serving static files, proxying /api |
| Migrations | Embedded SQL run by store.New() | golang-migrate, run on API startup |
| Config | CLI flags + env vars | All env vars via .env / Docker Compose |
| Dev workflow | make dev (two processes) | docker compose up (three containers) |

## What Stays the Same

- All API handlers, middleware, router logic
- All frontend React code (pages, components, hooks)
- Business logic, auth, notification system
- Models, API types, test structure

## Project Structure

```
hire/
├── docker-compose.yml
├── .env.example                          # DATABASE_URL, JWT_SECRET
├── Dockerfile                            # Multi-stage Go API build
├── cmd/server/main.go                    # Env var config, no embed, runs migrations on startup
├── internal/
│   ├── store/
│   │   ├── store.go                      # PostgreSQL via pgx, connection pooling
│   │   ├── users.go                      # $1/$2 params
│   │   ├── candidates.go
│   │   ├── competencies.go
│   │   ├── loops.go
│   │   ├── interviews.go
│   │   ├── feedback.go
│   │   └── notifications.go
│   ├── api/                              # Unchanged except router.go (remove SPA serving)
│   ├── models/                           # Unchanged
│   └── notify/                           # Unchanged
├── migrations/                           # Moved back to root for golang-migrate
│   ├── 000001_initial_schema.up.sql      # PostgreSQL DDL
│   └── 000001_initial_schema.down.sql    # DROP tables
├── seed/seed.go                          # Updated for PostgreSQL connection
├── frontend/
│   ├── Dockerfile                        # Multi-stage: npm build → nginx
│   ├── nginx.conf                        # Serve static + proxy /api
│   └── src/                              # Unchanged
└── Makefile                              # Updated targets for docker compose
```

**Removed:** `embed.go`, `internal/store/migrations/` (moved back to root)

## Docker Compose

Three services:

### db (PostgreSQL 16 Alpine)
- Named volume `pgdata` for data persistence
- Health check via `pg_isready` so API waits for readiness
- Exposes port 5432 for local tooling (psql, etc.)

### api (Go backend)
- Multi-stage Dockerfile: `golang:1.22-alpine` build → `alpine:3.20` runtime
- Copies compiled binary + migrations directory into runtime image
- Runs golang-migrate on startup before starting the HTTP server
- Environment: `DATABASE_URL`, `JWT_SECRET`
- Depends on db with `condition: service_healthy`
- Exposes port 8080

### frontend (Nginx + React)
- Multi-stage Dockerfile: `node:22-alpine` build → `nginx:alpine` runtime
- `npm ci && npm run build` → copies dist to nginx html root
- Custom `nginx.conf`: serves static files with SPA fallback, proxies `/api/` to `api:8080`
- Exposes port 3000
- Depends on api

## Store Layer Changes (SQLite → PostgreSQL)

### Driver
`github.com/jackc/pgx/v5/stdlib` — registers as `"pgx"` with `database/sql`.

### Connection
```go
func New(databaseURL string) (*Store, error) {
    db, err := sql.Open("pgx", databaseURL)
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    if err := db.Ping(); err != nil { ... }
    return &Store{db: db}, nil
}
```

No PRAGMA calls. No embedded migration runner. Store.New takes a connection string only.

### SQL Syntax Changes

Every store file needs these mechanical transformations:

| SQLite | PostgreSQL |
|--------|-----------|
| `?` placeholders | `$1, $2, $3` positional params |
| `INTEGER PRIMARY KEY AUTOINCREMENT` | `SERIAL PRIMARY KEY` |
| `DATETIME` | `TIMESTAMPTZ` |
| `BOOLEAN DEFAULT 0` | `BOOLEAN DEFAULT false` |
| `res.LastInsertId()` after Exec | `INSERT ... RETURNING id` with QueryRow.Scan |

The `RETURNING id` pattern is the biggest change. Every `Create*` method changes from:
```go
res, err := s.db.Exec(`INSERT INTO ... VALUES (?, ?)`, ...)
id, _ = res.LastInsertId()
```
To:
```go
err := s.db.QueryRow(`INSERT INTO ... VALUES ($1, $2) RETURNING id`, ...).Scan(&id)
```

### Tests
Tests run against a real PostgreSQL instance (the Docker Compose db service). The test helper:
1. Connects to the db container
2. Creates a test-specific database (or uses a transaction that rolls back)
3. Runs migrations
4. Returns a Store
5. Cleans up on test completion

Test DATABASE_URL comes from the environment, defaulting to the Docker Compose db service.

## Migration Setup (golang-migrate)

### Files
- `migrations/000001_initial_schema.up.sql` — PostgreSQL DDL for all 8 tables
- `migrations/000001_initial_schema.down.sql` — DROP TABLE in reverse dependency order

### Startup Flow
In `cmd/server/main.go`:
1. Read `DATABASE_URL` from environment
2. Run golang-migrate against that URL using the `migrations/` directory
3. Open store connection
4. Start HTTP server

golang-migrate tracks applied migrations in a `schema_migrations` table. Idempotent — re-running on an already-migrated database is a no-op.

### Future Migrations
```bash
migrate create -ext sql -dir migrations -seq add_some_column
# Produces 000002_add_some_column.up.sql and .down.sql
```

## Nginx Configuration

```nginx
server {
    listen 80;

    location / {
        root /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://api:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

SPA routing handled by `try_files` fallback to `index.html`. API proxying uses Docker Compose service name `api` for DNS resolution.

## cmd/server/main.go Changes

All configuration from environment variables:
- `DATABASE_URL` — PostgreSQL connection string (required)
- `JWT_SECRET` — JWT signing key (required)
- `PORT` — HTTP listen port (default 8080)

Remove:
- CLI flag parsing
- embed.FS frontend serving
- SPA fallback handler
- Import of root `hire` package

The server becomes an API-only process.

## Makefile Updates

```makefile
up:          docker compose up --build
down:        docker compose down
logs:        docker compose logs -f
test:        DATABASE_URL=postgres://hire:devpassword@localhost:5432/hire?sslmode=disable go test ./internal/... -v
seed:        DATABASE_URL=postgres://hire:devpassword@localhost:5432/hire?sslmode=disable go run ./seed/seed.go
migrate-new: migrate create -ext sql -dir migrations -seq $(name)
```

Tests and seed run on the host (not inside the container) against the Compose PostgreSQL instance exposed on port 5432.

## .env.example

```
DATABASE_URL=postgres://hire:devpassword@localhost:5432/hire?sslmode=disable
JWT_SECRET=change-me-to-a-real-secret
DB_PASSWORD=devpassword
```

`.env` is gitignored.
