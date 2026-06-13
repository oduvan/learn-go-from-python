# `go tool trace` — execution tracer

`go tool trace` is the viewer for **execution traces** produced by Go's runtime tracer (the `runtime/trace` package). A trace is a binary log of low-level runtime events with **nanosecond-precision timestamps and stack traces**.

## Tracer vs profiler

The tracer is **not** a profiler. From the official diagnostics page:

> "However, it is not great for identifying hot spots such as analyzing the cause of excessive memory or CPU usage. Use profiling tools instead first to address them."

- **`pprof`** answers *"where is my program spending CPU or allocating memory?"* via statistical sampling.
- **`go tool trace`** answers *"how are my goroutines interacting over time?"* — scheduling, blocking, GC pauses, syscalls, contention, parallelism. It's the timeline view.

## What the tracer records

Per the `runtime/trace` package documentation:

- Goroutine creation, blocking, and unblocking
- Syscall enter / exit / block events
- GC events
- Heap size changes
- Processor (P) start/stop events
- CPU profiling samples (when active)
- Optional user annotations:
  - **Tasks** — logical operations that span multiple goroutines
  - **Regions** — time intervals within one goroutine; can nest
  - **Logs** — timestamped, categorized messages

## Producing a trace

There are three common ways.

### A. From code

Wrap the workload you want to trace with `trace.Start` and `trace.Stop`:

```go
package main

import (
    "log"
    "os"
    "runtime/trace"
)

func main() {
    f, err := os.Create("trace.out")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    if err := trace.Start(f); err != nil {
        log.Fatal(err)
    }
    defer trace.Stop()

    // ... workload to trace ...
}
```

### B. From tests

The simplest path — no code change:

```bash
go test -trace=trace.out ./...
```

### C. From a running server

Import `net/http/pprof` and the server exposes `/debug/pprof/trace?seconds=5`, which streams a 5-second trace:

```go
import _ "net/http/pprof"
```

Then:

```bash
curl -o trace.out 'http://localhost:6060/debug/pprof/trace?seconds=5'
```

## Viewing a trace

```bash
go tool trace trace.out
```

This starts a local web server and opens a browser. The UI offers several views:

- **View trace** — the timeline. Goroutines on the Y axis, time on the X axis, color-coded events. You can zoom in to individual goroutine state transitions.
- **Goroutine analysis** — per-goroutine breakdown of time spent in different states (running, blocked on syscall, blocked on sync primitive, blocked on scheduler, etc.).
- **Network / sync / syscall / scheduler blocking profiles** — flame-graph-style breakdowns of *why* goroutines were waiting.
- **User-defined tasks/regions** — appears when your code used `trace.NewTask` or `trace.WithRegion`.

## When to reach for it

- You suspect goroutine contention or poor parallelism.
- Tail latency is bad and you want to see GC pauses on a timeline.
- You added concurrency but throughput didn't improve and you want to know whether goroutines are actually running in parallel.
- A request occasionally takes much longer than usual and you want to see what blocked it.

For "this code is slow on average," reach for `go tool pprof` first.

## Sources

- [Diagnostics — go.dev/doc/diagnostics](https://go.dev/doc/diagnostics)
- [`runtime/trace` package — pkg.go.dev/runtime/trace](https://pkg.go.dev/runtime/trace)
- [`cmd/trace` reference — pkg.go.dev/cmd/trace](https://pkg.go.dev/cmd/trace)
