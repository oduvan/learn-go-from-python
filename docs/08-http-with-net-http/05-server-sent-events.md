# Server-sent events

Sometimes the server has to tell the client something before the client
thinks to ask: a job's progress, a live log, a notification. Server-sent
events are the smallest way to do that — a normal HTTP response that
stays open and keeps writing.

```go
w.Header().Set("Content-Type", "text/event-stream")
rc := http.NewResponseController(w)

fmt.Fprintf(w, "event: tick\ndata: %d\n\n", i)
rc.Flush()
```

No new protocol, no upgrade handshake, and the client is a few lines of
JavaScript. Unlike a WebSocket it is one-directional — server to client
only — which is all most progress reporting needs.

## The wire format

Plain text. Each message is a set of `field: value` lines, terminated by
a **blank line**:

```
event: tick
data: 1

event: done
data: bye

```

| Field | Meaning |
|---|---|
| `data:` | the payload; repeat the line for multiline content |
| `event:` | a name the client can listen for; defaults to `message` |
| `id:` | lets the browser resume with `Last-Event-ID` after a drop |
| `retry:` | reconnect delay in milliseconds |

The double newline is what ends a message. Send `data: x\n` with one
newline and the client sits waiting for the rest of it — the single
most common SSE bug.

Keep `data:` on one line by encoding structured payloads as JSON.

## The handler

```go
func sseHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    rc := http.NewResponseController(w)

    for i := 1; i <= 3; i++ {
        select {
        case <-r.Context().Done():
            return
        default:
        }

        fmt.Fprintf(w, "event: tick\ndata: %d\n\n", i)
        if err := rc.Flush(); err != nil {
            return
        }
        time.Sleep(20 * time.Millisecond)
    }
}
```

Consuming it shows the shape:

```
content-type: text/event-stream
  "event: tick"
  "data: 1"
  "event: tick"
  "data: 2"
  "event: tick"
  "data: 3"
```

## Flushing is not optional

Responses are buffered. Without a flush, nothing reaches the client
until the handler returns — which for a stream is never, so the client
hangs with an open connection and no data.

`http.NewResponseController(w)` is the current way to reach the
underlying writer. The older approach was a type assertion to
`http.Flusher`:

```go
flusher, ok := w.(http.Flusher)   // older form
```

Prefer the response controller: it works through middleware that wraps
the `ResponseWriter`, which a bare type assertion does not. That is the
catch mentioned in [middleware](03-middleware.md) — a wrapper that
embeds `http.ResponseWriter` without forwarding `Flush` silently breaks
streaming.

`Flush` returning an error means the client is gone. Treat it as the
signal to stop.

## Noticing the client has left

A browser tab closes and nothing tells your handler directly — except
the request context, which is cancelled:

```go
select {
case <-r.Context().Done():
    return
default:
}
```

```
  got "event: tick" then hanging up
server: client went away: context canceled
```

Without that check the loop runs to completion, doing work for nobody
and holding a goroutine. For a long-lived stream, make the context a
real case in the `select` rather than a `default` poll:

```go
for {
    select {
    case <-r.Context().Done():
        return
    case msg := <-events:
        fmt.Fprintf(w, "data: %s\n\n", msg)
        if err := rc.Flush(); err != nil {
            return
        }
    case <-time.After(30 * time.Second):
        fmt.Fprint(w, ": keepalive\n\n")   // a comment line
        rc.Flush()
    }
}
```

A line starting with `:` is a comment the client ignores. Sending one
periodically keeps proxies from closing an idle connection, which they
will typically do after 30 to 60 seconds.

## Timeouts must not apply

An `http.Server` with a `WriteTimeout` will cut your stream off at
exactly that point. Either leave it unset on a server that streams, or
extend the deadline per request:

```go
rc := http.NewResponseController(w)
rc.SetWriteDeadline(time.Time{})   // no deadline for this response
```

The same goes for compression middleware — buffering to compress
defeats flushing, so exclude streaming routes from it.

## Never block on a slow subscriber

Each connected client gets a goroutine and usually a channel. If one
client stops reading, a blocking send stalls the goroutine feeding it —
and if a single producer fans out to many clients, one slow reader
stalls everyone.

Send without blocking and accept the loss:

```go
select {
case ch <- msg:
default:
    slog.Warn("dropping event for slow subscriber")
}
```

```go
full := make(chan string, 1)
fmt.Println(dropSlow(full, "a"), dropSlow(full, "b"))
// output: true false
```

The first send fits in the buffer; the second finds it full and gives up
rather than waiting. Dropping an event for one slow client is almost
always better than degrading everyone — and if the client needs the full
history, it should be fetching state, not tailing a stream.

## When it does not fit

- **Two-way communication** — use a WebSocket.
- **More than a handful of concurrent streams per replica** — each one
  holds a connection and a goroutine for its whole life.
- **Multiple replicas** — a client is connected to one instance, so an
  event raised on another never reaches it. That needs a shared pub/sub
  layer behind the handlers, which is a third-party concern.

> **From Python:** this is a streaming response with `yield`, but
> written as an explicit loop over a writer, and with `Flush` as
> something you call rather than something the framework infers.

## Quick reference

| Concern | Do |
|---|---|
| content type | `text/event-stream`, plus `Cache-Control: no-cache` |
| message format | `data: ...\n\n` — the **blank line** terminates it |
| named events | `event: name` before the `data:` line |
| push it out | `http.NewResponseController(w).Flush()` every message |
| client gone | `<-r.Context().Done()`, or a `Flush` error |
| idle proxies | a `: keepalive\n\n` comment every ~30s |
| server write timeout | unset it, or `rc.SetWriteDeadline(time.Time{})` |
| slow client | non-blocking `select` send with a `default` that drops |
| middleware | must forward `Flush`, and must not compress the route |

## Sources

- [`http.NewResponseController` — pkg.go.dev/net/http#NewResponseController](https://pkg.go.dev/net/http#NewResponseController)
- [`http.Flusher` — pkg.go.dev/net/http#Flusher](https://pkg.go.dev/net/http#Flusher)
- [`http.Request.Context` — pkg.go.dev/net/http#Request.Context](https://pkg.go.dev/net/http#Request.Context)
- [Server-sent events — developer.mozilla.org/en-US/docs/Web/API/Server-sent_events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)
- [WHATWG HTML: server-sent events — html.spec.whatwg.org/multipage/server-sent-events.html](https://html.spec.whatwg.org/multipage/server-sent-events.html)
