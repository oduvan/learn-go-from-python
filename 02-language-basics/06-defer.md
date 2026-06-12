# `defer`

`defer` schedules a function call to run when the surrounding function returns. It's how Go does cleanup — close files, unlock mutexes, finish HTTP responses, stop timers.

## The basic shape

```go
func read(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()           // runs when read returns
    return io.ReadAll(f)
}
```

The `defer f.Close()` line guarantees `f.Close()` runs regardless of how `read` exits — normal return, early return, or panic. You no longer have to remember "close f at every exit point."

> **From Python:** `defer` does roughly what `try`/`finally` or a `with` statement does — it ties cleanup to the surrounding scope's exit. Difference: `defer` is per-*function*, not per-block.

## Three semantic rules

### 1. LIFO order

Deferred calls run in reverse of the order they were scheduled.

```go
func main() {
    defer fmt.Println("third")
    defer fmt.Println("second")
    defer fmt.Println("first")
    fmt.Println("main")
}
// output:
// main
// first
// second
// third
```

Think of a stack: each `defer` pushes; the return pops them all.

### 2. Arguments evaluate **at defer time**

This trips up everyone exactly once.

```go
func main() {
    x := 1
    defer fmt.Println(x)      // x is captured as 1 right now
    x = 2
}
// output: 1
```

The argument `x` is evaluated when the `defer` statement runs, not when the function eventually executes. If you want to capture the *current* value at return time, defer a closure:

```go
func main() {
    x := 1
    defer func() {
        fmt.Println(x)        // closes over x — read at call time
    }()
    x = 2
}
// output: 2
```

### 3. Runs on every kind of return — including panic

```go
func cleanup() {
    defer fmt.Println("running cleanup")
    panic("boom")
}
// output:
// running cleanup
// panic: boom
//   ... stack trace ...
```

This is why `defer` is the natural place to put `recover()` calls:

```go
func safe() {
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("recovered from:", r)
        }
    }()
    panic("oops")
}
```

## Idiomatic uses

### Closing a file

```go
f, err := os.Open(path)
if err != nil {
    return err
}
defer f.Close()
```

### Unlocking a mutex

```go
var mu sync.Mutex

func update() {
    mu.Lock()
    defer mu.Unlock()
    // ... critical section ...
}
```

### Restoring state

```go
func quiet() func() {
    oldLevel := log.Default().Flags()
    log.Default().SetFlags(0)
    return func() {
        log.Default().SetFlags(oldLevel)
    }
}

func main() {
    defer quiet()()           // note the double () — quiet returns the cleanup
    log.Println("hello")
}
```

### Timing a function

```go
func track(name string) func() {
    start := time.Now()
    return func() {
        fmt.Printf("%s took %v\n", name, time.Since(start))
    }
}

func work() {
    defer track("work")()
    time.Sleep(100 * time.Millisecond)
}
// output: work took 100.xxx ms
```

## Gotchas

### Defer inside a loop accumulates

`defer` runs at **function** return, not loop iteration:

```go
func processAll(paths []string) error {
    for _, p := range paths {
        f, err := os.Open(p)
        if err != nil {
            return err
        }
        defer f.Close()       // !!! all files stay open until processAll returns
        // ... do work ...
    }
    return nil
}
```

Fix by extracting the work into a function so the `defer` is per-call:

```go
func processAll(paths []string) error {
    for _, p := range paths {
        if err := processOne(p); err != nil {
            return err
        }
    }
    return nil
}

func processOne(p string) error {
    f, err := os.Open(p)
    if err != nil {
        return err
    }
    defer f.Close()           // closes per iteration
    // ... do work ...
    return nil
}
```

### Defer has a tiny cost

It's measured in nanoseconds. Don't worry about it in normal code. In a hot inner loop (millions of calls/sec) you might choose to inline cleanup.

### Don't defer the wrong return

```go
f, err := os.Open(path)
defer f.Close()               // !!! panics if err != nil and f is nil
if err != nil { ... }
```

Always check `err` first, then `defer`:

```go
f, err := os.Open(path)
if err != nil {
    return err
}
defer f.Close()
```

## Sources

- [Defer statements — go.dev/ref/spec#Defer_statements](https://go.dev/ref/spec#Defer_statements)
- [Defer, Panic, and Recover — go.dev/blog/defer-panic-and-recover](https://go.dev/blog/defer-panic-and-recover)
- [Effective Go: Defer — go.dev/doc/effective_go#defer](https://go.dev/doc/effective_go#defer)
