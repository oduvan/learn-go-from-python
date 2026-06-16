# Project layout and workspaces

Go is opinionated about a few directory conventions and deliberately silent
about the rest. This article covers the layout patterns the tooling
actually understands, and **workspaces** for developing several modules at
once.

## The conventions the tooling enforces

Two directory names have real meaning to the `go` command:

- **`internal/`** — packages under an `internal/` directory can be imported
  only by code rooted at `internal/`'s parent. It's compiler-enforced
  privacy at the package-tree level.
- **`testdata/`** — ignored by the build tooling; a place for test
  fixtures.

```text
example.com/shop/
├── go.mod
├── internal/
│   └── auth/         # importable only within example.com/shop/...
└── store/
    └── testdata/     # fixtures, ignored by the compiler
```

Everything else about layout is convention, not rule.

## `cmd/` and the common layout

A widely used (but optional) shape separates entry points from library
code:

- **`cmd/<name>/`** — one directory per executable, each with its own
  `package main`. The directory name becomes the binary name.
- **`internal/`** — private packages, the bulk of the code.
- top-level packages — the module's public API, if it's meant to be
  imported.

```text
myapp/
├── go.mod
├── cmd/
│   ├── server/main.go     # builds the "server" binary
│   └── cli/main.go        # builds the "cli" binary
├── internal/
│   ├── store/
│   └── auth/
└── api/                   # exported, importable by others
```

Build a specific command with its path:

```bash
go build ./cmd/server      # produces ./server
go install ./cmd/cli       # installs the "cli" binary
```

> **From Python:** there's no `src/` requirement and no `__init__.py`. A
> directory *is* a package by virtue of its `.go` files; `cmd/` and
> `internal/` are the rough analogues of a scripts/entrypoints folder and
> a private subpackage.

## Keep `main` thin

A strong convention: `package main` should do as little as possible — parse
flags, wire things together, call into `internal/` packages — so the real
logic stays testable and importable. The binary is glue; the packages are
the program.

## Workspaces: developing multiple modules together

When you're changing two modules at once — say an app and a library it
depends on — editing `go.mod` with a `replace` for each works but is
fiddly and easy to commit by accident. A **workspace** solves this with a
`go.work` file that tells the `go` command to use several local modules
together.

```bash
go work init ./app ./lib
```

```text
// go.work
go 1.25.0

use (
    ./app
    ./lib
)
```

Now, building or testing from anywhere in the workspace resolves imports of
`./lib` to your local checkout — no `replace` directives needed. Add more
with `go work use ./other`.

The key practice: **`go.work` is local-only**. It's for your machine's
multi-module dev loop, so it's typically **git-ignored**, never published.
Released builds still resolve dependencies through `go.mod`/`go.sum`.

## Quick reference

| Path / file | Meaning |
|---|---|
| `internal/` | importable only within the parent module subtree |
| `testdata/` | test fixtures, ignored by the build |
| `cmd/<name>/` | one executable per subdirectory (`package main`) |
| `go build ./cmd/x` | build a specific command |
| `go.work` (`go work init`/`use`) | use several local modules together |
| keep `main` thin | logic lives in importable packages |

## Sources

- [Internal packages — go.dev/ref/spec#Internal_packages](https://go.dev/ref/spec#Internal_packages)
- [Workspaces tutorial — go.dev/doc/tutorial/workspaces](https://go.dev/doc/tutorial/workspaces)
- [`go.work` reference — go.dev/ref/mod#workspaces](https://go.dev/ref/mod#workspaces)
- [Organizing a Go module — go.dev/doc/modules/layout](https://go.dev/doc/modules/layout)
