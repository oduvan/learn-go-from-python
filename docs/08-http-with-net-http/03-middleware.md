# Middleware

Middleware is one idea: a function that takes a handler and returns a
handler. Logging, recovery, authentication and request IDs are all the
same shape, and the shape is not specific to HTTP.

```go
func logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Printf("-> %s %s\n", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}
```

It works because `http.Handler` is an interface with one method, so a
wrapper is indistinguishable from the thing it wraps. This is the
decorator pattern, built out of [closures](../02-language-basics/06-functions.md)
and [interfaces](../03-object-oriented-go/02-interfaces.md).

## Composing a chain

Applying several by hand nests awkwardly:

```go
h := logging(recoverMW(auth(mux)))
```

A small helper reads better and makes the order explicit:

```go
func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
    for i := len(mws) - 1; i >= 0; i-- {
        h = mws[i](h)
    }
    return h
}

h := chain(mux, logging, recoverMW, auth)
```

The loop runs backwards so the **first argument is the outermost
wrapper**. A request passes through `logging`, then `recoverMW`, then
`auth`, then reaches the mux; the response comes back the other way.

Order is a correctness question, not a preference:

- `logging` outermost, so it sees every request including rejected ones.
- `recoverMW` outside anything that might panic, but inside `logging` so
  the recovered `500` still gets logged.
- `auth` last, so unauthenticated requests never touch your handlers.

## Recovering from a panic

A panic in a handler would otherwise kill the process. `net/http` has
its own recovery, but it closes the connection without a response —
your own gives the client a proper `500`:

```go
func recoverMW(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                slog.Error("panic", "err", rec, "stack", string(debug.Stack()))
                http.Error(w, "internal error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

```
-> GET /boom
recovered: kaboom
<- 500
   => 500 "internal error"
```

Two limits. This only covers panics in the *request's own goroutine* —
anything the handler spawns needs its own `recover`, as
[long-running goroutines](../05-concurrency/08-long-running-goroutines.md)
explains. And if the handler already wrote a response, the `500` arrives
too late to change the status.

## Passing values down the chain

Middleware communicates with handlers through the request context.
The key must be an **unexported type**, so no other package can collide
with it:

```go
type ctxKey int

const userKey ctxKey = 0

func auth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tok := r.Header.Get("X-Token")
        if tok == "" {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return   // do not call next
        }
        ctx := context.WithValue(r.Context(), userKey, "user:"+tok)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

```go
func me(w http.ResponseWriter, r *http.Request) {
    u, _ := r.Context().Value(userKey).(string)
    fmt.Fprintf(w, "hello %s", u)
}
// output: hello user:abc
```

Three details. A request is immutable, so you attach the new context
with `r.WithContext(ctx)` and pass *that* on. Rejecting means writing a
response and returning **without calling `next`**. And `Value` returns
`any`, so reading it is a type assertion — use the comma-ok form, since
a missing value gives `nil`.

Keep context values to request-scoped facts: the caller's identity, a
request ID, a trace span. Real dependencies should be fields on the
handler struct, where the compiler can check them.

## Capturing the status code

`http.ResponseWriter` will not tell you what status was written, which
is awkward for a logger. Wrap it, embedding the interface so you inherit
every method and override only one:

```go
type statusRecorder struct {
    http.ResponseWriter
    status int
}

func (s *statusRecorder) WriteHeader(code int) {
    s.status = code
    s.ResponseWriter.WriteHeader(code)
}
```

```go
rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
next.ServeHTTP(rec, r)
// rec.status now holds what the handler wrote
```

Defaulting to `200` matters: a handler that just calls `Write` never
calls `WriteHeader`, and the status is still `200`.

This is [embedding](../02-language-basics/10-structs.md) doing real
work. One caveat — wrapping hides optional interfaces the real writer
implements, such as `http.Flusher`. If anything downstream streams, the
wrapper must forward those too, which the
[server-sent events](05-server-sent-events.md) article runs into.

## Middleware for things that are not HTTP

The pattern is about the *shape*, not the protocol. Anything with a
uniform "handler" signature can be wrapped the same way:

```go
type Job func(ctx context.Context) error

func timed(name string, next Job) Job {
    return func(ctx context.Context) error {
        start := time.Now()
        err := next(ctx)
        slog.Info("job", "name", name, "ms", time.Since(start).Milliseconds())
        return err
    }
}
```

The same decorator gives a scheduled job, a queue consumer or a
tool-call handler its logging, metrics and recovery, written once.

> **From Python:** this is a decorator, but explicit — there is no `@`,
> you compose the functions yourself, which is why the ordering is
> visible in the code. The closest analogue overall is WSGI/ASGI
> middleware: an application that wraps another application.

## Quick reference

| Concern | Form |
|---|---|
| the shape | `func(http.Handler) http.Handler` |
| adapt a function | `http.HandlerFunc(fn)` |
| compose | a `chain` helper; first argument is outermost |
| recover | `defer` + `recover` inside the wrapper, log the stack |
| stop the chain | write a response and `return` without calling `next` |
| pass a value down | `r.WithContext(context.WithValue(...))`, unexported key type |
| read it | `v, ok := r.Context().Value(key).(T)` |
| observe the status | embed `http.ResponseWriter`, override `WriteHeader`, default 200 |
| non-HTTP handlers | same wrapper shape over your own func type |

## Sources

- [`http.Handler` — pkg.go.dev/net/http#Handler](https://pkg.go.dev/net/http#Handler)
- [`http.HandlerFunc` — pkg.go.dev/net/http#HandlerFunc](https://pkg.go.dev/net/http#HandlerFunc)
- [`context.WithValue` — pkg.go.dev/context#WithValue](https://pkg.go.dev/context#WithValue)
- [`http.Request.WithContext` — pkg.go.dev/net/http#Request.WithContext](https://pkg.go.dev/net/http#Request.WithContext)
- [Go blog: context — go.dev/blog/context](https://go.dev/blog/context)
