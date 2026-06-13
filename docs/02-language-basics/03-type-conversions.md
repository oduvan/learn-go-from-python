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

Converting float to int discards the fractional part:

```go
fmt.Println(int(3.9))    // 3
fmt.Println(int(-3.9))   // -3 — toward zero, not toward -infinity
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

An *untyped* constant doesn't need conversion as long as the target type can represent it:

```go
var x float64 = 42        // ok — 42 is an untyped int constant, fits float64
var y int     = 0.0       // ok — 0.0 is representable as int (no fractional part)
var z int     = 3.14      // compile error: 3.14 (untyped float) truncated to int
```

But once a constant has a type, the rule snaps back:

```go
const limit int = 10
var f float64 = limit               // compile error
var f float64 = float64(limit)      // ok
```

This is why most package-level constants are left untyped — they're maximally reusable.

## `T(x)` also works between custom types

The same `T(x)` syntax works for any two types that share an underlying
type — not just predeclared numerics. See
[07-custom-types.md](07-custom-types.md) for the full story; in short:

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
