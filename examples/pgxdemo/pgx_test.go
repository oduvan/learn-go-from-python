// Verifies docs/13-third-party-libraries/06-pgx-and-postgres.md
package pgxdemo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oduvan/learn-go-from-python/examples/infra"
)

func pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	infra.RequirePostgres(t) // skips when unreachable

	cfg, err := pgxpool.ParseConfig(infra.PostgresDSN())
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour

	p, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)

	ctx := context.Background()
	if _, err := p.Exec(ctx, `DROP TABLE IF EXISTS items`); err != nil {
		t.Fatal(err)
	}
	_, err = p.Exec(ctx, `CREATE TABLE items (
		id uuid PRIMARY KEY,
		name text NOT NULL UNIQUE,
		tags text[] NOT NULL DEFAULT '{}',
		meta jsonb,
		created_at timestamptz NOT NULL DEFAULT now())`)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseConfigReadsTheDSN(t *testing.T) {
	infra.RequirePostgres(t)
	cfg, err := pgxpool.ParseConfig(infra.PostgresDSN())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "book" {
		t.Errorf("database = %q, want \"book\"", cfg.ConnConfig.Database)
	}
}

func TestUUIDv7IsTimeOrdered(t *testing.T) {
	a, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	b, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	if a.Version() != 7 {
		t.Errorf("Version() = %v, want 7", a.Version())
	}
	if !(a.String() < b.String()) {
		t.Errorf("v7 ids are not time-ordered: %s !< %s", a, b)
	}
	if uuid.New().Version() != 4 {
		t.Error("uuid.New() should be v4")
	}
}

func TestUUIDParseRejectsGarbage(t *testing.T) {
	if _, err := uuid.Parse("not-a-uuid"); err == nil {
		t.Fatal("expected an error")
	} else if err.Error() != "invalid UUID length: 10" {
		t.Errorf("error = %q, does not match the article", err)
	}
}

// The reason to use pgx natively: arrays, jsonb and uuid map to plain Go
// values with no wrapper types.
func TestNativeTypesRoundTrip(t *testing.T) {
	p := pool(t)
	ctx := context.Background()

	id, _ := uuid.NewV7()
	_, err := p.Exec(ctx, `INSERT INTO items (id,name,tags,meta) VALUES ($1,$2,$3,$4)`,
		id, "first", []string{"a", "b"}, map[string]any{"k": 1})
	if err != nil {
		t.Fatal(err)
	}

	var (
		gotID   uuid.UUID
		name    string
		tags    []string
		meta    map[string]any
		created time.Time
	)
	err = p.QueryRow(ctx, `SELECT id,name,tags,meta,created_at FROM items WHERE id=$1`, id).
		Scan(&gotID, &name, &tags, &meta, &created)
	if err != nil {
		t.Fatal(err)
	}
	if gotID != id || name != "first" {
		t.Errorf("got %v %q", gotID, name)
	}
	if len(tags) != 2 || tags[0] != "a" || tags[1] != "b" {
		t.Errorf("tags = %v, want [a b]", tags)
	}
	if meta["k"] != float64(1) {
		t.Errorf("meta = %v", meta)
	}
	if time.Since(created) > time.Minute {
		t.Errorf("created_at looks wrong: %v", created)
	}
}

func TestErrNoRows(t *testing.T) {
	p := pool(t)
	var name string
	err := p.QueryRow(context.Background(), `SELECT name FROM items WHERE name='nope'`).Scan(&name)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("err = %v, want pgx.ErrNoRows", err)
	}
}

func TestPgErrorCarriesSQLSTATE(t *testing.T) {
	p := pool(t)
	ctx := context.Background()
	id, _ := uuid.NewV7()
	if _, err := p.Exec(ctx, `INSERT INTO items (id,name) VALUES ($1,$2)`, id, "dup"); err != nil {
		t.Fatal(err)
	}
	_, err := p.Exec(ctx, `INSERT INTO items (id,name) VALUES ($1,$2)`, uuid.New(), "dup")

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("err = %v, want a *pgconn.PgError", err)
	}
	if pgErr.Code != "23505" {
		t.Errorf("Code = %q, want \"23505\" (unique violation)", pgErr.Code)
	}
	if pgErr.ConstraintName != "items_name_key" {
		t.Errorf("ConstraintName = %q", pgErr.ConstraintName)
	}
}

func TestCollectRows(t *testing.T) {
	p := pool(t)
	ctx := context.Background()
	for _, n := range []string{"beta", "alpha"} {
		id, _ := uuid.NewV7()
		if _, err := p.Exec(ctx, `INSERT INTO items (id,name) VALUES ($1,$2)`, id, n); err != nil {
			t.Fatal(err)
		}
	}

	rows, _ := p.Query(ctx, `SELECT name FROM items ORDER BY name`)
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("names = %v, want [alpha beta]", names)
	}

	type Item struct {
		ID   uuid.UUID
		Name string
	}
	rows2, _ := p.Query(ctx, `SELECT id, name FROM items ORDER BY name`)
	items, err := pgx.CollectRows(rows2, pgx.RowToStructByPos[Item])
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Name != "alpha" {
		t.Errorf("items = %+v", items)
	}
}

func TestBatch(t *testing.T) {
	p := pool(t)
	ctx := context.Background()

	b := &pgx.Batch{}
	for i := range 3 {
		id, _ := uuid.NewV7()
		b.Queue(`INSERT INTO items (id,name) VALUES ($1,$2)`, id, string(rune('a'+i)))
	}
	if err := p.SendBatch(ctx, b).Close(); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := p.QueryRow(ctx, `SELECT count(*) FROM items`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("rows = %d, want 3", n)
	}
}
