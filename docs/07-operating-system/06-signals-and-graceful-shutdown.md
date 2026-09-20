# Signals and graceful shutdown

When an orchestrator stops your service it sends `SIGTERM` and starts a
clock. Handle it and in-flight work finishes; ignore it and you are
killed mid-request. The whole mechanism is one function plus the
concurrency patterns from
[long-running goroutines](../05-concurrency/08-long-running-goroutines.md).

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

<-ctx.Done()
fmt.Println("signal received:", ctx.Err())
// output: signal received: context canceled
```

## `signal.NotifyContext`

This is the modern form and the one to reach for. It returns a context
cancelled when any listed signal arrives, which plugs straight into
every `ctx.Done()` you have already written — no separate channel to
plumb through.

Note that `ctx.Err()` is `context.Canceled`, not something signal-shaped.
A shutdown looks like any other cancellation to the code being shut down,
which is the point.

Call the returned `stop` when you are done. It restores default signal
behaviour, so a second `Ctrl-C` during a slow shutdown kills the process
instead of being swallowed.

The older `signal.Notify` with a channel still works and still appears
in existing code:

```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
<-sigCh
```

The channel **must be buffered**. Signal delivery never blocks, so an
unbuffered channel with nobody receiving at that instant drops the
signal silently.

## Which signals

| Signal | Meaning | Catchable |
|---|---|---|
| `SIGTERM` | orchestrators' "please stop" | yes — handle this one |
| `SIGINT` | `Ctrl-C` | yes |
| `SIGHUP` | terminal closed; often "reload config" | yes |
| `SIGKILL` | forced kill | **no** |

`SIGKILL` cannot be caught, blocked, or deferred. That is precisely why
the grace period matters: a container runtime sends `SIGTERM`, waits,
and then sends `SIGKILL`. Whatever you have not finished by then is lost.

## The shape of a shutdown

Three steps, in order:

1. **Stop accepting new work.** A server stops listening; a consumer
   stops pulling.
2. **Drain what is in flight**, with a deadline.
3. **Release resources** — flush buffers, close pools.

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    srv := startServer()

    <-ctx.Done()
    stop()   // a second signal now kills us, as it should

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    if err := srv.Shutdown(shutdownCtx); err != nil {
        fmt.Fprintln(os.Stderr, "shutdown:", err)
    }
}
```

`http.Server` and its `Shutdown` method belong to the next topic; what
matters here is the shape — something that stops accepting work and
drains, given a deadline.

The shutdown context is built from `context.Background()`, **not** from
`ctx`. Deriving it from the already-cancelled `ctx` would give you a
context that is dead on arrival, and nothing would drain.

## Draining with a deadline

`wg.Wait()` has no timeout. Convert it to a channel and race it, the
trick from the concurrency topic:

```go
func drain(wg *sync.WaitGroup, budget time.Duration) string {
    done := make(chan struct{})
    go func() { wg.Wait(); close(done) }()

    select {
    case <-done:
        return "drained cleanly"
    case <-time.After(budget):
        return "budget expired"
    }
}
```

```go
// two workers, 30ms and 40ms, budget 500ms
// output:
// worker 1 finished
// worker 2 finished
// drained cleanly
```

When the budget runs out, cancel the workers' own context so they stop
early rather than being killed mid-write:

```go
// one worker needing 5s, budget 100ms
// output:
// budget expired
// worker 3 cancelled
```

That ordering is the whole design: ask nicely, wait a bounded time, then
insist.

## Budget arithmetic

Split the grace period between phases and make sure the total fits
inside it. If the orchestrator allows 30 seconds, a 25-second request
drain plus a 10-second background drain overruns and the remainder is
killed. Name the budgets as constants so the sum is visible:

```go
const (
    requestDrainBudget = 12 * time.Second
    edgeDrainBudget    = 8 * time.Second
)
```

## Things that quietly break it

- **`os.Exit` skips `defer`.** Any cleanup in a deferred function does
  not run, as [flags and environment](04-flags-and-environment.md)
  showed. Return from `main` instead.
- **A deferred `stop()` alone is not enough.** Call `stop()` explicitly
  after `<-ctx.Done()` so a second signal is honoured.
- **PID 1 in a container has no default signal handlers.** If your
  process is PID 1 and does not handle `SIGTERM`, the default action
  does not apply and the signal is ignored — the container then takes
  the full grace period and dies to `SIGKILL` every time.
- **Detached work loses its context.** Anything started from a request
  but outliving it needs its own lifetime, or it is cancelled the moment
  the request ends.

Distinguishing why a context ended uses `errors.Is`:

```go
fmt.Println(errors.Is(ctx.Err(), context.DeadlineExceeded))   // output: true
fmt.Println(errors.Is(ctx.Err(), context.Canceled))           // output: false
```

> **From Python:** `signal.NotifyContext` replaces
> `signal.signal(SIGTERM, handler)`, and it is better behaved — there is
> no separate handler running at an arbitrary point, just a context that
> becomes cancelled, observed wherever you already select on one.

## Quick reference

| Task | Call |
|---|---|
| catch signals as a context | `signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)` |
| restore default behaviour | the returned `stop()` — call it explicitly too |
| older channel form | `signal.Notify(ch, ...)` with a **buffered** channel |
| shutdown deadline | `context.WithTimeout(context.Background(), d)` — not from the dead ctx |
| stop a server | `srv.Shutdown(shutdownCtx)` |
| wait with a deadline | `wg.Wait()` in a goroutine, `close(done)`, `select` |
| after the budget | cancel the workers' context |
| why it ended | `errors.Is(ctx.Err(), context.DeadlineExceeded)` |

## Sources

- [`os/signal` package reference — pkg.go.dev/os/signal](https://pkg.go.dev/os/signal)
- [`signal.NotifyContext` — pkg.go.dev/os/signal#NotifyContext](https://pkg.go.dev/os/signal#NotifyContext)
- [`signal.Notify` — pkg.go.dev/os/signal#Notify](https://pkg.go.dev/os/signal#Notify)
- [`context` package reference — pkg.go.dev/context](https://pkg.go.dev/context)
- [`http.Server.Shutdown` — pkg.go.dev/net/http#Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)
