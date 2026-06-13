# Pointers

A pointer holds the **memory address** of a variable. You've already
seen `&` and `*` mentioned briefly in
[04-operators.md](04-operators.md); this article explains what they
actually do.

## Two operators, one type prefix

| Syntax | What it means |
|---|---|
| `*T` | the **type** "pointer to T" — appears in declarations and signatures |
| `&x` | the **address-of** operator — takes a pointer to the variable `x` |
| `*p` | the **dereference** operator — reads (or writes) the value at the address in `p` |

```go
var x int = 5
p := &x                 // p has type *int and points at x
fmt.Println(p)          // 0xc000018058 (some address)
fmt.Println(*p)         // 5 — the value through the pointer

*p = 42                 // write through the pointer
fmt.Println(x)          // 42 — x changed
```

## The zero value of a pointer is `nil`

A declared-but-uninitialised pointer is `nil`:

```go
var p *int
fmt.Println(p)          // <nil>
fmt.Println(p == nil)   // true
```

Dereferencing a `nil` pointer **panics at runtime**:

```go
var p *int
fmt.Println(*p)         // runtime error: invalid memory address or nil pointer dereference
```

This is the most common cause of Go panics in practice. Always check
`p != nil` if there's any chance a pointer wasn't initialised.

## `new(T)` — allocate and return a pointer

`new(T)` is a built-in that allocates a fresh zero-valued `T` and
returns its address. The pointer is non-nil.

```go
p := new(int)           // p is *int, *p == 0
*p = 99
fmt.Println(*p)         // 99
```

In practice you'll see `new` less often than `&Struct{...}` because
composite literals let you initialise fields at the same time.
`new(int)` is fine when you just want a writable integer behind a
pointer.

## Why pointers exist at all

Go usually passes arguments by **value** — the function receives a
copy. Two reasons to use a pointer instead:

1. **You want the function to modify the caller's variable.**

   ```go
   func zero(x int)   { x = 0 }    // operates on its own copy
   func zeroP(x *int) { *x = 0 }   // writes through the caller's pointer

   n := 7
   zero(n)
   fmt.Println(n)        // 7 — zero saw a copy; n unchanged

   zeroP(&n)
   fmt.Println(n)        // 0 — zeroP wrote through the pointer
   ```

2. **You want to avoid copying a large struct.** Passing a pointer
   transfers 8 bytes; passing a 1 KB struct by value transfers all
   1 KB.

   ```go
   type Snapshot struct { /* many fields, big */ }

   func process(s *Snapshot) { /* ... */ }     // cheap call
   ```

> **From Python:** Python doesn't expose pointers, but it does
> distinguish *mutable* (`list`, `dict`, custom classes) from
> *immutable* (`int`, `tuple`, `str`) values. Mutable Python objects
> behave a bit like Go values held behind an implicit pointer — the
> function and the caller share the same object. In Go, the sharing
> is explicit: you pass a `*T`.

## No pointer arithmetic

Unlike C, you cannot do `p++`, `p + 1`, or `p[3]` on a pointer.
The compiler rejects them outright.

```go
p := &x
p++                     // compile error: invalid operation: p++ (non-numeric type *int)
```

If you need pointer-like traversal over memory, use a slice — its
runtime knows the bounds and will panic on out-of-range access
instead of corrupting memory.

## Pointers vs. references

Some Go terms (slices, maps, channels, functions) are themselves
*reference-like* under the hood: copying a slice header doesn't copy
the underlying array. So you almost never need `*[]int` — passing a
plain `[]int` already shares the backing array.

| Type | Pass by value gives caller a shared view? |
|---|---|
| `int`, `bool`, `float64`, arrays, structs | **No** — full copy |
| `string` | Yes (string headers are tiny; data is immutable) |
| `[]T` (slice) | Yes |
| `map[K]V` | Yes |
| `chan T` | Yes |
| `func(...)` | Yes |
| Interfaces | Yes (interface values are essentially `(type, *value)` pairs) |

Use `*T` when the caller-modification or copy-avoidance reason
applies, and the type isn't already reference-like.

## Pointer comparison

Two pointers are `==` when they point at the same address (or are
both `nil`). They are **not** compared by the values they reference.

```go
a, b := 1, 1
fmt.Println(&a == &b)   // false — different variables
fmt.Println(&a == &a)   // true — same address

var x int
p := &x
q := &x
fmt.Println(p == q)     // true
```

To compare what they point at: `*p == *q`.

## Common gotcha — the address of a loop variable

A `for ... range` loop assigns to a fresh variable each iteration in
modern Go, so taking `&v` inside the loop *is* safe today. Older code
sometimes shows the opposite — be careful when reading legacy
examples.

```go
nums := []int{10, 20, 30}
var ps []*int
for _, v := range nums {
    ps = append(ps, &v)
}
for _, p := range ps {
    fmt.Print(*p, " ")  // 10 20 30 — each pointer points at its own copy
}
fmt.Println()
```

## Quick reference

| You want | Write |
|---|---|
| Declare a pointer-typed variable | `var p *int` |
| Take a pointer to an existing variable | `p := &x` |
| Read or write through a pointer | `*p`, `*p = newValue` |
| Check for nil | `if p != nil { ... }` |
| Allocate a fresh zero value behind a pointer | `p := new(int)` |
| Pass a struct cheaply | `func f(s *BigStruct)` |
| Let a function mutate a caller's variable | `func reset(n *int) { *n = 0 }` |

## Sources

- [Pointer types — go.dev/ref/spec#Pointer_types](https://go.dev/ref/spec#Pointer_types)
- [Address operators — go.dev/ref/spec#Address_operators](https://go.dev/ref/spec#Address_operators)
- [Allocation with `new` — go.dev/ref/spec#Allocation](https://go.dev/ref/spec#Allocation)
- [Tour of Go: Pointers — go.dev/tour/moretypes/1](https://go.dev/tour/moretypes/1)
