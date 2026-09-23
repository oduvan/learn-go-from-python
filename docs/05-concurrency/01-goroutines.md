# Goroutines

A **goroutine** is a function running concurrently with everything else in
the program. You start one by putting `go` in front of a call. That's the
entire syntax — the function runs independently, and the caller continues
without waiting.

```go
go doWork()        // starts doWork concurrently; returns immediately
```

Goroutines are not OS threads. The Go runtime multiplexes many goroutines
onto a small pool of threads (an *M:N scheduler*: M goroutines share N
OS threads), so they're extremely
cheap — a few kilobytes of stack each, and you can have hundreds of
thousands. Creating one is closer in cost to a function call than to
spawning a thread.

> **From Python:** a goroutine is like a task scheduled on an event loop,
> except there's no `async`/`await` colouring — *any* function can run in a
> goroutine — and the runtime uses real OS threads underneath, so
> goroutines run in **parallel** on multiple cores, not just concurrently.

## The main goroutine exits — and takes everything with it

`main` itself runs in a goroutine. When `main` returns, the program exits
**immediately**, without waiting for other goroutines to finish. So this
often prints nothing:

```go
func main() {
    go fmt.Println("hello from a goroutine")
    // main returns here; the program may exit before the goroutine runs
}
```

The goroutine *might* not get a chance to run. You need to **synchronise** —
to make `main` wait until the work is done. Sleeping is not synchronisation
(it's a guess); the right tool is `sync.WaitGroup`.

## Waiting with `sync.WaitGroup`

A `WaitGroup` counts outstanding goroutines. `Add(n)` raises the count,
each goroutine calls `Done()` when finished (via `defer`), and `Wait()`
blocks until the count hits zero.

```go
// checkHealth pings one service. The sleep stands in for real network latency.
func checkHealth(name string) string {
    time.Sleep(100 * time.Millisecond)
    return name + ": ok"
}

func main() {
    services := []string{"api", "db", "cache"}
    statuses := make([]string, len(services))

    var wg sync.WaitGroup
    for i, name := range services {
        wg.Add(1)
        go func() {
            defer wg.Done()
            statuses[i] = checkHealth(name)   // each goroutine writes its own slot
        }()
    }
    wg.Wait()                                 // block until all three checks finish

    for _, s := range statuses {
        fmt.Println(s)
    }
}
// output:
// api: ok
// db: ok
// cache: ok
```

This is *why* goroutines exist: **overlapping independent, slow work.** Each
`checkHealth` takes 100 ms. Run one after another that's ~300 ms; launched as
goroutines they all wait at the same time, so the whole batch finishes in
~100 ms. Swap the sleep for a real HTTP request or DB query and it's a
genuine concurrent health-checker.

Two things keep it correct:

- `Wait()` guarantees every check finished before we read `statuses`.
- Each goroutine writes a **different** slice slot, so there's no concurrent
  write to the same memory (no data race — confirm with `go run -race`).

## The loop variable is per-iteration

The loop above captured both `i` and `name` inside each goroutine, and every
iteration gets its **own** copy — so the goroutine for `"db"` really sees
`name == "db"`, not whatever value the loop finished on. This is the modern
Go behaviour: each iteration of a `for` loop has a fresh loop variable.

In older Go this was a notorious bug — the single shared variable was
usually the loop's *last* value by the time the goroutines ran, so every
goroutine would have used `"cache"`. People worked around it by passing the
value in as an argument:

```go
for i, name := range services {
    wg.Add(1)
    go func(i int, name string) {   // old workaround: pass copies as arguments
        defer wg.Done()
        statuses[i] = checkHealth(name)
    }(i, name)
}
```

That still works and you'll see it in older code, but it's no longer
necessary.

## Goroutines run in parallel

With more than one CPU available, goroutines genuinely run at the same
time. `GOMAXPROCS` controls how many can execute simultaneously (it
defaults to the number of CPUs). Because of real parallelism, **any shared
mutable state needs protection** — that's what channels and the `sync`
package are for, covered next.

```go
fmt.Println(runtime.NumGoroutine())   // how many goroutines exist right now
```

## A goroutine's panic crashes the program

A panic in a goroutine that isn't recovered **inside that same goroutine**
takes down the whole process — you can't recover it from the parent. Each
goroutine is responsible for its own `recover` (see panic and recover).

```go
func main() {
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        panic("boom")     // not recovered here
    }()
    wg.Wait()
}
// panic: boom
// (the whole program crashes — the deferred wg.Done runs during unwinding,
//  but nothing in main can catch this)
```

The fix is to `recover` **in the goroutine itself**:

```go
go func() {
    defer wg.Done()
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("recovered:", r)   // output: recovered: boom
        }
    }()
    panic("boom")
}()
// main keeps running
```

## Quick reference

| Construct | Meaning |
|---|---|
| `go f(args)` | run `f` concurrently; returns immediately |
| `func main()` returns | program exits, abandoning other goroutines |
| `var wg sync.WaitGroup` | count outstanding goroutines |
| `wg.Add(n)` / `wg.Done()` / `wg.Wait()` | raise / decrement / block-until-zero |
| per-iteration loop var | each iteration captures its own copy |
| `runtime.GOMAXPROCS` | max goroutines executing in parallel |
| `runtime.NumGoroutine()` | current goroutine count |

## Sources

- [Go statements — go.dev/ref/spec#Go_statements](https://go.dev/ref/spec#Go_statements)
- [sync.WaitGroup — pkg.go.dev/sync#WaitGroup](https://pkg.go.dev/sync#WaitGroup)
- [Effective Go: goroutines — go.dev/doc/effective_go#goroutines](https://go.dev/doc/effective_go#goroutines)
- [go.dev/blog/loopvar-preview — the loop variable change](https://go.dev/blog/loopvar-preview)
- [runtime.GOMAXPROCS — pkg.go.dev/runtime#GOMAXPROCS](https://pkg.go.dev/runtime#GOMAXPROCS)
