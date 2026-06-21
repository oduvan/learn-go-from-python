# Goroutines

A **goroutine** is a function running concurrently with everything else in
the program. You start one by putting `go` in front of a call. That's the
entire syntax — the function runs independently, and the caller continues
without waiting.

```go
go doWork()        // starts doWork concurrently; returns immediately
```

Goroutines are not OS threads. The Go runtime multiplexes many goroutines
onto a small pool of threads (an *M:N scheduler*), so they're extremely
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
var wg sync.WaitGroup
results := make([]int, 3)

for i := 0; i < 3; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        results[i] = i * i      // each goroutine writes its own index — no clash
    }()
}

wg.Wait()                        // block until all three call Done
fmt.Println(results)             // output: [0 1 4]
```

Two things make this correct and deterministic:

- `Wait()` guarantees all goroutines finished before we read `results`.
- Each goroutine writes a **different** slice element, so there's no
  concurrent write to the same memory (no data race).

## The loop variable is per-iteration

In the loop above, each iteration has its **own** `i`, so the goroutine's
closure captures the right value. This is the modern Go behaviour — every
iteration of a `for` loop gets a fresh copy of the loop variable.

```go
for _, s := range []string{"a", "b", "c"} {
    wg.Add(1)
    go func() {
        defer wg.Done()
        _ = s            // each goroutine sees its own s — "a", "b", "c"
    }()
}
```

In older Go this was a classic bug: the single shared `i` would often be
the loop's final value by the time the goroutines ran, so people passed it
as an argument (`go func(i int){...}(i)`). That workaround still works and
you'll see it in older code, but it's no longer necessary.

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
