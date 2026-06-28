# Operators and expressions

## The complete list

### Arithmetic

```go
+   -   *   /   %
```

### Comparison

```go
==  !=  <  <=  >  >=
```

Comparison always yields a `bool`.

### Logical (boolean)

```go
&&  ||  !
```

Short-circuit, like Python. Operands must be `bool` — no implicit truthy conversion.

### Bitwise

```go
&    // AND
|    // OR
^    // XOR (also unary NOT)
&^   // AND NOT — Go-specific, clears bits
<<   // left shift
>>   // right shift
```

### Compound assignment

```go
+=  -=  *=  /=  %=
&=  |=  ^=  &^=
<<= >>=
```

### Address-of / dereference

```go
&x   // pointer to x
*p   // value at p
```

## Integer overflow wraps

> Integer overflow uses two's complement arithmetic — no undefined behavior, no panic.

```go
var x uint8 = 255
x++
fmt.Println(x)   // 0  — wraps

var y int8 = 127
y++
fmt.Println(y)   // -128
```

If you need overflow detection, use the `math/bits` package's `Add64`/`Mul64`/etc., or check ranges manually before arithmetic.

> **From Python:** Python's `int` grows without bound. Go integers are fixed-width and wrap. For arbitrary precision you reach for `math/big.Int`.

## Integer division truncates toward zero

```go
fmt.Println(7 / 2)     // 3
fmt.Println(-7 / 2)    // -3 (toward zero, not toward -infinity)
fmt.Println(7 % 2)     // 1
fmt.Println(-7 % 2)    // -1
```

> **From Python:** Python 3's `/` always returns float (`7 / 2 == 3.5`); use `//` for integer floor division (`-7 // 2 == -4`). Go's `/` is integer division *when both operands are integers*, and truncates toward zero like C.

## No exponentiation operator

Go has no `**`. Use `math.Pow`:

```go
import "math"

fmt.Println(math.Pow(2, 10))    // 1024  — float64

// For integer powers of 2, use a shift:
fmt.Println(1 << 10)            // 1024
```

## No ternary operator

The community decided readability beats brevity. Use `if`/`else` or a helper:

```go
// Python: status := "even" if n%2 == 0 else "odd"
var status string
if n%2 == 0 {
    status = "even"
} else {
    status = "odd"
}
```

If you really want a one-liner, write a tiny generic helper. The
`[T any]` part is Go's generics syntax — a *type parameter* that
lets one function work with values of any type. Generics get their
own treatment later; the example below is just to show what such a
helper would look like.

```go
func If[T any](cond bool, a, b T) T {
    if cond { return a }
    return b
}

status := If(n%2 == 0, "even", "odd")
```

But most Go code just uses the four-line `if`/`else`. Don't fight the language.

## `++` and `--` are *statements*, not expressions

```go
i++             // ok — statement
j--             // ok — statement

x := i++        // compile error: i++ is not an expression
if i++ > 10 {}  // compile error
```

You can never use `i++` inside an expression. And there is no `++i` / `--i` prefix form.

## `&^` — AND NOT (bit clear)

Unique to Go. `a &^ b` is equivalent to `a & (^b)` — clears the bits in `a` that are set in `b`.

```go
const (
    Readable  = 1 << 0   // 0b001
    Writable  = 1 << 1   // 0b010
    Executable = 1 << 2  // 0b100
)

perms := Readable | Writable | Executable   // 0b111
perms = perms &^ Writable                   // 0b101 — write bit cleared
fmt.Printf("%03b\n", perms)                 // output: 101
```

## String concatenation

```go
s := "hello" + ", " + "world"        // works, but allocates per +
s := fmt.Sprintf("%s, %s", "hello", "world")

import "strings"
s := strings.Join([]string{"hello", "world"}, ", ")

// For many concatenations in a loop, use a Builder:
var b strings.Builder
for _, w := range words {
    b.WriteString(w)
    b.WriteString(" ")
}
result := b.String()
```

Concatenating with `+` in a tight loop is O(n²) — each `+` copies the whole prefix. `strings.Builder` is O(n).

> **From Python:** ≈ the `+` vs `''.join(...)` discussion. Same advice: use the builder for loops.

## Operator precedence

Five levels, lowest to highest:

```
||
&&
==  !=  <  <=  >  >=
+  -  |  ^
*  /  %  <<  >>  &  &^
```

Unary operators (`!`, `-`, `^`, `*`, `&`, `<-`) bind tighter than any binary operator.

When in doubt, **parenthesize**. Code-reviewers prefer explicit parens over relying on the table.

## Comparison gotchas

- **Floats** compare exactly, but with a twist. `0.1 + 0.2 == 0.3` written as bare literals is **`true`** — Go evaluates untyped constant expressions at compile time with arbitrary precision, so no IEEE 754 rounding ever happens. Once the same numbers live in `float64` variables, rounding kicks in and the answer flips:

  ```go
  fmt.Println(0.1 + 0.2 == 0.3)                                 // true  (untyped constants)
  var a, b, c float64 = 0.1, 0.2, 0.3
  fmt.Println(a + b == c)                                        // false (float64 variables)
  ```

  Use an epsilon comparison whenever you're comparing `float32`/`float64` variables.
- **Strings** compare byte-by-byte (lexicographic). `"a" < "b"` is `true`.
- **Slices, maps, functions** are *not* comparable with `==`/`!=` — only against `nil`. Use `reflect.DeepEqual` or a hand-written check.
  ```go
  a := []int{1, 2, 3}
  b := []int{1, 2, 3}
  fmt.Println(a == b)   // compile error
  ```
- **Structs** are comparable if all their fields are comparable. So a struct of ints/strings is fine; a struct containing a slice is not.

## Sources

- [Operators — go.dev/ref/spec#Operators](https://go.dev/ref/spec#Operators)
- [Arithmetic operators — go.dev/ref/spec#Arithmetic_operators](https://go.dev/ref/spec#Arithmetic_operators)
- [Comparison operators — go.dev/ref/spec#Comparison_operators](https://go.dev/ref/spec#Comparison_operators)
- [Integer overflow (two's complement) — go.dev/ref/spec#Integer_overflow](https://go.dev/ref/spec#Integer_overflow)
- [`strings.Builder` — pkg.go.dev/strings#Builder](https://pkg.go.dev/strings#Builder)
- [`math/bits` — pkg.go.dev/math/bits](https://pkg.go.dev/math/bits)
