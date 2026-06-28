# OOP patterns in Go

Go has no `class`, no inheritance, and no constructors. Yet it covers
everything object-oriented code reaches for — it just assembles the results
from four smaller pieces you've already met: **structs**, **methods**,
**interfaces**, and **embedding**. This article maps the familiar OOP ideas
onto the Go way of doing them.

| OOP idea | Go mechanism |
|---|---|
| class | struct + methods |
| constructor | a `NewT(...) T` function (plain convention) |
| instance method | method with a receiver |
| encapsulation | exported vs unexported names (per **package**) |
| inheritance | — none; use **embedding** for reuse |
| polymorphism | **interfaces** |
| abstract base class | an interface |

## "Objects" are structs with methods

A struct bundles the data; methods give it behaviour. The idiomatic
"constructor" is just a function named `New...` that returns a ready value:

```go
type Counter struct{ n int }

func NewCounter(start int) *Counter { return &Counter{n: start} }

func (c *Counter) Inc()       { c.n++ }
func (c *Counter) Value() int { return c.n }

c := NewCounter(10)
c.Inc()
fmt.Println(c.Value())   // output: 11
```

## Encapsulation is per-package, not per-class

Go's access control is **capitalisation**: an identifier starting with an
uppercase letter is *exported* (visible to other packages); lowercase is
*unexported* (visible only inside its own package). The privacy boundary is
the **package**, not the type.

```go
type Account struct {
    owner   string   // unexported: other packages can't touch it
    balance int      // unexported
}

func NewAccount(owner string) *Account { return &Account{owner: owner} }

func (a *Account) Deposit(amount int) { a.balance += amount }
func (a *Account) Balance() int       { return a.balance }

a := NewAccount("Ada")
a.Deposit(100)
fmt.Println(a.Balance())   // output: 100
```

Code in another package can call `Deposit` and `Balance` but cannot read or
write `balance` directly — that's encapsulation. (Within the *same* package
everything is visible, so the barrier is about package boundaries.)

## Composition over inheritance: embedding

Instead of subclassing, you **embed** one struct in another. The inner
type's fields *and methods* are promoted, so the outer type appears to
"have" them — reuse without an inheritance hierarchy.

```go
type Logger struct{ prefix string }

func (l Logger) Log(msg string) string { return l.prefix + ": " + msg }

type Server struct {
    Logger          // embedded — Server gains Log()
    addr string
}

s := Server{Logger: Logger{prefix: "srv"}, addr: ":8080"}
fmt.Println(s.Log("up"))   // output: srv: up  — promoted from Logger
```

You can **override** a promoted method by defining one with the same name
on the outer type; the outer one shadows the inner, which is still
reachable via the field name:

```go
func (s Server) Log(msg string) string {
    return "[" + s.addr + "] " + s.Logger.Log(msg)   // call the embedded one explicitly
}

s := Server{Logger: Logger{prefix: "srv"}, addr: ":8080"}
fmt.Println(s.Log("up"))   // output: [:8080] srv: up
```

> **From Python:** embedding looks like inheritance but isn't — there's no
> base class and no `super()`. It's *composition* with automatic
> forwarding; you reach the inner value explicitly as `s.Logger`.

## Polymorphism through interfaces

Different "subclasses" become different types satisfying one interface — no
shared base required:

```go
type Speaker interface{ Speak() string }

type Dog struct{}
func (Dog) Speak() string { return "woof" }

type Cat struct{}
func (Cat) Speak() string { return "meow" }

func chorus(speakers []Speaker) string {
    out := ""
    for _, s := range speakers {
        out += s.Speak() + " "
    }
    return out
}

fmt.Print(chorus([]Speaker{Dog{}, Cat{}}))   // output: woof meow 
```

## Interface embedding: building bigger from smaller

Interfaces embed interfaces, composing capabilities. This is exactly how
the standard library builds `io.ReadWriter` from `io.Reader` + `io.Writer`:

```go
type Reader interface{ Read() string }
type Writer interface{ Write(s string) }

type ReadWriter interface {
    Reader          // embedded interfaces
    Writer
}
```

A type satisfies `ReadWriter` automatically once it has both `Read` and
`Write` — no declaration needed:

```go
type Buf struct{ data string }

func (b *Buf) Read() string   { return b.data }
func (b *Buf) Write(s string) { b.data = s }

var rw ReadWriter = &Buf{}    // *Buf has Read + Write, so it qualifies
rw.Write("hi")
fmt.Println(rw.Read())        // output: hi
```

`*Buf` never mentions `ReadWriter`, `Reader`, or `Writer` — having the two
methods is enough.

## Why no inheritance?

Go leaves out inheritance on purpose. Deep class hierarchies couple a
subclass to its ancestors and grow brittle; Go pushes you toward small
interfaces (for polymorphism) plus embedding (for reuse), which compose
more flexibly. The practical guidance: **model "is-a" with interfaces and
"has-a" with embedding** — and you'll rarely miss classes.

## Quick reference

| Want… | Do this |
|---|---|
| an "object" | struct + methods |
| a constructor | `func NewT(...) *T` |
| private state | lowercase (unexported) fields + exported methods |
| reuse another type's behaviour | embed it |
| override a promoted method | define same-named method on the outer type |
| polymorphism | define an interface; many types satisfy it |
| combine capabilities | embed interfaces |

## Sources

- [Effective Go: embedding — go.dev/doc/effective_go#embedding](https://go.dev/doc/effective_go#embedding)
- [Struct types (embedded fields) — go.dev/ref/spec#Struct_types](https://go.dev/ref/spec#Struct_types)
- [Exported identifiers — go.dev/ref/spec#Exported_identifiers](https://go.dev/ref/spec#Exported_identifiers)
- [Interface types (embedding) — go.dev/ref/spec#Interface_types](https://go.dev/ref/spec#Interface_types)
- [Go FAQ: is Go object-oriented? — go.dev/doc/faq#Is_Go_an_object-oriented_language](https://go.dev/doc/faq#Is_Go_an_object-oriented_language)
