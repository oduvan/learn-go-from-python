# Special folders in a Go project

There are exactly **two** folder names that the `go` tool treats specially. Everything else is community convention.

## `internal/` — enforced visibility

The compiler enforces a rule about `internal/`. From the `cmd/go` reference:

> An import of a path containing the element "internal" is disallowed if the importing code is outside the tree rooted at the parent of the "internal" directory.

In practice, if your module is `example.com/app`:

```
app/
├── go.mod
├── api/
│   └── handler.go        # may import .../internal/auth — same module
└── internal/
    └── auth/
        └── token.go      # only packages under app/... can import this
```

A *different* module attempting `import "example.com/app/internal/auth"` will fail to compile.

This is how libraries declare "this is an implementation detail — I will not promise to keep it stable." It is the strongest visibility mechanism in Go.

## `testdata/` — ignored by the build

From the `cmd/go` reference:

> "The go tool will ignore a directory named 'testdata', making it available to hold ancillary data needed by the tests."

```
foo/
├── foo.go
├── foo_test.go
└── testdata/
    ├── input1.json
    └── golden/
        └── expected.txt
```

Anything inside `testdata/` is invisible to the build: Go won't try to compile it, won't complain about non-Go files inside, won't include it in dependency graphs. It's the standard place for test fixtures, golden files, malformed inputs to fuzz tests, etc.

## Conventions (not enforced) — still worth knowing

| Folder | Meaning |
|---|---|
| `cmd/<name>/` | Each subdirectory holds a `main` package producing one binary (`cmd/server`, `cmd/cli`). Standard layout when a repository contains multiple executables. |
| `pkg/` | Older convention for library packages. Modern Go doesn't need it — put packages at the module root. Don't add it just because you saw it elsewhere. |
| `vendor/` | If present, `go build` uses it instead of `$GOMODCACHE`. Created by `go mod vendor`. For air-gapped or fully reproducible builds. |
| `api/`, `web/`, `configs/`, etc. | From the unofficial "golang-standards/project-layout" repo. **Not endorsed by the Go team.** Treat as one team's opinion. |

## Files and directories ignored by the `go` tool

In addition to `testdata/`, the tool ignores:

- Any file or directory whose name starts with `_` (e.g. `_scratch.go`, `_drafts/`).
- Any file or directory whose name starts with `.` (e.g. `.idea/`).

These can be used for scratch work that should stay near the code without participating in builds.

## A common, recommended layout

For a project with multiple executables and shared internal code:

```
myapp/
├── go.mod
├── go.sum
├── README.md
├── cmd/
│   ├── api-server/
│   │   └── main.go
│   └── worker/
│       └── main.go
├── internal/
│   ├── auth/
│   │   └── auth.go
│   └── metrics/
│       └── metrics.go
├── api.go                 # public package at the module root
└── api_test.go
```

For a single-purpose library, the simplest form is fine:

```
greetings/
├── go.mod
├── greetings.go
└── greetings_test.go
```

## Sources

- [Organizing a Go module — go.dev/doc/modules/layout](https://go.dev/doc/modules/layout)
- [`internal/` packages rule — pkg.go.dev/cmd/go#hdr-Internal_Directories](https://pkg.go.dev/cmd/go#hdr-Internal_Directories)
- [`testdata` and ignored files — pkg.go.dev/cmd/go#hdr-Test_packages](https://pkg.go.dev/cmd/go#hdr-Test_packages)
