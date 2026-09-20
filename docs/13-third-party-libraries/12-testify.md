# testify

Assertions with readable failure messages. Go's
[`testing` package](../10-testing/01-the-testing-package.md) gives you
`if` and `t.Errorf`; testify gives you one line per check and a diff
when it fails.

> **Module:** `github.com/stretchr/testify`.

```go
require.NoError(t, err)
assert.Equal(t, "Ada", u.Name)
```

## `assert` stops nothing, `require` stops everything

Two packages, the same function names, one difference:

| Package | On failure |
|---|---|
| `assert` | records it and **continues** — like `t.Errorf` |
| `require` | records it and **stops the test** — like `t.Fatalf` |

The rule follows from that: `require` for anything the rest of the test
depends on, `assert` for independent checks.

```go
u, err := Load("1")
require.NoError(t, err)          // stop here if it failed
require.Equal(t, "Ada", u.Name)

assert.Len(t, u.Tags, 2)         // independent checks
assert.Contains(t, u.Tags, "a")
```

Using `assert.NoError` where `require.NoError` belongs is the common
mistake — the test continues into a nil dereference, and the panic
buries the real message.

## Argument order

```go
assert.Equal(t, expected, actual)
```

`t` first, then **expected, then actual**. Get it backwards and the
test still passes or fails correctly, but the diff is inverted and
reads as a lie.

## What a failure looks like

```
Error Trace:	lib_test.go:27
Error:      	Not equal:
            	expected: "Bob"
            	actual  : "Ada"

            	Diff:
            	--- Expected
            	+++ Actual
            	@@ -1 +1 @@
            	-Bob
            	+Ada
Test:       	TestFailureOutput
Messages:   	name should match the seeded value
```

This is the reason to use it. On a struct or a slice, the diff shows
the differing field rather than dumping both values and leaving you to
compare.

Every assertion takes an optional trailing message:

```go
assert.Equal(t, "Ada", u.Name, "name should match the seeded value")
```

Worth adding when the assertion alone does not say *why* it matters.

## The assertions you will actually use

```go
require.NoError(t, err)
require.Error(t, err)
assert.ErrorIs(t, err, ErrEmpty)      // unwraps, like errors.Is
assert.ErrorAs(t, err, &target)

assert.Equal(t, want, got)
assert.NotEqual(t, a, b)
assert.Nil(t, v)
assert.True(t, ok)

assert.Len(t, u.Tags, 2)
assert.Empty(t, list)
assert.Contains(t, u.Tags, "a")
assert.ElementsMatch(t, []string{"b", "a"}, u.Tags)   // ignores order

assert.InDelta(t, 3.14159, 3.1416, 0.001)             // floats
assert.JSONEq(t, `{"a":1,"b":2}`, `{"b":2,"a":1}`)    // ignores key order
```

`ErrorIs` is the one to reach for on errors — it unwraps, so it works
through `fmt.Errorf("...: %w", err)`.

`ElementsMatch` and `JSONEq` remove two whole categories of flaky test:
slice ordering and JSON key ordering, neither of which you usually mean
to assert.

`InDelta` exists because `assert.Equal` on floats compares exactly, and
exact float comparison is a bug.

## `Equal` distinguishes nil from empty

`assert.Equal` uses `reflect.DeepEqual` semantics, so:

```go
var a []string
b := []string{}
assert.Equal(t, a, b)
```

```
Error: Not equal:
       expected: []string(nil)
       actual  : []string{}
```

A nil slice and an empty slice behave identically in every Go
operation — `len`, `range`, `append` — and differ here. When you mean
"no elements", use `assert.Empty`, which accepts both.

The same applies to maps. This trips people constantly when comparing
a decoded JSON structure against a literal.

## Do not call `require` from a goroutine

`require` fails via `t.FailNow`, which calls `runtime.Goexit`. From a
goroutine your test started, that kills only that goroutine — the test
does not fail properly and may hang waiting for it.

Send the result back to the test goroutine and assert there, or use
`assert` (which only records) inside the goroutine.

## The rest of the library

`testify/mock` generates expectation-based mocks, and `testify/suite`
adds xUnit-style setup and teardown classes. Plenty of Go codebases use
`assert` and `require` and nothing else, for the reasons in
[fakes and stubs](../10-testing/04-fakes-and-stubs.md): hand-written
doubles are clearer than `EXPECT().Times(1)`, and `t.Cleanup` covers
what a suite's teardown would.

Adopt the two assertion packages first. Add the others only if you hit
something they genuinely solve.

## Is it worth a dependency

The honest case against: the standard library can do all of this, and
assertion libraries have a habit of growing until tests are written in
a DSL rather than in Go.

The case for: the diff output, and that `require.NoError(t, err)` is
one line where the plain version is three. Over a large suite that is a
real difference in how much of each test is signal.

It is also close to universal in Go codebases, so the idiom is one most
readers already know.

> **From Python:** this is `pytest`'s assertion rewriting, made
> explicit — you call `assert.Equal` instead of `assert x == y`, and
> get a comparable diff. `require` versus `assert` is the distinction
> `pytest` does not have, since there every failed assertion stops the
> test.

## Quick reference

| Need | Call |
|---|---|
| fail and stop | `require.X(t, ...)` |
| fail and continue | `assert.X(t, ...)` |
| order | `(t, expected, actual)` |
| no error | `require.NoError(t, err)` |
| a specific error | `assert.ErrorIs(t, err, ErrX)` — unwraps |
| length / membership | `assert.Len`, `assert.Contains` |
| order-insensitive slice | `assert.ElementsMatch` |
| JSON | `assert.JSONEq` |
| floats | `assert.InDelta` |
| nil *or* empty | `assert.Empty` — **not** `Equal` |
| inside a goroutine | never `require`; send the result back |

## Sources

- [testify — pkg.go.dev/github.com/stretchr/testify](https://pkg.go.dev/github.com/stretchr/testify)
- [`assert` — pkg.go.dev/github.com/stretchr/testify/assert](https://pkg.go.dev/github.com/stretchr/testify/assert)
- [`require` — pkg.go.dev/github.com/stretchr/testify/require](https://pkg.go.dev/github.com/stretchr/testify/require)
- [`testing.T.FailNow` — pkg.go.dev/testing#T.FailNow](https://pkg.go.dev/testing#T.FailNow)
