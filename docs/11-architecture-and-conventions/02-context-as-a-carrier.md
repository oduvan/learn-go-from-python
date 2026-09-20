# Context as a carrier

`context.Context` does three jobs: it carries cancellation, a deadline,
and request-scoped values. [Context](../05-concurrency/05-context.md)
covered the first two. The third is the one with the sharp edges.

```go
type userKey struct{}

func WithUser(ctx context.Context, name string) context.Context {
    return context.WithValue(ctx, userKey{}, name)
}

func UserFrom(ctx context.Context) (string, bool) {
    s, ok := ctx.Value(userKey{}).(string)
    return s, ok
}
```

## The key must be an unexported type

`Value` looks up by a key compared with `==`, across the whole context —
including values put there by libraries you do not control. A `string`
key called `"user"` will eventually collide with someone else's.

Use an unexported type. Another package cannot name it, so collision is
impossible:

```go
type userKey struct{}
```

`struct{}{}` costs no memory, and the type itself is the identity. An
unexported `type ctxKey int` with `iota` constants works equally well
when you have several.

**Export the accessors, not the key.** Callers use `WithUser` and
`UserFrom`; nobody outside can construct the key or read the raw value.
That also gives you one place to change the type later.

```go
ctx := WithUser(context.Background(), "ada")
u, ok := UserFrom(ctx)
fmt.Println(u, ok)              // output: ada true

_, ok = UserFrom(context.Background())
fmt.Println(ok)                 // output: false
```

Always return the `ok`. A missing value yields `nil`, and the comma-ok
assertion is how you distinguish "absent" from "present and empty".

## What belongs in there

Request-scoped facts that cut across many layers and that most functions
only pass along:

- the authenticated user or tenant
- a request or trace id
- a logger already tagged with those

What does not belong:

- **Dependencies.** A store, a client, a config. Those are constructor
  parameters, where the compiler checks them.
- **Optional arguments.** If a function needs a value, put it in the
  signature. A context value is invisible to the caller and unchecked.
- **Anything mutable.** A context is immutable by design; putting a
  pointer in and mutating through it is a race waiting to happen.

The test: if removing it from the context would turn a compile error
into a runtime surprise, it should have been a parameter.

## Carrying a transaction

The genuinely borderline case is a database transaction, from
[transactions](../09-database-sql/03-transactions.md). Putting it in
the context means store methods keep a uniform signature and any of
them can participate in a caller's transaction without knowing.

The cost is real: the signature no longer tells you whether a function
writes inside a transaction. It is a trade — uniform signatures and
composability against explicitness. Small codebases should pass `tx`
explicitly; it becomes worth it when the call tree is deep enough that
threading `tx` through touches everything.

If you do it, provide `WithTx` and a resolver, never raw `Value` calls
at the call sites.

## Detaching work that must outlive the request

When a handler returns, its context is cancelled. Background work
started with that context dies immediately — usually mid-write.

`context.WithoutCancel` keeps the values and drops the cancellation:

```go
ctx, cancel := context.WithCancel(WithUser(context.Background(), "bo"))
detached := context.WithoutCancel(ctx)
cancel()

fmt.Println("original err:", ctx.Err())      // output: original err: context canceled
fmt.Println("detached err:", detached.Err()) // output: detached err: <nil>

u, _ := UserFrom(detached)
fmt.Println(u)                               // output: bo
```

The detached context keeps the user — which is usually what you want,
since the background job still needs to know who triggered it.

Two cautions. **It copies every value**, including ones whose lifetime
was tied to the request. That is the trap where the two halves of this
article meet: a transaction carried in the context survives detachment.

```go
txCtx := context.WithValue(ctx, txKey{}, tx)
cancelCtx, cancel := context.WithCancel(txCtx)

detached := context.WithoutCancel(cancelCtx)
cancel()

fmt.Println(detached.Err())                  // output: <nil>
fmt.Println(detached.Value(txKey{}) != nil)  // output: true
```

The background work now resolves a transaction that the request has
already committed or rolled back. Writes through it either error or
are silently discarded — and "silently discarded" is the outcome you
will spend a day finding.

Strip anything request-bound as part of detaching:

```go
func detach(ctx context.Context) context.Context {
    return context.WithValue(context.WithoutCancel(ctx), txKey{}, nil)
}
```

Better still, make that the only way to detach in your codebase, so
nobody calls `WithoutCancel` directly and has to remember.

And a detached context has **no deadline at all**. Give it one, or you
have created work that can run forever:

```go
detached, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
defer cancel()
go doBackgroundWork(detached)
```

## The conventions

**First parameter, always named `ctx`.** Every function that does I/O or
can block:

```go
func (s *store) ByID(ctx context.Context, id int64) (User, error)
```

**Never store one in a struct.** A context describes one operation's
lifetime; a struct outlives it. The exception is `http.Request`, which
carries its own and hands it out via `r.Context()`.

**Never pass `nil`.** Use `context.Background()` at the top of `main` or
in a test, and `context.TODO()` as a deliberate marker that you have not
decided yet.

**Pass it down, do not squirrel it away.** A goroutine outliving the
call needs a context you chose for it, not the one you happened to have.

> **From Python:** the closest thing is `contextvars`, but explicit —
> there is no ambient current context, so a value only exists where it
> was passed. More typing, and no mystery about where a value came from.

## Quick reference

| Concern | Form |
|---|---|
| a key | an unexported `type k struct{}` — never a string |
| the API | exported `WithX(ctx, v)` and `XFrom(ctx) (T, bool)` |
| reading | comma-ok assertion, always check it |
| what goes in | user, request id, trace span |
| what does not | dependencies, required arguments, mutable state |
| outliving the request | `context.WithoutCancel`, then add a timeout |
| detaching a transaction | strip it first — `WithoutCancel` copies it |
| the signature | `ctx context.Context` first, always |
| in a struct | no |
| roots | `context.Background()`, or `context.TODO()` as a marker |

## Sources

- [`context` package reference — pkg.go.dev/context](https://pkg.go.dev/context)
- [`context.WithValue` — pkg.go.dev/context#WithValue](https://pkg.go.dev/context#WithValue)
- [`context.WithoutCancel` — pkg.go.dev/context#WithoutCancel](https://pkg.go.dev/context#WithoutCancel)
- [Go blog: context — go.dev/blog/context](https://go.dev/blog/context)
- [Go wiki: contexts — go.dev/wiki/CodeReviewComments#contexts](https://go.dev/wiki/CodeReviewComments#contexts)
