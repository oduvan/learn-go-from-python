# select

A `select` statement waits on **multiple channel operations at once** and
proceeds with whichever is ready first. It's the control structure that
makes channels composable — timeouts, cancellation, and multiplexing all
build on it.

```go
select {
case v := <-ch1:
    fmt.Println("from ch1:", v)
case v := <-ch2:
    fmt.Println("from ch2:", v)
}
```

Each `case` is a send or receive. `select` blocks until one can proceed,
runs that case, and continues. If **several** are ready at once, it picks
one **at random** — so don't rely on case order for priority.

```go
c1 := make(chan string, 1)
c2 := make(chan string, 1)
c1 <- "hello"            // only c1 has a value

select {
case m := <-c1:
    fmt.Println(m)       // output: hello
case m := <-c2:
    fmt.Println(m)
}
```

## `default`: don't block

A `default` case runs immediately if no other case is ready, turning
`select` into a **non-blocking** operation. This is how you poll a channel
or do a non-blocking send.

```go
ch := make(chan int)     // empty, no sender

select {
case v := <-ch:
    fmt.Println("got", v)
default:
    fmt.Println("nothing ready")   // output: nothing ready
}
```

## Timeouts with `time.After`

`time.After(d)` returns a channel that delivers a value after duration `d`.
Put it in a `select` and you get a timeout for free:

```go
ch := make(chan int)     // nobody will send

select {
case v := <-ch:
    fmt.Println("got", v)
case <-time.After(10 * time.Millisecond):
    fmt.Println("timeout")          // output: timeout
}
```

## The `for`-`select` loop with a done channel

The workhorse pattern for a long-running goroutine: loop on `select`,
handling work on one channel and a **stop signal** on another. A closed
channel makes every receive return immediately, so it broadcasts "stop" to
all listeners.

```go
nums := make(chan int)
done := make(chan struct{})
finished := make(chan struct{})

go func() {
    for {
        select {
        case n := <-nums:
            fmt.Println("got", n)
        case <-done:
            fmt.Println("stopping")
            close(finished)
            return
        }
    }
}()

nums <- 1
nums <- 2
close(done)              // signal the goroutine to stop
<-finished               // wait for it to actually finish
// output:
// got 1
// got 2
// stopping
```

## Disabling a case with a nil channel

A receive (or send) on a `nil` channel blocks forever, so setting a
channel variable to `nil` **removes** its case from a `select`. This is the
idiomatic way to stop listening on a channel mid-loop without restructuring
the `select`.

```go
var ch chan int          // nil
select {
case <-ch:               // never fires — ch is nil
    fmt.Println("unreachable")
case <-time.After(time.Millisecond):
    fmt.Println("only this can happen")   // output: only this can happen
}
```

## An empty `select` blocks forever

`select {}` with no cases blocks the goroutine permanently — occasionally
used to park `main` while background goroutines run, but in `main` it
triggers the deadlock detector if nothing else is runnable.

## Quick reference

| Form | Meaning |
|---|---|
| `select { case ...: }` | wait until one channel op is ready |
| several ready | one chosen **at random** |
| `default:` | run if nothing else ready (non-blocking) |
| `case <-time.After(d):` | timeout branch |
| `for { select { ... } }` | long-running multiplexed loop |
| `case <-done:` | stop signal (closed channel broadcasts) |
| nil channel case | disabled (blocks forever) |

## Sources

- [Select statements — go.dev/ref/spec#Select_statements](https://go.dev/ref/spec#Select_statements)
- [time.After — pkg.go.dev/time#After](https://pkg.go.dev/time#After)
- [Go blog: pipelines and cancellation — go.dev/blog/pipelines](https://go.dev/blog/pipelines)
- [Effective Go: concurrency — go.dev/doc/effective_go#concurrency](https://go.dev/doc/effective_go#concurrency)
