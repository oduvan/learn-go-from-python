# goose migrations

`AutoMigrate` is fine for a test database. A production schema needs
changes that are reviewable, ordered, repeatable and reversible. goose
gives you that in plain SQL files.

> **Module:** `github.com/pressly/goose/v3`.

```sql
-- +goose Up
CREATE TABLE users (
    id bigserial PRIMARY KEY,
    email text NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE users;
```

## The file format

One file per change, named `NNNNN_description.sql`, applied in numeric
order:

```
migrations/
  00001_create_users.sql
  00002_add_name.sql
  00003_fn.sql
```

The `-- +goose Up` and `-- +goose Down` markers split the file. Up
applies the change; Down reverses it.

Prefer a zero-padded sequence over timestamps. Sequence numbers make
the order obvious at a glance, and two developers adding a migration on
the same day produce a merge conflict — which is exactly what you want,
because it forces a decision about ordering instead of silently
interleaving.

## Statements containing semicolons

goose splits on `;`, which breaks anything with an internal one — a
function body, a `DO` block, a trigger. Wrap those:

```sql
-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION touch() RETURNS trigger AS $$
BEGIN
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
```

Forget the markers and you get a syntax error pointing at a fragment of
your function, which is confusing until you know the cause.

## Embedding them in the binary

The migrations should ship with the code that expects them, using
[`go:embed`](../07-operating-system/03-go-embed.md):

```go
//go:embed all:migrations
var embedMigrations embed.FS

func main() {
    db, err := sql.Open("pgx", dsn)
    defer db.Close()

    goose.SetBaseFS(embedMigrations)
    if err := goose.SetDialect("postgres"); err != nil {
        return err
    }
    if err := goose.Up(db, "migrations"); err != nil {
        return err
    }
}
```

Note **`all:migrations`**, not `migrations`. The plain form skips files
beginning with `_` or `.`, and a silently skipped migration means a
schema that diverges between environments — the worst possible place
for that rule to bite.

goose takes a `*sql.DB`, so it works with any driver.

## Running them

```go
goose.Up(db, "migrations")      // apply everything pending
goose.Down(db, "migrations")    // roll back exactly one
goose.UpTo(db, "migrations", 2) // apply up to a version
goose.Status(db, "migrations")  // what is applied
version, err := goose.GetDBVersion(db)
```

```
--up--
version: 3
--up is idempotent--
version still: 3
--down one--
version: 2
```

`Up` is idempotent: running it again applies nothing. That is what lets
you run it unconditionally at startup or as a deploy step.

Note `Down` rolls back **one** migration, not all of them.

## The version table

goose records what it has applied in `goose_db_version`:

```
  0 applied=true
  1 applied=true
  2 applied=true
```

Version 0 is the initial row it creates. This table is the state of
your schema — never edit it by hand, and include it when you clone a
database for testing.

## Go migrations

For a change needing logic — backfilling a column from computed
values — register a Go function instead:

```go
func init() {
    goose.AddMigrationContext(upBackfill, downBackfill)
}

func upBackfill(ctx context.Context, tx *sql.Tx) error {
    _, err := tx.ExecContext(ctx, `UPDATE users SET slug = lower(email)`)
    return err
}
```

The file lives alongside the SQL ones and is ordered by the same
numeric prefix. Registration happens in `init()`, so the package must
be imported — usually a blank import from the migration binary.

Use these sparingly. SQL migrations are reviewable by anyone; a Go
migration is code that runs once and then never again, which makes it
hard to test and easy to get wrong.

## Where to run them

| Approach | Trade |
|---|---|
| a separate `cmd/migrate` binary | explicit, ordered before the deploy; needs a job step |
| at service startup | nothing to orchestrate; racy with multiple replicas |

Running at startup with several replicas means several processes
migrating at once. goose takes a lock, so it is safer than it sounds,
but a failed migration now fails your rollout rather than a job you can
retry. For anything beyond a single instance, a separate binary run as
a deploy step is the calmer choice.

## Writing migrations that do not break a rollout

During a deploy, old and new code run against the same schema for a
while. That constrains what a single migration may do:

- **Adding a nullable column or one with a default** is safe.
- **Dropping a column** breaks old code still selecting it. Deploy the
  code that stops using it first, drop it in a later release.
- **Renaming** is a drop plus an add. Do it in three steps: add the new
  column and write both, backfill, then remove the old one.
- **Adding an index** locks the table. In Postgres use
  `CREATE INDEX CONCURRENTLY` — which cannot run inside a
  transaction, so it needs `-- +goose NO TRANSACTION` at the top of the
  file.
- **Backfilling a large table** in one statement holds a long lock.
  Batch it.

Every migration also needs a `Down` that actually works. The time to
discover otherwise is not during an incident — apply and roll back
locally before opening the pull request.

## Keeping models and schema in step

goose owns the schema; your structs describe what the code expects.
Nothing connects them, so a migration adding a column and a struct that
never gained the field will not complain. A startup check comparing the
two — or a test that runs the migrations and asserts the columns match
your models — closes that gap cheaply.

> **From Python:** goose is Alembic without autogeneration. You write
> the SQL yourself, which is more typing and produces migrations you
> can actually read in review — no generated diff that drops a column
> you meant to keep.

## Quick reference

| Task | Form |
|---|---|
| a migration | `NNNNN_name.sql` with `-- +goose Up` / `Down` |
| semicolons inside a statement | `-- +goose StatementBegin` / `StatementEnd` |
| ship them in the binary | `//go:embed all:migrations` + `goose.SetBaseFS` |
| dialect | `goose.SetDialect("postgres")` |
| apply | `goose.Up(db, "migrations")` — idempotent |
| roll back one | `goose.Down(db, "migrations")` |
| current version | `goose.GetDBVersion(db)` |
| logic, not just SQL | `goose.AddMigrationContext` in `init()` |
| no transaction | `-- +goose NO TRANSACTION` |
| safe changes | additive first; drop in a later release |

## Sources

- [goose — github.com/pressly/goose](https://github.com/pressly/goose)
- [goose documentation — pressly.github.io/goose/](https://pressly.github.io/goose/)
- [`goose` package — pkg.go.dev/github.com/pressly/goose/v3](https://pkg.go.dev/github.com/pressly/goose/v3)
- [PostgreSQL: CREATE INDEX CONCURRENTLY — postgresql.org/docs/current/sql-createindex.html](https://www.postgresql.org/docs/current/sql-createindex.html)
