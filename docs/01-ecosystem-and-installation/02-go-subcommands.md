# The `go` command — subcommands reference

The `go` command is the umbrella tool for the entire Go workflow. One binary handles compiling, running, testing, formatting, static analysis, dependency management, documentation, and more.

Run `go help` for the full list of subcommands and `go help <subcommand>` for details on any one.

## Building and running code

### `go run`

Compiles and runs a Go program in one step. The binary is written to a temporary location and discarded after execution. Use for quick experiments.

```bash
go run .                    # run the package in the current directory
go run main.go              # run a single file
go run ./cmd/server         # run the package at that path
```

> **From Python:** roughly `python script.py` — except Go always compiles first, then runs. There is no interpreter mode.

### `go build`

Compiles to a binary in the current directory (or to the path given by `-o`). Does not run it. The binary bundles the Go runtime, so there is no interpreter to install on the target.

It is **not** automatically fully static, though. **cgo** is the part of Go that lets Go code call C code, and it is on by default when a C toolchain is present. With cgo on, importing `net` or `os/user` links against the system libc:

```bash
$ go build -o app . && file app
app: ELF 64-bit LSB executable, x86-64, ..., dynamically linked, ...

$ CGO_ENABLED=0 go build -o app . && file app
app: ELF 64-bit LSB executable, x86-64, ..., statically linked, ...
```

Set `CGO_ENABLED=0` when you need a binary that runs in a `scratch` or distroless container. Add `-trimpath` to keep your absolute source paths out of the binary.

```bash
go build                    # produce ./<name>
go build -o bin/app .       # custom output path
go build ./...              # build every package in the module
```

Common flags:

- `-o <path>` — output file path.
- `-race` — enable the data race detector (adds ~5–10× memory overhead at runtime).
- `-tags <tag>` — enable build tags.
- `-ldflags '...'` — flags to the linker; commonly used to inject version strings, e.g. `-ldflags "-X main.Version=$(git rev-parse HEAD)"`.

> **From Python:** no direct analog. Closest is `pyinstaller`/`shiv`/`pex` — but Go builds are first-class and produce a single static binary with the runtime included.

### `go install`

Builds a binary and installs it into `$GOBIN` (defaults to `$HOME/go/bin`). The intended use is installing third-party CLI tools.

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

The `@version` (or `@latest`) suffix is required when running outside a module. Add `$GOBIN` to your `$PATH` to invoke the installed tools.

> **From Python:** ≈ `pipx install <package>` — drops a CLI tool somewhere on `$PATH` without polluting your project's dependencies.

## Dependencies and modules

### `go mod init <module-path>`

Creates a new `go.mod` file. The module path doubles as the import path under which other code references this module. For learning projects, any path works: `example.com/hello`.

> **From Python:** ≈ `poetry init` or `hatch new` — creates the project manifest. There is no equivalent to `pyproject.toml`'s `[tool.<thing>]` sprawl; `go.mod` is small and uniform.

### `go mod tidy`

Reconciles `go.mod` with the imports actually used in your source: adds missing dependencies, removes unused ones, refreshes `go.sum`. Run this whenever imports change.

> **From Python:** ≈ `poetry lock` or `pip-compile` — but driven by the actual `import` statements in your code, not a separate `requirements.in`.

### `go mod download`

Pre-fetches modules into the local cache without building. Useful in CI when you want to separate dependency fetching from compilation.

### `go mod why <pkg>`

Prints the import chain that pulls a dependency into your build. Indispensable when you want to understand why some unexpected module showed up.

> **From Python:** ≈ `pipdeptree --reverse --packages <pkg>`.

### `go mod graph`

Prints the full module dependency graph as edges.

> **From Python:** ≈ `pipdeptree` (plain output).

### `go mod vendor`

Copies all dependencies into a local `vendor/` directory. Subsequent builds use the vendored copies instead of `$GOMODCACHE`. Useful for air-gapped or fully reproducible builds. Optional.

### `go get`

Adds, upgrades, or downgrades a dependency in `go.mod`. `go get` only manages dependencies — it does **not** install CLI binaries (that role belongs to `go install`).

```bash
go get github.com/some/pkg                  # add at latest
go get github.com/some/pkg@v1.2.3           # pin to version
go get -u ./...                             # upgrade all deps to latest minor/patch
go get go@1.23.0                            # change the Go version in go.mod
```

> **From Python:** ≈ `poetry add <pkg>` — edits the manifest and refreshes the lockfile in one step. *Not* like `pip install`, which only mutates the environment.

### `go work`

Manages multi-module workspaces — when several local modules should see each other's source without `replace` directives. Subcommands: `go work init`, `go work use`, `go work sync`.

> **From Python:** loosely like `pip install -e <local-path>` across several sibling projects, or `uv`/`pdm` workspace mode.

## Quality and correctness

### `go test`

Runs tests in `*_test.go` files. Testing is built into the language — no external framework.

```bash
go test ./...                       # all packages
go test -v                          # verbose output
go test -run TestFoo                # filter test names (regex)
go test -race                       # run with the race detector
go test -cover                      # coverage summary
go test -coverprofile=c.out         # write coverage profile
go test -bench=.                    # also run benchmarks
go test -count=1                    # disable result caching
go test -trace=trace.out            # produce an execution trace
```

> **From Python:** ≈ `pytest`, but **shipped with the language** — no third-party install, no fixture/plugin ecosystem to learn. Benchmarks and fuzz tests are first-class citizens, not separate tools.

### `go vet`

Static analysis for common mistakes that compile but are likely bugs: printf format mismatches, unreachable code, lock copying, suspicious shadowing, etc. The analyzers are conservative — false positives are rare, so CI typically treats `go vet` failures as build failures.

```bash
go vet ./...
```

> **From Python:** ≈ `pyflakes` — a conservative subset of what `pylint`/`ruff` would catch. Tuned to almost never produce false positives, so CI treats failures as build breaks.

### `go fmt` / `gofmt`

Formats Go source to the canonical style. There is **one** Go style — `gofmt` has no formatting options. Most editors run it automatically on save.

```bash
go fmt ./...                # format all packages
gofmt -d file.go            # show diff without writing
gofmt -s file.go            # apply additional simplifications
```

> **From Python:** ≈ `black` — one canonical style, no options, no debates. `gofmt` predates `black` by several years and was the inspiration for it.

## Inspecting

### `go env`

Prints Go's environment variables: `GOROOT`, `GOPATH`, `GOOS`, `GOARCH`, `GOMODCACHE`, `GOPROXY`, and more. Use `-w` to persistently set a value (writes to `$HOME/.config/go/env`).

```bash
go env                              # print everything
go env GOPATH                       # one variable
go env -w GOPROXY=direct            # set persistently
```

Key variables:

- `GOROOT` — where the Go toolchain is installed. Usually managed automatically.
- `GOPATH` — historic workspace root; today mainly contains `pkg/mod` (cache) and `bin/` (installed tools).
- `GOMODCACHE` — module cache directory (default `$GOPATH/pkg/mod`).
- `GOPROXY` — module proxy URL (default `https://proxy.golang.org,direct`).
- `GOTOOLCHAIN` — controls toolchain version selection (see the multi-version article).
- `GOOS` / `GOARCH` — target OS and architecture for the next build.

> **From Python:** loosely like inspecting `python -m sysconfig` together with the relevant `PY*` env vars — there is no clean single-command analog because Python's tooling state is spread across several files and processes.

### `go version`

Prints the toolchain version. Can also report which Go version compiled an existing binary:

```bash
go version
go version ./mybinary
```

> **From Python:** ≈ `python --version`. The second form (reading a binary) has no Python parallel.

### `go list`

Prints metadata about packages or modules. Highly scriptable through Go templates.

```bash
go list ./...                              # every package in module
go list -m all                             # every module in dep graph
go list -m -u all                          # show available upgrades
go list -f '{{.ImportPath}}' ./...         # custom format
```

> **From Python:** the `-m` forms ≈ `pip list` / `pip list --outdated`. The non-`-m` form (listing packages inside the current project) has no direct analog — it's closer to walking `pkgutil.iter_modules`.

### `go doc`

Prints documentation for a package, type, function, or variable directly from source.

```bash
go doc fmt.Println
go doc fmt
go doc -all fmt
go doc -src fmt.Println
```

> **From Python:** ≈ `pydoc fmt.Println`. Same idea: read docstrings straight out of the installed source.

## Cross-compilation

Go cross-compiles natively — no separate toolchain to install per target — provided you don't use cgo.

```bash
GOOS=linux   GOARCH=amd64 go build -o app-linux .
GOOS=darwin  GOARCH=arm64 go build -o app-mac-m1 .
GOOS=windows GOARCH=amd64 go build -o app.exe .
```

`go tool dist list` prints every supported OS/architecture combination.

> **From Python:** no real analog. Python ships source and relies on a target-platform interpreter; Go produces a native binary for whatever OS/arch you ask for, from any host.

## Less-common but worth knowing

- **`go generate`** — runs commands found in `//go:generate` directives inside source files. Used to invoke code generators (protobuf, stringer, mockgen). *From Python:* ≈ running a codegen `pre-commit` hook or a `make generate` target.
- **`go tool`** — entrypoint for internal tools: `go tool pprof`, `go tool trace`, `go tool objdump`, etc.
- **`go clean`** — removes build artifacts; `go clean -modcache` wipes downloaded modules. *From Python:* ≈ `rm -rf __pycache__/ build/ dist/` plus clearing `~/.cache/pip`.
- **`go bug`** — opens a pre-filled bug report for the Go project.
- **`go fix`** — rewrites old code to use newer APIs; rarely needed today. *From Python:* the historic analog is `2to3`; the modern analog is `pyupgrade`.

## The 80% you'll reach for daily

`go run`, `go build`, `go test`, `go mod tidy`, `go fmt`, `go vet`, `go get`, `go install`, `go env`. Everything else surfaces when a specific need does.

## Sources

- [`go` command reference — pkg.go.dev/cmd/go](https://pkg.go.dev/cmd/go)
- [Tutorial: Get started with Go](https://go.dev/doc/tutorial/getting-started)
- [Go modules reference](https://go.dev/ref/mod)
