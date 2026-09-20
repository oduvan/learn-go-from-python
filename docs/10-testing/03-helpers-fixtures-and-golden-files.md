# Helpers, fixtures and golden files

Everything around the assertion: shared setup that reports failures
usefully, temporary files that clean themselves up, and comparing
against a recorded expected output.

```go
func mustTempFile(t *testing.T, content string) string {
    t.Helper()
    p := filepath.Join(t.TempDir(), "f.txt")
    if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
        t.Fatalf("writing temp file: %v", err)
    }
    return p
}
```

## `t.Helper` fixes the reported line number

Without it, a failure inside a helper is reported at the line *in the
helper*, which is the same line for every caller and tells you nothing.
`t.Helper()` marks the function so the failure is attributed to the
caller instead.

Call it as the **first statement**, and in every helper — including ones
that only call other helpers.

A helper should take `*testing.T` as its first parameter and fail with
`t.Fatalf` rather than returning an error. That is the one place the
usual "return errors" rule is inverted: a test helper failing means the
test cannot continue, and making every caller write `if err != nil`
buries the actual test.

## `t.TempDir`

Returns a fresh directory and registers its removal automatically:

```go
dir := t.TempDir()
```

It is per-test — a subtest calling it gets its own, deleted when *that
subtest* ends:

```go
var saved string
t.Run("inner", func(t *testing.T) {
    saved = t.TempDir()   // exists here
})
// by now it is gone
```

That makes it safe with `t.Parallel`, where a shared directory would
have cases stepping on each other. Never use a fixed path like
`/tmp/mytest`; parallel runs and `-count=2` will collide.

## `t.Cleanup` runs last-in-first-out

```go
t.Cleanup(func() { order = append(order, "first registered") })
t.Cleanup(func() { order = append(order, "second registered") })
// cleanup order: [second registered first registered]
```

Same ordering as `defer`, but it is attached to the *test* rather than
to the enclosing function — so a helper can register cleanup for work it
started, which a `defer` inside that helper could not.

It runs on failure and on panic too, which `defer` in the test body also
does. The reason to prefer `t.Cleanup` is the helper case and
compatibility with `t.Parallel`.

## `testdata/`

The `go` tool ignores any directory named `testdata`, so fixture files
live there and are never compiled or treated as a package:

```
store/
  store.go
  store_test.go
  testdata/
    input.json
    expected.json
```

Paths in a test are relative to the **package directory**, because
`go test` runs each package's tests with that as the working directory.
So `os.ReadFile("testdata/input.json")` works regardless of where you
invoked `go test` from.

## Golden files

When the expected output is large — rendered HTML, a formatted report,
a JSON document — inlining it makes the test unreadable. Record it in a
file and compare:

```go
var update = flag.Bool("update", false, "rewrite golden files")

func assertGolden(t *testing.T, name, got string) {
    t.Helper()
    p := filepath.Join("testdata", name+".golden")

    if *update {
        if err := os.WriteFile(p, []byte(got), 0o644); err != nil {
            t.Fatalf("updating golden: %v", err)
        }
        return
    }

    want, err := os.ReadFile(p)
    if err != nil {
        t.Fatalf("reading golden (run with -update to create): %v", err)
    }
    if got != string(want) {
        t.Errorf("golden mismatch for %s\n got: %q\nwant: %q", name, got, string(want))
    }
}
```

```go
func TestGolden(t *testing.T) {
    assertGolden(t, "greeting", Render("GOLDEN"))
}
```

Regenerate after an intentional change:

```bash
go test ./... -update
```

Then **read the diff before committing it**. That is the whole risk of
golden files: `-update` makes any change pass, so an unreviewed
regeneration silently blesses a bug. The files must be in version
control and the diff must be part of review.

A `flag.Bool` at package level in a test file is how you add a flag to
`go test`; the `testing` package parses it along with its own.

Two more rules. Golden files must be **deterministic** — no timestamps,
no map iteration order, no random ids. Normalise those before comparing
or you get a test that fails on Tuesdays. And keep them readable: a
golden diff is only useful if a human can see what changed.

## When a plain comparison is not enough

`==` works for strings and comparable structs. For maps and slices, use
`slices.Equal` and `maps.Equal`; `reflect.DeepEqual` also works but
treats a nil slice and an empty one as different, which is rarely what
a test means — the point made in
[XML, CSV and reflection](../06-text-time-and-data/08-xml-csv-and-reflection.md).

For JSON, compare the *decoded* values rather than the text. Key order
and whitespace are not part of what you are testing.

> **From Python:** `t.TempDir` is the `tmp_path` fixture and `t.Cleanup`
> is `addfinalizer`, but there is no fixture injection — helpers are
> ordinary functions you call. `t.Helper` is the equivalent of
> `__tracebackhide__`. Golden files are `pytest-regressions`, hand-rolled
> in fifteen lines.

## Quick reference

| Task | Form |
|---|---|
| shared setup | a helper taking `*testing.T` first |
| correct failure line | `t.Helper()` as the first statement |
| a scratch directory | `t.TempDir()` — per test, auto-removed |
| teardown | `t.Cleanup(fn)` — LIFO, survives helpers and `t.Parallel` |
| fixture files | `testdata/`, ignored by the `go` tool |
| working directory | the package directory, always |
| large expected output | a `.golden` file under `testdata/` |
| regenerate | a package-level `flag.Bool("update", ...)` |
| the danger | review the golden diff; `-update` makes anything pass |
| compare collections | `slices.Equal` / `maps.Equal` |

## Sources

- [`testing.T.Helper` — pkg.go.dev/testing#T.Helper](https://pkg.go.dev/testing#T.Helper)
- [`testing.T.TempDir` — pkg.go.dev/testing#T.TempDir](https://pkg.go.dev/testing#T.TempDir)
- [`testing.T.Cleanup` — pkg.go.dev/testing#T.Cleanup](https://pkg.go.dev/testing#T.Cleanup)
- [Package names and testdata — pkg.go.dev/cmd/go#hdr-Package_lists_and_patterns](https://pkg.go.dev/cmd/go#hdr-Package_lists_and_patterns)
- [`flag` package reference — pkg.go.dev/flag](https://pkg.go.dev/flag)
