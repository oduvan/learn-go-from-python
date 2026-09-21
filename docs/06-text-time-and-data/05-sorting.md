# Sorting

Sorting lives in two packages. `slices` is the one to use; `sort` is the
older API you will still meet in existing code. The comparison helpers in
`cmp` are what make the new one pleasant.

```go
s := []int{3, 1, 2}
slices.Sort(s)
fmt.Println(s)   // output: [1 2 3]
```

## `slices.Sort` for ordered types

`slices.Sort` works on any slice whose elements have a natural order —
numbers, strings, and anything whose underlying type is one of those. It
sorts in place and returns nothing:

```go
w := []string{"pear", "Apple", "fig"}
slices.Sort(w)
fmt.Println(w)   // output: [Apple fig pear]
```

`Apple` sorts first because comparison is byte-wise and `A` is 65 while
`f` is 102. Sorting strings is not alphabetical ordering, and it is not
locale-aware.

## `slices.SortFunc` returns an int, not a bool

For anything else you supply a comparison. It returns a **negative
number, zero, or a positive number** — not a boolean:

```go
type user struct {
    Name string
    Age  int
}

us := []user{{"Bo", 30}, {"Ada", 25}, {"Cy", 30}}
slices.SortFunc(us, func(a, b user) int {
    return cmp.Compare(a.Age, b.Age)
})
fmt.Println(us)   // output: [{Ada 25} {Bo 30} {Cy 30}]
```

`cmp.Compare` produces exactly that three-way result, so you rarely
write the comparison by hand:

```go
fmt.Println(cmp.Compare(1, 2), cmp.Compare(2, 2), cmp.Compare(3, 2))
// output: -1 0 1
```

To sort descending, swap the arguments rather than negating the result —
negation misbehaves at the extreme of an integer range:

```go
d := []int{3, 1, 2}
slices.SortFunc(d, func(a, b int) int { return cmp.Compare(b, a) })
fmt.Println(d)   // output: [3 2 1]
```

Sorting case-insensitively is a comparison on transformed values:

```go
w := []string{"pear", "Apple", "fig"}
slices.SortFunc(w, func(a, b string) int {
    return cmp.Compare(strings.ToLower(a), strings.ToLower(b))
})
fmt.Println(w)   // output: [Apple fig pear]
```

## `cmp.Or` for a second sort key

`cmp.Or` returns its first non-zero argument. Since "equal" is zero, it
expresses "sort by age, then by name" directly:

```go
slices.SortFunc(us, func(a, b user) int {
    return cmp.Or(
        cmp.Compare(a.Age, b.Age),
        cmp.Compare(a.Name, b.Name),
    )
})
fmt.Println(us)   // output: [{Ada 25} {Bo 30} {Cy 30}]
```

It is not sorting-specific — it is the general "first non-zero value"
helper, which also reads well for defaults:

```go
fmt.Println(cmp.Or(0, 0, -1, 5))      // output: -1
fmt.Println(cmp.Or("", "fallback"))   // output: fallback
```

## Stable sorting

`slices.Sort` and `SortFunc` may reorder elements your comparison calls
equal. `SortStableFunc` keeps their original relative order:

```go
us := []user{{"Bo", 30}, {"Ada", 25}, {"Cy", 30}}
slices.SortStableFunc(us, func(a, b user) int { return cmp.Compare(a.Age, b.Age) })
fmt.Println(us)   // output: [{Ada 25} {Bo 30} {Cy 30}]
```

Bo stays ahead of Cy because it started that way. Stability costs
performance, so take it only when you need it — for instance when
sorting by one column on top of an existing order. The alternative is to
make the comparison total with `cmp.Or`, which is usually clearer.

## Searching a sorted slice

```go
fmt.Println(slices.IsSorted([]int{1, 2, 3}))    // output: true
fmt.Println(slices.BinarySearch([]int{1, 3, 5}, 3))   // output: 1 true
```

`BinarySearchFunc` takes a *target* that need not be the element type,
so you can search by one field. Its comparison receives the element
first and the target second:

```go
fmt.Println(slices.BinarySearchFunc(us, 30, func(u user, age int) int {
    return cmp.Compare(u.Age, age)
}))
// output: 1 true
```

The slice must already be sorted by the same ordering, or the result is
meaningless — this is never checked for you.

## Minimum and maximum

`min` and `max` are builtins taking any number of arguments. The `slices`
versions take a slice and panic on an empty one:

```go
fmt.Println(max(3, 1, 2), min(3, 1, 2))                 // output: 3 1
fmt.Println(slices.Max([]int{3, 1, 2}), slices.Min([]int{3, 1, 2}))
// output: 3 1
```

## The older `sort` package

`sort` predates generics. You will meet `sort.Slice`, which takes a
**less** function of indices returning a bool:

```go
l := []int{3, 1, 2}
sort.Slice(l, func(i, j int) bool { return l[i] < l[j] })
fmt.Println(l)   // output: [1 2 3]

ss := []string{"b", "a"}
sort.Strings(ss)
fmt.Println(ss)   // output: [a b]
```

Note it closes over the slice rather than receiving elements, which is
easy to get wrong after a reassignment. There is also `sort.Interface`,
requiring `Len`, `Less` and `Swap` methods on a named type.

Prefer `slices` in new code. The two differ in three ways worth keeping
straight: `slices` compares **elements**, `sort` compares **indices**;
`slices` wants a three-way `int`, `sort` wants a `bool`; and `slices` is
type-safe without a wrapper type.

> **From Python:** `slices.Sort` is `list.sort()`. The big difference is
> that Go has no `key=` — there is no `sort(key=lambda u: u.age)`, so you
> write a comparison instead, and `cmp.Or` is how you express what a
> tuple key would have done in Python. Note the direction: `cmp.Compare`
> returns three-way like the comparator you would hand to
> `functools.cmp_to_key`, whereas the older `sort.Slice` wants a plain
> "is a less than b" bool.

## Quick reference

| Task | Call |
|---|---|
| sort numbers or strings | `slices.Sort(s)` |
| sort by a field | `slices.SortFunc(s, func(a, b T) int { ... })` |
| three-way compare | `cmp.Compare(a, b)` → `-1`, `0`, `1` |
| descending | `cmp.Compare(b, a)` — swap, do not negate |
| second sort key | `cmp.Or(cmp.Compare(...), cmp.Compare(...))` |
| keep equal elements in order | `slices.SortStableFunc` |
| is it sorted | `slices.IsSorted`, `slices.IsSortedFunc` |
| find in a sorted slice | `slices.BinarySearch`, `BinarySearchFunc` |
| largest / smallest | `max(a, b, c)`, `slices.Max(s)` |
| older code | `sort.Slice` (bool over indices), `sort.Interface` |

## Sources

- [`slices` package reference — pkg.go.dev/slices](https://pkg.go.dev/slices)
- [`cmp` package reference — pkg.go.dev/cmp](https://pkg.go.dev/cmp)
- [`sort` package reference — pkg.go.dev/sort](https://pkg.go.dev/sort)
- [`slices.SortFunc` — pkg.go.dev/slices#SortFunc](https://pkg.go.dev/slices#SortFunc)
- [`cmp.Or` — pkg.go.dev/cmp#Or](https://pkg.go.dev/cmp#Or)
