# testcontainers

Some things cannot honestly be faked. A query with a window function, a
unique constraint, a transaction isolation level — the only way to know
those work is to run them against the real database. testcontainers
starts one from your test.

> **Modules:** `github.com/testcontainers/testcontainers-go` and its
> per-technology modules.

[Fakes and stubs](../10-testing/04-fakes-and-stubs.md) argued that the
store's own tests cannot use a fake store. This is what they use
instead.

```go
c, err := postgres.Run(ctx, "postgres:16-alpine",
    postgres.WithDatabase("app"),
    postgres.WithUsername("u"),
    postgres.WithPassword("p"),
)
dsn, err := c.ConnectionString(ctx, "sslmode=disable")
```

## Wait for it to be ready

A started container is not a ready database. Without a wait strategy
you connect to a port that is open but not yet accepting queries, and
get a test that fails one run in twenty:

```go
testcontainers.WithWaitStrategy(
    wait.ForLog("database system is ready to accept connections").
        WithOccurrence(2).
        WithStartupTimeout(60*time.Second),
)
```

`WithOccurrence(2)` is not superstition. The Postgres image starts the
server once to run initialisation scripts, shuts it down, and starts it
again — so the first occurrence of that line is the throwaway instance.

Other strategies: `wait.ForListeningPort`, `wait.ForHTTP("/health")`,
`wait.ForSQL`. Prefer one that proves the service is usable over one
that proves a port is open.

## Start it once per test binary

Container startup dominates everything. Do it once behind a
`sync.Once`:

```go
var (
    once     sync.Once
    baseDSN  string
    setupErr error
)

func sharedPostgres(t *testing.T) string {
    t.Helper()
    once.Do(func() {
        c, err := postgres.Run(ctx, "postgres:16-alpine", /* ... */)
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
```

The difference is stark:

```
--- PASS: TestInsertAndRead (1.52s)            ← started the container
--- PASS: TestSecondUsesSameContainer (0.01s)  ← reused it
```

Note the error is captured in a variable rather than failing inside
`once.Do`. Calling `t.Fatalf` in there would abort the first test
holding the lock and leave `once` marked done, so every later test
would silently get an empty DSN.

## One entry point for tests

Wrap it so tests say one line:

```go
func SetupTestDB(t *testing.T) *sql.DB {
    t.Helper()

    dsn := sharedPostgres(t)
    db, err := sql.Open("pgx", dsn)
    require.NoError(t, err)
    t.Cleanup(func() { db.Close() })

    // fresh schema for this test
    _, err = db.Exec(`DROP TABLE IF EXISTS users;
                      CREATE TABLE users (id bigserial PRIMARY KEY, name text NOT NULL)`)
    require.NoError(t, err)

    return db
}
```

```go
func TestInsertAndRead(t *testing.T) {
    db := SetupTestDB(t)
    // ...
}
```

Isolating tests from each other matters more than isolating them from
the container. Options, cheapest first: truncate the tables you
touched; run each test in a transaction that always rolls back; or
clone a database from a migrated template.

The template clone is the one that scales. Migrate once into a template
database, then per test binary:

```sql
CREATE DATABASE test_7 TEMPLATE app_template;
```

Postgres copies the files, so it is far faster than re-running
migrations, and each binary gets full isolation including DDL. Between
tests within a binary, truncating the touched tables is usually enough.

### Let CI reuse a service container

CI often already provides a database, and starting another inside it is
slow or impossible. Check for an externally-supplied DSN first and only
fall back to testcontainers:

```go
func testDSN(t *testing.T) string {
    if dsn := os.Getenv("TEST_DATABASE_DSN"); dsn != "" {
        return dsn                      // CI service container
    }
    return sharedPostgres(t)            // local Docker
}
```

One helper, two environments, no build tags. This is often the
difference between database tests running in CI and quietly not.

Run your real [migrations](09-goose-migrations.md) against the
container rather than hand-writing the schema — then the tests also
prove the migrations work.

## Cleanup

`testcontainers.CleanupContainer(t, c)` ties a container's lifetime to
a test. For a shared one, the Ryuk reaper sidecar removes it when the
test process exits, including after a panic or a `SIGKILL`.

Ryuk needs to reach the Docker socket. Under a non-standard runtime —
Colima, Podman, a remote host — that usually means setting
`TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE` or `DOCKER_HOST`, or containers
leak. It is worth putting that in your Makefile rather than each
developer rediscovering it.

## Gate at runtime, never with a build tag

These tests need Docker, so some environments cannot run them. The
temptation is `//go:build integration`. Resist it: if nothing passes
`-tags integration`, the tests never run *anywhere*, and they look
present in the repo the whole time.

Skip at runtime, where the skip is visible in the output:

```go
if os.Getenv("SKIP_DB_TESTS") != "" {
    t.Skip("SKIP_DB_TESTS set")
}
```

Better still, invert it in CI: make the absence of Docker a failure
there, so a misconfigured pipeline is loud rather than quietly green.
This is the same argument as in
[build and codegen](../11-architecture-and-conventions/05-build-codegen-and-cgo.md).

## Beyond Postgres

Modules exist for Redis, Kafka, RabbitMQ, MongoDB, LocalStack and
many more, all with the same shape. `testcontainers.GenericContainer`
runs any image, so anything with a container has a test double.

## What it costs

- **Seconds, not milliseconds.** Keep these tests separate from your
  fast unit tests so `go test ./...` on a package stays quick.
- **Docker must be available**, including in CI.
- **Images must be pulled**, which is slow the first time and needs
  network. Pin tags — `postgres:16-alpine`, never `latest` — so the
  version cannot change under you.
- **Parallelism needs thought.** Tests sharing one container are not
  isolated by default; that is what the per-test reset is for.

Use it for the store layer and anything genuinely integration-shaped.
Business logic should still be tested against interfaces, in
milliseconds.

> **From Python:** this is `pytest-docker` or `testcontainers-python`,
> with the container lifecycle managed from the test binary rather than
> from a fixture. `sync.Once` plays the role of a session-scoped
> fixture.

## Quick reference

| Task | Form |
|---|---|
| start Postgres | `postgres.Run(ctx, "postgres:16-alpine", opts...)` |
| wait until usable | `wait.ForLog(...).WithOccurrence(2)` |
| connection string | `c.ConnectionString(ctx, "sslmode=disable")` |
| start once | `sync.Once`, capturing the error in a variable |
| per-test entry point | a `SetupTestDB(t)` helper with `t.Cleanup` |
| isolation | truncate, roll back, or a template database |
| schema | run your real migrations |
| teardown | `CleanupContainer`, or Ryuk for a shared one |
| non-standard Docker | `TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE` |
| skipping | a runtime env check — **never a build tag** |
| images | pinned tags, never `latest` |

## Sources

- [testcontainers-go — golang.testcontainers.org](https://golang.testcontainers.org/)
- [Postgres module — golang.testcontainers.org/modules/postgres/](https://golang.testcontainers.org/modules/postgres/)
- [Wait strategies — golang.testcontainers.org/features/wait/introduction/](https://golang.testcontainers.org/features/wait/introduction/)
- [`testcontainers-go` — pkg.go.dev/github.com/testcontainers/testcontainers-go](https://pkg.go.dev/github.com/testcontainers/testcontainers-go)
