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

Inference is not limited to calls. Wherever a generic function is
**assigned to a variable of a matching function type** — or converted to
one — the compiler works the type arguments out from that type:

```go
func Map[T, U any](s []T, f func(T) U) []U { /* ... */ }

var g func([]int, func(int) string) []string = Map   // infers T=int, U=string
fmt.Println(g([]int{1, 2}, func(i int) string { return fmt.Sprint(i * 10) }))
// output: [10 20]
```

That means you can hand a generic function straight to anything expecting
a concrete function type — a struct field, a callback parameter, a map of
handlers — without spelling out `Map[int, string]`.

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
literally `float64`, only *based on* it:

```go
type StrictFloat interface{ float64 }   // no ~

func StrictSum[T StrictFloat](xs []T) T { /* ... */ }

StrictSum([]Celsius{1, 2})
// compile error: Celsius does not satisfy StrictFloat
//   (possibly missing ~ for float64 in StrictFloat)
```

The compiler even suggests the fix. Add the `~` and `Celsius` qualifies.

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

## Generic methods

A method may declare **its own type parameters**, separate from any the
receiver carries. That matters whenever an operation has to *change* the
element type: a set of user IDs turned into a set of usernames, a cache
keyed one way re-keyed another. The receiver's `T` is fixed by the value
you call it on, so the new type needs a parameter of its own.

```go
type Set[T comparable] map[T]struct{}

func (s Set[T]) Add(v T)      { s[v] = struct{}{} }
func (s Set[T]) Has(v T) bool { _, ok := s[v]; return ok }

// MapTo declares U for itself — T comes from the receiver.
func (s Set[T]) MapTo[U comparable](f func(T) U) Set[U] {
    out := Set[U]{}
    for v := range s {
        out.Add(f(v))
    }
    return out
}
```

Calling it infers `U` from the function you pass, exactly as for a generic
function:

```go
ids := Set[int]{}
ids.Add(1)
ids.Add(2)

names := ids.MapTo(func(id int) string { return fmt.Sprintf("user-%d", id) })
fmt.Println(names.Has("user-1"), names.Has("user-9"))   // output: true false
```

You can instantiate the method explicitly when inference can't help, which
also gives you a reusable method value:

```go
toString := ids.MapTo[string]
fmt.Println(toString(func(id int) string { return fmt.Sprint(id) }).Has("2"))
// output: true
```

The payoff is namespacing. Without a type parameter of its own, a method
like this has to be a package-level function — `MapSet`, `MapStack`,
`MapList` — one per container, all competing for names in the package.
As a method it lives on the type it belongs to.

### Interfaces stay non-generic

The one firm limit: **an interface method may not declare type
parameters**, and a generic method cannot implement a non-generic one.

```go
type Doer interface {
    Do[T any](T) T   // compile error: interface method must have no type parameters
}
```

A generic method has no single fixed signature, so it can't satisfy a
method the interface pins down:

```go
type Doer interface{ Do(int) int }

type T struct{}
func (T) Do[U any](u U) U { return u }

var _ Doer = T{}
// compile error: T does not implement Doer (wrong type for method Do)
//   have Do[U any](U) U
//   want Do(int) int
```

Dynamic dispatch needs one concrete signature per method; a generic method
is a family of them. Keep interface methods concrete, and put the generic
work on the implementing type.

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
| `func (b Box[T]) To[U any](...)` | generic method — its own type parameter |
| `var f func(int) string = G` | inference from assignment to a function type |
| `var zero T` | the zero value of a type parameter |

## Sources

- [Type parameters — go.dev/ref/spec#Type_parameter_declarations](https://go.dev/ref/spec#Type_parameter_declarations)
- [Type constraints — go.dev/ref/spec#Type_constraints](https://go.dev/ref/spec#Type_constraints)
- [The `comparable` constraint — go.dev/ref/spec#Comparison_operators](https://go.dev/ref/spec#Comparison_operators)
- [cmp.Ordered — pkg.go.dev/cmp#Ordered](https://pkg.go.dev/cmp#Ordered)
- [Go blog: an introduction to generics — go.dev/blog/intro-generics](https://go.dev/blog/intro-generics)
- [Tutorial: getting started with generics — go.dev/doc/tutorial/generics](https://go.dev/doc/tutorial/generics)
- [Method declarations — go.dev/ref/spec#Method_declarations](https://go.dev/ref/spec#Method_declarations)
- [Type inference — go.dev/ref/spec#Type_inference](https://go.dev/ref/spec#Type_inference)
