# Choosing a data structure

You now have four building blocks for modelling data: **custom types**,
**structs**, **slices**, and **maps** (plus arrays, which you rarely reach
for directly). Real programs combine them. This article is a practical
guide to picking the right one — and the right *combination* — for a given
job.

## The one-line guide

| You need to… | Reach for |
|---|---|
| keep an ordered sequence, iterate, allow duplicates, index by position | **slice** `[]T` |
| look something up by key, test membership, count, or group | **map** `map[K]V` |
| describe one *thing* with a fixed set of named, differently-typed fields | **struct** |
| give a value a distinct name/behaviour over an existing type (units, IDs, enums) | **custom type** `type T U` |
| a fixed-size, comparable, value-copied buffer | **array** `[N]T` (rare) |

The rest of the article expands each of these and then shows how they
compose.

## Slices — the default collection

If you just need "a bunch of values in order," use a slice. It is the
workhorse: ordered, growable with `append`, cheap to iterate, and laid out
contiguously in memory (cache-friendly).

```go
scores := []int{90, 85, 90}
scores = append(scores, 70)
total := 0
for _, s := range scores {
    total += s
}
fmt.Println(total)   // output: 335
```

Its weakness is **lookup**: finding whether a value is present, or finding
it by some key, is an O(n) scan. That's fine for small or rarely-searched
collections — don't reach for a map just because you have a slice.

## Maps — keyed access

Use a map when you look things up by a key, rather than walk the whole
collection. Average O(1) get/set. Four classic jobs:

**Lookup by key:**

```go
age := map[string]int{"alice": 30, "bob": 25}
fmt.Println(age["bob"])   // output: 25
```

**Counting (frequency):** the missing-key zero value makes `++` just work:

```go
counts := map[string]int{}
for _, w := range []string{"go", "py", "go"} {
    counts[w]++
}
fmt.Println(counts["go"])   // output: 2
```

**Grouping (one-to-many)** with `map[K][]V`:

```go
byParity := map[string][]int{}
for _, n := range []int{1, 2, 3, 4} {
    if n%2 == 0 {
        byParity["even"] = append(byParity["even"], n)
    } else {
        byParity["odd"] = append(byParity["odd"], n)
    }
}
fmt.Println(byParity["even"])   // output: [2 4]
```

**Set / membership** with `map[T]struct{}` (or `map[T]bool`):

```go
seen := map[string]struct{}{}
seen["go"] = struct{}{}
_, ok := seen["go"]
fmt.Println(ok)   // output: true
```

Map weaknesses: iteration order is randomised, there's per-entry hashing
overhead, and a map is the wrong tool for a *fixed, known* set of
fields — that's a struct's job.

## Structs — one thing with named fields

When a value has a **fixed, known shape** — several attributes of
different types that belong together — use a struct. The field names and
types are checked by the compiler.

```go
type User struct {
    Name string
    Age  int
}
u := User{Name: "Ada", Age: 36}
fmt.Println(u.Name)   // output: Ada
```

Resist modelling a known record as `map[string]any`: you lose type
safety, field-name checking, and clarity. A map is for *open-ended* keys;
a struct is for a *closed* set of fields.

## Custom types — meaning and behaviour

Use a defined type (`type T U`) when an underlying type isn't enough on
its own: to stop incompatible values mixing, to attach methods, or to
build an enumeration with `iota`.

```go
type Celsius float64
type Fahrenheit float64    // distinct: can't accidentally add the two

type Suit int
const (
    Spades Suit = iota
    Hearts
    Diamonds
    Clubs
)
fmt.Println(Hearts)   // output: 1
```

## Combining them — the real patterns

Most data models are compositions. The common ones:

| Pattern | Meaning |
|---|---|
| `[]User` | an ordered list of records — *the* everyday shape |
| `map[int]User` | records indexed by id, for O(1) lookup |
| `map[string][]Order` | one-to-many grouping (orders per customer) |
| `map[string]struct{}` | a set |
| a struct with slice/map fields | an aggregate that owns collections |

A struct that owns collections, indexed for fast access:

```go
type Library struct {
    Name  string
    Books []string
}
shelf := map[string]Library{}
shelf["scifi"] = Library{Name: "Sci-Fi", Books: []string{"Dune"}}
fmt.Println(shelf["scifi"].Books[0])   // output: Dune
```

## Head-to-head: which works better when

**Slice vs. map for lookup.** For a handful of elements, a linear scan
over a slice is simpler and often *faster* (no hashing, cache-friendly).
Switch to a map when the collection is large or you look up by key
frequently.

**Map vs. struct.** Known field names → struct (compile-time safety).
Keys decided at runtime → map. Don't fake a struct with `map[string]any`.

**Array vs. slice.** Default to a slice. Use an array only when the length
is genuinely fixed and part of the type, or you want value-copy /
comparability (e.g. `[32]byte` for a hash).

**Custom type vs. raw primitive.** Wrap a primitive when its *meaning*
matters (a `UserID` shouldn't be addable to an `OrderID`) or when you want
methods on it. Otherwise the plain type is fine.

> **From Python:** the mapping is loose — Go's slice ≈ `list`, map ≈
> `dict`, struct ≈ a `@dataclass` (or a class with fixed attributes), and
> the `map[T]struct{}` set ≈ `set`. The big difference is that Go fixes a
> struct's fields at compile time, whereas a Python object can grow
> attributes at will.

## A worked example

Counting word lengths groups several of these together — a map keyed by
length, whose values are slices of structs:

```go
type Word struct {
    Text   string
    Length int
}
byLen := map[int][]Word{}
for _, t := range []string{"go", "rust", "py", "java"} {
    w := Word{Text: t, Length: len(t)}
    byLen[w.Length] = append(byLen[w.Length], w)
}
fmt.Println(byLen[2][0].Text, byLen[2][1].Text)   // output: go py
```

That's `map[int][]Word`: a **map** for keyed grouping, a **slice** for the
ordered group, and a **struct** for each record — each chosen for what it
does best.

> When different *types* need to share behaviour behind one abstraction,
> none of these is the answer — that's what **interfaces** are for,
> covered in [a later article](../03-object-oriented-go/02-interfaces.md).

## Cheat sheet

- Ordered, iterate, duplicates → **slice**
- Lookup / membership / count / group → **map**
- Fixed named fields for one entity → **struct**
- Distinct meaning, methods, or enum → **custom type**
- Fixed-size comparable buffer → **array**
- Real models **combine** these — `[]Struct`, `map[K]Struct`,
  `map[K][]V`, `map[K]struct{}`.

## Sources

- [Effective Go: data — go.dev/doc/effective_go#data](https://go.dev/doc/effective_go#data)
- [Slice types — go.dev/ref/spec#Slice_types](https://go.dev/ref/spec#Slice_types)
- [Map types — go.dev/ref/spec#Map_types](https://go.dev/ref/spec#Map_types)
- [Struct types — go.dev/ref/spec#Struct_types](https://go.dev/ref/spec#Struct_types)
- [Type definitions — go.dev/ref/spec#Type_definitions](https://go.dev/ref/spec#Type_definitions)
- [Go blog: arrays, slices — go.dev/blog/slices-intro](https://go.dev/blog/slices-intro)
