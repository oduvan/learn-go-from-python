# Type assertions and type switches

An interface value hides the concrete type inside it. When you need that
concrete type back — to call a method the interface doesn't expose, or to
branch on what's really stored — Go gives you two tools: the **type
assertion** and the **type switch**.

## Type assertions

A type assertion `x.(T)` claims that the interface value `x` holds a value
of type `T`, and extracts it. It comes in two forms.

The **single-result** form returns the value, and **panics** if the
dynamic type isn't `T`:

```go
var x any = "hello"
s := x.(string)
fmt.Println(s)        // output: hello

n := x.(int)          // panic: interface conversion: interface {} is string, not int
```

The **comma-ok** form never panics — it returns the value plus a boolean
reporting whether the assertion held:

```go
var x any = "hello"

s, ok := x.(string)
fmt.Println(s, ok)    // output: hello true

n, ok := x.(int)
fmt.Println(n, ok)    // output: 0 false   — n is the zero value
```

Prefer comma-ok unless you are certain of the type; the panicking form is
for cases where a wrong type is a genuine bug.

## Asserting to an interface type

`T` doesn't have to be a concrete type — it can be **another interface**.
Then the assertion asks "does the stored value also satisfy *this*
interface?" This is how you probe for an optional capability.

```go
var w any = bytes.NewBufferString("hi")

if s, ok := w.(fmt.Stringer); ok {
    fmt.Println(s.String())   // output: hi
}
```

Here `w` holds a `*bytes.Buffer`; the assertion succeeds because that type
has a `String() string` method, so it satisfies `fmt.Stringer`.

## Type switches

When you want to branch across several possible types, a chain of
assertions is clumsy. A **type switch** does it in one construct: the
special form `x.(type)` (legal only inside a `switch`) tests the dynamic
type, and `v := x.(type)` binds `v` to the value with the matching type in
each case.

```go
func describe(x any) string {
    switch v := x.(type) {
    case nil:
        return "nil"
    case int:
        return fmt.Sprintf("int: %d", v)        // v is an int here
    case string:
        return fmt.Sprintf("string of len %d", len(v))   // v is a string here
    default:
        return fmt.Sprintf("other: %T", v)      // v keeps its original type
    }
}

fmt.Println(describe(42))      // output: int: 42
fmt.Println(describe("hi"))    // output: string of len 2
fmt.Println(describe(nil))     // output: nil
fmt.Println(describe(3.14))    // output: other: float64
```

A few rules worth knowing:

- A `case nil` matches a nil interface value.
- In a case listing **one** type, `v` has that concrete type. In a case
  listing **multiple** types (`case int, int64:`) or in `default`, `v`
  keeps the original interface type.
- The cases are tested top to bottom; the first match wins.

## A real use: inspecting error types

A common place this shows up is examining an error's concrete type. A type
switch reads cleanly:

```go
switch e := err.(type) {
case *os.PathError:
    fmt.Println("path problem:", e.Path)
case nil:
    fmt.Println("no error")
default:
    fmt.Println("some error:", e)
}
```

For *wrapped* errors, the standard library's `errors.As` is preferred over
a bare assertion because it unwraps the chain. It takes a pointer to a
variable of the target type and fills it in if any error in the chain
matches:

```go
_, err := os.Open("/nope/nope")

var pe *os.PathError
if errors.As(err, &pe) {
    fmt.Println("path:", pe.Path)   // output: path: /nope/nope
}
```

The mechanism underneath is the same idea as a type assertion — `errors.As`
just walks the wrapped chain for you.

> **From Python:** a type switch is the idiomatic stand-in for an
> `isinstance(x, T)` ladder, and comma-ok assertions play the role of a
> guarded `isinstance` check before using a value as a specific type.

## Quick reference

| Form | Result |
|---|---|
| `v := x.(T)` | extract `T`; **panics** on mismatch |
| `v, ok := x.(T)` | extract `T`; `ok` is false (and `v` zero) on mismatch |
| `x.(SomeInterface)` | succeeds if the dynamic type satisfies that interface |
| `switch v := x.(type) { ... }` | branch on the dynamic type |
| `case nil:` | matches a nil interface value |
| multi-type case / `default` | `v` keeps the interface type |

## Sources

- [Type assertions — go.dev/ref/spec#Type_assertions](https://go.dev/ref/spec#Type_assertions)
- [Type switches — go.dev/ref/spec#Type_switches](https://go.dev/ref/spec#Type_switches)
- [errors.As — pkg.go.dev/errors#As](https://pkg.go.dev/errors#As)
- [Effective Go: interface conversions and type assertions — go.dev/doc/effective_go#interface_conversions](https://go.dev/doc/effective_go#interface_conversions)
