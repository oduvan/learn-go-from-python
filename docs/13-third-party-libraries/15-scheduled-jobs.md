# Scheduled jobs

Running something on a schedule is easy. Running it exactly once when
three replicas of your service are all scheduling it is the actual
problem, and the standard library has no answer for it.

> **Module:** `github.com/go-co-op/gocron/v2`, plus PostgreSQL
> advisory locks for the coordination.

```go
s, err := gocron.NewScheduler()
s.NewJob(gocron.CronJob("0 3 * * *", false), gocron.NewTask(reindex, ctx))
s.Start()
```

## gocron

```go
s, err := gocron.NewScheduler()
if err != nil {
    return err
}
defer func() { _ = s.Shutdown() }()

j, err := s.NewJob(
    gocron.DurationJob(5*time.Minute),
    gocron.NewTask(pollUpstream, ctx),
    gocron.WithSingletonMode(gocron.LimitModeReschedule),
)

s.Start()
```

`NewJob` takes a schedule, a task, and options. Job definitions:

| Definition | Meaning |
|---|---|
| `gocron.DurationJob(d)` | every `d` |
| `gocron.CronJob("0 3 * * *", false)` | a cron expression; `true` for seconds |
| `gocron.DailyJob(1, atTimes)` | every *n* days at given times |
| `gocron.OneTimeJob(...)` | once, at a time you specify |

A malformed expression is rejected at `NewJob`, not at the first
firing — so a typo fails at startup:

```go
_, err := s2.NewJob(gocron.CronJob("not a cron", false), gocron.NewTask(func() {}))
// err != nil
```

`NewTask(fn, args...)` binds arguments, which is how you pass a
context. `Shutdown` waits for running jobs; call it, or a job in
flight is killed when the process exits.

### `WithSingletonMode` — within one process

```go
gocron.WithSingletonMode(gocron.LimitModeReschedule)
```

Stops a job overlapping with itself when a run takes longer than the
interval. `LimitModeReschedule` skips the missed tick;
`LimitModeWait` queues it.

This only coordinates **inside one process**. Three replicas each have
their own scheduler and will all fire.

## Exactly once across replicas

The general solutions are a distributed lock or a leader election. If
you already have PostgreSQL, its advisory locks give you one for free —
no extra infrastructure.

```go
func LockedRun(ctx context.Context, db *sql.DB, job string, fn func(context.Context) error) (bool, error) {
    conn, err := db.Conn(ctx)
    if err != nil {
        return false, err
    }
    defer conn.Close()

    key := JobLockKey(job)

    var acquired bool
    if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
        return false, err
    }
    if !acquired {
        return false, nil          // someone else has it
    }

    defer func() {
        unlockCtx := context.WithoutCancel(ctx)
        conn.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", key)
    }()

    return true, fn(ctx)
}
```

Five goroutines racing for the same job:

```
ran=1 skipped=4
```

And the lock is released afterwards, so the next scheduled run works:

```go
ok, err := LockedRun(ctx, db, "nightly-reindex", work)
// true <nil>
```

Three details carry this, and each one is a bug if you skip it.

**`db.Conn(ctx)`, not `db`.** An advisory lock is held by a *session*.
Taking it on one pooled connection and releasing it on another leaves
it held until that connection dies — which is the pool-exhaustion bug
that looks like the job "just stopped running". This is what
[`database/sql`](../09-database-sql/01-database-sql.md) means by
reserving a connection for session state.

**`pg_try_advisory_lock`, not `pg_advisory_lock`.** The `try` version
returns false immediately. The blocking version makes the other
replicas queue up and all run the job in sequence, which is the
opposite of what you wanted.

**`context.WithoutCancel` on the unlock.** If the job's context was
cancelled — a shutdown, a timeout — the unlock must still run, or the
lock stays held. That is the detaching pattern from
[context as a carrier](../11-architecture-and-conventions/02-context-as-a-carrier.md).

### The key must be a number

Advisory locks are keyed by `bigint`, so hash the job name. FNV is
fast and, unlike Go's map hashing, deterministic across processes:

```go
func JobLockKey(name string) int64 {
    h := fnv.New64a()
    h.Write([]byte(name))
    return int64(h.Sum64())
}
```

```go
JobLockKey("nightly-reindex") == JobLockKey("nightly-reindex")   // true
JobLockKey("a") != JobLockKey("b")                               // true
```

Determinism is the whole point: every replica must compute the same
number for the same job name. A collision between two job names would
mean one blocks the other, so keep the names distinct and few.

## Putting it together

```go
s.NewJob(
    gocron.CronJob("0 3 * * *", false),
    gocron.NewTask(func(ctx context.Context) {
        ran, err := LockedRun(ctx, db, "nightly-reindex", reindex)
        switch {
        case err != nil:
            slog.Error("nightly-reindex", "error", err)
        case !ran:
            slog.Debug("nightly-reindex skipped, another replica holds the lock")
        }
    }, ctx),
)
```

Note that "another replica ran it" is `Debug`, not `Warn`. It is the
expected case on every replica but one, and logging it louder means
every deploy looks like an incident.

## Practical notes

- **Recover inside the job.** A panic in a scheduled goroutine takes
  the process down, as
  [long-running goroutines](../05-concurrency/08-long-running-goroutines.md)
  covers. gocron may not save you; wrap the task.
- **Give each job a timeout.** `context.WithTimeout` around the work,
  or a stuck job holds its lock until the process restarts.
- **Time zones.** Cron expressions run in the scheduler's location; set
  it explicitly with `gocron.WithLocation` rather than inheriting the
  container's, which is usually UTC and occasionally is not.
- **Make jobs idempotent.** A missed tick, a retry after a crash, or
  two replicas briefly disagreeing about the lock should not corrupt
  anything.
- **Expose the last success.** A gauge with the last-run timestamp
  lets you alert on a job that has silently stopped — far more useful
  than alerting on failures, because the dangerous case is a job that
  never runs at all.

## When to use something else

This pattern suits periodic maintenance inside a service: a nightly
reindex, a stale-record sweep, a report.

For work that must survive a restart, retry with backoff, or be
observable as individual units, you want a job queue with durable
storage, not a scheduler. And if the platform already provides one — a
Kubernetes `CronJob`, a cloud scheduler — a separate binary run on a
schedule is simpler than coordinating locks, because the exactly-once
problem becomes somebody else's.

> **From Python:** gocron is APScheduler, and `LockedRun` is the
> distributed lock you would reach for Redis to get. Postgres advisory
> locks are the equivalent of `SELECT ... FOR UPDATE` used as a
> mutex — without needing a row.

## Quick reference

| Task | Form |
|---|---|
| a scheduler | `gocron.NewScheduler()`, `s.Start()`, `defer s.Shutdown()` |
| interval | `gocron.DurationJob(d)` |
| cron | `gocron.CronJob("0 3 * * *", false)` |
| pass a context | `gocron.NewTask(fn, ctx)` |
| no self-overlap | `WithSingletonMode(LimitModeReschedule)` — one process only |
| across replicas | `pg_try_advisory_lock` on a dedicated `db.Conn` |
| the key | `fnv.New64a()` over the job name |
| release | `defer` with `context.WithoutCancel` |
| skipped run | log at `Debug` — it is the normal case |
| alerting | a gauge of the last successful run |

## Sources

- [gocron — pkg.go.dev/github.com/go-co-op/gocron/v2](https://pkg.go.dev/github.com/go-co-op/gocron/v2)
- [PostgreSQL advisory locks — postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS)
- [`sql.DB.Conn` — pkg.go.dev/database/sql#DB.Conn](https://pkg.go.dev/database/sql#DB.Conn)
- [`hash/fnv` — pkg.go.dev/hash/fnv](https://pkg.go.dev/hash/fnv)
- [`context.WithoutCancel` — pkg.go.dev/context#WithoutCancel](https://pkg.go.dev/context#WithoutCancel)
