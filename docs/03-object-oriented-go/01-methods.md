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

## A nil receiver is a normal case

Calling a pointer-receiver method on a `nil` pointer is **not** an error.
The call itself is fine — only *dereferencing* the nil pointer faults. So a
method can check for `nil` and treat it as a meaningful state, which is how
recursive structures avoid nil-guards at every call site.

```go
type Tree struct {
    Val         int
    Left, Right *Tree
}

func (t *Tree) Sum() int {
    if t == nil {        // the receiver being nil is a normal case, not a bug
        return 0
    }
    return t.Val + t.Left.Sum() + t.Right.Sum()
}

t := &Tree{Val: 1, Left: &Tree{Val: 2}}
fmt.Println(t.Sum())     // output: 3

var empty *Tree
fmt.Println(empty.Sum()) // output: 0
```

`t.Left.Sum()` works even when `Left` is nil — that's what makes the
recursion terminate without checking every branch before descending.

> **From Python:** calling a method on `None` is always an
> `AttributeError`. In Go the receiver is just an argument, so a nil
> pointer arrives at the method perfectly intact; you decide what it means.

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

### The range-loop trap

The same addressability rule causes the single most common silent bug with
pointer receivers. `for _, x := range s` binds `x` to a **copy** of each
element, so mutating it changes nothing — and it compiles cleanly:

```go
type Counter struct{ n int }
func (c *Counter) Inc() { c.n++ }

cs := []Counter{{}, {}}

for _, c := range cs {
    c.Inc()          // c is a copy — compiles, mutates nothing
}
fmt.Println(cs)      // output: [{0} {0}]

for i := range cs {
    cs[i].Inc()      // cs[i] is addressable — mutates the slice
}
fmt.Println(cs)      // output: [{1} {1}]
```

Index through the slice (or make it `[]*Counter`) when you need the
mutation to stick.

> **From Python:** `for x in lst` hands you the object itself, so
> `x.inc()` sticks. Go hands you a copy, and nothing warns you.

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
primitive-backed types. The function-backed case is the surprising one — a
*function type* can have methods:

```go
type Handler func(string) string

func (h Handler) Twice(s string) string { return h(h(s)) }

var exclaim Handler = func(s string) string { return s + "!" }
fmt.Println(exclaim.Twice("go"))   // output: go!!
```

`exclaim` is a function value, yet `.Twice` calls it twice — the receiver
`h` *is* the function. (This is exactly how `http.HandlerFunc` adapts a
plain function into an interface.)

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
[custom types](../02-language-basics/09-custom-types.md): define your own type with
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

```go
type Counter int
func (c Counter) Get() int { return int(c) }   // value receiver
func (c *Counter) Inc()    { *c++ }             // pointer receiver

type Incrementer interface{ Inc() }

var c Counter = 5
c.Inc()                  // ok: c is addressable, so Go takes &c for you
fmt.Println(c.Get())     // output: 6

var i Incrementer = &c   // only *Counter satisfies Incrementer
i.Inc()
fmt.Println(c.Get())     // output: 7

// var bad Incrementer = c   // compile error: Counter does not implement
//                           // Incrementer (method Inc has pointer receiver)
```

Calling `c.Inc()` directly works because `c` is an addressable variable —
Go silently rewrites it to `(&c).Inc()`. But storing a *value* in the
interface doesn't get that help, so only `&c` satisfies `Incrementer`.

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

**The receiver is evaluated when the method value is created, not when it's
called.** With a value receiver that means you capture a *copy*, frozen at
that instant; with a pointer receiver you capture the address, so later
changes are visible.

```go
type Counter struct{ n int }
func (c Counter) Get() int { return c.n }   // value receiver
func (c *Counter) Inc()    { c.n++ }         // pointer receiver

c := Counter{}
get := c.Get      // copies c right now
inc := c.Inc      // captures &c

inc()
inc()
fmt.Println(get(), c.Get())   // output: 0 2
```

`get()` still reports 0 — it's reading the copy taken before the
increments. This is the classic bug when method values are stashed in a
callback or a `defer`.

> **From Python:** `obj.method` is a bound method that always sees current
> state. A Go method value with a value receiver does not.

### Method expression — receiver is the first parameter

```go
g := Celsius.Fahrenheit                 // method *expression*
fmt.Println(g(Celsius(100)))            // 212
```

`g` has type `func(Celsius) float64`. The receiver becomes an explicit
first parameter at the call site.

For a **pointer-receiver** method you must name the pointer type — and the
compiler tells you so:

```go
inc := (*Counter).Inc     // type: func(*Counter)
c := Counter{}
inc(&c)
inc(&c)
fmt.Println(c.n)          // output: 2

// f := Counter.Inc
// compile error: invalid method expression Counter.Inc
//   (needs pointer receiver (*Counter).Inc)
```

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

### What promotion puts in the method set

Promotion follows the same pointer/value split as ordinary methods, one
level deeper. Embedding a **value** `T` promotes `T`'s value-receiver
methods to both `S` and `*S` — but `T`'s **pointer**-receiver methods land
only in `*S`:

```go
type Logger struct{ n int }
func (l *Logger) Log() { l.n++ }      // pointer receiver

type Server struct{ Logger }          // embeds the value

type Loggable interface{ Log() }

var _ Loggable = &Server{}   // ok
var _ Loggable = Server{}
// compile error: Server does not implement Loggable (method Log has pointer receiver)
```

The error is confusing the first time, because `Log` isn't even declared on
`Server` — it's promoted. The fix is to use `*Server`.

### When two embedded types collide

Embedding two types that provide the same method name is legal to
*declare*. The error only fires where you actually select it:

```go
type Reader struct{}
func (Reader) Close() string { return "reader" }

type Writer struct{}
func (Writer) Close() string { return "writer" }

type File struct {
    Reader
    Writer
}

var f File
fmt.Println(f.Close())
// compile error: ambiguous selector f.Close
```

Shallower depth wins, so declaring `Close` directly on `File` resolves it —
and that is exactly how "overriding" works in Go. The inner ones stay
reachable by name:

```go
func (File) Close() string { return "file" }

fmt.Println(f.Close(), f.Reader.Close())   // output: file reader
```

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
