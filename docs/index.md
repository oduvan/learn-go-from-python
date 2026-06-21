# Learn Go from Python

A personal, opinionated set of conspect notes for a Python developer
picking up Go. The goal is to explain Go **on its own terms**, with
short Python analogies thrown in only when they sharpen the contrast.

All material targets the current stable Go release, **Go 1.26.4**.

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

### [Packages and modules](04-packages-and-modules/01-packages-and-visibility.md)

- [Packages and visibility](04-packages-and-modules/01-packages-and-visibility.md) — package rules, exported vs unexported, `init`.
- [Creating and publishing a module](04-packages-and-modules/02-creating-and-publishing-a-module.md) — `go.mod`, versioning, `replace`, publishing.
- [Project layout and workspaces](04-packages-and-modules/03-project-layout-and-workspaces.md) — `internal/`, `cmd/`, `go.work`.

### [Concurrency](05-concurrency/01-goroutines.md)

- [Goroutines](05-concurrency/01-goroutines.md) — `go`, scheduling, `WaitGroup`, the main-exits trap.
- [Channels](05-concurrency/02-channels.md) — send/receive, buffering, `close`, `range`, deadlocks.

## Source

- Source repository: <https://github.com/oduvan/learn-go-from-python>.
- Each conspect cites the official sources it consulted at the bottom
  of the page — typically [go.dev](https://go.dev/), [pkg.go.dev](https://pkg.go.dev/),
  or the [Go specification](https://go.dev/ref/spec).
