# Synchronization: sync and atomic

Channels are Go's preferred way to coordinate goroutines, but sometimes you
just need to protect a piece of shared state. The `sync` and `sync/atomic`
packages provide the classic tools: mutexes, one-time initialisation, and
lock-free counters.

## The problem: data races

When two goroutines touch the same variable and at least one writes,
without synchronisation, the result is a **data race** — undefined
behaviour. This loop *looks* like it counts to 1000 but doesn't reliably:

```go
count := 0
var wg sync.WaitGroup
for i := 0; i < 1000; i++ {
    wg.Add(1)
    go func() { defer wg.Done(); count++ }()   // RACE: concurrent writes
}
wg.Wait()
fmt.Println(count)   // unpredictable: often < 1000
```

`count++` is read-modify-write — three steps that can interleave and lose
updates.

## `sync.Mutex`: mutual exclusion

A `Mutex` lets only one goroutine into the guarded section at a time. `Lock`
before touching the shared state, `Unlock` after (usually via `defer`).

```go
var mu sync.Mutex
count := 0
var wg sync.WaitGroup

for i := 0; i < 1000; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        mu.Lock()
        count++
        mu.Unlock()
    }()
}
wg.Wait()
fmt.Println(count)   // output: 1000
```

Now the increments are serialised, so the result is always 1000.

## `sync.RWMutex`: many readers or one writer

When reads vastly outnumber writes, an `RWMutex` lets any number of readers
proceed in parallel (`RLock`/`RUnlock`) while writes (`Lock`/`Unlock`) get
exclusive access.

```go
var mu sync.RWMutex
mu.RLock()           // multiple readers can hold this simultaneously
_ = sharedValue
mu.RUnlock()
```

## `sync.Once`: run exactly once

`Once.Do(f)` runs `f` a single time, no matter how many goroutines call it
or how often — the standard way to do lazy, thread-safe initialisation.

```go
var once sync.Once
setup := func() { fmt.Println("init") }

for i := 0; i < 3; i++ {
    once.Do(setup)
}
// output:
// init
```

## `sync/atomic`: lock-free counters

For a single integer, an atomic type is simpler and faster than a mutex.
The typed atomics (`atomic.Int64`, `atomic.Bool`, …) carry their own
synchronisation:

```go
var count atomic.Int64
var wg sync.WaitGroup

for i := 0; i < 1000; i++ {
    wg.Add(1)
    go func() { defer wg.Done(); count.Add(1) }()
}
wg.Wait()
fmt.Println(count.Load())   // output: 1000
```

Reach for atomics for simple counters and flags; reach for a mutex when you
must keep **several** values consistent together.

## The race detector

Go ships a **race detector** that instruments memory access and reports
races at runtime. Run your program or tests with `-race`:

```bash
go run -race .
go test -race ./...
```

On a real race it prints a `WARNING: DATA RACE` with both stacks. It only
catches races that actually occur during the run, so use it with tests that
exercise concurrency. It's one of the most valuable tools in Go — make a
habit of running tests with `-race` in CI.

> **From Python:** there's no GIL serialising bytecode, so Go code really
> does race. The flip side: real parallelism, plus a first-class detector
> to catch the mistakes the GIL would have hidden.

## Quick reference

| Tool | Use |
|---|---|
| `sync.Mutex` (`Lock`/`Unlock`) | exclusive access to shared state |
| `sync.RWMutex` (`RLock`/`RUnlock`) | many readers or one writer |
| `sync.Once` (`Do`) | run an init exactly once |
| `sync.WaitGroup` | wait for goroutines to finish |
| `atomic.Int64` etc. (`Add`/`Load`/`Store`) | lock-free counters & flags |
| `go run -race` / `go test -race` | detect data races |

## Sources

- [sync — pkg.go.dev/sync](https://pkg.go.dev/sync)
- [sync/atomic — pkg.go.dev/sync/atomic](https://pkg.go.dev/sync/atomic)
- [The Go Memory Model — go.dev/ref/mem](https://go.dev/ref/mem)
- [Data Race Detector — go.dev/doc/articles/race_detector](https://go.dev/doc/articles/race_detector)
