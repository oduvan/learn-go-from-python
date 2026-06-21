# Creating and publishing a module

A **module** is a collection of packages versioned together — the unit Go
uses to distribute and depend on code. It's defined by a `go.mod` file at
its root, which names the module and records its dependencies. One module
typically maps to one repository.

## Starting a module

`go mod init` creates `go.mod`. The argument is the **module path** — the
import prefix for every package inside, and (for published modules) the
place it can be downloaded from.

```bash
go mod init example.com/shop
```

```go
// go.mod
module example.com/shop

go 1.25.0
```

The `module` line sets the import prefix: a package in `store/` is now
imported as `example.com/shop/store`. The `go` line records the language
version the module targets.

> **From Python:** `go.mod` is the rough equivalent of `pyproject.toml` —
> it names the project and pins dependencies. There's no virtualenv: the
> module graph is resolved per-build and dependencies are cached globally.

## Adding a dependency

Import a package and let the tooling fetch it. `go get` adds it; `go mod
tidy` reconciles `go.mod`/`go.sum` with what the code actually imports.

```bash
go get github.com/google/uuid
go mod tidy        # add missing, drop unused
```

This records the dependency and its exact version in `go.mod`, and writes
cryptographic checksums to **`go.sum`** so future builds verify they got
the identical bytes.

```go
// go.mod
require github.com/google/uuid v1.6.0
```

`go.sum` is not a lockfile in the npm sense — `go.mod` already pins exact
versions; `go.sum` is the integrity record. Commit both.

## Semantic versioning and the module cache

Go modules use **semantic versioning** (`vMAJOR.MINOR.PATCH`). Version
selection is **Minimal Version Selection**: a build uses the *lowest*
version that satisfies all requirements — builds are reproducible without
a separate lockfile. Downloaded modules live in a global, read-only cache
shared across projects.

Upgrades are explicit:

```bash
go get github.com/google/uuid@latest   # newest
go get github.com/google/uuid@v1.6.0    # a specific version
go get github.com/google/uuid@none      # remove
```

## Major versions: the `/vN` rule

A breaking change means a new **major** version, and from `v2` onward the
major version becomes part of the module path:

```go
module example.com/shop/v2
```

Importers then write `example.com/shop/v2/store`. This lets `v1` and `v2`
coexist in one build — a different answer to dependency conflicts than
Python's single-version-per-environment model.

## Publishing

There is no central registry to upload to. **Publishing a module is just
pushing a tagged commit** to a public repository whose URL matches the
module path.

```bash
git tag v1.0.0
git push origin v1.0.0
```

The first time someone runs `go get example.com/shop@v1.0.0`, the Go
tooling fetches it straight from the repo (often via the module proxy) and
records the checksum. The module path must match the repo location so the
tooling knows where to look.

## Replacing a dependency (local or forked)

The `replace` directive swaps a dependency for another version, a fork, or
a local path — invaluable while developing two modules side by side:

```go
// go.mod
require example.com/lib v1.2.0
replace example.com/lib => ../lib    // use local checkout instead
```

`replace` is build-local: it affects *this* module's builds only, not
anyone who depends on you.

## Quick reference

| Command / directive | Purpose |
|---|---|
| `go mod init <path>` | create `go.mod`, set the module path |
| `go get <pkg>@<ver>` | add / upgrade / pin a dependency |
| `go get <pkg>@none` | remove a dependency |
| `go mod tidy` | sync `go.mod`/`go.sum` with imports |
| `go.sum` | checksums for integrity (commit it) |
| `require` | a dependency and its version |
| `replace A => B` | swap a dependency (local/fork) |
| `/vN` in module path | major versions ≥ 2 |
| `git tag vX.Y.Z` + push | publish a version |

## Sources

- [Go Modules Reference — go.dev/ref/mod](https://go.dev/ref/mod)
- [Managing dependencies — go.dev/doc/modules/managing-dependencies](https://go.dev/doc/modules/managing-dependencies)
- [Module version numbering — go.dev/doc/modules/version-numbers](https://go.dev/doc/modules/version-numbers)
- [Publishing a module — go.dev/doc/modules/publishing](https://go.dev/doc/modules/publishing)
- [Minimal Version Selection — go.dev/ref/mod#minimal-version-selection](https://go.dev/ref/mod#minimal-version-selection)
