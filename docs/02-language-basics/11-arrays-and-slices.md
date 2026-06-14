# Arrays and slices

Go has two sequence types that look similar and behave completely
differently. **Arrays** have a fixed length baked into their type and copy
by value. **Slices** are growable views into an array and are what you
reach for ~99% of the time. Understanding the relationship between them is
the key to using slices without surprises.

## Arrays: fixed length, part of the type

An array's length is part of its type. `[3]int` and `[4]int` are two
*different, incompatible* types.

```go
var a [3]int          // three ints, all zeroed
fmt.Println(a)        // output: [0 0 0]
fmt.Println(len(a))   // output: 3
```

Literals, and `[...]` to let the compiler count:

```go
b := [3]int{10, 20, 30}
c := [...]int{1, 2, 3, 4}     // length inferred as 4
fmt.Println(b, len(c))         // output: [10 20 30] 4
```

Arrays are **value types** — assigning or passing one copies all the
elements:

```go
x := [3]int{1, 2, 3}
y := x          // full copy
y[0] = 99
fmt.Println(x[0], y[0])   // output: 1 99
```

Arrays are comparable with `==` if their element type is:

```go
fmt.Println([2]int{1, 2} == [2]int{1, 2})   // output: true
```

In practice you rarely declare arrays directly. Their fixed size is too
rigid, and the copy-on-pass behaviour surprises people. They mostly show
up as the *backing store* behind a slice, or for fixed-size data like a
hash digest (`[32]byte`).

## Slices: the workhorse

A slice is a lightweight three-word header — a **pointer** to a backing
array, a **length**, and a **capacity** — that describes a contiguous
section of that array. The slice itself holds no elements; it points at
them.

The zero value of a slice is `nil`: length 0, capacity 0, no backing
array. A `nil` slice is safe to read the length of, to `range` over, and
to `append` to.

```go
var s []int           // nil slice — no [N] in the type
fmt.Println(s == nil, len(s))   // output: true 0
```

### Building slices

**Literal** — creates the backing array and the slice in one step:

```go
s := []int{1, 2, 3}
fmt.Println(s, len(s))   // output: [1 2 3] 3
```

**`make`** — allocate a slice of a given length (all zero), optionally
with extra capacity reserved up front:

```go
s := make([]int, 3)        // len 3, cap 3 → [0 0 0]
t := make([]int, 0, 10)    // len 0, cap 10 — empty but room for 10
fmt.Println(len(s), len(t), cap(t))   // output: 3 0 10
```

### Length vs capacity

`len` is how many elements the slice currently holds; `cap` is how many it
can hold before the backing array must be reallocated. Reserving capacity
with `make` avoids repeated reallocation when you know roughly how big the
slice will get.

## `append`: growing a slice

`append` returns a (possibly new) slice — **you must assign the result
back**. If the backing array has spare capacity, `append` writes in place;
if not, it allocates a bigger array, copies the elements over, and returns
a slice pointing at the new array.

```go
s := []int{1, 2}
s = append(s, 3)          // one element
s = append(s, 4, 5)       // several at once
fmt.Println(s)            // output: [1 2 3 4 5]
```

Spread another slice into `append` with `...`:

```go
a := []int{1, 2}
b := []int{3, 4}
a = append(a, b...)
fmt.Println(a)            // output: [1 2 3 4]
```

> **From Python:** `append` is not a method that mutates in place like
> `list.append`. It is a function that *returns* the grown slice, because
> growth may move the data. Forgetting `s = append(s, ...)` is the classic
> beginner bug.

## Slicing: `s[low:high]`

`s[low:high]` produces a new slice header covering indices `low` up to
**but not including** `high`. Both bounds are optional (`s[:2]`, `s[1:]`,
`s[:]`).

```go
s := []int{0, 1, 2, 3, 4}
fmt.Println(s[1:3])   // output: [1 2]
fmt.Println(s[:2])    // output: [0 1]
fmt.Println(s[3:])    // output: [3 4]
```

The crucial part: slicing does **not** copy. The new slice shares the same
backing array, so writing through one is visible through the other.

```go
s := []int{0, 1, 2, 3, 4}
mid := s[1:3]
mid[0] = 99
fmt.Println(s)        // output: [0 99 2 3 4]  — s changed too
```

## The shared-backing-array gotcha

Because slices share storage, `append` can mutate data you didn't expect.
If a sub-slice has spare capacity, appending to it overwrites the
neighbouring elements of the original:

```go
s := []int{1, 2, 3, 4}
head := s[:2]                 // len 2, but cap is still 4
head = append(head, 99)       // writes into s[2] — there's room
fmt.Println(s)                // output: [1 2 99 4]
```

To force an independent copy, either `copy` into a fresh slice or use a
**three-index slice** `s[low:high:max]`, which caps the capacity at
`max-low` so the next `append` is guaranteed to reallocate:

```go
s := []int{1, 2, 3, 4}
head := s[:2:2]               // len 2, cap 2 — capacity capped
head = append(head, 99)       // cap exceeded → new backing array
fmt.Println(s)                // output: [1 2 3 4]  — original untouched
```

## `copy`: explicit element copy

`copy(dst, src)` copies `min(len(dst), len(src))` elements and returns
that count. It is the idiomatic way to duplicate a slice's data:

```go
src := []int{1, 2, 3}
dst := make([]int, len(src))
n := copy(dst, src)
dst[0] = 99
fmt.Println(n, src, dst)   // output: 3 [1 2 3] [99 2 3]
```

## Iterating

`for range` gives index and a **copy** of each element. Drop the value
with `_`, or drop both and keep just the index:

```go
s := []string{"a", "b", "c"}
for i, v := range s {
    fmt.Println(i, v)
}
// output:
// 0 a
// 1 b
// 2 c
```

Because `v` is a copy, assigning to it does nothing to the slice — index
through `s[i]` to mutate.

## Removing an element

There is no `remove` builtin; the idiom is `append` with a spread to close
the gap (order-preserving):

```go
s := []int{10, 20, 30, 40}
i := 1
s = append(s[:i], s[i+1:]...)
fmt.Println(s)            // output: [10 30 40]
```

## Multidimensional slices

Go has no true 2D slice — you build a slice of slices, and each inner
slice is allocated separately, so rows can even have different lengths:

```go
grid := make([][]int, 2)
for i := range grid {
    grid[i] = make([]int, 3)
}
grid[1][2] = 7
fmt.Println(grid)        // output: [[0 0 0] [0 0 7]]
```

## Quick reference

| Operation | Result |
|---|---|
| `[3]int{...}` | array — fixed length, copies by value |
| `[]int{...}` | slice literal |
| `make([]T, n)` | slice of length `n`, zeroed |
| `make([]T, n, c)` | length `n`, capacity `c` |
| `len(s)` / `cap(s)` | current length / backing capacity |
| `s = append(s, x)` | grow (reassign the result!) |
| `s[low:high]` | sub-slice, **shares** backing array |
| `s[low:high:max]` | sub-slice with capped capacity |
| `copy(dst, src)` | copy elements, returns count |
| `append(s[:i], s[i+1:]...)` | delete index `i` |

## Sources

- [Array types — go.dev/ref/spec#Array_types](https://go.dev/ref/spec#Array_types)
- [Slice types — go.dev/ref/spec#Slice_types](https://go.dev/ref/spec#Slice_types)
- [Appending and copying slices — go.dev/ref/spec#Appending_and_copying_slices](https://go.dev/ref/spec#Appending_and_copying_slices)
- [Slice expressions — go.dev/ref/spec#Slice_expressions](https://go.dev/ref/spec#Slice_expressions)
- [Go blog: slices intro — go.dev/blog/slices-intro](https://go.dev/blog/slices-intro)
- [Go blog: arrays and slices usage — go.dev/blog/slices](https://go.dev/blog/slices)
