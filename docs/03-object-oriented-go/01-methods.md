# Methods

A **method** is a function with a *receiver* — a typed parameter that
appears between `func` and the method name. Methods turn a defined
type into something that has behaviour attached to it.

```go
type Celsius float64

func (c Celsius) Fahrenheit() float64 {
    return float64(c)*9/5 + 32
}

func main() {
    c := Celsius(100)
    fmt.Println(c.Fahrenheit())     // 212
}
```

The receiver `(c Celsius)` is just a normal parameter; the only thing
different is its position. Inside the body, `c` behaves like any other
variable of type `Celsius`.

> **From Python:** ≈ a class method, with one subtle difference —
> there is no class. The receiver is just an extra parameter the
> compiler binds to a specific type. You also get to *name* the
> receiver; there's no implicit `self`.

## Value receivers

`func (c Celsius)` is a **value receiver**. The method gets a *copy* of
the value. Modifying the receiver inside the method doesn't affect
the original.

```go
type Counter int

func (c Counter) Inc() {        // value receiver — operates on a copy
    c++
}

func main() {
    var n Counter = 0
    n.Inc()
    n.Inc()
    fmt.Println(n)              // 0 — Inc never touched the caller's n
}
```

Use a value receiver when:

- The method doesn't need to modify the receiver.
- The receiver is small (a primitive-typed defined type, a small
  struct).

## Pointer receivers

`func (c *Counter)` is a **pointer receiver**. The method receives a
pointer to the original, and writes through it modify the caller's
value.

```go
type Counter int

func (c *Counter) Inc() {       // pointer receiver — writes through
    *c++
}

func main() {
    var n Counter = 0
    n.Inc()
    n.Inc()
    fmt.Println(n)              // 2
}
```

Use a pointer receiver when:

- The method needs to mutate the receiver.
- The receiver is large (a multi-field struct you don't want to copy
  on every call).
- The struct contains a field that must not be copied (a `sync.Mutex`,
  for example).
- *Consistency*: if any method on the type needs a pointer receiver,
  give them **all** pointer receivers so the type's method set is
  consistent.

## Auto-addressing and auto-dereferencing

You don't write `(&n).Inc()` or `(*p).Inc()`. Go inserts the `&` or
`*` for you when the call site has a value of one form and the method
needs the other.

```go
type Counter int
func (c *Counter) Inc() { *c++ }

func main() {
    var n Counter = 0
    n.Inc()                     // Go silently rewrites as (&n).Inc()
    p := &n
    p.Inc()                     // already a pointer; no rewrite needed
    fmt.Println(n)              // 2
}
```

One condition for the rewrite: the value must be **addressable** (a
named variable, a field of an addressable struct, or `*` something).
A map element or the return value of a function call is **not**
addressable.

```go
type Counter int
func (c *Counter) Inc() { *c++ }

m := map[string]Counter{"x": 0}
m["x"].Inc()                    // compile error: cannot call pointer method Inc on Counter
```

The fix: read into a local, mutate, write back; or change the map to
hold `*Counter` values.

## Methods on non-struct types

The receiver type can be **any defined type in your package** — not
just structs.

```go
type Names []string

func (n Names) Contains(s string) bool {
    for _, x := range n {
        if x == s {
            return true
        }
    }
    return false
}

func main() {
    n := Names{"Ada", "Linus"}
    fmt.Println(n.Contains("Ada"))      // true
    fmt.Println(n.Contains("Grace"))    // false
}
```

This is how you attach behaviour to slice, map, function, or
primitive-backed types.

## The "same package" restriction

The receiver type must be **defined in the same package as the
method**:

```go
package mine
func (t time.Time) Foo() { ... }        // compile error
func (i int) Double() int { ... }       // compile error
```

You cannot bolt methods onto `int`, `time.Time`, or anything else
from another package. The workaround is the same one from
[09-custom-types.md](../02-language-basics/09-custom-types.md): define your own type with
the foreign type as its underlying type, and attach the method there.

```go
type Stamp time.Time

func (s Stamp) Unix() int64 {
    return time.Time(s).Unix()
}
```

## Method sets — preview

Every type has a **method set**: the methods that can be called on
values of that type. The rule:

- The method set of `T` contains all methods with receiver type `T`.
- The method set of `*T` contains all methods with receiver type
  `*T` **and** all methods with receiver type `T`.

In practice you rarely think about method sets explicitly — until you
start implementing interfaces. An **interface** (covered properly in
a later article) is a named set of method signatures; a type
*satisfies* an interface when its method set contains all those
methods. Interfaces get their own topic; remember the rule for then:

> If any method has a pointer receiver, only `*T` (not `T`) satisfies
> interfaces that include that method.

## Method values and method expressions

A method can be detached from its receiver in two ways.

### Method value — receiver is baked in

```go
type Celsius float64
func (c Celsius) Fahrenheit() float64 { return float64(c)*9/5 + 32 }

c := Celsius(100)
f := c.Fahrenheit                       // method *value* — c is captured
fmt.Println(f())                        // 212
```

`f` has type `func() float64`. The receiver `c` is closed over.

### Method expression — receiver is the first parameter

```go
g := Celsius.Fahrenheit                 // method *expression*
fmt.Println(g(Celsius(100)))            // 212
```

`g` has type `func(Celsius) float64`. The receiver becomes an explicit
first parameter at the call site.

Method values are far more common in real code; expressions show up
in plumbing libraries and tests.

## Embedding and method promotion

If a struct **embeds** another type (a field with a type name and no
field name), the embedded type's methods become callable on the outer
struct.

```go
type Logger struct{ prefix string }
func (l Logger) Log(msg string) { fmt.Println(l.prefix, msg) }

type Server struct {
    Logger              // embedded — no field name
    addr string
}

func main() {
    s := Server{Logger: Logger{prefix: "[srv]"}, addr: ":8080"}
    s.Log("starting")                   // [srv] starting
}
```

`s.Log(...)` is shorthand for `s.Logger.Log(...)`. The method has been
**promoted** to `Server`. Compose behaviour by embedding; Go has no
inheritance.

## Quick reference

| You want | Write |
|---|---|
| Method that reads the receiver | `func (c Celsius) F() float64` (value receiver) |
| Method that mutates the receiver | `func (c *Counter) Inc()` (pointer receiver) |
| Call a pointer-receiver method on a value variable | Just write `n.Inc()` — Go inserts `&` |
| Method on a slice / map / int-backed type | Define `type X []int`, then `func (x X) Foo() {}` |
| Bind a method to a fixed receiver | `f := c.Fahrenheit` (method value) |
| Treat a method as an unbound function | `g := Celsius.Fahrenheit` (method expression) |

## Sources

- [Method declarations — go.dev/ref/spec#Method_declarations](https://go.dev/ref/spec#Method_declarations)
- [Method sets — go.dev/ref/spec#Method_sets](https://go.dev/ref/spec#Method_sets)
- [Method values & method expressions — go.dev/ref/spec#Method_values](https://go.dev/ref/spec#Method_values)
- [Struct types: embedded fields & promotion — go.dev/ref/spec#Struct_types](https://go.dev/ref/spec#Struct_types)
- [Effective Go: Methods — go.dev/doc/effective_go#methods](https://go.dev/doc/effective_go#methods)
