# Bounded concurrency

Goroutines are cheap, so the obvious loop starts one per item. With a
thousand items that is a thousand goroutines all hitting the same database
or the same API at once:

```go
var wg sync.WaitGroup
for _, job := range jobs {
    wg.Add(1)
    go func() {
        defer wg.Done()
        process(job)
    }()
}
wg.Wait()
```

Nothing here is wrong for ten items. For ten thousand it is a way to run
out of file descriptors, exhaust a connection pool, or get rate-limited —
and the failure arrives under load, not in testing.

## A buffered channel is a semaphore

A buffered channel of capacity *n* admits *n* senders before it blocks.
That is all a counting semaphore is. Acquire by sending, release by
receiving:

```go
sem := make(chan struct{}, 3)        // at most 3 at once

var wg sync.WaitGroup
for _, job := range jobs {
    wg.Add(1)
    go func() {
        defer wg.Done()
        sem <- struct{}{}            // acquire: blocks when 3 are running
        defer func() { <-sem }()     // release
        process(job)
    }()
}
wg.Wait()
```

`struct{}` is the zero-byte type from the [structs article](../02-language-basics/10-structs.md) —
the channel carries no data, only permission. Run the loop above over a
thousand jobs while tracking how many are inside `process` at once and the
peak is exactly 3.

All the goroutines still get created — they just queue at the acquire.
That is fine: a goroutine parked on a channel holds a small stack and
no OS thread, which is the whole reason goroutines are cheap. If even
creating them is too much, use the [worker pool](06-concurrency-patterns.md)
instead, which starts a fixed number of goroutines and feeds them from a
channel. The rule of thumb:

| Shape | Use |
|---|---|
| a known slice of work, want to cap parallelism | semaphore |
| an open-ended stream of work | worker pool |
| work arriving faster than you can handle it | worker pool, so the queue is visible |

## Where to put the acquire

Acquire *inside* the goroutine, as above, not before the `go` statement.
Putting it outside blocks the loop itself, which means the goroutines are
created one at a time and anything after the loop waits too.

Release with `defer` so a panic or an early `return` inside `process`
cannot leak a slot. A leaked slot shrinks the limit permanently, and once
enough leak the whole thing deadlocks — a bug that looks like a hang, with
no error anywhere.

## Collecting results

Writing to distinct indices of a pre-sized slice needs no lock at all.
Different indices are different memory, so there is no race:

```go
jobs := []string{"a", "b", "c", "d"}
results := make([]string, len(jobs))

sem := make(chan struct{}, 2)
var wg sync.WaitGroup
for i, job := range jobs {
    wg.Add(1)
    go func() {
        defer wg.Done()
        sem <- struct{}{}
        defer func() { <-sem }()
        results[i] = strings.ToUpper(job)
    }()
}
wg.Wait()
fmt.Println(results)   // output: [A B C D]
```

Results stay in input order, which a channel would not give you. Note
that `i` and `job` are per-iteration variables, so the closure captures
the right ones — see [goroutines](01-goroutines.md).

Anything *shared* does need a lock. Keeping the first error is the common
case:

```go
var mu sync.Mutex
var firstErr error

// inside each goroutine:
if _, err := process(job); err != nil {
    mu.Lock()
    if firstErr == nil {
        firstErr = err
    }
    mu.Unlock()
}
```

This pattern is common enough that the Go team ships a helper for it in
`golang.org/x/sync/errgroup`. That is a separate module rather than part
of the standard library, so it is covered with the other third-party
libraries.

## `sync.Map`

A plain map guarded by a `sync.Mutex` is the right default. `sync.Map` is
a separate type for two specific shapes: a key is written once and read
many times, or different goroutines touch mostly disjoint keys. It trades
compile-time types for `any`, so reach for it only when profiling says so.

```go
var cache sync.Map

cache.Store("a", 1)
v, ok := cache.Load("a")
fmt.Println(v, ok)              // output: 1 true

actual, loaded := cache.LoadOrStore("b", 2)
fmt.Println(actual, loaded)     // output: 2 false

actual, loaded = cache.LoadOrStore("b", 99)
fmt.Println(actual, loaded)     // output: 2 true

cache.Delete("a")
_, ok = cache.Load("a")
fmt.Println(ok)                 // output: false
```

`LoadOrStore` is the one that earns its keep: it returns the existing
value if there is one and stores yours otherwise, atomically, so two
goroutines racing to populate the same key agree on a winner.

> **From Python:** there is no `ThreadPoolExecutor(max_workers=3)` in the
> standard library. The buffered channel *is* the `max_workers`, and you
> assemble the rest yourself — which is why the same six lines show up in
> every Go codebase.

## Quick reference

| Form | Meaning |
|---|---|
| `sem := make(chan struct{}, n)` | semaphore allowing `n` at once |
| `sem <- struct{}{}` | acquire (blocks when full) |
| `defer func() { <-sem }()` | release, panic-safe |
| `results[i] = v` | lock-free result collection by index |
| `mu.Lock()` around shared state | anything that is not per-index |
| `sync.Map` | write-once-read-many, or disjoint keys |

## Sources

- [Effective Go: channels — go.dev/doc/effective_go#channels](https://go.dev/doc/effective_go#channels)
- [`sync` package reference — pkg.go.dev/sync](https://pkg.go.dev/sync)
- [`sync.Map` — pkg.go.dev/sync#Map](https://pkg.go.dev/sync#Map)
- [Go blog: pipelines and cancellation — go.dev/blog/pipelines](https://go.dev/blog/pipelines)
- [Go memory model — go.dev/ref/mem](https://go.dev/ref/mem)
