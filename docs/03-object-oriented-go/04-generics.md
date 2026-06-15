# Generics: type parameters and constraints

**Generics** let you write a single function or type that works across many
types, while keeping full compile-time type safety. Where an interface
abstracts over *behaviour*, a generic abstracts over the *type itself* — no
`any`, no runtime type assertions, no boxing.

## Type parameters on functions

A function gains **type parameters** in square brackets *before* the
ordinary parameter list. Each type parameter has a **constraint** that
limits which types may be substituted.

```go
func Max[T cmp.Ordered](a, b T) T {
    if a > b {
        return a
    }
    return b
}

fmt.Println(Max(3, 7))         // output: 7
fmt.Println(Max("go", "py"))   // output: py
```

`T` is the type parameter; `cmp.Ordered` is its constraint — the set of
types that support `<`, `>`, and so on. The same `Max` now works for ints,
floats, and strings, each checked at compile time.

## Type inference

You usually don't write the type argument — the compiler infers `T` from
the call's arguments. You *can* spell it out when inference can't (or for
clarity):

```go
fmt.Println(Max(3, 7))         // inferred: T = int
fmt.Println(Max[float64](3, 7)) // explicit: T = float64 → prints 7
```

## Constraints are interfaces

A constraint is just an **interface** used in a type-parameter position.
The two built-in ones you'll meet first:

- `any` — no restriction (every type qualifies; it's literally `interface{}`)
- `comparable` — types that support `==` and `!=`

```go
func Index[T comparable](s []T, target T) int {
    for i, v := range s {
        if v == target {     // == is allowed because T is comparable
            return i
        }
    }
    return -1
}

fmt.Println(Index([]string{"a", "b", "c"}, "b"))   // output: 1
```

## Custom constraints: type sets and `~`

A constraint interface can list a **set of types** with `|`. That lets the
body use operators those types share. The `~` prefix means "any type whose
*underlying* type is this," so your own defined types qualify too.

```go
type Number interface {
    ~int | ~int64 | ~float64
}

func Sum[T Number](nums []T) T {
    var total T          // zero value of T
    for _, n := range nums {
        total += n       // + is allowed: every type in the set supports it
    }
    return total
}

type Celsius float64     // underlying type is float64
fmt.Println(Sum([]int{1, 2, 3}))            // output: 6
fmt.Println(Sum([]Celsius{1.5, 2.5}))       // output: 4
```

Without the `~`, `Sum[Celsius]` would be rejected — `Celsius` is not
literally `float64`, only *based on* it.

## Generic types

Types take type parameters too. The classic example is a container that
holds any element type:

```go
type Stack[T any] struct {
    items []T
}

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *Stack[T]) Pop() (T, bool) {
    var zero T
    if len(s.items) == 0 {
        return zero, false
    }
    last := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    return last, true
}

var s Stack[int]
s.Push(1)
s.Push(2)
v, ok := s.Pop()
fmt.Println(v, ok)   // output: 2 true
```

Note `var zero T` — since you don't know `T`, that's how you produce its
zero value. Methods on a generic type repeat the type parameter in the
receiver: `(s *Stack[T])`.

## A generic set

Combining a generic type with `comparable` gives a reusable set — better
than re-coding `map[T]struct{}` for each element type:

```go
type Set[T comparable] map[T]struct{}

func (s Set[T]) Add(v T)      { s[v] = struct{}{} }
func (s Set[T]) Has(v T) bool { _, ok := s[v]; return ok }

s := Set[string]{}
s.Add("go")
fmt.Println(s.Has("go"), s.Has("py"))   // output: true false
```

## When not to reach for generics

Generics shine for **containers and algorithms** that are identical across
element types (collections, `Map`/`Filter`/`Reduce`, min/max). They are
*not* a replacement for interfaces: when you want different types to supply
different behaviour behind one abstraction, that's an interface's job. Rule
of thumb — if the only thing varying is the *type*, use a generic; if the
*behaviour* varies, use an interface.

> **From Python:** this is `typing.TypeVar` / `Generic[T]` territory, but
> enforced by the compiler rather than by an optional checker — and with
> zero runtime cost, since the types are resolved at build time.

## Quick reference

| Form | Meaning |
|---|---|
| `func F[T any](x T)` | function with a type parameter |
| `[T cmp.Ordered]` | constraint allowing `<`, `>` |
| `[T comparable]` | constraint allowing `==`, `!=` |
| `interface{ ~int \| ~float64 }` | type-set constraint; `~` = underlying type |
| `type Box[T any] struct{ v T }` | generic type |
| `func (b Box[T]) Get() T` | method on a generic type |
| `var zero T` | the zero value of a type parameter |

## Sources

- [Type parameters — go.dev/ref/spec#Type_parameter_declarations](https://go.dev/ref/spec#Type_parameter_declarations)
- [Type constraints — go.dev/ref/spec#Type_constraints](https://go.dev/ref/spec#Type_constraints)
- [The `comparable` constraint — go.dev/ref/spec#Comparison_operators](https://go.dev/ref/spec#Comparison_operators)
- [cmp.Ordered — pkg.go.dev/cmp#Ordered](https://pkg.go.dev/cmp#Ordered)
- [Go blog: an introduction to generics — go.dev/blog/intro-generics](https://go.dev/blog/intro-generics)
- [Tutorial: getting started with generics — go.dev/doc/tutorial/generics](https://go.dev/doc/tutorial/generics)
