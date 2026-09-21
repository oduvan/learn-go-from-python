# Table-driven tests

The dominant test style in Go: a slice of cases, one loop, one subtest
each. It is how the standard library is tested, and adding a case costs
one line.

```go
tests := []struct {
    name string
    in   string
    want string
}{
    {"trims and lowers", "  Go  ", "go"},
    {"already fine", "go", "go"},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        if got := Normalize(tt.in); got != tt.want {
            t.Errorf("got %q, want %q", got, tt.want)
        }
    })
}
```

## The anonymous struct

The case type is declared inline because it exists only for this test.
Field names make each case readable at the call site, and a `name`
field comes first by convention.

Keep one behaviour per table. If half the cases need a different set of
fields, that is two tests wearing one coat — split them.

## `t.Run` gives each case its own test

Subtests are not cosmetic. Each one:

- **is named**, so a failure says which case broke rather than which
  line;
- **fails independently**, so `t.Fatalf` in one case does not hide the
  rest;
- **can be run alone** with `-run`.

```
=== RUN   TestNormalizeTable
=== RUN   TestNormalizeTable/trims_and_lowers
=== RUN   TestNormalizeTable/already_fine
--- PASS: TestNormalizeTable (0.00s)
    --- PASS: TestNormalizeTable/trims_and_lowers (0.00s)
    --- PASS: TestNormalizeTable/already_fine (0.00s)
```

Note that **spaces in a name become underscores**. That is the name you
pass to `-run`:

```bash
go test -run 'TestNormalizeTable/trims_and_lowers' ./...
```

The pattern is a regex per path segment, so `-run 'TestX/.*empty'` works
too.

## Testing errors in the table

Put the expected error in the case and compare with `errors.Is`:

```go
tests := []struct {
    name    string
    in      string
    want    string
    wantErr error
}{
    {"trims and lowers", "  Go  ", "go", nil},
    {"empty", "   ", "", ErrEmpty},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := Normalize(tt.in)
        if !errors.Is(err, tt.wantErr) {
            t.Fatalf("err = %v, want %v", err, tt.wantErr)
        }
        if got != tt.want {
            t.Errorf("got %q, want %q", got, tt.want)
        }
    })
}
```

`errors.Is(nil, nil)` is `true`, so the success cases work with the same
comparison — no separate `wantErr bool` field needed.

Do not compare error *strings*. Message wording is not API; the sentinel
or type is, which is the point of
[custom error types](../03-object-oriented-go/06-custom-error-types.md).

## `t.Parallel`

Adding one line runs the cases concurrently:

```go
t.Run(tt.name, func(t *testing.T) {
    t.Parallel()
    // ...
})
```

```
=== RUN   TestParallelSubtests/lower
=== PAUSE TestParallelSubtests/lower
=== RUN   TestParallelSubtests/already
=== PAUSE TestParallelSubtests/already
=== CONT  TestParallelSubtests/lower
=== CONT  TestParallelSubtests/already
```

The `PAUSE`/`CONT` pairs show the mechanism: a parallel subtest pauses,
the parent finishes starting all of them, then they resume together.

Two consequences follow, and both matter:

**The parent finishes before the children.** Code after the loop runs
while the subtests are still going. Anything the subtests need must be
cleaned up with `t.Cleanup`, not with a `defer` in the parent.

**Cases must not share mutable state.** Parallel cases touching the same
map or counter is a data race. Run `-race` in CI and it will be caught;
without it, the test passes until it does not.

Parallelism pays off when cases do I/O. For pure functions it adds
scheduling noise for no gain — do not add it by reflex.

## Keep the loop dumb

The loop body should do the same thing for every case. Once it grows an
`if tt.special { ... }`, the table has stopped describing data and
started encoding control flow, and a separate test is clearer.

A function field is the honest way to vary behaviour:

```go
tests := []struct {
    name  string
    setup func(*testing.T) *Store
    want  int
}{
    {"empty", func(t *testing.T) *Store { return NewStore() }, 0},
    {"seeded", seededStore, 3},
}
```

That keeps the loop uniform while letting each case arrange itself.

> **From Python:** this is `@pytest.mark.parametrize`, written out by
> hand. More verbose, but the cases are ordinary values you can build,
> filter or generate in Go, and the `name` field does what `pytest`'s
> generated ids do — except you choose it.

## Quick reference

| Concern | Form |
|---|---|
| the table | a slice of an inline anonymous struct, `name` first |
| one subtest per case | `t.Run(tt.name, func(t *testing.T) { ... })` |
| run one case | `-run 'TestX/case_name'` — spaces become underscores |
| expected errors | a `wantErr error` field plus `errors.Is` |
| concurrency | `t.Parallel()` inside the subtest |
| cleanup with parallel cases | `t.Cleanup`, never `defer` in the parent |
| a case that differs | a `setup func(*testing.T)` field, not an `if` |

## Sources

- [`testing.T.Run` — pkg.go.dev/testing#T.Run](https://pkg.go.dev/testing#T.Run)
- [`testing.T.Parallel` — pkg.go.dev/testing#T.Parallel](https://pkg.go.dev/testing#T.Parallel)
- [Go blog: subtests and sub-benchmarks — go.dev/blog/subtests](https://go.dev/blog/subtests)
- [Go wiki: table-driven tests — go.dev/wiki/TableDrivenTests](https://go.dev/wiki/TableDrivenTests)
- [`errors.Is` — pkg.go.dev/errors#Is](https://pkg.go.dev/errors#Is)
