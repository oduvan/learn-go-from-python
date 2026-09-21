# The `testing` package

Testing is built into the toolchain. There is no framework to install,
no assertion library required, and no runner to configure — a file
ending in `_test.go` and a function starting with `Test` is a test.

```go
func TestNormalize(t *testing.T) {
    got, err := Normalize("  Hello  ")
    if err != nil {
        t.Fatalf("Normalize() unexpected error: %v", err)
    }
    if got != "hello" {
        t.Errorf("Normalize() = %q, want %q", got, "hello")
    }
}
```

## The rules

- The file name ends in `_test.go`. Those files are excluded from
  normal builds, so test code never ships.
- The function is `func TestXxx(t *testing.T)`. The name after `Test`
  must start with a capital letter — `Testthing` is not a test, and
  nothing warns you.
- Test files live **next to** the code they test, in the same package.
  There is no separate `tests/` tree.

Run them with `go test ./...` from anywhere in the module.

## No assertions, just `if`

Go ships no `assertEqual`. You compare with `if` and report with the
methods on `*testing.T`:

| Method | Effect |
|---|---|
| `t.Errorf` | record a failure, **keep going** |
| `t.Fatalf` | record a failure and stop this test |
| `t.Logf` | note something; shown only with `-v` or on failure |
| `t.Skipf` | skip, with a reason |

The choice between `Errorf` and `Fatalf` is about whether continuing
makes sense. If a later line would panic — a nil result, a failed
setup — use `Fatalf`. Otherwise `Errorf` reports several problems in one
run instead of making you fix them one at a time.

`t.Fatalf` calls `runtime.Goexit`, so it only stops the goroutine it
runs on. **Calling it from a goroutine your test started does not fail
the test properly** — send the failure back to the test goroutine
instead.

## Write the failure message for the person reading it

A test that fails saying `false` tells you nothing. The convention is
to print what you called, what you got, and what you wanted:

```go
t.Errorf("Sum(%v) = %d, want %d", xs, got, want)
```

```
    calc_test.go:44: Sum([1 2]) = 3, want 4
--- FAIL: TestFailureMessage (0.00s)
```

The file and line come for free. Use `%q` for strings so whitespace
differences are visible — the reason `"hello "` and `"hello"` are
otherwise indistinguishable in output.

## Running a subset

```bash
go test ./...                    # everything
go test ./internal/store/...     # one package tree
go test -v ./...                 # show each test
go test -run TestNormalize ./... # regex match on the name
go test -count=1 ./...           # defeat the result cache
```

`-run` takes a **regular expression**, not a literal, so
`-run TestNormalize` also matches `TestNormalizeTable`. Anchor it with
`-run '^TestNormalize$'` when that matters.

Results are cached: an unchanged package prints `(cached)` and does not
re-run. That is usually what you want, and `-count=1` is the way to
force a real run.

## Internal and external test packages

A test file may declare either package:

```go
package store        // internal: sees unexported identifiers
package store_test   // external: only the public API
```

`store_test` is the one exception to one-package-per-directory, and the
compiler allows both in the same folder.

Use the internal form by default. Reach for `store_test` when you want
to be sure the exported API is usable on its own, or to break an import
cycle — a test needing a package that imports the one under test.

## Setup and cleanup

`t.Cleanup` registers work to run when the test finishes, including on
failure. It runs last-in-first-out, like `defer`, but survives across
helper functions:

```go
func TestThing(t *testing.T) {
    srv := startServer()
    t.Cleanup(srv.Close)
    // ...
}
```

Prefer it to `defer` in tests: a helper can register its own cleanup,
which `defer` inside that helper could not do.

`TestMain` gives a package one-time setup, and **must** call `m.Run`
and exit with its code:

```go
func TestMain(m *testing.M) {
    // setup
    code := m.Run()
    // teardown
    os.Exit(code)
}
```

Note the `os.Exit`, which means deferred functions in `TestMain` do not
run — put teardown before the exit.

## Coverage, benchmarks and the race detector

```bash
go test -cover ./...
# ok   scratch   0.399s   coverage: 44.4% of statements

go test -race ./...
```

`-race` instruments the binary to detect concurrent access to the same
memory. It is slower, and it only reports races that actually occur
during the run — but it finds real bugs that no amount of reading will.
Run it in CI.

A quick benchmark, covered properly in
[benchmarks, fuzzing and the race detector](06-benchmarks-fuzzing-and-race.md):

```bash
go test -run XXX -bench . -benchmem ./...
# BenchmarkSum-12   3177772   370.7 ns/op   0 B/op   0 allocs/op
```

`-run XXX` matches no test, so only benchmarks run.

## Example functions

A function named `ExampleXxx` with an `// Output:` comment is compiled,
run, and its output compared:

```go
func ExampleNormalize() {
    s, _ := Normalize("  Go  ")
    fmt.Println(s)
    // Output: go
}
```

These appear in `go doc` and on pkg.go.dev, so they are documentation
that cannot go stale. Omit the `// Output:` comment and it is compiled
but not run.

> **From Python:** closest to `pytest`, minus the magic. No fixtures,
> no `assert` rewriting, no plugins, no `conftest.py`. You write plain
> comparisons and plain error messages, which is more typing and much
> easier to read when something breaks at 3am.

## Quick reference

| Task | Form |
|---|---|
| a test | `func TestXxx(t *testing.T)` in `*_test.go` |
| fail and continue | `t.Errorf("got %v, want %v", got, want)` |
| fail and stop | `t.Fatalf(...)` |
| cleanup | `t.Cleanup(fn)` — beats `defer` in tests |
| package-level setup | `TestMain(m *testing.M)`, must `os.Exit(m.Run())` |
| public-API-only tests | `package foo_test` |
| run everything | `go test ./...` |
| verbose / filtered | `-v`, `-run '^TestX$'` |
| defeat the cache | `-count=1` |
| coverage / races | `-cover`, `-race` |
| runnable docs | `ExampleXxx` with `// Output:` |

## Sources

- [`testing` package reference — pkg.go.dev/testing](https://pkg.go.dev/testing)
- [`go test` command — pkg.go.dev/cmd/go#hdr-Test_packages](https://pkg.go.dev/cmd/go#hdr-Test_packages)
- [`testing.T.Cleanup` — pkg.go.dev/testing#T.Cleanup](https://pkg.go.dev/testing#T.Cleanup)
- [Go blog: the race detector — go.dev/blog/race-detector](https://go.dev/blog/race-detector)
- [Add a test — go.dev/doc/tutorial/add-a-test](https://go.dev/doc/tutorial/add-a-test)
