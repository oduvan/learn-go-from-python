# `database/sql`

`database/sql` is the standard interface to a SQL database. It is not an
ORM and does not generate queries — you write SQL, it manages
connections and maps results into Go values.

```go
var name string
err := db.QueryRowContext(ctx, `SELECT name FROM users WHERE id = $1`, id).Scan(&name)
```

> **One note on this topic.** `database/sql` needs a driver to talk to an
> actual database, and every driver is a third-party module. The
> examples here were run against PostgreSQL, but nothing in this article
> is driver-specific except the placeholder syntax. Choosing and
> importing a driver is covered with the other external libraries.

## `Open` gives you a pool, not a connection

```go
db, err := sql.Open("pgx", dsn)
defer db.Close()
```

Two surprises. `sql.Open` **does not connect** — it validates arguments
and returns immediately, so a wrong host gives you no error here. And
`*sql.DB` is a *pool*, not a connection: it is safe for concurrent use,
opens connections as needed, and should be created once at startup and
passed around. Opening one per request is a serious bug.

To find out whether the database is actually reachable, ask:

```go
if err := db.PingContext(ctx); err != nil {
    return fmt.Errorf("database unreachable: %w", err)
}
```

Do that at startup so a misconfigured DSN fails immediately rather than
on the first request.

### Pool limits

The defaults are unbounded open connections and only two idle ones,
which is wrong in both directions for a server:

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(25)
db.SetConnMaxLifetime(5 * time.Minute)
```

- `SetMaxOpenConns` caps total connections. Unlimited means a traffic
  spike exhausts the server's connection limit instead of queueing.
  Keep it comfortably under what the database allows, divided by the
  number of replicas.
- `SetMaxIdleConns` should usually match it, or connections are closed
  and reopened constantly.
- `SetConnMaxLifetime` retires connections periodically, which lets a
  load balancer rebalance and avoids server-side idle timeouts.

`db.Stats()` exposes the pool's state, which is worth putting on a
dashboard: `WaitCount` climbing means requests are queueing for a
connection.

## The three query calls

| Call | For |
|---|---|
| `ExecContext` | `INSERT`/`UPDATE`/`DELETE` — no rows back |
| `QueryRowContext` | exactly one row |
| `QueryContext` | many rows |

Always the `Context` variants. The plain `Exec`, `Query` and `QueryRow`
exist for compatibility and give you no cancellation — a slow query
outlives the request that asked for it.

```go
res, err := db.ExecContext(ctx, `INSERT INTO users (name, age) VALUES ($1, $2)`, "Ada", 36)
n, _ := res.RowsAffected()
fmt.Println(n, err)   // output: 1 <nil>
```

`LastInsertId` is not supported by every driver — PostgreSQL does not
have it. Use `RETURNING` instead, which is a single round trip:

```go
var id int64
err := db.QueryRowContext(ctx,
    `INSERT INTO users (name, age) VALUES ($1, $2) RETURNING id`, "Bo", 7).Scan(&id)
```

## Placeholders are not string formatting

Arguments are sent separately from the SQL. The database never parses
them as code, so there is nothing to escape and injection is impossible:

```go
evil := "Ada'; DROP TABLE users; --"

var cnt int
db.QueryRowContext(ctx, `SELECT count(*) FROM users WHERE name = $1`, evil).Scan(&cnt)
fmt.Println(cnt)   // output: 0
// the table is still there
```

It searched for a user literally named `Ada'; DROP TABLE users; --` and
found none.

Placeholder syntax is driver-specific: `$1, $2` for PostgreSQL, `?` for
MySQL and SQLite. **Never** build a query with `fmt.Sprintf` and user
input. Where an identifier must vary — a column to sort by — validate it
against an allow-list of known names, because identifiers cannot be
parameterised.

## `Scan` copies columns into your variables

Pass a pointer per column, in the order the query selects them:

```go
var u User
err := db.QueryRowContext(ctx,
    `SELECT id, name, email, age FROM users WHERE name = $1`, "Ada").
    Scan(&u.ID, &u.Name, &u.Email, &u.Age)
```

Prefer naming columns over `SELECT *`. With `*`, adding a column to the
table breaks every `Scan` at runtime.

A query matching nothing returns a sentinel error, which is an expected
outcome rather than a failure:

```go
err := db.QueryRowContext(ctx, `... WHERE name = $1`, "Nobody").Scan(&u.Name)
fmt.Println(errors.Is(err, sql.ErrNoRows))   // output: true
// err: sql: no rows in result set
```

Handle it explicitly — usually by turning it into your own "not found"
error, so callers are not coupled to `database/sql`.

## NULL needs a nullable target

Scanning a `NULL` into a plain `string` fails:

```go
var email string
err := db.QueryRowContext(ctx, `SELECT email FROM users WHERE name = $1`, "Ada").Scan(&email)
fmt.Println(err)
// output: sql: Scan error on column index 0, name "email": converting NULL to string is unsupported
```

Two fixes. A **pointer**, where `nil` means NULL:

```go
var email *string
// ... Scan(&email)
fmt.Println(email == nil)   // output: true
```

Or `sql.NullString`, which carries the flag explicitly:

```go
var ns sql.NullString
// ... Scan(&ns)
fmt.Printf("%v %q\n", ns.Valid, ns.String)   // output: false ""
```

There is a `sql.Null*` for each basic type, plus the generic
`sql.Null[T]`. Pointers read more naturally in structs; the `Null` types
are clearer when you must not confuse NULL with a zero value. The third
option is to fix it in SQL with `COALESCE(email, '')`.

## Iterating rows

```go
rows, err := db.QueryContext(ctx, `SELECT id, name, age FROM users ORDER BY name`)
if err != nil {
    return err
}
defer rows.Close()

var users []User
for rows.Next() {
    var u User
    if err := rows.Scan(&u.ID, &u.Name, &u.Age); err != nil {
        return err
    }
    users = append(users, u)
}
return rows.Err()
```

Four rules, and the last is the one people miss:

1. `defer rows.Close()` — an unclosed `Rows` holds a pooled connection.
   Leak enough and the pool is exhausted, which looks like a hang.
2. Check the error from `QueryContext` before touching `rows`.
3. Check the error from each `Scan`.
4. **Check `rows.Err()` after the loop.** `Next` returns false both at
   the end of the results and on failure. Without this check, a
   connection dropped mid-iteration looks exactly like a short result
   set — you return partial data and no error.

`rows.Close` is idempotent and safe alongside the `defer`.

## Errors from the database

A constraint violation comes back as a driver error carrying the
database's own message:

```go
_, err := db.ExecContext(ctx, `INSERT INTO users (name, age) VALUES ($1, $2)`, "Ada", 1)
fmt.Println(err)
// output: ERROR: duplicate key value violates unique constraint "users_name_key" (SQLSTATE 23505)
```

Matching on the message text is fragile. Drivers expose a typed error
you can inspect with `errors.As` to read the SQLSTATE code — `23505` is
unique violation, `23503` foreign key — which is how you turn a
duplicate into a clean "already exists" response. That type is
driver-specific, so it belongs with the driver.

## Prepared statements

`database/sql` prepares and caches statements for you when you pass
arguments, so an explicit `Prepare` is rarely worth it. It matters for a
statement executed many times in a tight loop:

```go
stmt, err := db.PrepareContext(ctx, `INSERT INTO users (name, age) VALUES ($1, $2)`)
defer stmt.Close()

for _, u := range users {
    if _, err := stmt.ExecContext(ctx, u.Name, u.Age); err != nil {
        return err
    }
}
```

A `*sql.Stmt` is bound to the pool and will reprepare itself on a
different connection as needed.

## When you need one connection

Anything with session state — a session variable, an advisory lock, a
temporary table — must run on a single connection. Pool calls give you
an arbitrary one each time, so reserve one:

```go
conn, err := db.Conn(ctx)
defer conn.Close()   // returns it to the pool
```

This matters more than it sounds: a lock taken on one pooled connection
and released on another is simply not released.

> **From Python:** `*sql.DB` is a connection *pool*, not DB-API's
> connection — closer to SQLAlchemy's engine. There is no cursor; you
> get `Rows`. And `Scan` is the opposite direction from `fetchone()`:
> you hand it pointers rather than receiving a tuple, which is what lets
> it be type-safe without reflection.

## Quick reference

| Task | Call |
|---|---|
| open the pool | `sql.Open(driver, dsn)` once — does not connect |
| verify connectivity | `db.PingContext(ctx)` at startup |
| pool limits | `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime` |
| write | `db.ExecContext(ctx, q, args...)` → `RowsAffected` |
| new id | `RETURNING id` + `QueryRowContext(...).Scan(&id)` |
| one row | `db.QueryRowContext(ctx, q, args...).Scan(&a, &b)` |
| no match | `errors.Is(err, sql.ErrNoRows)` |
| many rows | `QueryContext`, `defer rows.Close()`, **`rows.Err()`** |
| NULL | `*string`, or `sql.NullString` |
| user input | always a placeholder, never `fmt.Sprintf` |
| session state | `db.Conn(ctx)` |
| pool health | `db.Stats()` |

## Sources

- [`database/sql` package reference — pkg.go.dev/database/sql](https://pkg.go.dev/database/sql)
- [`sql.DB` — pkg.go.dev/database/sql#DB](https://pkg.go.dev/database/sql#DB)
- [`sql.Rows` — pkg.go.dev/database/sql#Rows](https://pkg.go.dev/database/sql#Rows)
- [`sql.ErrNoRows` — pkg.go.dev/database/sql#pkg-variables](https://pkg.go.dev/database/sql#pkg-variables)
- [Go wiki: SQL database drivers — go.dev/wiki/SQLDrivers](https://go.dev/wiki/SQLDrivers)
- [Accessing databases — go.dev/doc/database/](https://go.dev/doc/database/)
