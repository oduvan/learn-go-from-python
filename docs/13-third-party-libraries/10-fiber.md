# Fiber

A web framework built on fasthttp rather than `net/http`. It trades
standard-library compatibility for speed and a more compact handler
API. Everything in
[an HTTP server](../08-http-with-net-http/01-http-server.md) still
applies conceptually — the shapes just change.

> **Module:** `github.com/gofiber/fiber/v3`.

```go
app := fiber.New()
app.Get("/users/:id", func(c fiber.Ctx) error {
    return c.JSON(fiber.Map{"id": c.Params("id")})
})
app.Listen(":8080")
```

## v3 changed the handler signature

In v2 a handler took `*fiber.Ctx`. In **v3 `fiber.Ctx` is an
interface**, passed by value:

```go
func(c fiber.Ctx) error
```

Most v2 examples online will not compile. If a snippet says
`*fiber.Ctx`, it is v2.

The signature's real advantage over `net/http` is the returned
`error`. Instead of writing a response and remembering to `return`,
you return the error and a central handler deals with it — which
removes the most common `net/http` handler bug.

## Routing

```go
app.Get("/users/:id", h.get)
app.Post("/users", h.create)
app.Get("/search", h.search)
```

Method-named functions rather than patterns, and `:id` for parameters.
Read them with `Params`, `Query` and `FormValue`:

```go
c.Params("id")           // /users/:id
c.Query("q")             // ?q=go
c.Get("Content-Type")    // a request header
```

`c.Get` reads a *request* header; `c.Set` writes a *response* header.
That asymmetry catches people.

Groups keep prefixes and middleware together:

```go
api := app.Group("/api/v1")
api.Use(requireAuth)
api.Get("/users/:id", h.get)
```

Unmatched routes and wrong methods behave sensibly with no work:

```
GET    /nope      404 {"error":"Not Found"}
DELETE /users/7   405 {"error":"Method Not Allowed"}
```

## Handlers as methods

Same dependency discipline as anywhere else — a struct holding the
narrow interfaces it needs:

```go
type Handler struct{ users Store }

func (h Handler) get(c fiber.Ctx) error {
    u, err := h.users.ByID(c.Params("id"))
    if err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
    }
    return c.JSON(u)
}
```

```
GET /users/7    200 {"name":"Ada:7","age":36}
GET /users/404  404 {"error":"not found"}
```

`c.JSON` sets the content type and encodes. `fiber.Map` is
`map[string]any` with a shorter name.

## Binding

```go
var u User
if err := c.Bind().Body(&u); err != nil {
    return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bad body"})
}
```

`Bind().Body` picks the decoder from the content type. There are also
`Bind().Query`, `Bind().Params` and `Bind().Header`.

```
POST /users {"name":"Bo","age":7}   201 {"name":"Bo","age":7}
POST /users {                       400 {"error":"bad body"}
```

Binding is not validation. It fills the struct; it does not check that
`Age` is plausible or `Name` non-empty. Validate after binding,
either by hand or with a validation library.

## One place for errors

Returning an error routes it to the app's `ErrorHandler`:

```go
app := fiber.New(fiber.Config{
    ErrorHandler: func(c fiber.Ctx, err error) error {
        code := fiber.StatusInternalServerError
        if e, ok := err.(*fiber.Error); ok {
            code = e.Code
        }
        return c.Status(code).JSON(fiber.Map{"error": err.Error()})
    },
})
```

```go
return fiber.NewError(fiber.StatusTeapot, "teapot")
// 418 {"error":"teapot"}
```

This is where the `error` return pays off: one place decides the
response shape, and handlers just fail. Extend it to recognise your own
domain errors — `errors.Is(err, ErrNotFound)` becoming a `404` — and
handlers stop constructing responses at all.

Be careful what reaches the client. The default handler above echoes
`err.Error()`, which will happily leak an internal message. Map known
errors to safe text and return a generic message for the rest.

## Middleware

```go
app.Use(requestid.New())
app.Use(recover.New())
app.Use(compress.New(compress.Config{
    Next: func(c fiber.Ctx) bool {
        return strings.HasSuffix(c.Path(), "/stream")   // never compress SSE
    },
}))
```

Middleware is `func(c fiber.Ctx) error`, the same shape as a handler.
Call `c.Next()` to continue; return without it to stop.

`recover.New()` turns a panic into a `500` through your error handler:

```
GET /boom   500 {"error":"kaboom"}
```

Most bundled middleware takes a `Next` predicate to skip specific
routes — the compression example above is the one that matters if you
stream, since buffering to compress defeats flushing.

## What fasthttp costs you

Fiber does not use `net/http`, and that has consequences:

- **`net/http` middleware does not work.** No `otelhttp`, no
  third-party handler wrappers, without an adaptor.
- **`httptest` does not work.** Use `app.Test(req)`, which takes an
  `*http.Request` and returns an `*http.Response` — convenient, and
  what the examples here use.
- **No context ends when the client hangs up.** `c.Context()` returns
  the `context.Context` you stored with `c.SetContext`, or an empty
  `context.Background()` if you stored none. `c.RequestCtx()` returns
  fasthttp's own `*fasthttp.RequestCtx`. It also satisfies
  `context.Context`, but its `Done` channel closes only when the server
  shuts down. Under `net/http`, `r.Context()` ends with the request;
  under Fiber, add your own deadline with `context.WithTimeout`.
- **Request and response values are reused between requests.** A
  `[]byte` or string from `c.Params` or `c.Body` is only valid during
  the handler. Keeping one past the return — in a goroutine, a cache, a
  struct field — gives you data from an unrelated request later.
  Copy it: `string(append([]byte(nil), b...))`. The `append` onto a
  `nil` slice builds a new array that shares nothing with fasthttp's
  buffer. Or just use `c.Params`'s string if your version already copies. This is the bug that is
  hardest to find, because it only appears under concurrency.

That last point is the real trade. Fiber is fast partly because it
recycles buffers, and recycled buffers require discipline.

## Streaming: SSE is fasthttp, not `http.Flusher`

[Server-sent events](../08-http-with-net-http/05-server-sent-events.md)
teaches the `net/http` mechanism: get an `http.ResponseController` or
an `http.Flusher` and call `Flush` after each message. Under Fiber that
interface is simply not there:

```go
_, isFlusher := any(c.Response().BodyWriter()).(http.Flusher)
// false
```

fasthttp streams through a body-stream writer instead. `c.RequestCtx()`
gives you fasthttp's request object, and its `SetBodyStreamWriter`
method takes a `fasthttp.StreamWriter`: a function that receives a
`*bufio.Writer` and flushes it. Both names belong to fasthttp, not
Fiber, so fasthttp's own documentation is the reference for them:

```go
app.Get("/stream", func(c fiber.Ctx) error {
    c.Set("Content-Type", "text/event-stream")
    c.Set("Cache-Control", "no-cache")

    c.RequestCtx().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
        for i := 1; i <= 3; i++ {
            fmt.Fprintf(w, "event: tick\ndata: %d\n\n", i)
            if err := w.Flush(); err != nil {
                return          // client went away
            }
            time.Sleep(10 * time.Millisecond)
        }
    }))
    return nil
})
```

```
ct: text/event-stream
  "event: tick"
  "data: 1"
  "event: tick"
  "data: 2"
  "event: tick"
  "data: 3"
```

Everything else from the core article still holds: the wire format, the
blank line terminating each message, keepalive comments, and dropping
slow subscribers with a non-blocking send.

Three Fiber-specific points. The handler **returns immediately** —
`SetBodyStreamWriter` registers a callback that runs after you return,
so anything you need must be captured before then, and a value taken
from `c` must be copied for the reason above. A failing `w.Flush()` is
your disconnect signal, since fasthttp's per-connection cancellation
does not behave like `r.Context()`. And exclude the route from
compression, or buffering defeats the flush.

## Should you use it

If you need a `net/http` ecosystem — OpenTelemetry instrumentation,
existing middleware, `httptest` — the standard library with a router is
the calmer choice, and fast enough for almost everything.

Fiber makes sense when its ergonomics suit the team, or when the
request rate genuinely justifies fasthttp. Decide once, because mixing
is not practical.

> **From Python:** Fiber is FastAPI-shaped — decorator-ish routing,
> binding into a typed struct, a central exception handler — running on
> a non-standard server, the way `uvloop` replaces asyncio's loop. The
> buffer-reuse rule has no Python equivalent and is the thing to
> remember.

## Quick reference

| Task | Form |
|---|---|
| app | `fiber.New(fiber.Config{...})` |
| handler | `func(c fiber.Ctx) error` — **value, not pointer, in v3** |
| routes | `app.Get("/users/:id", h)`, `app.Group("/api")` |
| path / query | `c.Params("id")`, `c.Query("q")` |
| request vs response header | `c.Get(...)` vs `c.Set(...)` |
| decode a body | `c.Bind().Body(&v)` — then validate |
| respond | `c.JSON(v)`, `c.Status(code).JSON(...)` |
| fail | `return fiber.NewError(code, msg)` |
| one place for errors | `Config.ErrorHandler` |
| middleware | `app.Use(...)`, with `Next` to skip routes |
| testing | `app.Test(req)` — not `httptest` |
| **buffer reuse** | copy anything kept past the handler |

## Sources

- [Fiber documentation — docs.gofiber.io](https://docs.gofiber.io/)
- [`fiber/v3` — pkg.go.dev/github.com/gofiber/fiber/v3](https://pkg.go.dev/github.com/gofiber/fiber/v3)
- [What's new in v3 — docs.gofiber.io/next/whats_new](https://docs.gofiber.io/next/whats_new)
- [fasthttp — pkg.go.dev/github.com/valyala/fasthttp](https://pkg.go.dev/github.com/valyala/fasthttp)
