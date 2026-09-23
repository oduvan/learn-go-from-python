# An HTTP client

Calling another service is the other half of `net/http`. The package
gets you a long way, but its defaults have one dangerous gap and one
behaviour that surprises everybody.

```go
client := &http.Client{Timeout: 10 * time.Second}

req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
resp, err := client.Do(req)
```

## Never use `http.DefaultClient`

`http.Get`, `http.Post` and `http.DefaultClient` all share one client
with **no timeout**. A server that accepts your connection and then
never replies will hang that goroutine forever, and under load the
process runs out of them. It is the most common way a Go service falls
over.

Make your own, once, and reuse it:

```go
var client = &http.Client{Timeout: 10 * time.Second}
```

An `http.Client` is safe for concurrent use and pools connections
internally. Creating one per request throws the pool away and leaks file
descriptors — build it at startup and pass it in.

`Timeout` covers the whole exchange: connect, send, wait, and read the
body. Set it longer than your slowest legitimate call.

## Build requests with a context

`client.Get(url)` is fine for a throwaway. Real calls use
`NewRequestWithContext`, so the request dies with whatever triggered it:

```go
req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
if err != nil {
    return err
}
req.Header.Set("Content-Type", "application/json")
req.Header.Set("X-Token", token)

resp, err := client.Do(req)
```

Passing `r.Context()` from an inbound handler means a client hanging up
cancels the outbound call too, instead of leaving it to finish for
nobody.

The body is an `io.Reader`, so `bytes.NewReader`, `strings.NewReader`
and an open file all work:

```go
body, _ := json.Marshal(User{Name: "Bo"})
req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
// the server receives a POST with the body {"name":"Bo","age":0}
```

## A 404 is not an error

This catches everyone once. `err` is non-nil only when the exchange
failed — DNS, connection, timeout. **Any response the server sent, at
any status, is a success:**

```go
resp, err := client.Get(url + "/404")
fmt.Println("err:", err, "status:", resp.StatusCode)
// output: err: <nil> status: 404
```

So every call needs two checks:

```go
resp, err := client.Do(req)
if err != nil {
    return fmt.Errorf("calling %s: %w", url, err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
    return fmt.Errorf("%s: unexpected status %s", url, resp.Status)
}
```

## Always close the body, always drain it

`resp.Body` is an open network stream. Not closing it leaks the
connection:

```go
defer resp.Body.Close()
```

`defer` it immediately after the error check — before anything that
might return early.

There is a second, quieter rule: a connection is only reused if the body
was read to the end. On a path where you ignore the body, drain it:

```go
io.Copy(io.Discard, resp.Body)
resp.Body.Close()
```

Skip that and every such request opens a fresh connection, which shows
up as mysterious socket exhaustion under load rather than as an error.

Decoding straight from the stream drains it naturally:

```go
var u User
if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
    return fmt.Errorf("decoding response: %w", err)
}
```

Use `io.LimitReader` if the peer is not fully trusted.

## Recognising a timeout

A client error is wrapped in `*url.Error`, and `errors.Is` sees the
cause through it:

```go
_, err := tc.Get(slowURL)
fmt.Println(errors.Is(err, context.DeadlineExceeded))   // output: true

var ue *url.Error
fmt.Println(errors.As(err, &ue), ue.Timeout())          // output: true true
```

Both the client's own `Timeout` and a cancelled context surface as
`context.DeadlineExceeded`, so one check covers both.

## Retrying

Nothing retries for you. Retry only what is safe to repeat — a `GET`, or
a request carrying an idempotency key — and back off between attempts:

```go
for attempt := range 5 {
    resp, err := client.Do(req)
    if err == nil && resp.StatusCode < 500 {
        return resp, nil
    }
    if resp != nil {
        io.Copy(io.Discard, resp.Body)
        resp.Body.Close()
    }
    time.Sleep(time.Duration(1<<attempt) * 100 * time.Millisecond)
}
// third attempt succeeds: attempt 2 status 200 recovered
```

`1<<attempt` doubles the wait each time. Three details matter: close the
body on the failed attempt too, do not retry a `4xx` (it will fail
identically), and check `ctx.Err()` in the loop so a cancelled request
stops retrying.

A request body that is an `io.Reader` is consumed by the first attempt.
To retry a `POST`, keep the bytes and build a fresh `bytes.NewReader`
each time.

Real systems add jitter so that many clients recovering from the same
outage do not retry in lockstep.

## Building URLs

Never concatenate query strings. `url.Values` escapes for you:

```go
u, _ := url.Parse(base + "/search")
q := u.Query()
q.Set("q", "go & rust")
q.Set("page", "2")
u.RawQuery = q.Encode()

fmt.Println(u.String())
// output: http://127.0.0.1:8080/search?page=2&q=go+%26+rust
```

`Encode` sorts keys, so the output is stable — handy for caching and
tests. The `&` became `%26`, which is the whole point.

```go
fmt.Println(url.QueryEscape("a b&c"))   // output: a+b%26c
```

Parsing gives you the pieces:

```go
pu, _ := url.Parse("https://x.example/a/b?k=v#frag")
fmt.Println(pu.Scheme, pu.Host, pu.Path, pu.Query().Get("k"), pu.Fragment)
// output: https x.example /a/b v frag
```

## Tuning the transport

`http.Client` handles policy — timeouts, redirects, cookies. The
`Transport` handles connections. The defaults suit a handful of hosts;
a service hammering one upstream usually needs a bigger per-host pool:

```go
transport := http.DefaultTransport.(*http.Transport).Clone()
transport.MaxIdleConnsPerHost = 100
transport.IdleConnTimeout = 90 * time.Second

client := &http.Client{Timeout: 10 * time.Second, Transport: transport}
```

`DefaultMaxIdleConnsPerHost` is 2, so without this every extra
concurrent request to the same host opens and discards a connection.
`Clone` matters — mutating `http.DefaultTransport` changes it for
everything in the process.

> **From Python:** this is `requests`, minus the conveniences. There is
> no `raise_for_status()`, so you check the code yourself; no
> `resp.json()`, so you decode; and no session by default, so you must
> hold onto one client to get connection pooling. The upside is the
> context, which gives cancellation `requests` has no equivalent for.

## Quick reference

| Task | Call |
|---|---|
| a client | `&http.Client{Timeout: d}` at startup — never the default |
| a request | `http.NewRequestWithContext(ctx, method, url, body)` |
| send | `client.Do(req)` |
| check | `err != nil` **and** `resp.StatusCode` — a 404 is not an error |
| close | `defer resp.Body.Close()`, right after the error check |
| reuse the connection | drain with `io.Copy(io.Discard, resp.Body)` |
| decode | `json.NewDecoder(resp.Body).Decode(&v)` |
| detect a timeout | `errors.Is(err, context.DeadlineExceeded)` |
| retry | only idempotent calls, back off, re-create the body |
| query strings | `url.Values` + `q.Encode()` |
| bigger connection pool | `DefaultTransport.Clone()`, `MaxIdleConnsPerHost` |

## Sources

- [`http.Client` — pkg.go.dev/net/http#Client](https://pkg.go.dev/net/http#Client)
- [`http.NewRequestWithContext` — pkg.go.dev/net/http#NewRequestWithContext](https://pkg.go.dev/net/http#NewRequestWithContext)
- [`http.Transport` — pkg.go.dev/net/http#Transport](https://pkg.go.dev/net/http#Transport)
- [`net/url` package reference — pkg.go.dev/net/url](https://pkg.go.dev/net/url)
- [`url.Values` — pkg.go.dev/net/url#Values](https://pkg.go.dev/net/url#Values)
