package store

import (
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func testDSN() string {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://hire:devpassword@localhost:5433/hire_test?sslmode=disable"
	}
	return dsn
}

func TestMain(m *testing.M) {
	dsn := testDSN()
	mig, err := migrate.New("file://../../migrations", dsn)
	if err != nil {
		panic("migrate.New: " + err.Error())
	}
	mig.Up() // ignore "no change" errors
	os.Exit(m.Run())
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(testDSN())
	if err != nil {
		t.Fatalf("newTestStore: %v", err)
	}
	s.db.Exec("TRUNCATE competency_ratings, notifications, feedback, interviews, interview_loops, competencies, candidates, users RESTART IDENTITY CASCADE")
	t.Cleanup(func() { s.Close() })
	return s
}
