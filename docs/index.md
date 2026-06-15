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

- [Variables and constants](02-language-basics/01-variables-and-constants.md)
- [Basic types](02-language-basics/02-basic-types.md)
- [Type conversions](02-language-basics/03-type-conversions.md)
- [Operators](02-language-basics/04-operators.md)
- [Control flow](02-language-basics/05-control-flow.md)
- [Functions](02-language-basics/06-functions.md)
- [Errors](02-language-basics/07-errors.md)
- [Pointers](02-language-basics/08-pointers.md)
- [Custom types](02-language-basics/09-custom-types.md)
- [Structs](02-language-basics/10-structs.md)
- [Arrays and slices](02-language-basics/11-arrays-and-slices.md)
- [Maps](02-language-basics/12-maps.md)
- [Choosing a data structure](02-language-basics/13-choosing-a-data-structure.md)
- [Defer](02-language-basics/14-defer.md)
- [Panic and recover](02-language-basics/15-panic-and-recover.md)
- [Imports](02-language-basics/16-imports.md)

### [Object-oriented Go](03-object-oriented-go/01-methods.md)

- [Methods](03-object-oriented-go/01-methods.md)
- [Interfaces](03-object-oriented-go/02-interfaces.md)
- [Type assertions and type switches](03-object-oriented-go/03-type-assertions-and-type-switches.md)
- [Generics](03-object-oriented-go/04-generics.md)
- [OOP patterns](03-object-oriented-go/05-oop-patterns.md)

## Source

- Source repository: <https://github.com/oduvan/learn-go-from-python>.
- Each conspect cites the official sources it consulted at the bottom
  of the page — typically [go.dev](https://go.dev/), [pkg.go.dev](https://pkg.go.dev/),
  or the [Go specification](https://go.dev/ref/spec).
