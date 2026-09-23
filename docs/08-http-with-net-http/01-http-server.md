# An HTTP server

`net/http` is a production-capable web server in the standard library.
No framework is required to serve real traffic, and every framework you
will meet is built on the interfaces in this article.

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "hello")
})
http.ListenAndServe(":8080", mux)
```

## Handlers

Everything that serves a request satisfies one interface:

```go
type Handler interface {
    ServeHTTP(w http.ResponseWriter, r *http.Request)
}
```

Writing a struct with that method is one option. Usually you write a
function and let `http.HandlerFunc` adapt it — an adapter type whose
`ServeHTTP` simply calls the function:

```go
func hello(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "hello")
}

var h http.Handler = http.HandlerFunc(hello)
```

`mux.HandleFunc` does that conversion for you. `w` is an `io.Writer`, so
everything from
[readers and writers](../07-operating-system/02-readers-and-writers.md)
applies: `fmt.Fprintf`, `io.Copy`, `json.NewEncoder(w)`.

When a handler needs dependencies, make it a method on a struct. That is
how you inject a database or a logger without reaching for globals:

```go
type API struct{ store Store }

func (a *API) getUser(w http.ResponseWriter, r *http.Request) {
    // a.store is available here
}

mux.HandleFunc("GET /users/{id}", a.getUser)
```

## Routing with method and wildcards

`ServeMux` patterns take an optional method and can capture path
segments:

```go
mux.HandleFunc("GET /hello", helloHandler)
mux.HandleFunc("GET /users/{id}", getUser)
mux.HandleFunc("POST /users", createUser)
```

```go
func getUser(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "user %s", r.PathValue("id"))
}
// GET /users/42  →  200 "user 42"
```

`r.PathValue("id")` reads the captured segment. `{rest...}` at the end
of a pattern matches everything remaining.

A pattern ending in `/` matches a subtree; without the slash it matches
exactly. The most specific pattern wins, so `/users/{id}` beats `/users/`.

Naming the method gets you correct behaviour for free:

```
GET  /users   →  405 Method Not Allowed
GET  /nope    →  404 page not found
```

A `405` with no work from you is the reason to write the method in the
pattern rather than checking `r.Method` inside the handler.

## Reading the request

```go
q := r.URL.Query().Get("q")          // ?q=go
id := r.PathValue("id")              // /users/{id}
ct := r.Header.Get("Content-Type")   // header, case-insensitive
```

`Query()` parses on every call, so hold the result if you need several
values. A missing key gives `""` — use `r.URL.Query().Has("k")` to tell
absent from empty.

Decode a JSON body straight off the stream rather than reading it all
first:

```go
func createUser(w http.ResponseWriter, r *http.Request) {
    var u User
    if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
        http.Error(w, "bad json", http.StatusBadRequest)
        return
    }
    // ...
}
```

Note the `return` after `http.Error`. Writing a response does not stop
the handler, and forgetting to return is the most common bug in this
code — you end up writing a second response on top of the error.

The server closes `r.Body` for you; a *client* response body is your
job, which the [next article](02-http-client.md) covers.

Cap what you will read, or a large upload becomes an out-of-memory
crash:

```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20)   // 1 MiB
```

## Writing the response, in the right order

Three steps, and they must happen in this sequence:

```go
w.Header().Set("Content-Type", "application/json")   // 1. headers
w.WriteHeader(http.StatusCreated)                    // 2. status
json.NewEncoder(w).Encode(u)                         // 3. body
```

The first `Write` sends a `200` and freezes the headers. Anything set
after that is ignored, and the server logs a complaint rather than
failing:

```go
w.Write([]byte("body first"))
w.WriteHeader(http.StatusTeapot)   // too late
w.Header().Set("X-Late", "yes")    // too late
// log: http: superfluous response.WriteHeader call from ...
// the client still gets 200
```

Skip `WriteHeader` entirely for a `200`. Use the named constants —
`http.StatusCreated`, `http.StatusBadRequest` — not bare numbers.

With no `Content-Type` set, the server sniffs it from the first bytes,
which is how plain text becomes `text/plain; charset=utf-8`. Set it
yourself for anything structured.

`http.Error(w, msg, code)` writes a plain-text error and the status in
one call.

## Configure the server, don't use the default

`http.ListenAndServe` is fine for an example. Real code builds an
`http.Server`, because the defaults have **no timeouts at all** — a slow
client can hold a connection open indefinitely:

```go
srv := &http.Server{
    Addr:              ":8080",
    Handler:           mux,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       60 * time.Second,
}

if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
    return err
}
```

`ListenAndServe` blocks until the server stops, and returns
`http.ErrServerClosed` after a clean shutdown — which is a normal
outcome, not a failure, hence the `errors.Is`.

Avoid `http.HandleFunc` and `http.ListenAndServe(addr, nil)`. Those use
a package-level default mux, which is global mutable state any imported
package can register on. Make your own `ServeMux`.

## Shutting down

`Shutdown` stops accepting connections and waits for in-flight requests,
bounded by the context — the pattern from
[signals and graceful shutdown](../07-operating-system/06-signals-and-graceful-shutdown.md).
The code logs errors with `slog`, Go's standard structured logger, which
[Structured logging with slog](../12-observability/01-structured-logging-with-slog.md)
covers later:

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

go func() {
    if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        slog.Error("listen", "error", err)
    }
}()

<-ctx.Done()
stop()

shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()
return srv.Shutdown(shutdownCtx)
```

`Shutdown` does not wait for hijacked or long-lived streaming
connections. Those need their own cancellation.

## Every request carries a context

`r.Context()` is cancelled when the client disconnects, so pass it down
to anything slow and the work stops when nobody is listening:

```go
func (a *API) getUser(w http.ResponseWriter, r *http.Request) {
    u, err := a.store.User(r.Context(), r.PathValue("id"))
    // ...
}
```

## Static files

```go
mux.Handle("GET /static/", http.StripPrefix("/static/",
    http.FileServer(http.Dir("assets"))))
```

`StripPrefix` is needed because `FileServer` resolves the whole request
path. Serving from an `embed.FS` instead of disk uses `http.FS` and
keeps the binary self-contained.

> **From Python:** `ServeMux` is roughly Flask's routing with far fewer
> features, but there is no WSGI layer and no separate server to deploy
> behind — this *is* the production server. A handler takes the writer
> as a parameter rather than returning a response, so "return early
> after writing" is a discipline the type system will not enforce.

## Quick reference

| Task | Call |
|---|---|
| a router | `http.NewServeMux()` — not the default mux |
| register | `mux.HandleFunc("GET /users/{id}", fn)` |
| path segment | `r.PathValue("id")` |
| query | `r.URL.Query().Get("q")`, `.Has("q")` |
| JSON in | `json.NewDecoder(r.Body).Decode(&v)` |
| limit the body | `http.MaxBytesReader(w, r.Body, n)` |
| headers, then status, then body | `w.Header().Set` → `w.WriteHeader` → `w.Write` |
| an error response | `http.Error(w, msg, code)` — then **`return`** |
| timeouts | build an `http.Server`, never the defaults |
| clean stop | `srv.Shutdown(ctx)`; expect `http.ErrServerClosed` |
| per-request cancellation | `r.Context()` |
| static files | `http.StripPrefix` + `http.FileServer` |

## Sources

- [`net/http` package reference — pkg.go.dev/net/http](https://pkg.go.dev/net/http)
- [`http.ServeMux` — pkg.go.dev/net/http#ServeMux](https://pkg.go.dev/net/http#ServeMux)
- [`http.Server` — pkg.go.dev/net/http#Server](https://pkg.go.dev/net/http#Server)
- [`http.Server.Shutdown` — pkg.go.dev/net/http#Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)
- [Routing enhancements — go.dev/blog/routing-enhancements](https://go.dev/blog/routing-enhancements)
