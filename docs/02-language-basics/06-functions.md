# Functions

A function in Go is a named (or anonymous) block of code with typed
parameters and zero or more typed return values. The keyword is
`func`. You've already seen `func main()` as the program entrypoint;
this article covers every other shape a function can take.

## The basic shape

```go
func add(x int, y int) int {
    return x + y
}

fmt.Println(add(2, 3))     // 5
```

Three things to notice:

1. The `func` keyword starts the declaration.
2. Each parameter is written `name type` — the **type comes after the
   name**, opposite to C/Java.
3. The return type goes **after** the parameter list, with no `:` or
   `->` arrow.

When several parameters share a type, you can list them once at the end:

```go
func add(x, y int) int {        // same as (x int, y int)
    return x + y
}
```

## No return value

Omit the return type entirely. The function returns when execution
falls off the end (or hits an early `return`).

```go
func greet(name string) {
    fmt.Println("Hello,", name)
}
```

## Multiple return values

A function can return **more than one value**. The return types go in
parentheses.

```go
func divmod(a, b int) (int, int) {
    return a / b, a % b
}

q, r := divmod(7, 3)
fmt.Println(q, r)               // 2 1
```

This is the foundation of Go's error-handling convention — the
canonical signature for a fallible operation is `(result, error)`:

```go
func parseAge(s string) (int, error) {
    n, err := strconv.Atoi(s)
    if err != nil {
        return 0, err
    }
    return n, nil
}
```

The `error` type is covered in its own article ([07-errors.md](07-errors.md));
for now, treat it as "Go's standard way to signal something went wrong."

> **From Python:** Python supports multiple returns through tuple
> unpacking (`return a, b`). Go's multi-return is a first-class part
> of the type system — the function's signature *declares* it returns
> two values, the compiler checks every caller, and you can't
> accidentally lose the second one.

## Named return values

You can name the return values in the signature. They behave like
pre-declared variables initialized to their zero values, and a bare
`return` (with no expression) returns whatever they currently hold.

```go
func split(sum int) (x, y int) {
    x = sum * 4 / 9
    y = sum - x
    return                      // returns x, y
}

fmt.Println(split(17))          // 7 10
```

Use named returns sparingly. They're most useful when:

- The signature otherwise has two values of the same type and the
  names clarify intent (`(width, height int)`).
- The function body is short enough that the implicit assignment is
  obvious.

For longer functions, prefer explicit `return x, y` — readers
shouldn't have to scan the whole body to learn what's being returned.

## Variadic parameters

A parameter typed `...T` accepts zero or more arguments and binds them
as a `[]T` slice inside the function. **Only the last parameter** can
be variadic.

```go
func sum(nums ...int) int {
    total := 0
    for _, n := range nums {
        total += n
    }
    return total
}

fmt.Println(sum())              // 0
fmt.Println(sum(1, 2, 3))       // 6

xs := []int{1, 2, 3, 4}
fmt.Println(sum(xs...))         // 10 — spread an existing slice with ...
```

> **From Python:** `*args` and `**kwargs` rolled into one — but typed.
> Go has no keyword-argument equivalent; if you need named options,
> use a struct or the "functional options" pattern.

## Functions are values

Function types are first-class. You can assign a function to a
variable, pass it as an argument, or return it from another function.

```go
func apply(f func(int) int, x int) int {
    return f(x)
}

double := func(n int) int { return n * 2 }
fmt.Println(apply(double, 5))   // 10
```

The type `func(int) int` describes "a function taking one `int` and
returning one `int`." Any function with that signature can fill that
slot.

## Anonymous functions and closures

A function literal (no name) can be defined inline. It can also
**close over** variables in the surrounding scope — capturing
references to them, not copies.

```go
func makeCounter() func() int {
    count := 0
    return func() int {
        count++
        return count
    }
}

c := makeCounter()
fmt.Println(c(), c(), c())      // 1 2 3
```

`count` lives for as long as the returned closure references it, even
after `makeCounter` itself has returned. This is the same closure
semantics as Python's nested functions.

### Immediately-invoked function expression

A common pattern with `defer` (covered in [11-defer.md](11-defer.md)):
define a function literal and call it right away with `()`.

```go
func() {
    fmt.Println("runs once, right now")
}()
// output: runs once, right now
```

## Functions don't have generic syntax sugar

Go doesn't have keyword arguments, optional positional arguments, or
default values. If you want any of those ergonomics, the idiomatic
options are:

- Multiple functions with descriptive names (`NewReader` /
  `NewReaderSize`).
- A struct parameter (`func Open(opts Options)`).
- The "functional options" pattern (`func Open(opts ...Option)`).

> **From Python:** no `def f(x=42, *, debug=False)`. Idiomatic Go
> favours an extra constructor or an options struct over keyword
> arguments. It feels verbose at first; you stop noticing quickly.

## Recursion

Recursion is allowed and unsurprising. Go does **not** perform
tail-call optimisation — a deeply recursive function can run out of
stack. For most practical depths this is irrelevant; goroutine stacks
start small (~8 KB) and grow on demand.

```go
func factorial(n int) int {
    if n <= 1 {
        return 1
    }
    return n * factorial(n-1)
}

fmt.Println(factorial(10))      // 3628800
```

## Quick reference

| Form | Looks like |
|---|---|
| Single return | `func f(x int) int { ... }` |
| Multiple returns | `func f() (int, error) { ... }` |
| Same-type parameter list | `func f(x, y, z int) { ... }` |
| Named returns + bare return | `func f() (n int, err error) { ... return }` |
| Variadic | `func f(xs ...int) { ... }`; call with `f(1, 2, 3)` or `f(slice...)` |
| Function as parameter | `func apply(f func(int) int, x int) int` |
| Function literal | `func(x int) int { return x*x }` |
| Closure | nested `func` capturing an outer variable |

## Sources

- [Function declarations — go.dev/ref/spec#Function_declarations](https://go.dev/ref/spec#Function_declarations)
- [Function types — go.dev/ref/spec#Function_types](https://go.dev/ref/spec#Function_types)
- [Function literals — go.dev/ref/spec#Function_literals](https://go.dev/ref/spec#Function_literals)
- [Passing arguments to `...` parameters — go.dev/ref/spec#Passing_arguments_to_..._parameters](https://go.dev/ref/spec#Passing_arguments_to_..._parameters)
- [Effective Go: Functions — go.dev/doc/effective_go#functions](https://go.dev/doc/effective_go#functions)
