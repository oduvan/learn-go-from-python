# Demo project

A small Go module that exercises most of the topics covered in this folder:

```
09-demo-project/
├── go.mod                          # module manifest
├── main.go                         # package main — the program entrypoint
├── counter/
│   ├── counter.go                  # public package, importable from outside
│   ├── counter_test.go             # internal tests + benchmark
│   └── testdata/                   # ignored by the build, used by tests
│       ├── alice.txt
│       └── lorem.txt
└── internal/
    └── workpool/
        └── workpool.go             # only importable from inside this module
```

What each part demonstrates:

- **`go.mod`** — module path is `example.com/demo`; created with `go mod init`.
- **`main.go`** — `package main` produces an executable. It imports both `counter` (sibling package) and uses the runtime tracer.
- **`counter/`** — an ordinary package, importable as `example.com/demo/counter`.
- **`counter/counter_test.go`** — same-package tests (can see unexported symbols), plus a `BenchmarkCountConcurrent`.
- **`counter/testdata/`** — the `go` tool refuses to compile anything inside `testdata/`, so fixtures live here safely.
- **`internal/workpool/`** — Go's compiler enforces that `example.com/demo/internal/workpool` can only be imported by packages rooted at `example.com/demo`. If you copied this module to another path, external code could not reach `workpool`.

## Running it

From this directory:

```bash
go run .
```

Expected output:

```
Counted 377 words across 2 files (workload amplified to 400 jobs over 4 workers so the trace shows visible parallelism)
```

The real work is small — two text files in `counter/testdata/`, 377 words
total. The program deliberately repeats that tiny workload 200× (→ 400
jobs) and spreads it over 4 worker goroutines for one reason only: so the
recorded trace has enough concurrent activity to be worth looking at in
`go tool trace`. It also writes `trace.out`.

## Running the tests

```bash
go test ./...
```

To run the benchmark:

```bash
go test -bench=. ./counter
```

## Viewing the trace

After `go run .` has produced `trace.out`:

```bash
go tool trace trace.out
```

This starts a local web server and prints a URL like `http://127.0.0.1:NNNNN/...`. Open it in a browser and explore:

- **View trace** — the goroutine timeline.
- **Goroutine analysis** — what each goroutine spent its time on.
- **Sync / scheduler blocking profiles** — where goroutines were waiting.

Press Ctrl-C in the terminal to stop the viewer.

## Files relating back to the conspect

| Topic | Conspect file | Where you see it here |
|---|---|---|
| `go.mod` directives | [04-go-file-types.md](../04-go-file-types.md) | `go.mod` |
| `_test.go`, `testdata/` | [04-go-file-types.md](../04-go-file-types.md) | `counter/counter_test.go`, `counter/testdata/` |
| `internal/` rule | [05-special-folders.md](../05-special-folders.md) | `internal/workpool/` |
| `runtime/trace` and `go tool trace` | [03-go-tool-trace.md](../03-go-tool-trace.md) | `main.go`'s trace.Start / trace.Stop, plus `trace.out` |
| `go run`, `go test`, `go build` | [02-go-subcommands.md](../02-go-subcommands.md) | the commands above |
