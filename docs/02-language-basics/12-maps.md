# Maps

A **map** is Go's built-in hash table: an unordered collection of
key→value pairs with average O(1) lookup, insert, and delete. The type is
written `map[K]V` — `K` is the key type, `V` the value type.

```go
ages := map[string]int{
    "alice": 30,
    "bob":   25,
}
fmt.Println(ages["alice"])   // output: 30
```

## Keys must be comparable

A key type must support `==`: that covers all the basic types (strings,
numbers, booleans), pointers, and structs/arrays whose fields are
themselves comparable. Slices, maps, and functions are **not** comparable
and so cannot be keys — trying makes it a compile error.

```go
m := map[[]int]string{}   // compile error: invalid map key type []int
```

Value types have no such restriction — `map[string][]int` is fine.

### A struct as the key

Because a comparable struct supports `==`, it works as a key directly —
handy for a **composite key** (a sparse grid, memoization keyed on several
values, deduplication by tuple). The lookup compares the key field-by-field,
so equal structs map to the same entry.

```go
type Point struct{ X, Y int }

grid := map[Point]string{
    {0, 0}: "origin",     // inside the literal you can drop the "Point"
    {1, 2}: "somewhere",
}
fmt.Println(grid[Point{X: 1, Y: 2}])   // output: somewhere

v, ok := grid[Point{9, 9}]
fmt.Println(v, ok)                     // output:  false
```

If the struct has any non-comparable field (a slice, map, or function),
the whole type is non-comparable and the map declaration itself is a
compile error: `invalid map key type`.

## Creating maps

**Literal**, including the empty literal `map[K]V{}`:

```go
m := map[string]int{"a": 1, "b": 2}
empty := map[string]int{}      // non-nil, ready to use
```

**`make`** — an empty map ready for writes:

```go
m := make(map[string]int)
m["x"] = 1
fmt.Println(m["x"])    // output: 1
```

### The nil map trap

The zero value of a map is `nil`. You can **read** a nil map (you get zero
values) and take its `len` — but **writing to a nil map panics**.

```go
var m map[string]int     // nil
fmt.Println(m["missing"], len(m))   // output: 0 0  — reads are fine
m["x"] = 1               // panic: assignment to entry in nil map
```

Always initialise with `make` or a literal before writing. Declaring
`var m map[string]int` and forgetting to `make` it is the classic map bug.

## Reading: the missing-key zero value

Indexing a key that isn't present returns the value type's **zero value**,
not an error and not a panic:

```go
ages := map[string]int{"alice": 30}
fmt.Println(ages["charlie"])   // output: 0  — absent, so zero
```

That's ambiguous: did `charlie` map to `0`, or is he absent? Use the
**comma-ok** form to tell them apart — the second value is a bool.

```go
ages := map[string]int{"alice": 30}
v, ok := ages["charlie"]
fmt.Println(v, ok)             // output: 0 false

v, ok = ages["alice"]
fmt.Println(v, ok)             // output: 30 true
```

> **From Python:** indexing a missing key does **not** raise `KeyError`.
> It quietly returns the zero value — closer to `dict.get(key, default)`
> than `dict[key]`. Reach for comma-ok when "absent" and "present-but-zero"
> must be distinguished.

## Updating, deleting, sizing

```go
m := map[string]int{"a": 1}
m["a"] = 100          // overwrite
m["b"] = 2            // insert
delete(m, "a")        // remove; no-op if key absent, never panics
fmt.Println(len(m), m["b"])   // output: 1 2
```

## Iteration order is randomised

`for range` visits every pair, but the order is **deliberately
randomised** — it differs from run to run. Never rely on map order. To
iterate in a stable order, collect the keys into a slice and sort it.

```go
m := map[string]int{"a": 1, "b": 2, "c": 3}

keys := make([]string, 0, len(m))
for k := range m {           // one variable → keys only
    keys = append(keys, k)
}
sort.Strings(keys)
for _, k := range keys {
    fmt.Println(k, m[k])
}
// output:
// a 1
// b 2
// c 3
```

Ranging with one variable yields keys; with two, keys and values.

## Maps are reference-like

A map value is a small header pointing at the underlying hash table.
Copying a map — assigning it or passing it to a function — copies that
header, **not** the data, so both names refer to the same table.
Mutations through one are visible through the other.

```go
func add(m map[string]int) {
    m["new"] = 1          // mutates the caller's map
}

m := map[string]int{}
add(m)
fmt.Println(m["new"])     // output: 1
```

This is unlike structs and arrays (which copy wholesale). There is no
"copy a map" builtin — to get an independent copy you allocate a new map
and copy entries in a loop.

## A set via `map[T]struct{}`

Go has no built-in set type. The idiom is a map with a zero-byte
`struct{}` value, so only the keys carry meaning:

```go
set := map[string]struct{}{}
set["go"] = struct{}{}
set["go"] = struct{}{}        // idempotent
_, exists := set["go"]
fmt.Println(exists, len(set)) // output: true 1
```

Using `map[string]bool` is a common, slightly heavier alternative that
reads a touch more naturally (`set["go"] = true`).

## Quick reference

| Operation | Code |
|---|---|
| literal | `map[string]int{"a": 1}` |
| empty, ready to write | `make(map[string]int)` or `map[string]int{}` |
| read (zero if absent) | `v := m[k]` |
| read with presence | `v, ok := m[k]` |
| insert / update | `m[k] = v` |
| delete | `delete(m, k)` |
| size | `len(m)` |
| iterate (random order) | `for k, v := range m` |
| **panic** | writing to a `nil` map |

## Sources

- [Map types — go.dev/ref/spec#Map_types](https://go.dev/ref/spec#Map_types)
- [Index expressions — go.dev/ref/spec#Index_expressions](https://go.dev/ref/spec#Index_expressions)
- [The `delete` builtin — go.dev/ref/spec#Deletion_of_map_elements](https://go.dev/ref/spec#Deletion_of_map_elements)
- [For statements with range — go.dev/ref/spec#For_range](https://go.dev/ref/spec#For_range)
- [Go blog: Go maps in action — go.dev/blog/maps](https://go.dev/blog/maps)
