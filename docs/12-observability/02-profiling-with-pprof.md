# Profiling with pprof

When something is slow or using too much memory, pprof tells you where
rather than where you guessed. It is built into the runtime, costs
almost nothing when idle, and works on a live production process.

```bash
go tool pprof -top -nodecount=5 cpu.prof
```

The [execution tracer](../01-ecosystem-and-installation/03-go-tool-trace.md)
answers a different question. pprof samples *where time and memory
go*; the tracer shows *when things happened* — scheduling, blocking,
GC pauses. Reach for pprof first.

## The profiles

| Profile | Answers |
|---|---|
| `cpu` | which functions burn CPU |
| `heap` | what is allocating, and what is still live |
| `goroutine` | how many goroutines exist and where they are stuck |
| `allocs` | every allocation since start |
| `mutex` | which locks are contended |
| `block` | what is blocking on channels and syscalls |

`goroutine` is the one that finds leaks: a count climbing steadily is a
goroutine never returning.

`mutex` and `block` are off by default and need enabling:

```go
runtime.SetMutexProfileFraction(5)
runtime.SetBlockProfileRate(10000)
```

## From a test or benchmark

The easiest way, and the one to reach for while developing:

```bash
go test -run XXX -bench . -cpuprofile cpu.prof -memprofile mem.prof ./...
```

You now have a profile of a reproducible workload, which beats
profiling a whole server when you already suspect one function.

## From code

For a batch job or a CLI, wrap the work:

```go
f, err := os.Create("cpu.prof")
if err != nil {
    return err
}
if err := pprof.StartCPUProfile(f); err != nil {
    return err
}
defer pprof.StopCPUProfile()
```

A heap profile is a snapshot, so take it at the interesting moment —
and call `runtime.GC()` first, or you are measuring garbage that has
not been collected yet:

```go
runtime.GC()
pprof.WriteHeapProfile(mf)
```

`pprof.Profiles()` lists what is available:

```
allocs (20)
block (0)
goroutine (1)
heap (20)
mutex (0)
threadcreate (9)
```

## From a running server

`net/http/pprof` registers handlers as an import side effect — the
blank import from [imports](../02-language-basics/16-imports.md):

```go
import _ "net/http/pprof"
```

That attaches to the **default mux**, so if you built your own
`ServeMux` you must register explicitly:

```go
mux.HandleFunc("/debug/pprof/", pprof.Index)
mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
mux.HandleFunc("/debug/pprof/heap", pprof.Index)
```

Then point the tool at the live process:

```bash
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
go tool pprof http://localhost:8080/debug/pprof/heap
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

**Never expose these endpoints publicly.** They leak function names,
let anyone trigger a 30-second CPU profile, and the goroutine dump
shows your internal structure. Bind them to a separate internal port,
put them behind authentication, or gate them on an environment
variable.

## Reading the output

```
Duration: 204.38ms, Total samples = 30ms (14.68%)
Showing nodes accounting for 30ms, 100% of 30ms total
      flat  flat%   sum%        cum   cum%
      10ms 33.33% 33.33%       10ms 33.33%  runtime.kevent
         0     0%   100%       10ms 33.33%  main.main
```

Two columns matter:

- **flat** — time in *this* function's own code.
- **cum** — time in this function *and everything it called*.

A function with high `cum` and near-zero `flat` is a caller; follow it
down. High `flat` is where the work actually happens.

Note `Total samples = 30ms (14.68%)` — this program was mostly idle, so
the sample count is low and the results are noise. **A profile needs
enough samples to mean anything.** Profile under real load, or for
long enough, or the top entries will be runtime scheduling functions
like `kevent` and `usleep` rather than your code.

Memory profiles are clearer on a short run:

```
   37.76MB 90.87% 90.87%    37.76MB 90.87%  strings.(*Builder).WriteString
```

That is unambiguous: one call site is 91% of allocation.

A heap profile has four views, selected with `-sample_index`:

| Index | Shows |
|---|---|
| `inuse_space` | bytes currently live — **the default**, for leaks |
| `inuse_objects` | object count currently live |
| `alloc_space` | bytes ever allocated — for GC pressure |
| `alloc_objects` | allocation count ever |

`inuse_*` finds what is retained; `alloc_*` finds what churns. A
program with no leak can still spend all its time in GC.

## The interactive and web views

```bash
go tool pprof cpu.prof
(pprof) top
(pprof) list work        # annotated source, line by line
(pprof) web              # SVG call graph, needs graphviz
```

`list` is the most useful command: it prints the function's source with
per-line cost, which usually ends the investigation.

```bash
go tool pprof -http=:8081 cpu.prof
```

That opens a browser UI with a flame graph. Read it as: width is time,
and each bar sits on top of its caller. Look for wide bars, not deep
stacks — depth is just call nesting.

You can also diff two profiles, which is how you prove an optimisation
worked:

```bash
go tool pprof -base before.prof after.prof
```

## Order of operations

1. **Measure first.** Intuition about Go performance is usually wrong;
   the bottleneck is frequently allocation or a lock, not the
   arithmetic you were looking at.
2. **Benchmark the suspect** so you have a number that moves.
3. **Profile it** and find the actual line.
4. **Change one thing**, re-benchmark, compare with `benchstat`.

Profiling a program that is fast enough is a way to spend an afternoon.

> **From Python:** `cProfile` plus `memory_profiler`, but sampling
> rather than instrumenting, so the overhead is low enough to leave on
> in production — and it works on a live process over HTTP, which
> Python has no standard equivalent for.

## Quick reference

| Task | Command |
|---|---|
| from a benchmark | `go test -bench . -cpuprofile cpu.prof` |
| from code | `pprof.StartCPUProfile(f)` / `defer StopCPUProfile()` |
| heap snapshot | `runtime.GC()` then `pprof.WriteHeapProfile(f)` |
| from a server | `_ "net/http/pprof"`, then `go tool pprof <url>` |
| live CPU profile | `.../debug/pprof/profile?seconds=30` |
| goroutine leak | `.../debug/pprof/goroutine` |
| enable mutex/block | `SetMutexProfileFraction`, `SetBlockProfileRate` |
| top functions | `go tool pprof -top -nodecount=10 cpu.prof` |
| per-line cost | `(pprof) list FuncName` |
| flame graph | `go tool pprof -http=:8081 cpu.prof` |
| compare | `go tool pprof -base before.prof after.prof` |
| leak vs churn | `-sample_index=inuse_space` vs `alloc_space` |

## Sources

- [`runtime/pprof` — pkg.go.dev/runtime/pprof](https://pkg.go.dev/runtime/pprof)
- [`net/http/pprof` — pkg.go.dev/net/http/pprof](https://pkg.go.dev/net/http/pprof)
- [Go blog: profiling Go programs — go.dev/blog/pprof](https://go.dev/blog/pprof)
- [Diagnostics — go.dev/doc/diagnostics](https://go.dev/doc/diagnostics)
- [`go tool pprof` — github.com/google/pprof/blob/main/doc/README.md](https://github.com/google/pprof/blob/main/doc/README.md)
