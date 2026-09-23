# `golang.org/x/sync` and `golang.org/x/time`

Two Go-team modules that sit just outside the standard library.
`errgroup` is the one you will reach for most: bounded concurrency with
error propagation, in about six lines.

> **Modules:** `golang.org/x/sync` and `golang.org/x/time`. Maintained
> by the Go team, versioned separately from the toolchain — which is
> why they are here and not in topic 05.

Builds on [bounded concurrency](../05-concurrency/07-bounded-concurrency.md),
which showed the hand-rolled version.

## `errgroup` replaces WaitGroup plus error plumbing

The standard-library version needs a `WaitGroup`, a mutex and a
first-error variable. `errgroup` is the same thing packaged:

```go
g, ctx := errgroup.WithContext(context.Background())

for _, job := range jobs {
    g.Go(func() error {
        return process(ctx, job)
    })
}

if err := g.Wait(); err != nil {
    return err
}
```

`Wait` blocks until every goroutine returns and yields the **first
non-nil error**. Later errors are discarded — if you need them all,
collect them yourself with `errors.Join`.

```go
// four jobs, job 2 fails
// err: job 2 failed
```

## `WithContext` cancels the rest

The context it returns is cancelled as soon as any goroutine returns an
error. Work still running sees `ctx.Done()` and can stop:

```go
g, ctx := errgroup.WithContext(context.Background())

g.Go(func() error { return errors.New("boom") })
g.Go(func() error {
    <-ctx.Done()
    return nil
})

err := g.Wait()
// boom, and ctx.Err() is context canceled
```

This is the main reason to use it. Without cancellation, one failure
leaves every sibling running to completion for a result nobody wants.

**The goroutines must actually watch the context.** A `time.Sleep`
or a blocking call with no context ignores the cancellation entirely,
and `Wait` still waits for it.

Note: use the `ctx` that `WithContext` returns, not the one you passed
in. Passing the outer context to the goroutines defeats the whole
mechanism.

## `SetLimit` caps concurrency

```go
g := new(errgroup.Group)
g.SetLimit(3)

for _, job := range jobs {
    g.Go(func() error { return process(job) })
}
// peak concurrency: 3
```

`g.Go` blocks when the limit is reached, so a hundred jobs run three at
a time. This replaces the buffered-channel semaphore entirely.

Call `SetLimit` **before** the first `Go`; changing it while goroutines
are running panics. `SetLimit(-1)` means unlimited.

`TryGo` starts a goroutine only if a slot is free, returning whether it
did:

```go
g.SetLimit(1)
fmt.Println(g.TryGo(slowJob))   // output: true
fmt.Println(g.TryGo(quickJob))  // output: false
```

Useful for "do this if we have spare capacity, otherwise skip it".

## Collecting results

Distinct slice indices need no lock, exactly as before:

```go
g := new(errgroup.Group)
g.SetLimit(2)

out := make([]int, len(jobs))
for i, job := range jobs {
    g.Go(func() error {
        v, err := process(job)
        out[i] = v
        return err
    })
}
err := g.Wait()
// [0 1 4 9 16]
```

Results stay in input order. Anything genuinely shared still needs a
mutex.

## What it does not do

- **It does not recover panics.** A panic in a `g.Go` function takes
  the process down, exactly as it would in a bare goroutine — the point
  [long-running goroutines](../05-concurrency/08-long-running-goroutines.md)
  makes. If the work can panic, recover inside the function.
- **`Wait` may be called only once**, and the group is not reusable
  afterwards.
- **It does not time anything out.** That is the context's job.

`golang.org/x/sync` also has `semaphore` for weighted limits and
`singleflight` for collapsing duplicate concurrent calls to the same
key — worth knowing exist, rarely the first tool.

## `x/time/rate`: token buckets

A rate limiter for calling something with a published limit:

```go
lim := rate.NewLimiter(rate.Limit(100), 1)   // 100/second, burst 1

for _, req := range requests {
    if err := lim.Wait(ctx); err != nil {
        return err   // context cancelled
    }
    send(req)
}
```

`Wait` blocks until a token is available or the context ends. The
second argument is the **burst**: how many can go at once after an
idle period. Burst 1 gives evenly spaced requests; a larger burst
allows a spike.

A per-minute quota converts directly:

```go
rpm := 600
lim := rate.NewLimiter(rate.Limit(float64(rpm)/60.0), 1)
fmt.Println(float64(lim.Limit()))   // output: 10
```

`Allow` is the non-blocking form — it answers immediately and consumes
a token only if one was free:

```go
l := rate.NewLimiter(rate.Limit(1), 1)
fmt.Println(l.Allow(), l.Allow())   // output: true false
```

Use `Wait` for outbound calls you want to pace, `Allow` for inbound
requests you want to reject with a `429`. A limiter is safe for
concurrent use, so one per upstream, created at startup.

`Reserve` sits between them: it tells you how long you *would* wait, so
you can decide whether to.

## Putting them together

The two compose into the shape you want when calling a rate-limited API
concurrently:

```go
// ctx comes from the caller, for example the incoming request's context
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(8)                                   // 8 in flight
lim := rate.NewLimiter(rate.Limit(20), 5)       // 20/s, burst 5

for _, item := range items {
    g.Go(func() error {
        if err := lim.Wait(ctx); err != nil {
            return err
        }
        return fetch(ctx, item)
    })
}
return g.Wait()
```

Concurrency and rate are different limits and both matter: eight
in-flight requests can still exceed twenty per second if each is fast.

> **From Python:** `errgroup` is `asyncio.gather` with a concurrency
> cap and automatic cancellation on first error, or a
> `ThreadPoolExecutor` that propagates exceptions properly.
> `rate.Limiter` is the token bucket you would otherwise hand-roll.

## Quick reference

| Task | Form |
|---|---|
| a group | `g, ctx := errgroup.WithContext(ctx)` |
| start work | `g.Go(func() error { ... })` |
| wait | `g.Wait()` → the first error |
| cap concurrency | `g.SetLimit(n)` before the first `Go` |
| only if free | `g.TryGo(fn)` |
| use which context | the one `WithContext` returned |
| panics | not recovered — do it yourself |
| pace outbound calls | `rate.NewLimiter(rate.Limit(n), burst)` + `Wait(ctx)` |
| reject inbound | `lim.Allow()` |
| per-minute quota | `rate.Limit(float64(rpm)/60.0)` |

## Sources

- [`errgroup` — pkg.go.dev/golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup)
- [`semaphore` — pkg.go.dev/golang.org/x/sync/semaphore](https://pkg.go.dev/golang.org/x/sync/semaphore)
- [`singleflight` — pkg.go.dev/golang.org/x/sync/singleflight](https://pkg.go.dev/golang.org/x/sync/singleflight)
- [`rate` — pkg.go.dev/golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate)
- [Token bucket — en.wikipedia.org/wiki/Token_bucket](https://en.wikipedia.org/wiki/Token_bucket)
