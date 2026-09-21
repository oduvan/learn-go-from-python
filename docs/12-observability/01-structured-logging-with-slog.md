# Structured logging with `log/slog`

`log/slog` is the standard library's structured logger. Every entry is
a message plus typed key-value attributes, which means logs can be
queried rather than grepped.

```go
slog.Info("server started", "port", 8080, "tls", false)
// level=INFO msg="server started" port=8080 tls=false
```

## Two handlers

The logger formats nothing itself — a `Handler` does. Two ship with the
standard library:

```go
slog.New(slog.NewTextHandler(os.Stdout, nil))   // human-readable
slog.New(slog.NewJSONHandler(os.Stdout, nil))   // machine-readable
```

```go
jl.Info("request", "method", "GET", "path", "/users", "ms", 12)
// {"level":"INFO","msg":"request","method":"GET","path":"/users","ms":12}
```

The usual arrangement is text locally and JSON in production, decided
once at startup:

```go
func init() {
    if os.Getenv("ENVIRONMENT") != "local" {
        opts := &slog.HandlerOptions{Level: parseLevel(os.Getenv("LOG_LEVEL"))}
        slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, opts)))
    }
}
```

`slog.SetDefault` makes the package-level `slog.Info` and friends use
your handler, so library code logs through it without being passed a
logger.

Log to **stdout**, not a file. In a container something else collects
it; writing to a file means log rotation is now your problem.

## Attributes

The loose form alternates keys and values:

```go
slog.Info("cache miss", "key", "u:1")
```

It is concise, and unchecked — an odd number of arguments produces a
`!BADKEY` entry rather than a compile error. The typed form avoids
that and is faster, since nothing has to be boxed:

```go
slog.Info("started",
    slog.Int("workers", 4),
    slog.Duration("timeout", 5*time.Second),
)
// {"level":"INFO","msg":"started","workers":4,"timeout":5000000000}
```

Note `Duration` serialises as nanoseconds in JSON. If your log platform
wants milliseconds, pass `slog.Int64("timeout_ms", d.Milliseconds())`
instead.

`slog.Any` covers types without a dedicated constructor.

## Levels

Four levels, as `Debug`, `Info`, `Warn`, `Error`. They are ordered
integers, so filtering is a comparison:

```go
fmt.Println(slog.LevelWarn > slog.LevelInfo)   // output: true
fmt.Println(int(slog.LevelError))              // output: 8
```

Set the threshold on the handler:

```go
slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
```

Anything below it is dropped cheaply — the arguments are not even
formatted. Make the level configurable by environment variable so
production can be turned up without a deploy.

`Error` means a human should act. See
[project conventions](../11-architecture-and-conventions/06-project-conventions.md)
for what each level is for; the short version is log a failure once, at
the level that handles it.

## `With` for context that repeats

`With` returns a logger carrying attributes, so you set them once:

```go
log := slog.Default().With(
    slog.String("service", "api"),
    slog.String("version", "1.2.3"),
)
log.Info("started", slog.Int("workers", 4))
// {"level":"INFO","msg":"started","service":"api","version":"1.2.3","workers":4}
```

This is the main tool for correlating entries. A logger created per
request with the request id attached means every line from that request
is findable, without threading the id through every call.

`WithGroup` and `slog.Group` nest attributes, which keeps names from
colliding:

```go
slog.Info("db query",
    slog.Group("db", slog.String("table", "users"), slog.Int("rows", 3)),
)
// {"level":"INFO","msg":"db query","db":{"table":"users","rows":3}}
```

## `LogValuer` keeps secrets out

A type can control its own logged representation. This is the reliable
way to stop a password reaching the logs — reliable because it works
everywhere the value is logged, not just where someone remembered:

```go
type User struct {
    Name     string
    Password string
}

func (u User) LogValue() slog.Value {
    return slog.GroupValue(slog.String("name", u.Name))
}
```

```go
slog.Info("login", "user", User{Name: "ada", Password: "hunter2"})
// {"level":"INFO","msg":"login","user":{"name":"ada"}}
```

The password is gone. Implement `LogValue` on every type that holds a
credential, a token, or personal data, and the redaction follows the
type around.

It also defers work: `LogValue` is only called if the entry is actually
emitted, so an expensive representation costs nothing at a filtered
level.

## The `Context` variants

`InfoContext`, `ErrorContext` and friends pass a context to the
handler. On their own they do nothing visible — the point is that a
custom handler can pull values out of it:

```go
type ctxHandler struct{ slog.Handler }

func (h ctxHandler) Handle(ctx context.Context, r slog.Record) error {
    if id, ok := ctx.Value(ctxKey{}).(string); ok {
        r.AddAttrs(slog.String("request_id", id))
    }
    return h.Handler.Handle(ctx, r)
}
```

```go
cl.InfoContext(ctx, "handled")
// {"level":"INFO","msg":"handled","request_id":"req-42"}
```

Embedding `slog.Handler` means you inherit `Enabled`, `WithAttrs` and
`WithGroup` and override only `Handle` — the same embedding trick as
the [middleware](../08-http-with-net-http/03-middleware.md) status
recorder.

This is how a trace id gets onto every log line automatically. Use the
`Context` variants by default; they cost nothing and enable this later.

## Asserting on logs in a test

Swap the default handler for one writing to a buffer:

```go
var buf bytes.Buffer
old := slog.Default()
slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
defer slog.SetDefault(old)

slog.Warn("careful", "n", 1)
// level=WARN msg=careful n=1
```

With a JSON handler you can decode each line and assert on fields
rather than matching text. Restore the previous default — `t.Cleanup` is
the right place — or you leak the buffer into every later test.

`HandlerOptions.ReplaceAttr` is what makes output deterministic: drop
the `time` attribute and the lines become comparable.

## The older `log` package

`log.Printf` and `log.Fatal` still exist and still appear in small
programs. Two things to know: `log.Fatal` calls `os.Exit`, so deferred
functions do not run; and its output is unstructured, so it cannot be
queried. For a service, use `slog`.

> **From Python:** this is `structlog`, in the standard library, and
> with no logger hierarchy — no `getLogger(__name__)`, no propagation,
> no `dictConfig`. You build a handler, set a default, and pass loggers
> explicitly. `LogValuer` is `__repr__` for logs, and it is the piece
> Python's logging has no good equivalent for.

## Quick reference

| Task | Form |
|---|---|
| a handler | `slog.NewJSONHandler(os.Stdout, opts)` / `NewTextHandler` |
| make it the default | `slog.SetDefault(slog.New(h))` |
| log | `slog.Info("msg", "key", value)` |
| typed, checked attrs | `slog.Int("n", 4)`, `slog.String(...)` |
| threshold | `&slog.HandlerOptions{Level: slog.LevelInfo}` |
| repeated fields | `logger.With(slog.String("service", "api"))` |
| nesting | `slog.Group("db", ...)` |
| redact a type | implement `LogValue() slog.Value` |
| values from a context | `InfoContext` plus a handler that reads it |
| test it | swap the default for a buffer handler, restore after |
| deterministic output | `HandlerOptions.ReplaceAttr` to drop `time` |

## Sources

- [`log/slog` package reference — pkg.go.dev/log/slog](https://pkg.go.dev/log/slog)
- [`slog.Handler` — pkg.go.dev/log/slog#Handler](https://pkg.go.dev/log/slog#Handler)
- [`slog.LogValuer` — pkg.go.dev/log/slog#LogValuer](https://pkg.go.dev/log/slog#LogValuer)
- [`slog.HandlerOptions` — pkg.go.dev/log/slog#HandlerOptions](https://pkg.go.dev/log/slog#HandlerOptions)
- [Go blog: structured logging with slog — go.dev/blog/slog](https://go.dev/blog/slog)
