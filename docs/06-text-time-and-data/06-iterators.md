# Iterators

The [control flow](../02-language-basics/05-control-flow.md) article
showed `range` over a function and promised the producing side later.
This is it. An iterator in Go is just a function you can `range` over,
and writing one is less machinery than it sounds.

```go
for v := range Countdown(3) {
    fmt.Print(v, " ")
}
// output: 3 2 1
```

## The shape

Two type aliases in the `iter` package name the signatures:

```go
type Seq[V any]     func(yield func(V) bool)
type Seq2[K, V any] func(yield func(K, V) bool)
```

An iterator is a function that *takes* a function. `range` supplies the
`yield`, your code calls it once per element, and the loop body is what
runs inside `yield`:

```go
func Countdown(n int) iter.Seq[int] {
    return func(yield func(int) bool) {
        for i := n; i > 0; i-- {
            if !yield(i) {
                return
            }
        }
    }
}
```

The direction is inside out compared with most languages: the loop does
not pull values from you, you push values into the loop.

## `yield` returns false when the loop stops

That `if !yield(i) { return }` is the whole contract. A `break`,
`return`, or `panic` in the loop body makes `yield` return `false`, and
you must stop and return:

```go
for v := range Countdown(10) {
    if v < 8 {
        break
    }
    fmt.Print(v, " ")
}
// output: 10 9 8
```

Ignoring that result is the one real bug you can write here, and the
runtime catches it:

```go
func Bad(n int) iter.Seq[int] {
    return func(yield func(int) bool) {
        for i := n; i > 0; i-- {
            yield(i)   // result ignored
        }
    }
}
// panic: runtime error: range function continued iteration after
//        function for loop body returned false
```

A corollary: the loop body can exit at any time, so a resource you open
before yielding still needs `defer` to be released when a caller breaks
early.

## Two values: `Seq2`

`Seq2` is the same idea with a pair, which is what `range` over a map or
slice already gives you:

```go
func Enumerate[T any](s []T) iter.Seq2[int, T] {
    return func(yield func(int, T) bool) {
        for i, v := range s {
            if !yield(i, v) {
                return
            }
        }
    }
}

for i, v := range Enumerate([]string{"a", "b"}) {
    fmt.Printf("%d=%s ", i, v)
}
// output: 0=a 1=b
```

## Iterators compose

Because an iterator is an ordinary value, a function can take one and
return another. Nothing is buffered — values still flow one at a time:

```go
func Filter[T any](seq iter.Seq[T], keep func(T) bool) iter.Seq[T] {
    return func(yield func(T) bool) {
        for v := range seq {
            if keep(v) && !yield(v) {
                return
            }
        }
    }
}

even := Filter(Countdown(6), func(n int) bool { return n%2 == 0 })
fmt.Println(slices.Collect(even))   // output: [6 4 2]
```

## Where they pay off: recursive structures

Exposing a tree's contents used to mean building a slice or accepting a
callback. An iterator gives callers a plain `for` loop over a structure
that is awkward to walk:

```go
type Tree struct {
    Val         int
    Left, Right *Tree
}

func (t *Tree) All() iter.Seq[int] {
    return func(yield func(int) bool) {
        t.walk(yield)
    }
}

func (t *Tree) walk(yield func(int) bool) bool {
    if t == nil {
        return true
    }
    return t.Left.walk(yield) && yield(t.Val) && t.Right.walk(yield)
}
```

```go
tr := &Tree{Val: 2, Left: &Tree{Val: 1}, Right: &Tree{Val: 3}}
fmt.Println(slices.Collect(tr.All()))   // output: [1 2 3]
```

The `&&` chain does double duty: it sequences left, self, right, and it
short-circuits the moment `yield` returns `false`. Note the method calls
on a nil `*Tree` — safe, as [methods](../03-object-oriented-go/01-methods.md)
explains.

## The standard library produces and consumes them

You have already used these. `maps.Keys` returns an iterator, which is
why it pairs with `slices.Sorted`:

```go
m := map[string]int{"b": 2, "a": 1}
fmt.Println(slices.Sorted(maps.Keys(m)))   // output: [a b]
```

| Producer | Gives |
|---|---|
| `slices.Values(s)` | each element |
| `slices.All(s)` | index and element |
| `slices.Backward(s)` | index and element, last to first |
| `maps.Keys(m)` / `maps.Values(m)` | keys / values |
| `maps.All(m)` | key and value |
| `strings.SplitSeq(s, sep)` | pieces, without allocating a slice |

| Consumer | Gives |
|---|---|
| `slices.Collect(seq)` | a `[]T` |
| `slices.Sorted(seq)` | a sorted `[]T` |
| `maps.Collect(seq2)` | a `map[K]V` |

```go
fmt.Println(slices.Collect(slices.Values([]int{1, 2, 3})))     // output: [1 2 3]
fmt.Println(slices.Collect(strings.SplitSeq("a,b,c", ",")))    // output: [a b c]

for i, v := range slices.Backward([]int{1, 2, 3}) {
    fmt.Printf("%d:%d ", i, v)
}
// output: 2:3 1:2 0:1
```

`SplitSeq` is the point of the whole feature in miniature: `strings.Split`
allocates a slice you then throw away, while `SplitSeq` hands you the
pieces as it finds them.

## `iter.Pull` when you need to drive

Sometimes you cannot use a `for` loop — you want to advance two
sequences in step. `iter.Pull` turns a push iterator into a `next`
function:

```go
next, stop := iter.Pull(slices.Values([]int{1, 2, 3}))
defer stop()

for {
    v, ok := next()
    if !ok {
        break
    }
    fmt.Print(v, " ")
}
// output: 1 2 3
```

`stop` must always be called — hence the `defer` — because `Pull` runs
the iterator in a separate goroutine and `stop` is what releases it.
With two of them you can zip:

```go
a, stopA := iter.Pull(slices.Values([]string{"x", "y"}))
defer stopA()
b, stopB := iter.Pull(slices.Values([]int{10, 20}))
defer stopB()

for {
    s, ok1 := a()
    n, ok2 := b()
    if !ok1 || !ok2 {
        break
    }
    fmt.Printf("%s=%d ", s, n)
}
// output: x=10 y=20
```

`Pull` costs more than ranging directly, so use it only when the control
flow genuinely demands it.

## When not to write one

If you already have a slice, return the slice. An iterator earns its
place when the sequence is expensive, unbounded, or awkward to
materialise — a tree walk, a paged API, lines of a large file. For a
handful of values in memory it is indirection for its own sake.

> **From Python:** this is a generator, but built the other way round.
> Python's `yield` suspends your function; Go's `yield` is a callback the
> loop gives you, and returning `false` is what `GeneratorExit` does.
> `iter.Pull` is the closest thing to holding the generator object and
> calling `next()` yourself.

## Quick reference

| Form | Meaning |
|---|---|
| `iter.Seq[V]` | `func(yield func(V) bool)` |
| `iter.Seq2[K, V]` | `func(yield func(K, V) bool)` |
| `if !yield(v) { return }` | stop when the loop body breaks — mandatory |
| `slices.Collect(seq)` | drain into a slice |
| `slices.Sorted(maps.Keys(m))` | sorted map keys |
| `iter.Pull(seq)` | `next, stop` — always `defer stop()` |

## Sources

- [`iter` package reference — pkg.go.dev/iter](https://pkg.go.dev/iter)
- [`slices` iterator functions — pkg.go.dev/slices](https://pkg.go.dev/slices)
- [`maps` iterator functions — pkg.go.dev/maps](https://pkg.go.dev/maps)
- [For statements with range clause — go.dev/ref/spec#For_range](https://go.dev/ref/spec#For_range)
- [Go blog: range over function types — go.dev/blog/range-functions](https://go.dev/blog/range-functions)
