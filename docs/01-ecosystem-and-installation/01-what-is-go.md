# What is Go (the ecosystem)

Go (often written "Golang" for searchability) is a statically typed, compiled programming language designed at Google starting in 2007 and released as open source in 2009. It was created by Robert Griesemer, Rob Pike, and Ken Thompson.

## Design goals

Go was designed to make working on large codebases tractable:

- Compile fast even on huge programs.
- Run fast — close to C/C++ for many workloads.
- Be simple enough that engineers can read each other's code without surprises.
- Built-in concurrency primitives (goroutines, channels).
- Strong, opinionated tooling shipped with the compiler.

## Stewardship and release cadence

- Open source under a BSD-3 license.
- Developed in the open at [github.com/golang/go](https://github.com/golang/go).
- Sponsored by Google; design decisions go through the public Go proposal process.
- New minor versions ship roughly every six months (February and August).
- The latest two minor versions receive security patches.
- **Compatibility promise:** code written for Go 1.0 (March 2012) is expected to still compile and run on the latest Go 1.x release.

## What ships with Go

Installing Go gives you:

1. The `go` command — one binary that drives building, running, testing, dependency management, code formatting, documentation, profiling, and more.
2. The Go compiler and linker.
3. The standard library — large by design. HTTP server/client, JSON, crypto, templating, SQL interface, testing, profiling, concurrency primitives, file I/O, networking, and much more.

A typical Go project depends on a handful of external modules — or zero — because the standard library covers so much ground.

## How code is organized

Go uses two terms that often confuse newcomers:

- **Package** — a directory of `.go` files that share the same `package <name>` declaration on their first line. It is the unit of compilation and the unit of visibility (capitalized identifiers are exported, lowercase are package-private).
- **Module** — a collection of packages versioned and distributed together. Defined by a `go.mod` file at the module root. It is the unit of dependency management.

> Coming from Python: **package** ≈ Python module/subpackage; **module** ≈ Python distribution package (the thing you'd publish to PyPI). The terminology is inverted.

## Dependencies

- No central package index like PyPI or npm.
- You import by URL: `import "github.com/gorilla/mux"`.
- The toolchain fetches modules directly from version control.
- A public proxy at `proxy.golang.org` caches every public module ever requested. By default `go` uses this proxy; it can be disabled or replaced via `GOPROXY`.
- Checksums of every module are recorded in `go.sum` and additionally verified against the public checksum database at `sum.golang.org`.

## Tooling around the language

These ship outside the Go toolchain but are de facto standard:

- **`gopls`** — official Go language server; powers IDE features in VS Code, Neovim, JetBrains GoLand, etc.
- **`dlv`** (Delve) — the debugger.
- **`golangci-lint`** — meta-linter aggregating dozens of checks; standard in CI.

## Community resources

- **[go.dev](https://go.dev/)** — official site: blog, release notes, tutorials, the Tour of Go.
- **[pkg.go.dev](https://pkg.go.dev/)** — searchable documentation for every published Go module.
- `r/golang` on Reddit, the Gophers Slack, the `golang-nuts` mailing list.

## Sources

- [go.dev — official site](https://go.dev/)
- [Go release history](https://go.dev/doc/devel/release)
- [Go 1 compatibility promise](https://go.dev/doc/go1compat)
- [Standard library index](https://pkg.go.dev/std)
