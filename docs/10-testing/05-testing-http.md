# Testing HTTP

`net/http/httptest` covers both directions: calling a handler without a
network, and standing up a real server so your client has something to
talk to.

```go
rec := httptest.NewRecorder()
req := httptest.NewRequest(http.MethodGet, "/users/1", nil)

api.Handler().ServeHTTP(rec, req)
```

## Testing a handler: `NewRecorder`

A handler is just a function taking a writer and a request, so you can
call it directly. `httptest.NewRecorder` is a `ResponseWriter` that
remembers what was written, and `httptest.NewRequest` builds a request
without parsing a URL over the wire:

```go
api := API{Users: stubUsers{ByIDFn: func(_ context.Context, id string) (User, error) {
    return User{Name: "Ada"}, nil
}}}

rec := httptest.NewRecorder()
api.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/users/1", nil))

// status=200 ct="application/json" body="{\"name\":\"Ada\"}"
```

No port, no listener, no cleanup. Read the result from `rec.Code`,
`rec.Body` and `rec.Result().Header`.

Call `ServeHTTP` on the **mux**, not on the handler function, when you
want routing and method matching tested too — otherwise a wrong path
never gets exercised.

`httptest.NewRequest` panics on a malformed target rather than
returning an error, which is what you want in a test.

Error paths are where the stub earns its place:

```go
api := API{Users: stubUsers{ByIDFn: func(context.Context, string) (User, error) {
    return User{}, ErrNotFound
}}}
// status=404 body="not found"
```

## Testing a client: `NewServer`

For the other direction, run a real HTTP server on a real loopback
port. The client under test is then exercised over an actual connection:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(User{Name: "Bo"})
}))
t.Cleanup(srv.Close)

c := Client{BaseURL: srv.URL, HTTP: srv.Client()}
```

`srv.URL` is the assigned address — the port is chosen by the OS, so
parallel tests never collide. `srv.Client()` returns a client configured
for that server, which matters for `NewTLSServer` where it carries the
test certificate.

`t.Cleanup(srv.Close)` rather than `defer srv.Close()`: it works with
`t.Parallel` and inside helpers.

This is why a client type should take its base URL and its
`*http.Client` as fields. A client that hardcodes a URL cannot be
tested this way at all.

## Scripting responses

The fake server is an ordinary handler, so it can behave however the
test needs. Counting calls with an atomic makes retry logic testable:

```go
var calls atomic.Int32

srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if calls.Add(1) < 3 {
        w.WriteHeader(http.StatusServiceUnavailable)
        return
    }
    json.NewEncoder(w).Encode(User{Name: "Bo"})
}))
```

```go
u, err := c.Fetch(context.Background(), "1")
// calls=3 user={Name:Bo} err=<nil>
```

That asserts the interesting thing: the client retried twice and
succeeded on the third attempt. Use an atomic rather than a plain `int`
— the handler runs on the server's goroutine, so a plain counter is a
race the detector will flag.

The same shape covers a `500` that never recovers, a timeout
(`time.Sleep` longer than the client's deadline), malformed JSON, and a
connection closed mid-body.

## Asserting what the client sent

Capture it in the handler:

```go
var gotPath, gotToken string

srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    gotPath = r.URL.Path
    gotToken = r.Header.Get("X-Token")
    json.NewEncoder(w).Encode(User{Name: "x"})
}))
// path="/users/7" token="abc"
```

This is how you check that authentication headers, query parameters and
request bodies are built correctly — without inspecting the client's
internals.

If the client is called concurrently, guard those variables or use
channels; assigning from the handler goroutine and reading from the
test goroutine is a race.

## Which one to reach for

| Testing | Use |
|---|---|
| a handler or middleware | `NewRecorder` + `NewRequest` |
| a client, or retry and timeout behaviour | `NewServer` |
| TLS-specific behaviour | `NewTLSServer` + `srv.Client()` |
| the full stack end to end | `NewServer` wrapping your real router |

`NewRecorder` is faster and simpler, so prefer it for handler logic.
`NewServer` is the honest choice whenever the thing you are testing
involves the transport: timeouts, retries, connection reuse, streaming.

## What a recorder does not do

`httptest.NewRecorder` is a stand-in, not a real server. It does not
run the HTTP state machine, so it will not tell you about a handler
writing headers after the body, and its `Flush` does nothing useful for
streaming. For
[server-sent events](../08-http-with-net-http/05-server-sent-events.md)
or anything else where flushing matters, use `NewServer` and read the
response body as a stream.

> **From Python:** `NewRecorder` is a test client that bypasses the
> network, and `NewServer` is `responses`/`httpretty` turned inside
> out — instead of intercepting the client's calls, you run a real
> server and point the client at it. Fewer surprises, because nothing is
> monkey-patched.

## Quick reference

| Task | Call |
|---|---|
| fake response writer | `httptest.NewRecorder()` |
| build a request | `httptest.NewRequest(method, target, body)` |
| invoke a handler | `h.ServeHTTP(rec, req)` — use the mux for routing |
| read the result | `rec.Code`, `rec.Body`, `rec.Result().Header` |
| real server on a free port | `httptest.NewServer(handler)` |
| its address / client | `srv.URL`, `srv.Client()` |
| shut it down | `t.Cleanup(srv.Close)` |
| count calls | `atomic.Int32` in the handler |
| check what was sent | capture from inside the handler |
| streaming or TLS | `NewServer` / `NewTLSServer`, not the recorder |

## Sources

- [`net/http/httptest` — pkg.go.dev/net/http/httptest](https://pkg.go.dev/net/http/httptest)
- [`httptest.NewRecorder` — pkg.go.dev/net/http/httptest#NewRecorder](https://pkg.go.dev/net/http/httptest#NewRecorder)
- [`httptest.NewServer` — pkg.go.dev/net/http/httptest#NewServer](https://pkg.go.dev/net/http/httptest#NewServer)
- [`httptest.NewRequest` — pkg.go.dev/net/http/httptest#NewRequest](https://pkg.go.dev/net/http/httptest#NewRequest)
- [`sync/atomic` — pkg.go.dev/sync/atomic](https://pkg.go.dev/sync/atomic)
