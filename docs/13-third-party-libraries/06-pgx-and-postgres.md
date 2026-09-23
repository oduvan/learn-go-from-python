# pgx and PostgreSQL

`database/sql` is driver-agnostic, which means it speaks the subset of
SQL every database shares. pgx is a PostgreSQL-specific driver, and
using it directly gets you the types Postgres actually has.

> **Modules:** `github.com/jackc/pgx/v5` and
> `github.com/google/uuid`.

Builds on [`database/sql`](../09-database-sql/01-database-sql.md),
which is the API this replaces — or plugs into, if you prefer.

```go
pool, err := pgxpool.New(ctx, dsn)
defer pool.Close()
```

## Two ways to use it

pgx can be the driver *behind* `database/sql`:

```go
import _ "github.com/jackc/pgx/v5/stdlib"

db, err := sql.Open("pgx", dsn)
```

Everything from the core article then applies unchanged. Choose this
when you want portability, or when a library you use expects `*sql.DB`.

Or you use its native API, which is what the rest of this article
covers:

```go
pool, err := pgxpool.New(ctx, dsn)
```

The native API gives you Postgres types, batching and `COPY`. The
`database/sql` route gives you a standard interface. Both are
reasonable; mixing them in one codebase is not.

## The pool

```go
cfg, err := pgxpool.ParseConfig(dsn)
cfg.MaxConns = 10
cfg.MaxConnLifetime = time.Hour

pool, err := pgxpool.NewWithConfig(ctx, cfg)
defer pool.Close()

if err := pool.Ping(ctx); err != nil {
    return fmt.Errorf("database unreachable: %w", err)
}
```

`ParseConfig` reads the DSN and gives you a struct to adjust, which is
how you set pool limits without encoding them in a connection string.
It also reads the standard `PG*` environment variables.

`*pgxpool.Pool` is the concurrency-safe thing to pass around — the
equivalent of `*sql.DB`. `pool.Stat()` exposes `MaxConns`,
`AcquiredConns` and `EmptyAcquireCount`, worth putting on a dashboard.

## Postgres types just work

This is the reason to use the native API. No `pq.Array`, no wrapper
types — arrays, `jsonb`, `uuid` and `timestamptz` map to ordinary Go
values:

```go
_, err = pool.Exec(ctx,
    `INSERT INTO items (id, name, tags, meta) VALUES ($1,$2,$3,$4)`,
    id, "first", []string{"a", "b"}, map[string]any{"k": 1})

var (
    gotID   uuid.UUID
    tags    []string
    meta    map[string]any
    created time.Time
)
err = pool.QueryRow(ctx, `SELECT id,name,tags,meta,created_at FROM items WHERE id=$1`, id).
    Scan(&gotID, &name, &tags, &meta, &created)
// [a b]  map[k:1]
```

A `text[]` column scans into `[]string` and a `jsonb` column into
`map[string]any` directly. Through `database/sql` both would need a
custom [`sql.Scanner`](../09-database-sql/02-custom-column-types.md).

## UUIDs, and why v7

```go
id, err := uuid.NewV7()
fmt.Println(id.Version())   // output: VERSION_7
```

Version 4 is fully random. Version 7 puts a timestamp in the high bits,
so ids generated later sort later:

```go
id1, _ := uuid.NewV7()
id2, _ := uuid.NewV7()
fmt.Println(id1.String() < id2.String())   // output: true
```

That matters for a primary key. Random v4 keys scatter inserts across
the whole B-tree, fragmenting the index; time-ordered v7 keys append,
which keeps inserts cheap and recent rows physically together.

`uuid.Parse` validates:

```go
_, err := uuid.Parse("not-a-uuid")
fmt.Println(err)   // output: invalid UUID length: 10
```

Postgres 18 has a built-in `uuidv7()`, so `DEFAULT uuidv7()` in the
schema is an alternative to generating ids in Go.

## Errors

No rows has its own sentinel:

```go
err := pool.QueryRow(ctx, `SELECT name FROM items WHERE name='nope'`).Scan(&name)
fmt.Println(errors.Is(err, pgx.ErrNoRows))   // output: true
// no rows in result set
```

Note it is `pgx.ErrNoRows`, not `sql.ErrNoRows`. Through the
`database/sql` adapter you get the latter.

Everything else the server rejects comes back as a `*pgconn.PgError`
carrying the SQLSTATE code:

```go
var pgErr *pgconn.PgError
if errors.As(err, &pgErr) {
    fmt.Println("code:", pgErr.Code, "constraint:", pgErr.ConstraintName)
}
// code: 23505 constraint: items_name_key
```

This is what turns a duplicate into a clean "already exists" response
instead of a `500`:

| SQLSTATE | Meaning |
|---|---|
| `23505` | unique violation |
| `23503` | foreign key violation |
| `23502` | not-null violation |
| `23514` | check constraint violation |
| `40001` | serialisation failure — retry |
| `57014` | query cancelled (statement timeout) |

Match on the code, never on the message text. And translate at the
store boundary, as
[the repository pattern](../09-database-sql/04-the-repository-pattern.md)
argues — `pgErr` should not escape into your handlers.

## Collecting rows

`pgx.CollectRows` removes the scan loop:

```go
rows, _ := pool.Query(ctx, `SELECT name FROM items ORDER BY name`)
names, err := pgx.CollectRows(rows, pgx.RowTo[string])
// [first second]
```

And into structs:

```go
type Item struct {
    ID   uuid.UUID
    Name string
}

rows, _ := pool.Query(ctx, `SELECT id, name FROM items ORDER BY name`)
items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Item])
```

`RowToStructByPos` matches by **position**, so the `SELECT` order must
match the field order — reordering the struct silently breaks it.
`RowToStructByName` matches on `db` tags instead, which is safer when
either side changes.

`CollectRows` closes the rows for you and returns any error, including
the one `rows.Err()` would have carried — removing the easiest mistake
in the core article's loop.

## Batching

Several statements in one round trip:

```go
b := &pgx.Batch{}
for _, item := range items {
    b.Queue(`INSERT INTO items (id,name) VALUES ($1,$2)`, item.ID, item.Name)
}

br := pool.SendBatch(ctx, b)
if err := br.Close(); err != nil {
    return err
}
```

`Close` reports the first error, and **must** be called. For bulk
loading, `pool.CopyFrom` implements the Postgres `COPY` protocol and is
much faster again — thousands of rows rather than dozens. `COPY` is
Postgres's bulk-load command: instead of one `INSERT` per row, the
client streams all the rows to the server in a single operation.

## Transactions and listen/notify

```go
tx, err := pool.Begin(ctx)
defer tx.Rollback(ctx)   // no-op after commit
// ...
return tx.Commit(ctx)
```

Same shape as the core article, except the methods take a context.
`pgx.BeginFunc` wraps the closure pattern for you.

pgx also exposes `LISTEN`/`NOTIFY` through `conn.WaitForNotification`,
which gives you push notifications from the database without polling —
something `database/sql` cannot express at all.

## pgvector

For embeddings, `pgvector` adds a `vector` column type and a
`github.com/pgvector/pgvector-go` package that registers with pgx. A
`vector(1536)` column then scans into a `[]float32`, and
`ORDER BY embedding <=> $1 LIMIT 10` does nearest-neighbour search in
SQL.

> **From Python:** pgx is psycopg3 — a Postgres-native driver with real
> type adaptation — while the `stdlib` shim is closer to using it
> through SQLAlchemy Core. `CollectRows` is `fetchall()` with row
> factories, and `pgErr.Code` is psycopg's `e.sqlstate`.

## Quick reference

| Task | Form |
|---|---|
| native pool | `pgxpool.New(ctx, dsn)` / `NewWithConfig` |
| tune it | `pgxpool.ParseConfig`, then set `MaxConns` |
| behind `database/sql` | `_ "github.com/jackc/pgx/v5/stdlib"`, `sql.Open("pgx", …)` |
| arrays / jsonb / uuid | plain `[]string`, `map[string]any`, `uuid.UUID` |
| no rows | `errors.Is(err, pgx.ErrNoRows)` |
| constraint violation | `errors.As(err, &pgErr)`, check `pgErr.Code` |
| collect results | `pgx.CollectRows(rows, pgx.RowToStructByName[T])` |
| many writes | `pgx.Batch` + `SendBatch`, or `CopyFrom` for bulk |
| transactions | `pool.Begin(ctx)`, or `pgx.BeginFunc` |
| primary keys | `uuid.NewV7()` — time-ordered, index-friendly |

## Sources

- [pgx — pkg.go.dev/github.com/jackc/pgx/v5](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [`pgxpool` — pkg.go.dev/github.com/jackc/pgx/v5/pgxpool](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool)
- [`pgconn.PgError` — pkg.go.dev/github.com/jackc/pgx/v5/pgconn#PgError](https://pkg.go.dev/github.com/jackc/pgx/v5/pgconn#PgError)
- [`google/uuid` — pkg.go.dev/github.com/google/uuid](https://pkg.go.dev/github.com/google/uuid)
- [PostgreSQL error codes — postgresql.org/docs/current/errcodes-appendix.html](https://www.postgresql.org/docs/current/errcodes-appendix.html)
- [pgvector — github.com/pgvector/pgvector-go](https://github.com/pgvector/pgvector-go)
