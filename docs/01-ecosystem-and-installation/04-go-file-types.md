# File types in a Go source tree

## Source files the `go` tool recognizes

| Filename pattern | What it means |
|---|---|
| `foo.go` | Ordinary source file. Compiled into the package. |
| `foo_test.go` | Test file. Compiled only by `go test`, never in normal builds. |
| `foo_linux.go` | Implicit build constraint: only compiled when `GOOS=linux`. |
| `foo_amd64.go` | Only compiled when `GOARCH=amd64`. |
| `foo_linux_amd64.go` | Only compiled when both apply. |
| `foo_test_linux.go` | A test file restricted to linux. |
| `_foo.go`, `.foo.go` | **Ignored entirely** by the go tool. The leading `_` or `.` is the off-switch. |
| `doc.go` | Convention only: a file holding the package-level doc comment. |
| `go.mod` | Module manifest. |
| `go.sum` | Module checksum database / lockfile. |
| `go.work`, `go.work.sum` | Workspace file for multi-module development. |

## Anatomy of a `.go` source file

Every Go source file follows the same skeleton:

```go
// Package greet provides a friendly greeting.
package greet                    // 1. package clause — required, first non-comment line

import (                         // 2. imports
    "fmt"
    "strings"
)

const Exclamation = "!"          // 3. top-level declarations
                                 //    (const, var, type, func — any order)

// Hello returns a greeting for name.
func Hello(name string) string {
    name = strings.TrimSpace(name)
    return fmt.Sprintf("Hello, %s%s", name, Exclamation)
}
```

Two non-obvious rules:

- **Capitalization controls visibility.** Identifiers starting with an uppercase letter are exported from the package; lowercase ones are package-private. There is no `public`/`private` keyword — the compiler enforces this purely from the name.
- **Unused imports and unused local variables are compile errors**, not warnings. This is enforced rigorously.

## Build constraints

Build constraints decide whether a file is included in a build.

### Explicit constraints (`//go:build`)

```go
//go:build linux && (amd64 || arm64)

package cache

// ... linux-on-64-bit-only code ...
```

Rules:

- The `//go:build` line must appear **before** the `package` clause, preceded only by blank lines and other comments.
- It must be followed by a **blank line** to distinguish it from the package doc comment.
- **At most one** `//go:build` line per file.
- The expression uses boolean operators: `||`, `&&`, `!`, and parentheses.

### Implicit constraints from filenames

These constraints come from filename suffixes — no comment needed.

- `server_linux.go` → effective constraint `//go:build linux`
- `math_amd64.go` → `//go:build amd64`
- `utils_windows_386.go` → `//go:build windows && 386`

### Automatically satisfied tags

During any build, these tags are true:

- The current `GOOS` (e.g. `linux`, `darwin`, `windows`).
- The current `GOARCH` (e.g. `amd64`, `arm64`).
- The `unix` tag, on any Unix-like OS.
- The compiler in use: `gc` or `gccgo`.
- `cgo`, if cgo is enabled.
- A tag for every Go version up to the current one: `go1.1`, `go1.21`, `go1.22`, ...
- Any custom tags passed via `-tags` to the `go` command.

## Test files — two flavors

A test file can declare the same package it tests, or a separate `_test` package.

```go
// foo_test.go — internal test (can see unexported identifiers)
package foo

import "testing"

func TestAdd(t *testing.T) {
    if got := add(2, 3); got != 5 {
        t.Errorf("add(2,3) = %d; want 5", got)
    }
}
```

```go
// foo_test.go — external test (tests the public API only)
package foo_test

import (
    "testing"
    "example.com/foo"
)

func TestPublic(t *testing.T) { /* ... */ }
```

A package can have both. Test files can also define:

- `BenchmarkXxx(b *testing.B)` — performance benchmarks.
- `FuzzXxx(f *testing.F)` — fuzz tests.
- `ExampleXxx()` — runnable examples that double as documentation.

## `go.mod` — what each directive looks like

A `go.mod` is a line-oriented, UTF-8 text file. A full example with every directive:

```
module example.com/myapp

go 1.23.0

toolchain go1.23.4

require (
    github.com/gorilla/mux v1.8.1
    golang.org/x/crypto v0.21.0 // indirect
)

exclude github.com/some/pkg v1.4.0

replace github.com/some/pkg => ../local/pkg

retract v1.0.1                            // do not depend on this version of MY module

godebug panicnil=1
```

| Directive | Purpose |
|---|---|
| `module` | The module's canonical import path. Exactly once. |
| `go` | Minimum Go language version this module requires. |
| `toolchain` | Suggested toolchain version for development and CI. |
| `require` | A dependency at a specific minimum version. `// indirect` marks transitive deps. |
| `exclude` | Refuse a specific version of a dependency. |
| `replace` | Redirect a module to a local path or to a fork. |
| `retract` | Mark versions of *your own* module as do-not-use. |
| `godebug` | Pin a `GODEBUG` toggle when this module is the main one. |
| `tool` | Declare a Go tool runnable via `go tool`. |

Do **not** hand-edit `go.mod` for routine changes. Use `go get`, `go mod tidy`, etc. — they are aware of `go.sum` and will keep both consistent.

## `go.sum` — what it looks like

Two lines per `(module, version)` pair:

```
golang.org/x/text v0.3.0 h1:g61tztE5qeGQ89tm6NTjjM9VPIm088od1l6aSorWRWg=
golang.org/x/text v0.3.0/go.mod h1:NqM8EUOU14njkJ3fqMW+pc6Ldnwhi/IjpwHt7yyuwOQ=
```

- First line: hash of the **whole module zip**.
- Second line: hash of **only that module's `go.mod` file**.

The second line lets Go verify transitive metadata without downloading those modules. Both lines are written and checked by the `go` command automatically. Never hand-edit `go.sum`.

## Sources

- [Go modules reference (`go.mod` / `go.sum`) — go.dev/ref/mod](https://go.dev/ref/mod)
- [Build constraints — pkg.go.dev/cmd/go#hdr-Build_constraints](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- [Test packages — pkg.go.dev/cmd/go#hdr-Test_packages](https://pkg.go.dev/cmd/go#hdr-Test_packages)
- [`testing` package — pkg.go.dev/testing](https://pkg.go.dev/testing)
