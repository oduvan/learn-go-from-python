# Benchmarks, fuzzing and the race detector

Three tools that come with `go test` and find things a normal test
cannot: how fast code is, inputs you never thought of, and concurrency
bugs that only sometimes happen.

```go
func BenchmarkConcatBuilder(b *testing.B) {
    for b.Loop() {
        ConcatBuilder(input)
    }
}
```

## Benchmarks

A benchmark is `func BenchmarkXxx(b *testing.B)` in a `_test.go` file.
`b.Loop()` runs the body enough times to get a stable measurement — the
framework decides how many.

```bash
go test -run XXX -bench . -benchmem ./...
```

`-run XXX` matches no test, so only benchmarks run. `-bench` takes a
regex; `.` means all.

```
BenchmarkConcatPlus-12       37771     31376 ns/op   265138 B/op   499 allocs/op
BenchmarkConcatBuilder-12   506193      2366 ns/op     3320 B/op     9 allocs/op
```

Reading a line: the `-12` is `GOMAXPROCS`, then iterations, then
nanoseconds per operation, then — with `-benchmem` — bytes allocated
and allocation *count* per operation.

This is the [operators article's](../02-language-basics/04-operators.md)
warning about `+=` in a loop, measured: the `strings.Builder` version is
about thirteen times faster and allocates nine times instead of 499.
`allocs/op` is often the most actionable number, because allocation
count is what drives garbage-collection pressure.

### Keep setup out of the measurement

```go
func BenchmarkWithSetup(b *testing.B) {
    data := strings.Repeat("x", 1000)
    b.ResetTimer()
    for b.Loop() {
        _ = strings.ToUpper(data)
    }
}
```

`b.ResetTimer()` discards everything timed so far. There is also
`b.StopTimer()`/`b.StartTimer()` for per-iteration setup, though that
is slow enough that restructuring is usually better.

### Do not let the compiler delete your work

If a result is unused, the optimiser may remove the call entirely and
you benchmark an empty loop. Assign to a package-level variable:

```go
var sink string

func BenchmarkSink(b *testing.B) {
    for b.Loop() {
        sink = ConcatBuilder(input)
    }
}
```

`b.Loop()` is more resistant to this than the older `for i := 0; i < b.N; i++`
form, but the sink remains the reliable habit. A benchmark reporting
implausibly few nanoseconds per operation has usually been optimised
away.

### Comparing honestly

One run is noise. Benchmark numbers move with CPU throttling, other
processes, and cache state. Take several runs (`-count=10`) and compare
with `benchstat`, which reports whether a difference is statistically
meaningful. Measure a change against the same machine in the same
session; never compare numbers from two different machines.

## Fuzzing

A fuzz target is `func FuzzXxx(f *testing.F)`. You provide seed inputs
and a property; the runtime generates mutations looking for one that
breaks it:

```go
func FuzzParseKV(f *testing.F) {
    f.Add("a=b")
    f.Add("noequals")
    f.Add("=")

    f.Fuzz(func(t *testing.T, s string) {
        k, v, ok := ParseKV(s)
        if ok && k+"="+v != s {
            t.Fatalf("round trip failed for %q: %q %q", s, k, v)
        }
    })
}
```

```bash
go test -run XXX -fuzz FuzzParseKV -fuzztime 4s ./...
```

Without `-fuzz` the target still runs as a normal test over the seed
corpus, so fuzz targets are useful in CI even when you are not fuzzing.
Only one target can be fuzzed at a time.

### What it finds, and what it writes

Point it at a function with an unchecked assumption and it finds the
input in milliseconds:

```
fuzz: minimizing 27-byte failing input file
--- FAIL: FuzzFirstByte (0.03s)
    testing.go:2076: panic: runtime error: index out of range [0] with length 0
    Failing input written to testdata/fuzz/FuzzFirstByte/5838cdfae7b16cde
```

Two things to notice. It **minimised** the input first — the 27 bytes
it stumbled on were reduced to the smallest thing that still fails. And
it **wrote the case to `testdata/`**:

```
go test fuzz v1
string("")
```

Commit that file. It becomes part of the seed corpus, so the bug is a
permanent regression test that runs on every `go test`, with no fuzzing
needed.

### What makes a good property

The assertion cannot be "the output equals X", since you do not know
the input. Useful shapes:

- **It does not panic** — often enough on its own for a parser.
- **Round trips** — `decode(encode(x)) == x`.
- **Agrees with a slower, obviously-correct implementation.**
- **Invariants hold** — output is sorted, length is preserved.

Fuzzing suits anything taking untrusted bytes: parsers, decoders,
validators, anything handling a request body.

## The race detector

`-race` instruments memory access and reports unsynchronised concurrent
access:

```bash
go test -race ./...
```

```
WARNING: DATA RACE
Read at 0x00c0000122c8 by goroutine 19:
      r_test.go:20 +0x68
Previous write at 0x00c0000122c8 by goroutine 8:
      r_test.go:20 +0x78
```

Both stacks are shown, which usually makes the bug obvious. Here it is
`counter++` from a hundred goroutines — the increment is a read and a
write, not an atomic operation, so it loses updates.

Three things to understand about it:

**It only reports races that actually occur.** It is not static
analysis. A race on a path your test never takes is invisible, which is
why running the whole suite under `-race` matters more than running one
test.

**It costs.** Roughly five to ten times slower and much more memory. Run
it in CI on every build; run it locally when touching concurrency.

**It has no false positives.** If `-race` reports something, it is a
real bug. Do not add a mutex just to silence it — understand what is
shared first.

`-race` also catches unsynchronised map access, which otherwise
manifests as the runtime killing your process with "concurrent map
writes" at an unpredictable moment.

> **From Python:** benchmarks are `timeit` wired into the test runner
> with allocation counts included. Fuzzing is Hypothesis, but
> mutation-based rather than strategy-based, and it persists failures to
> disk as regression tests automatically. The race detector has no
> Python equivalent — the GIL means most of these bugs cannot happen
> there, and in Go they very much can.

## Quick reference

| Task | Form |
|---|---|
| a benchmark | `func BenchmarkXxx(b *testing.B)`, `for b.Loop()` |
| run them | `go test -run XXX -bench . -benchmem ./...` |
| exclude setup | `b.ResetTimer()` |
| avoid elimination | assign to a package-level sink |
| compare runs | `-count=10` plus `benchstat` |
| a fuzz target | `func FuzzXxx(f *testing.F)`, `f.Add`, `f.Fuzz` |
| fuzz it | `go test -run XXX -fuzz FuzzXxx -fuzztime 30s` |
| a discovered failure | written to `testdata/fuzz/...` — **commit it** |
| good properties | no panic, round trip, invariants |
| find data races | `go test -race ./...` in CI |

## Sources

- [`testing.B` — pkg.go.dev/testing#B](https://pkg.go.dev/testing#B)
- [`testing.B.Loop` — pkg.go.dev/testing#B.Loop](https://pkg.go.dev/testing#B.Loop)
- [`testing.F` — pkg.go.dev/testing#F](https://pkg.go.dev/testing#F)
- [Go fuzzing — go.dev/doc/security/fuzz/](https://go.dev/doc/security/fuzz/)
- [Go blog: the race detector — go.dev/blog/race-detector](https://go.dev/blog/race-detector)
- [Data race detector — go.dev/doc/articles/race_detector](https://go.dev/doc/articles/race_detector)
