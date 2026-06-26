# Context

A `context.Context` carries **cancellation, deadlines, and request-scoped
values** across API boundaries and goroutines. It's how you tell a tree of
goroutines "stop now" — when a request is cancelled, a timeout fires, or a
server is shutting down.

The core idea: a `Context` exposes a `Done()` channel that **closes** when
the context is cancelled. Goroutines `select` on it and bail out.

## Roots: Background and TODO

Every context tree starts from a root. `context.Background()` is the usual
one (top of `main`, incoming requests). `context.TODO()` is a placeholder
for "I haven't wired context through here yet."

```go
ctx := context.Background()
```

You never cancel the root directly — instead you *derive* a child context
that can be cancelled.

## WithCancel: explicit cancellation

`context.WithCancel` returns a child context and a `cancel` function.
Calling `cancel` closes the context's `Done()` channel, which every
goroutine watching it observes.

```go
ctx, cancel := context.WithCancel(context.Background())
done := make(chan struct{})

go func() {
    <-ctx.Done()                       // blocks until cancelled
    fmt.Println("worker:", ctx.Err())  // worker: context canceled
    close(done)
}()

cancel()                               // trigger cancellation
<-done
// output:
// worker: context canceled
```

`ctx.Err()` reports *why* it ended: `context.Canceled` after `cancel`, or
`context.DeadlineExceeded` after a timeout. Always call `cancel`
(typically `defer cancel()`) to release resources, even if the work
finished normally.

## WithTimeout and WithDeadline

`WithTimeout` cancels automatically after a duration; `WithDeadline` at a
fixed time. Combine with `select` to bound any blocking operation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
defer cancel()

select {
case <-time.After(time.Second):
    fmt.Println("work finished")
case <-ctx.Done():
    fmt.Println("gave up:", ctx.Err())   // gave up: context deadline exceeded
}
```

The 10 ms timeout fires long before the 1 s work, so `ctx.Done()` wins and
`ctx.Err()` is `context.DeadlineExceeded`.

## Propagation: pass it down, don't store it

The conventions are firm and worth following exactly:

- **Pass `ctx` as the first parameter**, named `ctx`:
  `func Fetch(ctx context.Context, url string) (...)`.
- **Don't store a `Context` in a struct** — thread it through calls.
- Derive child contexts as work fans out; cancelling a parent cancels all
  its children.
- A function that respects context **selects on `ctx.Done()`** in its
  blocking loops and returns `ctx.Err()` when it fires.

```go
func work(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()      // stop promptly when cancelled
        default:
            // ... one unit of work ...
            return nil
        }
    }
}
```

## Request-scoped values (use sparingly)

`context.WithValue` attaches a key/value pair that travels with the
context — meant for request-scoped metadata like a request ID, **not** for
passing optional function arguments. Overusing it hides dependencies, so
prefer explicit parameters and reach for values only for cross-cutting
data.

Use an **unexported custom key type**, not a bare string, so keys from
different packages can't collide:

```go
type ctxKey string

ctx := context.WithValue(context.Background(), ctxKey("reqID"), "abc123")

fmt.Println(ctx.Value(ctxKey("reqID")))   // output: abc123
fmt.Println(ctx.Value(ctxKey("missing"))) // output: <nil>
```

`Value` returns `any`, so it's `nil` for an absent key and you usually
type-assert the result back to its concrete type before using it.

## Quick reference

| Call | Meaning |
|---|---|
| `context.Background()` | root context |
| `context.TODO()` | placeholder root |
| `ctx, cancel := WithCancel(parent)` | manual cancellation |
| `WithTimeout(parent, d)` | auto-cancel after duration |
| `WithDeadline(parent, t)` | auto-cancel at a time |
| `<-ctx.Done()` | closed when cancelled |
| `ctx.Err()` | `Canceled` or `DeadlineExceeded` |
| `WithValue(parent, k, v)` | request-scoped data (sparingly) |
| first arg `ctx context.Context` | the convention |

## Sources

- [context — pkg.go.dev/context](https://pkg.go.dev/context)
- [Go blog: context and structs — go.dev/blog/context-and-structs](https://go.dev/blog/context-and-structs)
- [Go blog: pipelines and cancellation — go.dev/blog/pipelines](https://go.dev/blog/pipelines)
- [Go Concurrency Patterns: Context — go.dev/blog/context](https://go.dev/blog/context)
