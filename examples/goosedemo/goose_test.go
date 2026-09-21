// Verifies docs/13-third-party-libraries/09-goose-migrations.md
package goosedemo

import (
	"database/sql"
	"embed"
	"sync"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/oduvan/learn-go-from-python/examples/infra"
)

//go:embed all:migrations
var embedMigrations embed.FS

var setDialect sync.Once

func fresh(t *testing.T) *sql.DB {
	t.Helper()
	db := infra.RequirePostgres(t)

	db.Exec(`DROP TABLE IF EXISTS goose_users CASCADE`)
	db.Exec(`DROP TABLE IF EXISTS goose_db_version`)
	db.Exec(`DROP FUNCTION IF EXISTS goose_touch()`)

	goose.SetBaseFS(embedMigrations)
	goose.SetLogger(goose.NopLogger())
	setDialect.Do(func() {
		if err := goose.SetDialect("postgres"); err != nil {
			t.Fatal(err)
		}
	})
	return db
}

func TestUpAppliesEverything(t *testing.T) {
	db := fresh(t)
	if err := goose.Up(db, "migrations"); err != nil {
		t.Fatal(err)
	}
	v, err := goose.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	}
	if v != 3 {
		t.Errorf("version = %d, want 3", v)
	}

	var cols int
	db.QueryRow(`SELECT count(*) FROM information_schema.columns WHERE table_name='goose_users'`).Scan(&cols)
	if cols != 3 {
		t.Errorf("columns = %d, want 3 (id, email, name)", cols)
	}
}

// The StatementBegin/End markers are what let a function body with
// internal semicolons survive goose's statement splitting.
func TestStatementBlockCreatesTheFunction(t *testing.T) {
	db := fresh(t)
	if err := goose.Up(db, "migrations"); err != nil {
		t.Fatal(err)
	}
	var n int
	db.QueryRow(`SELECT count(*) FROM pg_proc WHERE proname='goose_touch'`).Scan(&n)
	if n != 1 {
		t.Errorf("function count = %d, want 1 — check the StatementBegin markers", n)
	}
}

func TestUpIsIdempotent(t *testing.T) {
	db := fresh(t)
	if err := goose.Up(db, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		t.Fatalf("second Up returned %v, want nil", err)
	}
	v, _ := goose.GetDBVersion(db)
	if v != 3 {
		t.Errorf("version = %d, want 3", v)
	}
}

func TestDownRollsBackExactlyOne(t *testing.T) {
	db := fresh(t)
	if err := goose.Up(db, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Down(db, "migrations"); err != nil {
		t.Fatal(err)
	}
	v, _ := goose.GetDBVersion(db)
	if v != 2 {
		t.Errorf("version = %d, want 2 — Down rolls back one, not all", v)
	}
}

// all: matters — without it a file beginning with _ is silently skipped,
// which for migrations means a schema that quietly diverges.
func TestEmbedIncludesEveryMigration(t *testing.T) {
	ents, err := embedMigrations.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 3 {
		t.Errorf("embedded %d migrations, want 3", len(ents))
	}
}
