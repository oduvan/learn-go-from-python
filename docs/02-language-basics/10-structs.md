# Structs

A **struct** is a typed collection of named fields glued together into one
value. It is Go's primary tool for modelling "a thing with several
attributes" — a point, a user, an HTTP request. There is no class, no
inheritance, no constructor keyword: a struct is just data laid out in
memory, and you build behaviour around it separately (with functions and
[methods](13-methods.md)).

You almost always give a struct a name using the `type` keyword from
[custom types](09-custom-types.md):

```go
type Point struct {
    X int
    Y int
}
```

That declares a new type `Point` whose values carry two `int` fields, `X`
and `Y`. Fields of the same type can share a line:

```go
type Point struct {
    X, Y int
}
```

## Creating struct values

There are several ways to make a `Point`, and the difference matters.

**Keyed literal** — name the fields. This is the form you should reach
for almost always: it is order-independent and survives someone adding a
new field later.

```go
p := Point{X: 1, Y: 2}
fmt.Println(p)        // output: {1 2}
```

**Positional literal** — values in field-declaration order, no names.
Fragile: it breaks the moment the struct gains or reorders a field, and
it *requires* a value for every field.

```go
p := Point{1, 2}      // ok, but tied to declaration order
```

**Zero value** — declare without initialising and every field gets its
type's zero value (`0`, `""`, `nil`, `false`, …). A struct has no
separate "uninitialised" state; the zero struct is a complete, usable
value.

```go
var p Point
fmt.Println(p)        // output: {0 0}
```

You can omit fields in a keyed literal; the ones you leave out take their
zero value:

```go
p := Point{Y: 5}
fmt.Println(p)        // output: {0 5}
```

> **From Python:** there is no `__init__`. The zero value *is* your
> default constructor. When zero isn't a sensible default, the convention
> is a plain function named `NewPoint(...) Point` — a regular function,
> not special syntax.

## Reading and writing fields

Dot notation, and fields are addressable, so you can assign to them
directly:

```go
p := Point{X: 1, Y: 2}
p.X = 10
fmt.Println(p.X + p.Y)   // output: 12
```

## Structs are value types

Assigning a struct, passing it to a function, or returning it **copies
every field**. The copy is independent of the original.

```go
a := Point{X: 1, Y: 2}
b := a            // full copy
b.X = 99
fmt.Println(a.X)  // output: 1  — a is untouched
```

This is the single most important thing to internalise. If you want a
function to mutate the caller's struct, pass a pointer:

```go
func moveRight(p *Point) {
    p.X++         // p.X is shorthand for (*p).X — Go auto-dereferences
}

a := Point{X: 1, Y: 2}
moveRight(&a)
fmt.Println(a.X)  // output: 2
```

Note `p.X` on a `*Point`: Go automatically dereferences a struct pointer
for field access, so you never write `(*p).X`. See [pointers](08-pointers.md)
for the underlying rule.

## Comparing structs

A struct is comparable with `==` **if all of its fields are comparable**.
The comparison is field-by-field.

```go
p := Point{1, 2}
q := Point{1, 2}
fmt.Println(p == q)   // output: true
```

This also makes comparable structs usable as map keys. But if a struct
contains a non-comparable field — a slice, a map, or a function — the
whole struct becomes non-comparable and `==` is a **compile error**:

```go
type Bag struct {
    items []int
}
b1 := Bag{}
b2 := Bag{}
_ = b1 == b2          // compile error: struct containing []int cannot be compared
```

## Nested and embedded structs

A field can itself be a struct:

```go
type Line struct {
    Start Point
    End   Point
}

l := Line{
    Start: Point{0, 0},
    End:   Point{3, 4},
}
fmt.Println(l.End.Y)   // output: 4
```

If you declare a field with **no name** — just a type — that field is
*embedded*, and its fields are promoted so you can reach them directly:

```go
type Circle struct {
    Point      // embedded: no field name, just the type
    Radius int
}

c := Circle{Point: Point{X: 1, Y: 2}, Radius: 5}
fmt.Println(c.X)        // output: 1  — promoted from the embedded Point
fmt.Println(c.Point.Y)  // output: 2  — the explicit path still works
```

Embedding is Go's composition mechanism — it stands in for the data side
of what other languages do with inheritance. The *method* side of
embedding (method promotion) is covered in [methods](13-methods.md).

## Anonymous structs

You can create a struct value without ever declaring a named type. Handy
for a one-off grouping — a table-test row, a quick JSON shape — where a
top-level `type` would be noise.

```go
config := struct {
    Host string
    Port int
}{
    Host: "localhost",
    Port: 8080,
}
fmt.Println(config.Host, config.Port)   // output: localhost 8080
```

## Struct tags

Each field may carry a **tag**: a raw string literal after the type. Tags
are metadata — the compiler ignores them, but libraries read them at
runtime via reflection. The canonical use is controlling how
`encoding/json` names fields:

```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email,omitempty"`
}

u := User{Name: "Ada"}
b, _ := json.Marshal(u)
fmt.Println(string(b))   // output: {"name":"Ada"}
```

Here `Email` is dropped because of `omitempty` and its zero (empty)
value. Without tags the keys would be `"Name"` and `"Email"` — the Go
field names. Tags are conventionally backtick-quoted `key:"value"` pairs;
multiple keys are space-separated.

## The empty struct `struct{}`

A struct with no fields occupies **zero bytes**. It carries no data — it
is used purely as a signal. The two common uses are a set (a map whose
values you don't care about) and a channel that signals "an event
happened" without sending a payload:

```go
seen := map[string]struct{}{}
seen["go"] = struct{}{}
_, ok := seen["go"]
fmt.Println(ok)          // output: true
```

`struct{}{}` reads oddly at first: the inner `struct{}` is the *type*
(empty struct), the outer `{}` is the *literal* (a value of that type).

## Quick reference

| Form | Meaning |
|---|---|
| `type T struct { X, Y int }` | declare a named struct type |
| `T{X: 1, Y: 2}` | keyed literal (preferred) |
| `T{1, 2}` | positional literal (order-bound) |
| `var t T` | zero value — all fields zeroed |
| `t.X` | field access (auto-derefs through a `*T`) |
| `a == b` | field-by-field, only if all fields comparable |
| embedded field (type, no name) | promotes the inner fields |
| `` `json:"name"` `` | field tag, read by libraries via reflection |
| `struct{}{}` | the zero-byte empty struct value |

## Sources

- [Struct types — go.dev/ref/spec#Struct_types](https://go.dev/ref/spec#Struct_types)
- [Composite literals — go.dev/ref/spec#Composite_literals](https://go.dev/ref/spec#Composite_literals)
- [Comparison operators — go.dev/ref/spec#Comparison_operators](https://go.dev/ref/spec#Comparison_operators)
- [Struct tags — pkg.go.dev/reflect#StructTag](https://pkg.go.dev/reflect#StructTag)
- [encoding/json#Marshal — pkg.go.dev/encoding/json#Marshal](https://pkg.go.dev/encoding/json#Marshal)
- [Effective Go: embedding — go.dev/doc/effective_go#embedding](https://go.dev/doc/effective_go#embedding)
