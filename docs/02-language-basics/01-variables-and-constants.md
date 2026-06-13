# Variables and constants

## `var` — the long form

`var` declares one or more variables. Four shapes are supported.

```go
// 1. Type with initializer.
var name string = "Ada"
var u, v, w float64 = -1.0, -2.0, -3.0

// 2. Type without initializer — variable takes its zero value.
var counter int          // counter == 0
var greeting string      // greeting == ""
var active bool          // active  == false

// 3. Initializer without explicit type — type is inferred.
var k = 0                // k is int
var pi = 3.14            // pi is float64

// 4. Grouped declarations (common at package level).
var (
    host     = "localhost"
    port     = 8080
    verbose  bool
)
```

### Zero values

From the Go spec:

> "Otherwise, each variable is initialized to its **zero value**."

There is no "uninitialized" state in Go — every variable always has a value.

| Type | Zero value |
|---|---|
| numeric (`int`, `float64`, …) | `0` |
| `bool` | `false` |
| `string` | `""` |
| pointer, slice, map, channel, function, interface | `nil` |
| struct | a struct whose fields each hold their own zero value |

```go
var s []int           // s == nil, len(s) == 0
var m map[string]int  // m == nil — reading is fine, writing panics
var p *int            // p == nil
```

## `:=` — short declaration

Inside a function, `:=` declares and assigns in one go. The type is always inferred.

```go
func main() {
    name := "Ada"          // string
    count := 0             // int
    ratio := 0.5           // float64
    items := []string{"a", "b"}

    fmt.Println(name, count, ratio, items)
}
```

Two rules to remember:

1. **Function-scope only.** `:=` is illegal at package level — there you must use `var`.
2. **Redeclaration** is allowed if at least one name on the left is new and the reused names keep the same type:
   ```go
   field1, offset := nextField(s, 0)
   field2, offset := nextField(s, offset)  // ok: offset redeclared, field2 new

   x, y := 1, 2
   x, y := 3, 4                            // compile error: no new variables
   ```

> **From Python:** Python lets you assign anywhere without ceremony. Go has *two* ways (`var` and `:=`) because one is the only one allowed at package level and the other carries useful type-inference + multi-return ergonomics inside functions.

## Multiple assignment and the blank identifier `_`

Like Python tuple unpacking, but for function returns:

```go
n, err := strconv.Atoi("42")
if err != nil {
    return err
}
fmt.Println(n)

_, err = io.Copy(dst, src)   // ignore the byte count, keep the error
```

`_` is the **blank identifier** — a write-only slot that lets you discard a value you don't need.

## Shadowing pitfall

A new `:=` in an inner scope creates a *new* variable that shadows the outer one. Read carefully:

```go
err := doFirst()
if err != nil { return err }

if cond {
    err := doSecond()        // !!! NEW err, shadows the outer one
    log.Println(err)
}
// outer err here is still the one from doFirst, NOT from doSecond.
```

This is one of the most common Go bugs. `go vet`'s shadow analyzer and `golangci-lint` will warn you.

## `const` — compile-time constants

Constants are evaluated at compile time. They can be typed or untyped.

```go
const Pi float64 = 3.14159     // typed
const Greeting = "hello"       // untyped string

const (
    KB = 1024
    MB = 1024 * KB
    GB = 1024 * MB
)
```

A constant cannot be assigned to. You also cannot take its address (`&Pi` is illegal).

**Untyped constants** are flexible — they take on the type they need at use:

```go
const limit = 10
var i int     = limit          // limit used as int
var f float64 = limit          // limit used as float64
var s string  = "small"
const small   = "small"
fmt.Println(s == small)        // works — small fits string
```

If `limit` were declared as `const limit int = 10`, the second line would fail to compile — `int` is not assignable to `float64` without explicit conversion.

## `iota` — auto-incrementing constants

`iota` is a predeclared identifier that resets to 0 at the start of each `const` block and increments by 1 with each `ConstSpec`.

```go
const (
    Sunday = iota   // 0
    Monday          // 1
    Tuesday         // 2
    Wednesday       // 3
    Thursday        // 4
    Friday          // 5
    Saturday        // 6
)
```

The expression on each line is implicitly repeated from the line above. You can use it with bit shifts for flag enums:

```go
const (
    ReadPerm    = 1 << iota  // 1   (iota == 0)
    WritePerm                // 2   (iota == 1)
    ExecutePerm              // 4   (iota == 2)
)

perms := ReadPerm | WritePerm  // == 3
```

`iota` only exists inside `const` blocks — there is no general-purpose counter syntax.

## Sources

- [Variable declarations — go.dev/ref/spec#Variable_declarations](https://go.dev/ref/spec#Variable_declarations)
- [Short variable declarations — go.dev/ref/spec#Short_variable_declarations](https://go.dev/ref/spec#Short_variable_declarations)
- [Constant declarations — go.dev/ref/spec#Constant_declarations](https://go.dev/ref/spec#Constant_declarations)
- [Iota — go.dev/ref/spec#Iota](https://go.dev/ref/spec#Iota)
- [The zero value — go.dev/ref/spec#The_zero_value](https://go.dev/ref/spec#The_zero_value)
