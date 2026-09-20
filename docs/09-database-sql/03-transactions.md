# Transactions

A transaction makes several statements succeed or fail together. The
API is three methods; getting it right is about making sure the
rollback always happens, including on a panic.

```go
tx, err := db.BeginTx(ctx, nil)
defer tx.Rollback()   // no-op once committed

// ... statements on tx ...

return tx.Commit()
```

## `*sql.Tx` is one connection

`BeginTx` takes a connection out of the pool and holds it until you
commit or roll back. Everything in the transaction must go through the
`*sql.Tx` — a statement issued on `db` while a transaction is open runs
on a *different* connection and is not part of it.

A `*sql.Tx` that is never finished leaks that connection permanently.
Enough of them and the pool is exhausted, which presents as requests
hanging rather than as an error.

## The closure pattern

Rather than leaving commit and rollback to every call site, wrap it once
so the lifecycle cannot be forgotten:

```go
func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }

    if err := fn(tx); err != nil {
        if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
            return errors.Join(err, rbErr)
        }
        return err
    }
    return tx.Commit()
}
```

Two details in the error path. `errors.Join` keeps **both** failures
when the rollback itself fails — the original error explains what went
wrong, the rollback error explains why the data may now be
inconsistent. And `sql.ErrTxDone` is filtered out, because a
transaction the database already aborted reports that on rollback and
it is not a new problem:

```go
tx, _ := db.BeginTx(ctx, nil)
tx.Commit()
fmt.Println(errors.Is(tx.Rollback(), sql.ErrTxDone))   // output: true
```

That is also why the bare `defer tx.Rollback()` in the opening example
is safe: after a successful commit it does nothing.

Using it:

```go
err := WithTx(ctx, db, func(tx *sql.Tx) error {
    if _, err := tx.ExecContext(ctx, `UPDATE acct SET bal = bal - 50 WHERE id='a'`); err != nil {
        return err
    }
    _, err := tx.ExecContext(ctx, `UPDATE acct SET bal = bal + 50 WHERE id='b'`)
    return err
})
// a,b = 50 50
```

Return an error and nothing is written:

```go
err := WithTx(ctx, db, func(tx *sql.Tx) error {
    tx.ExecContext(ctx, `UPDATE acct SET bal = bal - 50 WHERE id='a'`)
    return errors.New("business rule failed")
})
// err: business rule failed
// balance unchanged
```

## A panic must roll back and keep panicking

Without this, a panic inside the closure unwinds past your commit and
rollback, and the transaction is left open until the connection dies:

```go
defer func() {
    if p := recover(); p != nil {
        _ = tx.Rollback()
        panic(p)          // re-raise: do not swallow it
    }
}()
```

Re-panicking matters. Swallowing turns a programmer error into a silent
no-op; re-raising lets the middleware
[recover](../08-http-with-net-http/03-middleware.md) turn it into a
`500` while the data stays consistent.

```go
// closure panics after an insert
// recovered: boom
// d inserted? false
```

## Carrying the transaction in the context

Passing `tx` explicitly means every function that might run inside a
transaction takes a `*sql.Tx` — which spreads through the whole call
tree and forces two versions of functions that could be either.

The alternative is to put it in the context and have each call resolve
what to use:

```go
type txKey struct{}

func dbOrTx(ctx context.Context, db *sql.DB) execer {
    if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
        return tx
    }
    return db
}
```

`execer` is a small interface with the methods both `*sql.DB` and
`*sql.Tx` already have, so the same code works either way:

```go
type execer interface {
    ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error)
    QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row
}
```

Now a function is transaction-agnostic:

```go
func count(ctx context.Context, db *sql.DB) int {
    var n int
    dbOrTx(ctx, db).QueryRowContext(ctx, `SELECT count(*) FROM acct`).Scan(&n)
    return n
}
```

**Be honest about the trade-off.** The signature no longer tells you
whether a function runs in a transaction, which is exactly the
information you want when reading unfamiliar code. It also makes a real
mistake possible: pass a context that has escaped the transaction's
lifetime and writes go to the wrong place. Explicit `tx` passing is
clearer in a small codebase; the context approach pays off once the call
tree is deep.

## Nesting has to join, not nest

SQL has no nested transactions. If an inner call begins another
transaction on the pool it takes a *second* connection, which then waits
on locks the first one holds — a self-deadlock that looks like a hang.

Checking the context first makes the inner call join the outer one:

```go
func WithTx(ctx context.Context, db *sql.DB, fn func(ctx context.Context) error) error {
    if _, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
        return fn(ctx)            // already in one: just run
    }
    // ... begin, defer, commit as above
}
```

```go
err := WithTx(ctx, db, func(ctx context.Context) error {
    return WithTx(ctx, db, func(ctx context.Context) error {
        _, err := dbOrTx(ctx, db).ExecContext(ctx, `INSERT INTO acct VALUES ('c', 5)`)
        return err
    })
})
// err: <nil> — one transaction, committed once
```

Note what this means: the inner block cannot roll back on its own. An
error anywhere aborts the whole outer transaction. If you genuinely need
a partial rollback, that is a `SAVEPOINT`, which you issue as SQL.

## Isolation and read-only

`BeginTx` takes options. `nil` uses the driver default, which for
PostgreSQL is read-committed:

```go
tx, err := db.BeginTx(ctx, &sql.TxOptions{
    Isolation: sql.LevelSerializable,
    ReadOnly:  true,
})
```

```go
_, err := tx.ExecContext(ctx, `INSERT INTO acct VALUES ('e',1)`)
fmt.Println(err != nil)   // output: true
```

`ReadOnly` is a cheap safety net for reporting queries. Stronger
isolation levels can fail at commit time with a serialisation error
that the *application* is expected to retry — so if you raise the level,
add the retry loop.

## Keep transactions short

The connection is held for the whole block, so never do anything slow
inside one:

- No HTTP calls. An unresponsive third party now holds a database
  connection open.
- No waiting on a channel or a lock.
- No work that could be done before `BeginTx`.

Read what you need, compute, then open the transaction and write.

## Cancellation

The context passed to `BeginTx` governs the whole transaction. If it is
cancelled, the transaction rolls back automatically and further use
returns an error — so a client disconnecting cannot leave a half-applied
write.

Background work that must complete regardless needs a context that does
*not* inherit the request's cancellation.

> **From Python:** there is no `with conn.begin():` and no
> autocommit-off mode — a transaction is an explicit object, and `defer`
> plus a wrapper function is how you get the `with` guarantee. The
> re-panic in the recover block is what `__exit__` does for you when an
> exception propagates.

## Quick reference

| Task | Call |
|---|---|
| begin | `db.BeginTx(ctx, nil)` |
| finish | `tx.Commit()` / `tx.Rollback()` |
| safety net | `defer tx.Rollback()` — a no-op after commit |
| already finished | `errors.Is(err, sql.ErrTxDone)` — ignore it |
| rollback also failed | `errors.Join(err, rbErr)` |
| panic | `recover`, roll back, **`panic(p)` again** |
| avoid passing `tx` everywhere | keep it in the context, resolve per call |
| nesting | detect and join; SQL has no nested transactions |
| partial rollback | a `SAVEPOINT`, issued as SQL |
| read-only / stronger isolation | `&sql.TxOptions{...}` |
| rule of thumb | no network calls, no waiting, keep it short |

## Sources

- [`sql.Tx` — pkg.go.dev/database/sql#Tx](https://pkg.go.dev/database/sql#Tx)
- [`sql.DB.BeginTx` — pkg.go.dev/database/sql#DB.BeginTx](https://pkg.go.dev/database/sql#DB.BeginTx)
- [`sql.TxOptions` — pkg.go.dev/database/sql#TxOptions](https://pkg.go.dev/database/sql#TxOptions)
- [`sql.ErrTxDone` — pkg.go.dev/database/sql#pkg-variables](https://pkg.go.dev/database/sql#pkg-variables)
- [Executing transactions — go.dev/doc/database/execute-transactions](https://go.dev/doc/database/execute-transactions)
