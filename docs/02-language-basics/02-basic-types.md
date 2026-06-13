# Basic (predeclared) types

Go ships a fixed, small set of built-in types. There are **no other primitive types** — anything else is built from these.

## Integers

### Fixed-size

```go
var a int8   = -128                          // -128 .. 127
var b int16  = 32_000                        // _ allowed as digit separator
var c int32  = 2_000_000_000                 // up to ~2.1×10⁹
var d int64  = 9_000_000_000_000_000_000     // up to ~9.2×10¹⁸

var u uint8  = 255                           // 0 .. 255
var v uint16 = 65_535
var w uint32 = 4_000_000_000                 // up to ~4.3×10⁹
var x uint64 = 18_000_000_000_000_000_000    // up to ~1.8×10¹⁹
```

### Architecture-dependent

```go
var n int        // 32 or 64 bits, matching the platform
var u uint       // same as int, but unsigned
var p uintptr    // big enough to hold a raw pointer value
```

`int` is **not** an alias for `int64`. Even on a 64-bit machine where they happen to have the same size, they are *distinct types* — see [03-type-conversions.md](03-type-conversions.md).

### Aliases

Two integer aliases exist:

```go
byte == uint8        // byte for raw byte data
rune == int32        // rune for Unicode code points
```

Use `byte` when working with binary data (`[]byte`) and `rune` when working with characters in a string.

## Floating-point

```go
var f32 float32 = 1.5e3
var f64 float64 = 3.14159265358979
```

Both are IEEE 754. `float64` is the default for untyped float constants and what you'll use 99% of the time. Don't use `float32` unless you have a measured reason (memory layout, GPU buffers, etc.).

There are **no decimal types** in the standard library — if you need exact decimal arithmetic (money), reach for `github.com/shopspring/decimal` or similar.

## Complex numbers

```go
var z complex128 = 1 + 2i
fmt.Println(real(z), imag(z))    // 1 2
```

Rarely needed — most code can ignore `complex64` / `complex128`.

## Boolean

```go
var ok bool = true
var done bool          // false (zero value)
```

`bool` does **not** convert to int. You cannot write `if 1 { ... }` or `if someInt { ... }` — it's a compile error. The condition in `if`/`for` *must* be a `bool`.

```go
var n int = 1
if n {                  // compile error: non-bool n (type int) used as if condition
    fmt.Println("nope")
}

if n != 0 {             // ok
    fmt.Println("yep")
}
```

> **From Python:** no truthy/falsy. Empty strings, empty slices, zero, and `nil` are not implicitly false. You always write an explicit comparison.

## String

```go
var greeting string = "hello, 世界"
fmt.Println(len(greeting))         // 16 — number of BYTES, not characters!
fmt.Println(greeting[0])           // 104 — the byte 'h'
```

Two non-obvious facts:

1. **Strings are immutable.** You cannot do `s[0] = 'H'` — compile error.
2. **`len(s)` is bytes, not characters.** Since Go strings are UTF-8 encoded, a single character like `世` may be 3 bytes. To count characters (runes):

```go
import "unicode/utf8"

s := "世界"
fmt.Println(len(s))                       // 6 (bytes)
fmt.Println(utf8.RuneCountInString(s))    // 2 (runes)
```

To iterate by rune, use `range`:

```go
for i, r := range "hi世" {
    fmt.Printf("byte index %d, rune %c (%d)\n", i, r, r)
}
// output:
// byte index 0, rune h (104)
// byte index 1, rune i (105)
// byte index 2, rune 世 (19990)
```

> **From Python:** Python 3 strings are sequences of Unicode code points; `s[0]` of `"世"` gives you `"世"`. Go strings are byte sequences interpreted as UTF-8; `s[0]` of `"世"` gives you the first byte, not the first character.

## Untyped constants — a gentle introduction

Constants can be **untyped** until they're used.

```go
const small = 10        // untyped integer constant

var a int     = small   // ok — small becomes int
var b int64   = small   // ok — small becomes int64
var c float64 = small   // ok — small becomes float64
```

If you had written `const small int = 10`, only the first line would compile. Untyped constants are why you can write `x := 1.5` and get `float64` without ceremony.

Defaults if context forces a type:

| Constant form | Default type |
|---|---|
| integer (`42`, `0x2a`) | `int` |
| float (`3.14`, `1e10`) | `float64` |
| rune (`'A'`) | `rune` (i.e. `int32`) |
| string (`"hi"`) | `string` |
| boolean (`true`) | `bool` |
| complex (`2i`) | `complex128` |

## What's *not* a built-in type

Things a Python developer might expect that don't exist:

- **No `list` / `dict` / `set` keyword** — `slice`, `map`, no built-in set (use `map[T]struct{}`).
- **No `tuple`** — use multiple return values or a struct.
- **No `decimal`** — `float64` or a third-party module.
- **No `bigint` keyword** — use `math/big.Int`.
- **No `None`** — use `nil` (only valid for pointer, slice, map, channel, function, interface).

## Sources

- [Numeric types — go.dev/ref/spec#Numeric_types](https://go.dev/ref/spec#Numeric_types)
- [Boolean types — go.dev/ref/spec#Boolean_types](https://go.dev/ref/spec#Boolean_types)
- [String types — go.dev/ref/spec#String_types](https://go.dev/ref/spec#String_types)
- [Constants — go.dev/ref/spec#Constants](https://go.dev/ref/spec#Constants)
- [Strings, bytes, runes and characters in Go — go.dev/blog/strings](https://go.dev/blog/strings)
