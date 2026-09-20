# Long-running goroutines

Most goroutines in the earlier articles are short: start, do one thing,
finish before `Wait` returns. A server also has goroutines that live for
the whole process — a poller, a queue consumer, a background writer.
Those need three things the short ones do not: their own panic handling,
a way to stop, and a way for shutdown to wait for them.

```go
go func() {
    panic("worker exploded")
}()

time.Sleep(time.Second)
fmt.Println("never reached")
// panic: worker exploded
// exit status 2
```

## A panic cannot cross a goroutine boundary

`recover` only works in a function deferred by the *same* goroutine that
is panicking. The goroutine that started the work is a different
goroutine, so its `defer`/`recover` never sees the panic — and an
unrecovered panic takes down the entire process:

```go
defer func() {
    if r := recover(); r != nil {
        fmt.Println("caller recovered:", r)   // never printed
    }
}()

go func() {
    panic("worker exploded")
}()
```

This follows from [panic and recover](../02-language-basics/15-panic-and-recover.md),
but the consequence is worth stating plainly: **every goroutine you start
that outlives one request needs its own recover**, or one bad input
anywhere takes the whole server with it.

The fix is a wrapper you start goroutines through:

```go
func safeGo(name string, fn func()) {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                fmt.Printf("goroutine %s panicked: %v\n", name, r)
            }
        }()
        fn()
    }()
}
```

```go
safeGo("worker", func() { panic("boom") })

time.Sleep(50 * time.Millisecond)
fmt.Println("main is still running")
// output:
// goroutine worker panicked: boom
// main is still running
```

The sleep is only there so the example has something to show: `main`
would otherwise return and end the process before the goroutine ever got
to print. That is a property of the example, not of `safeGo`.

In real code, log the stack alongside the value. A panic message on its
own tells you almost nothing about where it came from:

```go
if r := recover(); r != nil {
    fmt.Printf("goroutine %s panicked: %v\n%s", name, r, debug.Stack())
}
```

`debug.Stack()` comes from `runtime/debug` and returns the stack of the
goroutine that calls it — so it has to be called inside the recovering
`defer`, while that goroutine is still on the stack.

## Stopping: a `select` on `ctx.Done()`

A long-running loop needs an off switch, and the switch is the
[context](05-context.md) it was handed. A ticker loop is the shape you
will write most often:

```go
func poll(ctx context.Context, every time.Duration) {
    t := time.NewTicker(every)
    defer t.Stop()

    for {
        select {
        case <-ctx.Done():
            fmt.Println("poller stopping:", ctx.Err())
            return
        case <-t.C:
            fmt.Println("tick")
        }
    }
}
```

```go
ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
defer cancel()
poll(ctx, 100*time.Millisecond)
// output:
// tick
// tick
// poller stopping: context deadline exceeded
```

`defer t.Stop()` matters. A `time.Ticker` holds a runtime timer that
keeps firing until stopped, so a loop that returns without stopping its
ticker leaks one per call.

Put the `ctx.Done()` case first as a habit. `select` picks randomly among
ready cases, so ordering does not actually prioritise it — but reading it
first makes the exit path the first thing anyone sees.

## Draining: waiting for work to finish

Cancelling tells goroutines to stop. It does not wait for them, and it
does not stop new work being handed to them a microsecond later. A
component that owns background work needs to refuse new work, then wait —
with a deadline, because waiting forever is not a shutdown:

```go
type Pool struct {
    mu     sync.Mutex
    closed bool
    wg     sync.WaitGroup
}

func (p *Pool) Go(fn func()) bool {
    p.mu.Lock()
    defer p.mu.Unlock()
    if p.closed {
        return false
    }
    p.wg.Add(1)
    go func() {
        defer p.wg.Done()
        fn()
    }()
    return true
}
```

The lock is not optional. `wg.Add` must not run once `wg.Wait` has
started, and the `closed` flag under the same mutex is what guarantees
that.

```go
func (p *Pool) Shutdown(ctx context.Context) {
    p.mu.Lock()
    p.closed = true
    p.mu.Unlock()

    drained := make(chan struct{})
    go func() {
        p.wg.Wait()
        close(drained)
    }()

    select {
    case <-drained:
        fmt.Println("all work finished")
    case <-ctx.Done():
        fmt.Println("drain budget expired")
    }
}
```

`wg.Wait()` has no timeout, so it is moved into its own goroutine and
converted into a channel that closes when it returns. Now it can race a
context in a `select`. This "wait, but not forever" conversion is worth
remembering — it works for anything blocking that you cannot cancel.

```go
var p Pool
p.Go(func() { time.Sleep(50 * time.Millisecond) })
p.Shutdown(context.Background())
fmt.Println("accepted after close:", p.Go(func() {}))
// output:
// all work finished
// accepted after close: false
```

And when the work outlasts the budget:

```go
var p Pool
p.Go(func() { time.Sleep(2 * time.Second) })

ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()
p.Shutdown(ctx)
// output: drain budget expired
```

Expiring is a real outcome, not a bug. Say so in a log line — silently
dropping in-flight work is how a "clean" deploy loses data.

> **From Python:** `safeGo` is roughly what `Thread` gives you for free,
> since a thread's exception prints and dies without killing the process.
> Go is the opposite: the default is fatal, and you opt into surviving.

## Quick reference

| Concern | Form |
|---|---|
| panic in a background goroutine | its own `defer`/`recover`, inside that goroutine |
| stopping a loop | `select` with a `case <-ctx.Done(): return` |
| repeating work | `time.NewTicker` plus `defer t.Stop()` |
| refusing new work | a `closed` flag under the same mutex as `wg.Add` |
| waiting with a deadline | `wg.Wait()` in a goroutine, `close(ch)`, `select` on it |

## Sources

- [Handling panics — go.dev/ref/spec#Handling_panics](https://go.dev/ref/spec#Handling_panics)
- [`sync.WaitGroup` — pkg.go.dev/sync#WaitGroup](https://pkg.go.dev/sync#WaitGroup)
- [`time.Ticker` — pkg.go.dev/time#Ticker](https://pkg.go.dev/time#Ticker)
- [`context` package reference — pkg.go.dev/context](https://pkg.go.dev/context)
- [`runtime/debug.Stack` — pkg.go.dev/runtime/debug#Stack](https://pkg.go.dev/runtime/debug#Stack)
- [Go blog: pipelines and cancellation — go.dev/blog/pipelines](https://go.dev/blog/pipelines)
