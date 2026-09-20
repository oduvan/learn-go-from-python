# Learn Go from Python

A personal, opinionated set of conspect notes for a Python developer
picking up Go. The goal is to explain Go **on its own terms**, with
short Python analogies thrown in only when they sharpen the contrast.

All material targets the current stable Go release, **Go 1.27.1**.

## How the notes are organised

Each topic is a numbered folder. Inside each folder, conspects and
runnable code examples share one numeric sequence so the reading order
is always obvious.

### [Ecosystem and installation](01-ecosystem-and-installation/01-what-is-go.md)

- [What is Go](01-ecosystem-and-installation/01-what-is-go.md) — the language, the community, the ecosystem.
- [The `go` command](01-ecosystem-and-installation/02-go-subcommands.md) — every subcommand you'll touch.
- [`go tool trace`](01-ecosystem-and-installation/03-go-tool-trace.md) — the execution tracer.
- [File types](01-ecosystem-and-installation/04-go-file-types.md) — `.go`, `_test.go`, `go.mod`, `go.sum`, build constraints.
- [Special folders](01-ecosystem-and-installation/05-special-folders.md) — `internal/`, `testdata/`, conventions.
- [Multiple Go versions](01-ecosystem-and-installation/06-multiple-go-versions.md) — `GOTOOLCHAIN`, the `go` and `toolchain` directives.
- [Installation](01-ecosystem-and-installation/07-installation.md) — macOS, Linux, Windows.
- [Additional tools](01-ecosystem-and-installation/08-additional-tools.md) — `gopls`, `dlv`, `golangci-lint`, and friends.
- [Demo project](01-ecosystem-and-installation/09-demo-project/README.md) — a small runnable module that exercises the above.

### [Language basics](02-language-basics/01-variables-and-constants.md)

- [Variables and constants](02-language-basics/01-variables-and-constants.md) — `var`, `:=`, `const`, and `iota`.
- [Basic types](02-language-basics/02-basic-types.md) — integers, floats, strings, booleans; no truthiness.
- [Type conversions](02-language-basics/03-type-conversions.md) — explicit `T(x)`, `strconv`, no implicit coercion.
- [Operators](02-language-basics/04-operators.md) — arithmetic, overflow, integer division; no ternary.
- [Control flow](02-language-basics/05-control-flow.md) — `if`, `for` (the only loop), `switch`.
- [Functions](02-language-basics/06-functions.md) — multiple returns, named results, variadics, first-class values.
- [Errors](02-language-basics/07-errors.md) — the `error` value, wrapping with `%w`, `errors.Is`/`As`.
- [Pointers](02-language-basics/08-pointers.md) — `&`/`*`, `nil`, `new`, no pointer arithmetic.
- [Custom types](02-language-basics/09-custom-types.md) — `type` definitions vs aliases, underlying types.
- [Structs](02-language-basics/10-structs.md) — fields, literals, zero value, embedding, tags.
- [Arrays and slices](02-language-basics/11-arrays-and-slices.md) — len/cap, `append`, and the shared-backing gotcha.
- [Maps](02-language-basics/12-maps.md) — keyed lookup, comma-ok, the nil-map trap, sets.
- [Choosing a data structure](02-language-basics/13-choosing-a-data-structure.md) — slice vs map vs struct vs custom type.
- [Defer](02-language-basics/14-defer.md) — deferred calls, LIFO order, cleanup patterns.
- [Panic and recover](02-language-basics/15-panic-and-recover.md) — when to panic, recovering in deferred calls.
- [Imports](02-language-basics/16-imports.md) — import paths, aliases, blank and dot imports.

### [Object-oriented Go](03-object-oriented-go/01-methods.md)

- [Methods](03-object-oriented-go/01-methods.md) — value vs pointer receivers, method sets, promotion.
- [Interfaces](03-object-oriented-go/02-interfaces.md) — implicit satisfaction, polymorphism, the empty interface / `any`.
- [Type assertions and type switches](03-object-oriented-go/03-type-assertions-and-type-switches.md) — recovering the concrete type at runtime.
- [Generics](03-object-oriented-go/04-generics.md) — type parameters and constraints.
- [OOP patterns](03-object-oriented-go/05-oop-patterns.md) — encapsulation, composition over inheritance, polymorphism.
- [Custom error types](03-object-oriented-go/06-custom-error-types.md) — your own `error` types, `Unwrap`, `errors.As`, custom `Is`.

### [Packages and modules](04-packages-and-modules/01-packages-and-visibility.md)

- [Packages and visibility](04-packages-and-modules/01-packages-and-visibility.md) — package rules, exported vs unexported, `init`.
- [Creating and publishing a module](04-packages-and-modules/02-creating-and-publishing-a-module.md) — `go.mod`, versioning, `replace`, publishing.
- [Project layout and workspaces](04-packages-and-modules/03-project-layout-and-workspaces.md) — `internal/`, `cmd/`, `go.work`.

### [Concurrency](05-concurrency/01-goroutines.md)

- [Goroutines](05-concurrency/01-goroutines.md) — `go`, scheduling, `WaitGroup`, the main-exits trap.
- [Channels](05-concurrency/02-channels.md) — send/receive, buffering, `close`, `range`, deadlocks.
- [select](05-concurrency/03-select.md) — multiplexing, `default`, timeouts, done-channels.
- [Synchronization](05-concurrency/04-synchronization.md) — `Mutex`, `Once`, atomics, the race detector.
- [Context](05-concurrency/05-context.md) — cancellation, deadlines, propagation.
- [Concurrency patterns](05-concurrency/06-concurrency-patterns.md) — worker pools, fan-out/fan-in, pipelines.
- [Bounded concurrency](05-concurrency/07-bounded-concurrency.md) — channel semaphores, collecting results, `sync.Map`.
- [Long-running goroutines](05-concurrency/08-long-running-goroutines.md) — per-goroutine `recover`, tickers, draining on shutdown.

### [Text, time and data](06-text-time-and-data/01-strings-bytes-and-runes.md)

- [Strings, bytes and runes](06-text-time-and-data/01-strings-bytes-and-runes.md) — `strings`, `bytes`, and why `len` counts bytes.
- [Formatting with `fmt`](06-text-time-and-data/02-formatting-with-fmt.md) — the verbs, width and precision, `Stringer`, `%w`.
- [Regular expressions](06-text-time-and-data/03-regular-expressions.md) — `regexp`, named groups, and what RE2 leaves out.
- [Time](06-text-time-and-data/04-time.md) — the reference layout, durations, zones, `Equal` over `==`.
- [Sorting](06-text-time-and-data/05-sorting.md) — `slices.SortFunc`, `cmp.Compare`, `cmp.Or`, stability.
- [Iterators](06-text-time-and-data/06-iterators.md) — writing `iter.Seq`, the `yield` contract, `iter.Pull`.
- [Encoding JSON](06-text-time-and-data/07-encoding-json.md) — tags, `omitempty`, `RawMessage`, custom marshalling.
- [XML, CSV and reflection](06-text-time-and-data/08-xml-csv-and-reflection.md) — token streaming, `csv`, struct tags at runtime.

### [The operating system](07-operating-system/01-files-and-paths.md)

- [Files and paths](07-operating-system/01-files-and-paths.md) — `os`, `filepath`, `WalkDir`, testing errors not paths.
- [Readers and writers](07-operating-system/02-readers-and-writers.md) — `io.Copy`, `bufio.Scanner`, and its 64 KB limit.
- [`go:embed`](07-operating-system/03-go-embed.md) — files in the binary, `embed.FS`, `all:`, `fs.Sub`.
- [Flags and environment](07-operating-system/04-flags-and-environment.md) — `flag`, subcommands, `LookupEnv`, exit codes.
- [Running external commands](07-operating-system/05-running-external-commands.md) — `exec.CommandContext`, `ExitError`, no shell.
- [Signals and graceful shutdown](07-operating-system/06-signals-and-graceful-shutdown.md) — `NotifyContext`, draining on a budget.
- [Hashing and random values](07-operating-system/07-hashing-and-random-values.md) — sha256, HMAC, `crypto/rand`, base64, gzip.

### [HTTP with `net/http`](08-http-with-net-http/01-http-server.md)

- [An HTTP server](08-http-with-net-http/01-http-server.md) — handlers, `ServeMux` routing, timeouts, shutdown.
- [An HTTP client](08-http-with-net-http/02-http-client.md) — why a 404 is not an error, closing bodies, retries.
- [Middleware](08-http-with-net-http/03-middleware.md) — wrapping handlers, recovery, context values.
- [Templates](08-http-with-net-http/04-templates.md) — `text/template` vs `html/template` and contextual escaping.
- [Server-sent events](08-http-with-net-http/05-server-sent-events.md) — streaming, flushing, dropping slow clients.

### [Databases with `database/sql`](09-database-sql/01-database-sql.md)

- [`database/sql`](09-database-sql/01-database-sql.md) — the pool, `Scan`, `ErrNoRows`, NULL, `rows.Err()`.
- [Custom column types](09-database-sql/02-custom-column-types.md) — `driver.Valuer` and `sql.Scanner`.
- [Transactions](09-database-sql/03-transactions.md) — the closure wrapper, rollback on panic, nesting.
- [The repository pattern](09-database-sql/04-the-repository-pattern.md) — a contract package, translating storage errors.

### [Testing](10-testing/01-the-testing-package.md)

- [The `testing` package](10-testing/01-the-testing-package.md) — `TestXxx`, `Errorf` vs `Fatalf`, `go test` flags.
- [Table-driven tests](10-testing/02-table-driven-tests.md) — the case slice, `t.Run`, `t.Parallel`.
- [Helpers, fixtures and golden files](10-testing/03-helpers-fixtures-and-golden-files.md) — `t.Helper`, `t.TempDir`, `testdata/`.
- [Fakes and stubs](10-testing/04-fakes-and-stubs.md) — function-field doubles, faking the clock.
- [Testing HTTP](10-testing/05-testing-http.md) — `httptest` recorders and servers.
- [Benchmarks, fuzzing and the race detector](10-testing/06-benchmarks-fuzzing-and-race.md) — `b.Loop`, `f.Fuzz`, `-race`.

### [Architecture and conventions](11-architecture-and-conventions/01-wiring-and-package-structure.md)

- [Wiring and package structure](11-architecture-and-conventions/01-wiring-and-package-structure.md) — `cmd/`, `internal/`, constructor injection.
- [Context as a carrier](11-architecture-and-conventions/02-context-as-a-carrier.md) — unexported keys, `WithoutCancel`, what not to put in.
- [Configuration patterns](11-architecture-and-conventions/03-configuration-patterns.md) — one struct, defaults, validation with `errors.Join`.
- [Dependency direction](11-architecture-and-conventions/04-dependency-direction.md) — which package may import which, and why.
- [Build, code generation and cgo](11-architecture-and-conventions/05-build-codegen-and-cgo.md) — build tags, `-ldflags`, cross-compiling, cgo's cost.
- [Project conventions](11-architecture-and-conventions/06-project-conventions.md) — error wrapping, log levels, naming, comments.

### [Observability](12-observability/01-structured-logging-with-slog.md)

- [Structured logging with `slog`](12-observability/01-structured-logging-with-slog.md) — handlers, `With`, `LogValuer`, testing logs.
- [Profiling with pprof](12-observability/02-profiling-with-pprof.md) — CPU and heap profiles, flat vs cum, flame graphs.

## Source

- Source repository: <https://github.com/oduvan/learn-go-from-python>.
- Each conspect cites the official sources it consulted at the bottom
  of the page — typically [go.dev](https://go.dev/), [pkg.go.dev](https://pkg.go.dev/),
  or the [Go specification](https://go.dev/ref/spec).
