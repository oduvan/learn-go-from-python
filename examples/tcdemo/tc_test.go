// Verifies docs/13-third-party-libraries/13-testcontainers.md
package tcdemo

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/oduvan/learn-go-from-python/examples/infra"
)

var (
	once     sync.Once
	baseDSN  string
	setupErr error
)

// The article's pattern: one container per test binary, behind sync.Once,
// with the error captured rather than failing inside the Once.
func sharedPostgres(t *testing.T) string {
	t.Helper()
	once.Do(func() {
		ctx := context.Background()
		c, err := postgres.Run(ctx, "postgres:16-alpine",
			postgres.WithDatabase("app"),
			postgres.WithUsername("u"),
			postgres.WithPassword("p"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(90*time.Second)),
		)
		if err != nil {
			setupErr = err
			return
		}
		baseDSN, setupErr = c.ConnectionString(ctx, "sslmode=disable")
	})
	if setupErr != nil {
		t.Fatalf("starting postgres: %v", setupErr)
	}
	return baseDSN
}

// Prefer a DSN the environment already provides (a CI service container)
// and only fall back to starting one. No build tags anywhere.
func testDSN(t *testing.T) string {
	t.Helper()
	if dsn := os.Getenv("TEST_DATABASE_DSN"); dsn != "" {
		return dsn
	}
	infra.RequireDocker(t)
	return sharedPostgres(t)
}

func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`DROP TABLE IF EXISTS tc_users;
	                  CREATE TABLE tc_users (id bigserial PRIMARY KEY, name text NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestInsertAndRead(t *testing.T) {
	db := SetupTestDB(t)
	if _, err := db.Exec(`INSERT INTO tc_users (name) VALUES ($1)`, "Ada"); err != nil {
		t.Fatal(err)
	}
	var n string
	if err := db.QueryRow(`SELECT name FROM tc_users`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != "Ada" {
		t.Errorf("name = %q, want Ada", n)
	}
}

// The second test reuses the container and gets a clean schema, which is
// what makes the pattern cheap.
func TestSecondTestReusesTheContainer(t *testing.T) {
	db := SetupTestDB(t)
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM tc_users`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("rows = %d, want 0 — the table is recreated per test", n)
	}
}
