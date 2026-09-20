# Custom error types

`error` is an interface with exactly one method. Anything that has an
`Error() string` method is an error, which means you can define your own
error types and put whatever fields you like on them:

```go
type ValidationError struct {
    Field string
    Msg   string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Msg)
}

var err error = &ValidationError{Field: "email", Msg: "must contain @"}
fmt.Println(err)   // output: email: must contain @
```

No registration, no base class. The [errors article](../02-language-basics/07-errors.md)
promised this once methods and interfaces were in place — here it is.

## Carrying data, not just text

A sentinel like `errors.New("invalid field")` is a fixed string. The
moment the caller needs to *do* something with the detail — highlight one
form field, retry only on a 503, report the row number — a string forces
them to parse it back out. A type hands it over directly:

```go
func checkAge(s string) error {
    return &ValidationError{Field: "age", Msg: "not a number"}
}
```

The caller gets the fields, not a sentence.

## `errors.As` gets your type back

`errors.As` walks the error chain looking for something assignable to the
target, and assigns it if found:

```go
err := checkAge("x")

var ve *ValidationError
if errors.As(err, &ve) {
    fmt.Println(ve.Field, "|", ve.Msg)   // output: age | not a number
}
```

The second argument is a **pointer to** a variable of your error type —
`&ve`, where `ve` is already a `*ValidationError`. That double indirection
is how `As` writes the result back.

## `Unwrap` puts your type in the chain

An error type that wraps another error should expose it through an
`Unwrap() error` method. That single method is what lets `errors.Is` and
`errors.As` see through your type to what is underneath:

```go
var ErrPermission = errors.New("permission denied")

type QueryError struct {
    Query string
    Err   error
}

func (e *QueryError) Error() string { return e.Query + ": " + e.Err.Error() }
func (e *QueryError) Unwrap() error { return e.Err }

qe := &QueryError{Query: "SELECT 1", Err: ErrPermission}
fmt.Println(qe)                            // output: SELECT 1: permission denied
fmt.Println(errors.Is(qe, ErrPermission))  // output: true
```

Your type composes with `fmt.Errorf` wrapping too. A chain can be part
sentinel, part custom type, and both lookups still work from the outside:

```go
wrapped := fmt.Errorf("loading user: %w", qe)
fmt.Println(wrapped)
// output: loading user: SELECT 1: permission denied

fmt.Println(errors.Is(wrapped, ErrPermission))   // output: true

var q *QueryError
fmt.Println(errors.As(wrapped, &q), q.Query)     // output: true SELECT 1
```

## Pointer or value receiver — pick pointer

Define `Error()` on the pointer receiver and always return `&T{...}`. Two
reasons, and the second one bites hard.

Method sets: with `func (e *ValidationError) Error() string`, only
`*ValidationError` satisfies `error` — a plain `ValidationError` does not.
Mix the two and `errors.As` does not merely miss, it panics:

```go
var ve ValidationError          // value, not pointer
errors.As(err, &ve)
// panic: errors: *target must be interface or implement error
```

Identity: two separate `&T{}` values are different pointers, so
comparisons stay predictable. Two equal struct *values* would compare
equal, which quietly turns unrelated errors into the same error.

## A custom `Is` method

By default `errors.Is` compares with `==`, so two distinct pointers never
match even when their contents are identical. If comparing by value is
what you want, say so with an `Is` method:

```go
type HTTPError struct{ Code int }

func (e *HTTPError) Error() string { return fmt.Sprintf("http %d", e.Code) }

func (e *HTTPError) Is(target error) bool {
    t, ok := target.(*HTTPError)
    return ok && t.Code == e.Code
}

fmt.Println(errors.Is(&HTTPError{404}, &HTTPError{404}))   // output: true
fmt.Println(errors.Is(&HTTPError{500}, &HTTPError{404}))   // output: false
```

`errors.Is` calls your `Is` at each step of the chain, so this works
through wrapping as well.

## The typed-nil trap

This is the [typed-nil gotcha](02-interfaces.md) wearing an error costume,
and it is the most common way to ship a bug with a custom error type:

```go
func find() error {
    var e *ValidationError   // nil pointer
    return e                 // ...but not a nil error
}

fmt.Println(find() == nil)   // output: false
```

The returned interface holds the type `*ValidationError` and the value
`nil`, and an interface is only `nil` when *both* halves are. Declare the
result variable as `error`, not as your concrete type, and return a
literal `nil` on the success path.

## Sentinel error or error type?

| Use | When |
|---|---|
| sentinel — `var ErrX = errors.New(...)` | the caller only needs to know *which* failure it was |
| error type | the caller needs data from the failure |
| type with `Unwrap` | you are adding context around someone else's error |
| type with `Is` | two instances should count as equal by value |

Start with a sentinel. Reach for a type when a caller would otherwise
have to read the message.

> **From Python:** this is `class ValidationError(Exception)` with fields
> on it, and `errors.As` is the `except ValidationError as e:` that binds
> it to a name. The differences are that nothing unwinds the stack for
> you, and that the chain is explicit — `Unwrap` is `__cause__`, but you
> have to write it.

## Quick reference

| Form | Meaning |
|---|---|
| `func (e *T) Error() string` | makes `*T` an `error` |
| `func (e *T) Unwrap() error` | exposes the wrapped error to `Is`/`As` |
| `func (e *T) Is(target error) bool` | custom equality for `errors.Is` |
| `var t *T; errors.As(err, &t)` | pull your type out of the chain |
| `return &T{...}` | always a pointer; never return a nil `*T` as `error` |

## Sources

- [`errors` package reference — pkg.go.dev/errors](https://pkg.go.dev/errors)
- [`errors.As` — pkg.go.dev/errors#As](https://pkg.go.dev/errors#As)
- [Go blog: working with errors in Go 1.13 — go.dev/blog/go1.13-errors](https://go.dev/blog/go1.13-errors)
- [Errors are values — go.dev/blog/errors-are-values](https://go.dev/blog/errors-are-values)
- [Method sets — go.dev/ref/spec#Method_sets](https://go.dev/ref/spec#Method_sets)
- [Why is my nil error value not equal to nil? — go.dev/doc/faq#nil_error](https://go.dev/doc/faq#nil_error)
