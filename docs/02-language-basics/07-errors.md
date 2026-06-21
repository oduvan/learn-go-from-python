# Errors

Go has no exceptions. When a function can fail, it returns an extra
value of type `error`, and the caller checks it explicitly.

```go
n, err := strconv.Atoi("forty-two")
if err != nil {
    fmt.Println("parse failed:", err)
    return
}
fmt.Println(n)
```

This pattern — check `err`, react, return — is the most common
five-line block in any Go program.

> **From Python:** Python uses exceptions: a failure interrupts
> control flow and unwinds the stack until something catches it. Go
> makes failure an ordinary return value. The mental shift is real:
> every fallible operation is *visible* at the call site, and you
> handle it (or explicitly pass it along) right there.

## The `error` type

`error` is a built-in **interface** with one method. (An interface,
for now, is a named set of method signatures any type can satisfy by
implementing them — full treatment in a later article.)

```go
type error interface {
    Error() string
}
```

Anything with an `Error() string` method satisfies it. You don't need
to know how to define one yet — that comes after methods and
interfaces.

The zero value of `error` is `nil`, which means "no error." That's
why `if err != nil { ... }` is the standard check.

## Creating an error: `errors.New`

`errors.New` returns a brand-new error carrying just a message.

```go
import "errors"

err := errors.New("something went wrong")
fmt.Println(err)                    // something went wrong
fmt.Println(err.Error())            // something went wrong
```

Each call returns a **distinct** value — two errors with identical
text are not equal under `==`:

```go
e1 := errors.New("oops")
e2 := errors.New("oops")
fmt.Println(e1 == e2)               // false
```

If you want a comparable, package-level error to check against, declare
it once and reuse it:

```go
var ErrNotFound = errors.New("not found")

func lookup(id int) (string, error) {
    if id == 0 {
        return "", ErrNotFound
    }
    return "found", nil
}

if _, err := lookup(0); err == ErrNotFound {
    fmt.Println("nothing matched")
}
```

These package-level errors are called **sentinel errors**. Convention:
name them `ErrXxx`.

## Wrapping errors with context: `fmt.Errorf` + `%w`

When an error bubbles up through several layers, each layer usually
wants to add context. The `%w` verb in `fmt.Errorf` builds a
**wrapped** error that preserves the original underneath the new
message.

```go
func loadConfig(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("loadConfig %q: %w", path, err)
    }
    _ = data
    return nil
}
```

The string you get out reads naturally:

```
loadConfig "missing.toml": open missing.toml: no such file or directory
```

But more importantly, the underlying error is still **inspectable**
through `errors.Is` and `errors.As`.

Use `%w` exactly once per `fmt.Errorf` call. To embed an error message
*without* wrapping (rare), use `%s` or `%v`.

## Inspecting wrapped errors: `errors.Is` and `errors.As`

### `errors.Is(err, target) bool`

Walks the wrap chain looking for a value equal to `target`. Use this
to compare against a sentinel.

```go
_, err := os.Open("missing.txt")
if errors.Is(err, fs.ErrNotExist) {
    fmt.Println("file does not exist")
}
```

The check survives any number of `fmt.Errorf("...: %w", err)` wraps.
Prefer `errors.Is(err, ErrFoo)` over `err == ErrFoo` — it works the
same when nothing has wrapped, and keeps working when something does.

### `errors.As(err, &target) bool`

Walks the wrap chain looking for an error of a specific concrete
type, and on success copies that error into `target`. Use this when
you need to read fields off the underlying error.

```go
_, err := os.Open("missing.txt")
var pathErr *fs.PathError
if errors.As(err, &pathErr) {
    fmt.Println("failed at path:", pathErr.Path)
    fmt.Println("operation was:", pathErr.Op)
}
```

You always pass a **pointer** to the target variable.

## Joining multiple errors: `errors.Join`

Sometimes a function performs several independent steps and you want
to report every failure, not just the first. `errors.Join` bundles
multiple errors into one; `errors.Is`/`As` then traverse all of them.

```go
err1 := errors.New("disk full")
err2 := errors.New("network down")

both := errors.Join(err1, err2)
fmt.Println(both)
// disk full
// network down

fmt.Println(errors.Is(both, err1))   // true
fmt.Println(errors.Is(both, err2))   // true
```

`nil` arguments are skipped. If all are `nil`, the result is `nil`.

## The cardinal rules

1. **Check every error.** A discarded `err` is almost always a bug.
   Use `_` only when you have a deliberate reason and a comment.
2. **Wrap with context.** When you return an error you didn't
   originate, add what *you* know (which file, which user, which
   operation).
3. **Prefer `errors.Is` to `==`.** It works equivalently in the
   unwrapped case and protects you when wrapping is added later.
4. **Don't put error checks inside structures that hide them.** No
   `try/except`-style middleware. The `if err != nil { return err }`
   block is the idiom — repetition is fine; it makes failure paths
   obvious.

## What `error` is *not*

- Not an exception. There's no implicit propagation; you return it.
- Not a sum type / `Result<T, E>`. The two return values are
  independent — by convention, if `err != nil`, ignore the other
  value; if `err == nil`, trust it.
- Not the right tool for "the program reached an impossible state."
  That's what `panic` is for ([panic and recover](15-panic-and-recover.md)).

## Sources

- [Errors are values — go.dev/blog/errors-are-values](https://go.dev/blog/errors-are-values)
- [Working with errors in Go 1.13 — go.dev/blog/go1.13-errors](https://go.dev/blog/go1.13-errors)
- [`errors` package reference — pkg.go.dev/errors](https://pkg.go.dev/errors)
- [`fmt.Errorf` and `%w` — pkg.go.dev/fmt#Errorf](https://pkg.go.dev/fmt#Errorf)
