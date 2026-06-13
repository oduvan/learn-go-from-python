# Control flow

Three keywords cover almost everything: `if`, `for`, `switch`.

## `if`

Curly braces are **required**. No parentheses around the condition.

```go
if x > 0 {
    fmt.Println("positive")
} else if x < 0 {
    fmt.Println("negative")
} else {
    fmt.Println("zero")
}
```

The condition **must be a `bool`** — no truthy/falsy. See [02-basic-types.md](02-basic-types.md).

### `if` with init statement

A frequent Go idiom: declare a variable scoped to the `if`/`else if`/`else` chain.

```go
if n, err := strconv.Atoi(s); err == nil {
    fmt.Println("parsed:", n)
} else {
    fmt.Println("bad input:", err)
}
// n and err are NOT visible here.
```

This keeps short-lived names out of the surrounding scope. It's the canonical way to handle errors from a single function call.

## `for` — the only loop

Go has no `while` and no `do…while`. The `for` keyword covers all of it.

### Three-clause form (classic C-style)

```go
for i := 0; i < 5; i++ {
    fmt.Println(i)
}
```

### Condition-only form (Go's `while`)

```go
n := 100
for n > 1 {
    n /= 2
}
```

### Infinite

```go
for {
    if shouldStop() {
        break
    }
}
```

### `range` form — iterate

`for ... range` iterates over slices, arrays, strings, maps, channels.

```go
nums := []int{10, 20, 30}

for i, v := range nums {
    fmt.Println(i, v)
}
// 0 10
// 1 20
// 2 30

for _, v := range nums {        // ignore index
    fmt.Println(v)
}

for i := range nums {           // ignore value
    fmt.Println(i)
}

for range nums {                // just count iterations
    fmt.Println("tick")
}
```

Maps give you `(key, value)` — but **order is randomized** on each iteration:

```go
m := map[string]int{"a": 1, "b": 2, "c": 3}
for k, v := range m {
    fmt.Println(k, v)        // order may differ run to run
}
```

Strings give you `(byteIndex, rune)`:

```go
for i, r := range "hi世" {
    fmt.Printf("%d %c\n", i, r)
}
// 0 h
// 1 i
// 2 世
```

### Range over integers

```go
for i := range 5 {
    fmt.Println(i)       // 0 1 2 3 4
}
```

### Range over a function

Iterator functions enable custom iteration:

```go
// pkg func Lines returns an iter.Seq[string]
for line := range bufio.NewScanner(r).Lines() {
    fmt.Println(line)
}
```

> **From Python:** `for ... range` covers `for x in seq:`, `for i, x in enumerate(seq):`, `for k, v in dict.items():`. The integer form `for i := range 5` matches `for i in range(5):`.

### `break` and `continue`

```go
for i, v := range data {
    if v < 0 {
        continue            // skip this iteration
    }
    if v > 1000 {
        break               // exit the loop
    }
    process(i, v)
}
```

### Labels — escape nested loops

```go
Outer:
for i := 0; i < 10; i++ {
    for j := 0; j < 10; j++ {
        if data[i][j] == target {
            fmt.Println("found at", i, j)
            break Outer        // breaks both loops
        }
    }
}
```

`continue Outer` jumps to the next iteration of the outer loop.

## `switch`

### Expression switch

```go
switch x {
case 1:
    fmt.Println("one")
case 2, 3:                  // multiple values per case
    fmt.Println("two or three")
case 4:
    fmt.Println("four")
default:
    fmt.Println("other")
}
```

**No implicit fallthrough.** Each `case` ends with an implicit `break`. If you actually want C-style fallthrough, write it:

```go
switch x {
case 1:
    fmt.Println("one")
    fallthrough
case 2:
    fmt.Println("one or two")
}
```

### "Tagless" switch — replaces `if/else if` chains

```go
switch {
case x < 0:
    fmt.Println("negative")
case x == 0:
    fmt.Println("zero")
case x > 0:
    fmt.Println("positive")
}
```

Idiomatic in Go — preferred over long `if/else if/else if` chains.

### `switch` with init

Same pattern as `if`:

```go
switch n := len(s); {
case n == 0:
    fmt.Println("empty")
case n < 10:
    fmt.Println("short")
default:
    fmt.Println("long")
}
```

### Type switch

For interface values — covered properly in a later topic, but worth seeing now:

```go
func describe(x any) {
    switch v := x.(type) {
    case int:
        fmt.Println("int:", v)
    case string:
        fmt.Println("string of length", len(v))
    case nil:
        fmt.Println("nil")
    default:
        fmt.Printf("unknown type %T\n", v)
    }
}
```

## `goto`

Exists, has the usual restrictions (can't jump over variable declarations, can't jump into a block). Almost never used outside generated code or very specific state-machine implementations. Don't reach for it casually.

## What's missing from Python

| Python | Go equivalent |
|---|---|
| `while cond:` | `for cond { ... }` |
| `else` on a `for` loop | no equivalent — restructure |
| `match`/`case` (PEP 634) | `switch` (similar but simpler) |
| `try`/`except`/`finally` | no exceptions — see error handling later; `defer` covers `finally`-style cleanup |
| comprehensions | none — write a `for` loop |
| `pass` | not needed — `{}` block is fine |

## Sources

- [If statements — go.dev/ref/spec#If_statements](https://go.dev/ref/spec#If_statements)
- [For statements — go.dev/ref/spec#For_statements](https://go.dev/ref/spec#For_statements)
- [Switch statements — go.dev/ref/spec#Switch_statements](https://go.dev/ref/spec#Switch_statements)
- [Break/continue/labeled statements — go.dev/ref/spec#Break_statements](https://go.dev/ref/spec#Break_statements)
- [Range over integers (Go 1.22 release notes) — go.dev/doc/go1.22#language](https://go.dev/doc/go1.22#language)
- [Range over functions (Go 1.23 release notes) — go.dev/doc/go1.23#language](https://go.dev/doc/go1.23#language)
