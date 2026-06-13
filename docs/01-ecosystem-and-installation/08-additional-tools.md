# Additional Go tools

The `go` command covers building, testing, formatting, vet, and dependency management. Beyond that, the Go community (and the Go team itself) maintains a small set of tools you'll meet almost immediately in real projects. This article lists the ones worth knowing.

All of these are installed the same way:

```bash
go install <import-path>@latest
```

The binary lands in `$GOBIN` (default `~/go/bin`), which must be on your `$PATH`.

## Editor / IDE support

### `gopls` — official language server

Go's official Language Server Protocol implementation, maintained by the Go team. Provides autocomplete, go-to-definition, find-references, inline diagnostics, refactoring, and rename — to **any** LSP-compatible editor.

```bash
go install golang.org/x/tools/gopls@latest
```

Editors that use it out of the box: VS Code (Go extension), Neovim (via built-in LSP), JetBrains GoLand (uses its own engine, not gopls), Sublime, Emacs, Helix.

> **From Python:** ≈ `pyright` or `python-lsp-server` — except `gopls` is the *official* one, maintained alongside the compiler, so there is no fragmentation.

## Debugger

### `dlv` (Delve)

The Go debugger. Set breakpoints, step through, inspect variables and goroutines, attach to running processes, debug core dumps.

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

Common invocations:

```bash
dlv debug ./cmd/server          # build with debug info and start
dlv test ./mypkg                # debug tests
dlv attach <pid>                # attach to running process
dlv exec ./mybinary             # debug an existing binary
```

VS Code's Go extension and GoLand both drive Delve under the hood — you usually never call `dlv` directly once your editor is wired up.

> **From Python:** ≈ `pdb` or `debugpy` — but compiled-binary aware, with goroutine inspection built in.

## Linters

### `golangci-lint` — the meta-linter

Runs **dozens of linters in parallel**, with a single config file (`.golangci.yml`), caching, and IDE/CI integrations. De facto standard for Go CI pipelines.

```bash
# preferred install method per the project: dedicated installer, not `go install`
brew install golangci-lint
# or
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin latest
```

Run:

```bash
golangci-lint run ./...
```

Linters it aggregates include `govet`, `staticcheck`, `errcheck`, `ineffassign`, `unused`, `gosimple`, `revive`, `gosec`, and many more. Most teams turn on a tuned subset rather than all 100+.

> **From Python:** ≈ `ruff` — one fast aggregator that replaces a stack of single-purpose linters. `golangci-lint` predates `ruff` by several years.

### `staticcheck` — heavyweight static analysis

If `go vet` is the conservative built-in, `staticcheck` is the deep one — catches dead code, suspicious patterns, ineffective assignments, performance traps. Included as one analyzer inside `golangci-lint`, but also runnable standalone.

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

> **From Python:** ≈ `pylint` (without the configuration cliff). Conservative enough to use without much tuning.

## Formatters beyond `gofmt`

### `goimports` — `gofmt` plus import management

Drop-in replacement for `gofmt` that **also adds missing imports and removes unused ones automatically**.

```bash
go install golang.org/x/tools/cmd/goimports@latest
goimports -w .
```

Configure your editor to run `goimports` on save (most do this by default for Go).

> **From Python:** ≈ `isort` + `black` combined into one tool.

### `gofumpt` — stricter `gofmt`

A `gofmt` superset that enforces additional style rules `gofmt` doesn't (e.g., no empty lines at the start/end of blocks, consistent grouping). Opinionated and unmovable, in the same spirit as `gofmt` itself.

```bash
go install mvdan.cc/gofumpt@latest
gofumpt -w .
```

Most projects use either `gofmt` (the default) or `gofumpt` (a small but growing minority). Consistent within a codebase is what matters.

## Security

### `govulncheck` — official vulnerability scanner

Maintained by the Go security team. Scans your code (or a built binary) for **known CVEs in dependencies you actually call** — not just dependencies you've pulled in. This call-graph awareness drastically cuts false positives.

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

Powered by the Go vulnerability database at [vuln.go.dev](https://vuln.go.dev), browsable at [pkg.go.dev/vuln](https://pkg.go.dev/vuln). Integrated into the VS Code Go extension and available as a GitHub Action.

> **From Python:** ≈ `pip-audit` or `safety` — except `govulncheck` understands which vulnerable functions your code actually reaches.

## Code generators

Go encourages code generation for repetitive patterns (enum-like types, mocks, generated DB code). Generators are invoked via `//go:generate` directives or directly.

### `stringer` — generate `String()` methods for enum-like types

Given a `const` block of typed integer values, generates a `String()` method. (The Go syntax in the example below — defining a custom type `Pill` and using `iota` to auto-number constants — is covered in the language-basics topic; here it's just shown as the kind of code `stringer` consumes.)

```bash
go install golang.org/x/tools/cmd/stringer@latest
```

```go
//go:generate stringer -type=Pill
type Pill int
const (
    Placebo Pill = iota
    Aspirin
    Ibuprofen
)
```

Then `go generate ./...` regenerates the `_string.go` file.

### `mockgen` — generate mocks for interfaces

From the [Uber `gomock` fork](https://github.com/uber-go/mock), now the canonical version.

```bash
go install go.uber.org/mock/mockgen@latest
mockgen -source=foo.go -destination=mocks/foo_mock.go
```

> **From Python:** the equivalent of `unittest.mock` — but Go's static typing means mocks are *generated* against an interface signature, not assembled dynamically.

### `sqlc` — SQL to typed Go code

Write SQL queries in `.sql` files; `sqlc` generates fully typed Go functions to call them. Avoids both ORMs and string-formatted SQL.

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
sqlc generate
```

### `protoc-gen-go` — protobuf code generator

For gRPC / Protobuf workflows.

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Used by `protoc` (the protobuf compiler) or, more commonly today, [`buf`](https://buf.build/).

## Profiling and performance

### `go tool pprof` — built-in profiler

Already shipped with the toolchain. Reads CPU, memory, block, mutex, and goroutine profiles. UI mode:

```bash
go tool pprof -http=:8080 cpu.prof
```

Profiles come from `runtime/pprof` (in code), `go test -cpuprofile=cpu.prof`, or the `net/http/pprof` server endpoint.

### `benchstat` — compare benchmark results

```bash
go install golang.org/x/perf/cmd/benchstat@latest

go test -bench=. -count=10 > old.txt
# ... make changes ...
go test -bench=. -count=10 > new.txt
benchstat old.txt new.txt
```

Prints statistically meaningful deltas — essential for any "is this actually faster?" question.

## Development workflow

### `air` — live reload during development

Watches your source tree, rebuilds and restarts on changes.

```bash
go install github.com/air-verse/air@latest
air                                  # uses .air.toml in the current dir
```

> **From Python:** ≈ `watchmedo auto-restart` or what Django/Flask dev servers do.

### `mage` — Go-based build automation

A `Make` alternative where build steps are written in Go.

```bash
go install github.com/magefile/mage@latest
```

You write `magefile.go` with exported functions; `mage <target>` runs them. Useful when shell scripts get too gnarly.

> **From Python:** ≈ `invoke` or `nox` — task automation in the host language.

## Worth knowing exists

- **`yaegi`** — a Go interpreter that supports a REPL. Limited (no cgo, partial stdlib), but useful for exploration. `go install github.com/traefik/yaegi/cmd/yaegi@latest`.
- **`gore`** — another Go REPL, in a similar spirit.
- **`go-callvis`** — visualizes your program's call graph as a diagram.
- **`gopium`** — analyzes and optimizes struct field layout for memory packing.

These are nice to know but not daily tools.

## Recommended starting set

For a typical day of Go work, you want:

| Tool | Why |
|---|---|
| `gopls` | IDE features in any editor. |
| `dlv` | Debugging when print statements aren't enough. |
| `goimports` | Format-on-save that also fixes imports. |
| `golangci-lint` | One command that catches a wide net of issues. |
| `govulncheck` | Periodic security scan. |

Add `mockgen`, `stringer`, `sqlc`, or `protoc-gen-go` as your project demands them.

## Sources

- [`gopls` — pkg.go.dev/golang.org/x/tools/gopls](https://pkg.go.dev/golang.org/x/tools/gopls) and [go.dev/gopls](https://go.dev/gopls)
- [Delve — github.com/go-delve/delve](https://github.com/go-delve/delve)
- [`golangci-lint` — golangci-lint.run](https://golangci-lint.run/)
- [`staticcheck` — staticcheck.dev](https://staticcheck.dev/)
- [`goimports` — pkg.go.dev/golang.org/x/tools/cmd/goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)
- [`gofumpt` — github.com/mvdan/gofumpt](https://github.com/mvdan/gofumpt)
- [`govulncheck` — go.dev/blog/govulncheck](https://go.dev/blog/govulncheck) and [pkg.go.dev/vuln](https://pkg.go.dev/vuln)
- [`stringer` — pkg.go.dev/golang.org/x/tools/cmd/stringer](https://pkg.go.dev/golang.org/x/tools/cmd/stringer)
- [`mockgen` — github.com/uber-go/mock](https://github.com/uber-go/mock)
- [`sqlc` — sqlc.dev](https://sqlc.dev/)
- [`protoc-gen-go` — pkg.go.dev/google.golang.org/protobuf](https://pkg.go.dev/google.golang.org/protobuf)
- [`benchstat` — pkg.go.dev/golang.org/x/perf/cmd/benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [`pprof` — github.com/google/pprof](https://github.com/google/pprof)
- [`air` — github.com/air-verse/air](https://github.com/air-verse/air)
- [`mage` — magefile.org](https://magefile.org/)
