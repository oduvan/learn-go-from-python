# Panic and recover

[errors](07-errors.md) covers Go's primary failure mechanism —
returning an `error`. This article covers the **other** one: `panic`
and `recover`, which exist for the small set of cases where a normal
error return doesn't apply.

The rule of thumb up front: **don't use panic for ordinary errors**.
Use it for programmer mistakes (impossible states, contract
violations) and for unrecoverable runtime conditions. For everything
else, return an `error`.

## What `panic` does

Calling `panic(v)` halts the function's normal execution and starts
**unwinding the stack**:

1. The current function's deferred calls run, in LIFO order.
2. The function returns to its caller.
3. The caller's deferred calls run.
4. The caller returns to *its* caller.
5. This continues until either:
   - a deferred function calls `recover()` and stops the unwinding, or
   - the panic reaches `main` and crashes the program with a stack
     trace.

Runtime errors — nil-pointer dereference, out-of-range slice index,
divide by zero on an integer, sending to a closed channel — all
trigger an implicit panic.

```go
package main

import "fmt"

func main() {
    defer fmt.Println("deferred in main")
    panic("boom")
}
// output:
// deferred in main
// panic: boom
//
// goroutine 1 [running]:
// main.main()
//     .../main.go:7 +0x...
// exit status 2
```

The deferred `fmt.Println` ran (panic unwinding still runs defers),
then the program crashed.

## What `recover` does

`recover()` is a built-in that **only does anything inside a deferred
function**. Anywhere else it returns `nil` and is harmless.

- During normal execution (no active panic): `recover` returns `nil`.
- During an active panic: `recover` returns the value passed to
  `panic` and **stops the unwinding** at the current function. That
  function returns to its caller normally.

```go
package main

import "fmt"

func safe() {
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("recovered:", r)
        }
    }()
    panic("oops")
}

func main() {
    safe()
    fmt.Println("main keeps going")
}
// output:
// recovered: oops
// main keeps going
```

The `defer func() { ... }()` shape is the universal pattern:

- The deferred call must be a function (you can't `defer recover()`
  directly — well, you can, but it's almost always wrong).
- `recover()` must be called from a function that is **itself**
  directly deferred. If the deferred function calls *another*
  helper, and the helper calls `recover()`, it won't catch the
  panic — the panic is no longer active at that depth.

```go
func wrong() {
    defer func() {            // this is the deferred function
        helper()              // helper is called BY the deferred function
    }()
    panic("boom")
}

func helper() {
    if r := recover(); r != nil {       // never triggers — wrong frame
        fmt.Println("won't print")
    }
}
```

The fix is simple: call `recover` directly inside the deferred
function literal, not from a helper.

## The full unwinding example

```go
package main

import "fmt"

func main() {
    f()
    fmt.Println("returned normally from f")
}

func f() {
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("recovered in f:", r)
        }
    }()
    fmt.Println("calling g")
    g(0)
    fmt.Println("returned normally from g")
}

func g(i int) {
    if i > 3 {
        fmt.Println("panicking!")
        panic(fmt.Sprintf("%v", i))
    }
    defer fmt.Println("defer in g", i)
    fmt.Println("printing in g", i)
    g(i + 1)
}
```

Output:

```
calling g
printing in g 0
printing in g 1
printing in g 2
printing in g 3
panicking!
defer in g 3
defer in g 2
defer in g 1
defer in g 0
recovered in f: 4
returned normally from f
```

Notice:

- Every `defer` along the unwinding path ran.
- `g`'s "returned normally from g" line did **not** print — the
  panic skipped over it.
- `recover` in `f` stopped the unwinding; `main` saw `f` return
  normally and printed its final line.

## When to use panic — three legitimate cases

1. **A programmer mistake the type system can't catch.** Calling
   `divide(0, 0)`, indexing a slice you just confirmed has 3
   elements at position `5`, accessing a struct field you swore was
   set. These represent bugs; the right behaviour is to crash and
   surface them.

   The standard library exposes one helper for this:

   ```go
   if user == nil {
       panic("logic error: caller must initialise user")
   }
   ```

2. **An unrecoverable runtime condition.** The program literally
   cannot proceed. Loss of essential configuration on startup,
   for example.

3. **As an internal unwinding mechanism**, recovered at the package
   boundary. The standard library does this — `encoding/json` panics
   internally during recursive traversal and recovers at the top of
   `Marshal`/`Unmarshal`, then returns a normal `error` to the
   caller.

   The user-facing API still returns an `error`. The caller never
   sees the panic.

## When *not* to use panic

- An expected error case ("file doesn't exist", "input was invalid",
  "network call failed"). Return an `error`.
- To save typing on error checks. The repetitive `if err != nil`
  blocks are the *idiom*, not a smell.
- To approximate exceptions. Go's design rejects exceptions on
  purpose. Don't reinvent them.

> **From Python:** Python uses exceptions for both ordinary errors
> *and* programmer mistakes (`KeyError`, `IndexError`, `TypeError`).
> Go splits the responsibility — `error` for ordinary failures,
> `panic` only for "this should never happen." Catching a panic with
> `recover` is the rough equivalent of an `except:` at a process
> boundary, not a normal control-flow tool.

## `recover` at a goroutine boundary

(Skip ahead if you haven't seen goroutines yet — they're Go's
lightweight concurrent tasks, covered in a later topic. The keyword
`go` launches a function as one: `go work()` runs `work` alongside
the caller instead of blocking on it.)

A panic only unwinds the **current goroutine**. If a goroutine
panics and nothing in its stack recovers, the *entire program*
crashes — even if all the other goroutines were healthy.

So: if you launch a goroutine that might panic, the goroutine itself
needs a `recover` at the top (or you must be certain the goroutine
can't panic).

```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Println("worker crashed:", r)
        }
    }()
    runJob()
}()
```

This is a frequent source of production outages: an unhandled panic
in a background goroutine takes the whole process down.

## Panic value can be any type

`panic` takes an `any`. You can panic with a string, an error, a
custom struct — anything.

```go
panic("oops")
panic(errors.New("disk full"))
panic(struct{ Code int }{42})
```

Convention: if you panic with an error, you can `recover` it and
return it as one. The `r.(error)` part is a **type assertion** — read
"if `r` actually contains an `error`, give it to me as `e`, and set
`ok` to true; otherwise `ok` is false." Type assertions get their
own treatment in the interfaces topic; for now treat the form as
"unwrap an `any` value into a known type, safely."

```go
func safeRun(f func()) (err error) {
    defer func() {
        if r := recover(); r != nil {
            if e, ok := r.(error); ok {
                err = e
            } else {
                err = fmt.Errorf("panic: %v", r)
            }
        }
    }()
    f()
    return nil
}
```

## Quick reference

| You want | Write |
|---|---|
| Crash on an impossible state | `panic("invariant violated: ...")` |
| Catch a panic at a boundary | `defer func() { if r := recover(); r != nil { ... } }()` |
| Convert a panic to an error | recover into a named return: `defer func() { if r := recover(); r != nil { err = ... } }()` |
| Protect a background goroutine | wrap its body with `defer recover()` and log the value |
| Just return an error instead | always your first choice — see [errors](07-errors.md) |

## Sources

- [Defer, Panic, and Recover — go.dev/blog/defer-panic-and-recover](https://go.dev/blog/defer-panic-and-recover)
- [Handling panics — go.dev/ref/spec#Handling_panics](https://go.dev/ref/spec#Handling_panics)
- [`builtin.panic` — pkg.go.dev/builtin#panic](https://pkg.go.dev/builtin#panic)
- [`builtin.recover` — pkg.go.dev/builtin#recover](https://pkg.go.dev/builtin#recover)
