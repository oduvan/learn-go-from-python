# Type conversions

## The strict rule

> "Explicit conversions are required when different numeric types are mixed in an expression or assignment."
> — Go specification

There are **no implicit numeric conversions** in Go. Even between types of the same size, you must convert.

```go
var i int   = 42
var f float64 = i               // compile error: cannot use i (int) as float64
var f float64 = float64(i)      // ok

var a int32 = 1
var b int64 = a                 // compile error: cannot use a (int32) as int64
var b int64 = int64(a)          // ok
```

> **From Python:** Python freely mixes `int` and `float` in arithmetic. Go refuses — you must spell out the conversion every time. Once you accept it, you'll find Go programs have fewer "wait, where did this float come from?" bugs.

## Conversion syntax: `T(x)`

```go
var i int       = 42
var f float64   = float64(i)        // int → float64
var u uint32    = uint32(f)         // float64 → uint32 (truncates toward zero)
var b byte      = byte(i)           // int → uint8 (truncates upper bits)
```

Converting a float to int discards the fractional part. The float must
be a *variable*, not a bare literal — `int(3.9)` written with a literal
is a compile error, because Go refuses to silently drop the `.9` from
an untyped constant. Once the value lives in a `float64`, the
conversion is fine:

```go
f := 3.9
fmt.Println(int(f))     // 3
fmt.Println(int(-f))    // -3 — toward zero, not toward -infinity
```

Converting a larger integer to a smaller one drops the high-order bits — the result wraps:

```go
var big int32 = 257
var small int8 = int8(big)
fmt.Println(small)       // 1   (257 mod 256)
```

## Strings, bytes, and runes

These three conversions are common enough to memorize.

### `string ↔ []byte` — raw UTF-8 bytes

```go
s := "hello"
b := []byte(s)              // [104 101 108 108 111]
s2 := string(b)             // "hello"
```

### `string ↔ []rune` — Unicode code points

```go
s := "日本語"
r := []rune(s)              // [26085 26412 35486]
s2 := string(r)             // "日本語"

fmt.Println(len(s))         // 9 — bytes
fmt.Println(len(r))         // 3 — runes
```

### Single rune to string

```go
s := string('A')            // "A"
s := string(rune(0x4e2d))   // "中"
```

A subtle one: `string(65)` also gives `"A"` but `go vet` will flag it. Always cast to `rune` first to make intent clear.

## Strings ↔ numbers — use `strconv`, not conversion

The `T(x)` syntax does **not** parse a string into a number. For that, the standard library has `strconv`.

```go
import "strconv"

// string → number
n, err := strconv.Atoi("42")
if err != nil { /* ... */ }                 // n == 42

f, err := strconv.ParseFloat("3.14", 64)    // f == 3.14
b, err := strconv.ParseBool("true")         // b == true

// number → string
s := strconv.Itoa(42)                       // "42"
s := strconv.FormatFloat(3.14, 'f', 2, 64)  // "3.14"
s := strconv.FormatInt(255, 16)             // "ff"
```

`fmt.Sprintf` is the swiss-army alternative — slower, but flexible:

```go
s := fmt.Sprintf("%d items, %.2f each", 3, 9.5)
// "3 items, 9.50 each"
```

> **From Python:** Python's `int("42")` and `str(42)` are global builtins. Go puts them in `strconv` to make the cost (and the possibility of error) explicit. `strconv.Atoi` returns `(int, error)` — there are no exceptions to catch.

## Untyped constants are the one exception

This is the single biggest gotcha for a Python developer learning Go,
and it sneaks into several places later in the curriculum, so it's
worth spending a moment on.

### Untyped constants are typeless until used

A literal like `42`, `3.14`, or `"hi"` written in source code is an
*untyped constant*. It has no fixed Go type — it carries an *idealised*
numeric value (arbitrary precision for numerics). It picks up a type
only when the context demands one.

```go
var x float64 = 42        // ok — 42 is an untyped int constant, fits float64
var y int     = 0.0       // ok — 0.0 is representable as int (no fractional part)
var z int     = 3.14      // compile error: 3.14 (untyped float) truncated to int
```

The third line is the rule that bit you above with `int(3.9)` — Go
refuses to silently drop the fractional part of a constant.

### Constant arithmetic is also arbitrary-precision

A consequence that surprises Python developers: when you do arithmetic
on *constants*, Go does it at compile time with arbitrary precision,
not with IEEE 754 floats. The classic float-equality trap doesn't fire:

```go
fmt.Println(0.1 + 0.2 == 0.3)                       // true  (constants)

var a, b, c float64 = 0.1, 0.2, 0.3
fmt.Println(a + b == c)                             // false (float64 variables)
```

Same numbers, completely different answers, depending on whether the
expression survives until runtime or gets folded at compile time.

### Once a constant has a type, the rule snaps back

```go
const limit int = 10
var f float64 = limit               // compile error
var f float64 = float64(limit)      // ok
```

This is why most package-level constants are left **untyped** — they
adapt to whatever type their caller needs. Mark them with an explicit
type only when you want to constrain them.

> **From Python:** Python has no real analog. `42` is always `int`,
> `3.14` is always `float`, and operations follow runtime semantics.
> Go's "constant is a value, not a typed thing" model is genuinely
> different — and it's the source of several of the subtle behaviour
> differences you'll encounter (literal-vs-variable comparison
> results, what compiles and what doesn't, default types in `:=`).

## `T(x)` also works between custom types

The same `T(x)` syntax works for any two types that share an underlying
type — not just predeclared numerics. See
[09-custom-types.md](09-custom-types.md) for the full story; in short:

```go
type Celsius float64
type Fahrenheit float64

c := Celsius(20.0)
f := Fahrenheit(c)            // ok — same underlying type
var raw float64 = float64(c)  // ok — strip the named type
```

Interface conversions (type assertions) use different syntax — covered in a later topic.

## Quick reference

| What | Syntax | Notes |
|---|---|---|
| Numeric between types | `T(x)` | Always explicit. Truncates floats; wraps on narrowing. |
| `string` → bytes | `[]byte(s)` | UTF-8 bytes, immutable copy. |
| bytes → `string` | `string(b)` | Copies. |
| `string` → runes | `[]rune(s)` | Decodes UTF-8 into code points. |
| runes → `string` | `string(r)` | Re-encodes as UTF-8. |
| `string` → number | `strconv.Atoi`, `strconv.ParseInt`, `strconv.ParseFloat`, ... | Returns `(value, error)`. |
| number → `string` | `strconv.Itoa`, `strconv.FormatInt`, `strconv.FormatFloat`, or `fmt.Sprintf` | Pick by speed/flexibility. |
| Single rune → string | `string(rune(0x4e2d))` | Always cast to `rune` first. |

## Sources

- [Conversions — go.dev/ref/spec#Conversions](https://go.dev/ref/spec#Conversions)
- [Assignability — go.dev/ref/spec#Assignability](https://go.dev/ref/spec#Assignability)
- [`strconv` package — pkg.go.dev/strconv](https://pkg.go.dev/strconv)
- [`fmt` package — pkg.go.dev/fmt](https://pkg.go.dev/fmt)
- [Strings, bytes, runes and characters in Go — go.dev/blog/strings](https://go.dev/blog/strings)
