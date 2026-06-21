# Concurrency patterns

The building blocks — goroutines, channels, `select`, `sync`, and
`context` — combine into a handful of patterns you'll reach for again and
again. This article shows the canonical three: **worker pools**,
**fan-out/fan-in**, and **pipelines**.

## Worker pool

When you have many independent jobs and want to cap parallelism, start a
*fixed* number of workers that pull from a shared `jobs` channel and push to
a `results` channel. The pool size bounds how much runs at once.

```go
jobs := make(chan int, 100)
results := make(chan int, 100)
var wg sync.WaitGroup

for w := 0; w < 3; w++ {           // 3 workers
    wg.Add(1)
    go func() {
        defer wg.Done()
        for j := range jobs {       // each worker drains jobs
            results <- j * j
        }
    }()
}

for i := 1; i <= 5; i++ {
    jobs <- i
}
close(jobs)                         // no more jobs; workers' range loops end

go func() { wg.Wait(); close(results) }()   // close results once all workers done

sum := 0
for r := range results {            // gather (order is nondeterministic)
    sum += r
}
fmt.Println(sum)                    // output: 55
```

Two idioms make this robust: **close `jobs`** so the workers' `range` loops
terminate, and **close `results` in a separate goroutine after `wg.Wait()`**
so the gathering `range` ends. Because workers finish in arbitrary order,
aggregate in an order-independent way (here, a sum).

## Fan-out / fan-in

*Fan-out* = several goroutines reading from one channel (the worker pool
above is a fan-out). *Fan-in* = merging several channels into one. Here's
the merge half, using a `WaitGroup` to close the merged channel once every
source is drained:

```go
func merge(cs ...<-chan int) <-chan int {
    out := make(chan int)
    var wg sync.WaitGroup
    for _, c := range cs {
        wg.Add(1)
        go func(c <-chan int) {
            defer wg.Done()
            for v := range c {
                out <- v
            }
        }(c)
    }
    go func() { wg.Wait(); close(out) }()
    return out
}
```

## Pipeline

A pipeline is a chain of stages, each a function that **takes a
receive-only channel and returns one**, doing its work in a goroutine.
Values flow stage to stage; each stage closes its output when its input is
drained. A single chain preserves order.

```go
func gen(nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        for _, n := range nums {
            out <- n
        }
        close(out)
    }()
    return out
}

func sq(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        for n := range in {
            out <- n * n
        }
        close(out)
    }()
    return out
}

for v := range sq(gen(2, 3, 4)) {
    fmt.Println(v)
}
// output:
// 4
// 9
// 16
```

Each stage is independent and composable: `sq(sq(gen(...)))` just works,
and stages run concurrently while data streams through.

## Make goroutines stoppable

Every long-lived goroutine needs an exit path, or it **leaks** — it lives
until the program ends, holding memory and possibly blocking forever. Give
each one a way out: close its input channel, or pass a `context.Context`
and `select` on `ctx.Done()`. A goroutine you can't stop is a bug.

## Rules of thumb

- **Don't start a goroutine without knowing how it stops.**
- **The sender closes a channel, never the receiver** — and only once.
- **Aggregate results order-independently** unless a pipeline guarantees
  order.
- Prefer a **bounded** worker pool to spawning one goroutine per job when
  jobs are unbounded.
- Pass a **`context`** through long operations so callers can cancel.

## Quick reference

| Pattern | Shape |
|---|---|
| Worker pool | N goroutines `range` over a shared `jobs` channel |
| Fan-out | multiple goroutines read one channel |
| Fan-in (merge) | many channels → one, `WaitGroup` then close |
| Pipeline | stages: `func(<-chan T) <-chan U`, each closes its out |
| Stop a goroutine | close its input, or `select` on `ctx.Done()` |
| Close discipline | sender closes, once |

## Sources

- [Go blog: pipelines and cancellation — go.dev/blog/pipelines](https://go.dev/blog/pipelines)
- [Go Concurrency Patterns (talk) — go.dev/talks/2012/concurrency.slide](https://go.dev/talks/2012/concurrency.slide)
- [Effective Go: concurrency — go.dev/doc/effective_go#concurrency](https://go.dev/doc/effective_go#concurrency)
- [sync.WaitGroup — pkg.go.dev/sync#WaitGroup](https://pkg.go.dev/sync#WaitGroup)
