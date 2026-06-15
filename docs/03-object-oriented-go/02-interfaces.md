# Interfaces

An **interface** is a type that lists a set of method signatures. Any value
whose type has *all* those methods satisfies the interface — and can be
stored in a variable of that interface type. Interfaces are how Go does
polymorphism: code depends on *what a value can do* (its methods), not on
its concrete type.

```go
type Shape interface {
    Area() float64
}
```

`Shape` is now a type. A variable of type `Shape` can hold any value that
has an `Area() float64` method.

## Satisfaction is implicit

There is no `implements` keyword. A type satisfies an interface simply by
having the required methods — the compiler checks this structurally. You
never declare the relationship; it just exists.

```go
type Rectangle struct{ W, H float64 }
func (r Rectangle) Area() float64 { return r.W * r.H }

type Circle struct{ R float64 }
func (c Circle) Area() float64 { return math.Pi * c.R * c.R }

var s Shape = Rectangle{W: 3, H: 4}   // Rectangle satisfies Shape — no declaration needed
fmt.Println(s.Area())                 // output: 12
```

Both `Rectangle` and `Circle` satisfy `Shape` without ever mentioning it.

> **From Python:** this is duck typing — "if it has the methods, it fits" —
> but verified at **compile time**. A type that's missing a method simply
> won't compile where the interface is expected, instead of blowing up at
> runtime.

## Polymorphism: one function, many types

Because any `Shape` has `Area()`, a function can accept the interface and
work with every concrete type uniformly:

```go
func totalArea(shapes []Shape) float64 {
    sum := 0.0
    for _, s := range shapes {
        sum += s.Area()
    }
    return sum
}

shapes := []Shape{Rectangle{3, 4}, Circle{1}}
fmt.Printf("%.2f\n", totalArea(shapes))   // output: 15.14
```

## An interface value is a (type, value) pair

Internally an interface value holds two things: the **dynamic type** of
what's stored, and the **value** itself. The zero value of an interface is
`nil` — no type, no value.

```go
var s Shape          // nil interface
fmt.Println(s == nil)   // output: true
```

Calling a method on a `nil` interface panics, because there's no concrete
method to dispatch to.

## Pointer vs value receivers decide satisfaction

This is the most common interface gotcha. A type's **method set**
determines which interfaces it satisfies:

- methods with **value receivers** belong to both `T` and `*T`
- methods with **pointer receivers** belong only to `*T`

So if a method has a pointer receiver, only a **pointer** satisfies the
interface — a value does not:

```go
type Counter struct{ n int }
func (c *Counter) Add()        { c.n++ }      // pointer receiver
func (c Counter) Value() int   { return c.n }

type Adder interface{ Add() }

var a Adder = &Counter{}   // ok: *Counter has Add
// var a Adder = Counter{} // compile error: Counter does not implement Adder
//                         //   (method Add has pointer receiver)
a.Add()
```

Rule of thumb: if any method needs a pointer receiver, pass the pointer
when you want the value to satisfy an interface.

## The empty interface and `any`

An interface with no methods is satisfied by **every** type. Its modern
spelling is `any` (an alias for `interface{}`), and it's how you hold "a
value of unknown type."

```go
var x any
x = 42
x = "hello"
fmt.Println(x)   // output: hello
```

`any` is a tool of last resort — you lose all compile-time type
information. To get the concrete value back out, you use a **type
assertion** or **type switch** — covered in the
[next article](03-type-assertions-and-type-switches.md).

> **From Python:** `any` is the closest thing to a plain `object`
> reference — it can hold anything, and you must check the type before
> using it specifically.

## The typed-nil gotcha

An interface is `nil` only when **both** its type and value are nil. If you
store a nil *pointer* in an interface, the interface holds a type, so it is
**not** nil — a frequent source of bugs.

```go
type T struct{}
func (t *T) Foo() {}

var p *T = nil
var i interface{ Foo() } = p
fmt.Println(p == nil)   // output: true
fmt.Println(i == nil)   // output: false  — i has type *T, so it's non-nil
```

The lesson: return a literal `nil` for the interface, not a nil concrete
pointer, when you mean "nothing."

## Small interfaces are idiomatic

Go favours tiny interfaces — often one method — defined where they're
*used*, not where types are defined. The standard library is full of them:

| Interface | Method | Purpose |
|---|---|---|
| `fmt.Stringer` | `String() string` | custom text form |
| `error` | `Error() string` | the error type |
| `io.Reader` | `Read([]byte) (int, error)` | a source of bytes |
| `io.Writer` | `Write([]byte) (int, error)` | a sink for bytes |
| `sort.Interface` | `Len`/`Less`/`Swap` | custom sorting |

Implement `fmt.Stringer` and `fmt.Println` uses it automatically:

```go
type Color struct{ R, G, B int }
func (c Color) String() string {
    return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}
fmt.Println(Color{255, 165, 0})   // output: #FFA500
```

The guideline: **accept interfaces, return concrete types.** Take the
smallest interface your function actually needs as a parameter, and return
the specific struct so callers keep full information.

## Quick reference

| Concept | Syntax |
|---|---|
| declare an interface | `type Reader interface { Read(p []byte) (int, error) }` |
| satisfy it | just define the methods — no keyword |
| empty interface | `any` (= `interface{}`), holds any value |
| nil interface | both type and value nil |

Extracting the concrete value back out of an interface is covered next, in
[type assertions and type switches](03-type-assertions-and-type-switches.md).

## Sources

- [Interface types — go.dev/ref/spec#Interface_types](https://go.dev/ref/spec#Interface_types)
- [Method sets — go.dev/ref/spec#Method_sets](https://go.dev/ref/spec#Method_sets)
- [Effective Go: interfaces — go.dev/doc/effective_go#interfaces](https://go.dev/doc/effective_go#interfaces)
- [Go blog: errors are values / typed nil — go.dev/doc/faq#nil_error](https://go.dev/doc/faq#nil_error)
- [fmt.Stringer — pkg.go.dev/fmt#Stringer](https://pkg.go.dev/fmt#Stringer)
