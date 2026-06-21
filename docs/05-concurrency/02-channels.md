# Channels

A **channel** is a typed conduit that lets goroutines communicate and
synchronise by passing values. One goroutine sends; another receives. The
channel handles the hand-off safely, so you don't need locks to move data
between goroutines.

> **Go's motto:** *don't communicate by sharing memory; share memory by
> communicating.* Prefer passing values over a channel to guarding shared
> variables with a mutex.

Create one with `make`, send with `ch <- v`, receive with `v := <-ch`:

```go
ch := make(chan string)
go func() { ch <- "ping" }()   // send
msg := <-ch                     // receive
fmt.Println(msg)                // output: ping
```

## Unbuffered channels synchronise

The channel above is **unbuffered**: a send blocks until another goroutine
is ready to receive, and vice versa. The exchange is a *rendezvous* — both
sides meet at the same instant. That makes an unbuffered channel a
synchronisation tool, not just a pipe: the receive can't complete before
the send happens.

```go
done := make(chan struct{})
go func() {
    fmt.Println("working")
    done <- struct{}{}      // signal completion
}()
<-done                       // blocks until the goroutine signals
fmt.Println("finished")
// output:
// working
// finished
```

A `chan struct{}` is the idiomatic "signal only, no data" channel.

## Buffered channels hold values

`make(chan T, n)` gives the channel a buffer of `n`. Sends block only when
the buffer is **full**; receives block only when it's **empty**. This
decouples sender and receiver in bursts.

```go
ch := make(chan int, 2)
ch <- 1          // doesn't block — buffer has room
ch <- 2          // doesn't block — now full
fmt.Println(len(ch), cap(ch))   // output: 2 2
fmt.Println(<-ch, <-ch)          // output: 1 2
```

`len` is how many values are buffered right now; `cap` is the buffer size.
Use a buffer when you knowingly want slack; reach for unbuffered by default,
since it gives you synchronisation for free.

## Closing a channel

`close(ch)` marks that no more values will be sent. Receivers can still
drain whatever's buffered, then get the zero value. The **comma-ok**
receive distinguishes a real value from "closed and empty":

```go
ch := make(chan int, 2)
ch <- 10
close(ch)

v, ok := <-ch
fmt.Println(v, ok)   // output: 10 true   — a real value
v, ok = <-ch
fmt.Println(v, ok)   // output: 0 false   — closed and drained
```

Rules: only the **sender** should close, and only once. Sending on a closed
channel **panics**; closing an already-closed channel panics. Closing is a
broadcast — every receiver sees it.

## Ranging over a channel

`for v := range ch` receives values until the channel is **closed and
drained**, then ends the loop. It's the clean way to consume a stream:

```go
ch := make(chan int, 3)
ch <- 1
ch <- 2
ch <- 3
close(ch)                // without this, range would block forever

for v := range ch {
    fmt.Println(v)
}
// output:
// 1
// 2
// 3
```

## Channel direction in signatures

A function parameter can restrict a channel to **send-only** (`chan<- T`)
or **receive-only** (`<-chan T`). This documents intent and lets the
compiler stop misuse.

```go
func produce(out chan<- int) { out <- 42; close(out) }   // can only send
func consume(in <-chan int)  { fmt.Println(<-in) }        // can only receive

ch := make(chan int, 1)
produce(ch)
consume(ch)            // output: 42
```

## Deadlocks and nil channels

If every goroutine is blocked waiting on a channel, the runtime detects it
and aborts:

```go
func main() {
    ch := make(chan int)
    <-ch        // nothing will ever send
}
// fatal error: all goroutines are asleep - deadlock!
```

A `nil` channel (never `make`-d) blocks **forever** on both send and
receive — occasionally useful in `select` to disable a case, but otherwise
a bug.

## Quick reference

| Operation | Meaning |
|---|---|
| `make(chan T)` | unbuffered — send/receive rendezvous |
| `make(chan T, n)` | buffered — blocks only when full/empty |
| `ch <- v` / `v := <-ch` | send / receive |
| `v, ok := <-ch` | `ok` is false once closed and drained |
| `close(ch)` | no more sends; sender-only, once |
| `for v := range ch` | receive until closed |
| `chan<- T` / `<-chan T` | send-only / receive-only parameter |
| send on closed channel | **panic** |
| all goroutines blocked | **fatal: deadlock** |

## Sources

- [Channel types — go.dev/ref/spec#Channel_types](https://go.dev/ref/spec#Channel_types)
- [Send statements — go.dev/ref/spec#Send_statements](https://go.dev/ref/spec#Send_statements)
- [Receive operator — go.dev/ref/spec#Receive_operator](https://go.dev/ref/spec#Receive_operator)
- [Close — pkg.go.dev/builtin#close](https://pkg.go.dev/builtin#close)
- [Effective Go: channels — go.dev/doc/effective_go#channels](https://go.dev/doc/effective_go#channels)
- [Go blog: share memory by communicating — go.dev/blog/codelab-share](https://go.dev/blog/codelab-share)
