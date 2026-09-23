# golangci-lint configuration

[Dependency direction](../11-architecture-and-conventions/04-dependency-direction.md)
argued that which package may import which is the architectural
decision worth protecting. This is how you stop protecting it by hand.

> **Module:** `github.com/golangci/golangci-lint` — a tool, not an
> import. Install the binary; nothing enters your `go.mod`.

```bash
golangci-lint run ./...
```

## What it is

A runner that executes many linters over one parsed program, so you pay
the type-checking cost once. `go vet` is included; so are `staticcheck`,
`errcheck`, `ineffassign` and `unused`.

Pin the version in CI. Linters gain checks between releases, and an
unpinned tool means a build that passed yesterday fails today with no
code change.

## The config file

`.golangci.yml` at the repo root. Version 2 of the format:

```yaml
version: "2"

linters:
  enable:
    - depguard
    - forbidigo
```

The default set — `errcheck`, `govet`, `ineffassign`, `staticcheck`,
`unused` — is already on. Adding `enable` entries extends it.

Resist enabling everything. A linter whose findings you routinely
suppress is worse than one you never turned on, because it trains
people to add `//nolint` without reading.

## `depguard`: which package may import what

This is the linter that makes an architecture rule real. The rule
below says only the storage layer may touch `database/sql`:

```yaml
linters:
  settings:
    depguard:
      rules:
        upper-layers:
          files:
            - "$all"
            - "!$test"
            - "!**/internal/storage/**"
          deny:
            - pkg: database/sql
              desc: "only internal/storage may hold a database handle"
```

A handler importing it now fails the build with the reason attached:

```
internal/web/handler.go:5:2: import 'database/sql' is not allowed from
list 'upper-layers': only internal/storage may hold a database handle (depguard)
```

### Scope by exclusion, not inclusion

The `files` list is the load-bearing part. `"$all"` minus the packages
that *are* allowed means a **new package is covered the day someone
creates it**. List the allowed packages instead and every new one is
unprotected until somebody remembers to add it — which is precisely
when the rule stops working.

`$all` and `$test` are built-in selectors: `$all` matches every Go file,
and `$test` matches only `_test.go` files. A leading `!` excludes, so
`"!$test"` leaves tests out.

Always write `desc`. It becomes the error message, and a rule that
explains itself gets followed instead of worked around.

The same shape guards a contracts package:

```yaml
services:
  files: ["**/internal/services/**"]
  deny:
    - pkg: gorm.io/gorm
      desc: "internal/services declares store interfaces; it does not implement them"
```

## `forbidigo`: ban an identifier

Where `depguard` bans an import, `forbidigo` bans a name:

```yaml
forbidigo:
  analyze-types: true
  forbid:
    - pattern: '^fmt\.Print.*$'
      msg: "use log/slog, not fmt.Print"
```

```
internal/web/extra.go:5:16: use of `fmt.Println` forbidden because
"use log/slog, not fmt.Print" (forbidigo)
```

**`analyze-types: true` is not optional** when you are matching
qualified names. Without it, matching is textual: it sees the
identifier as written, so an aliased import slips through and a local
variable with a matching name produces a false positive. Turning it on
makes the match type-aware.

This is how you allow a library's error sentinel while banning its
types — permitting `gorm.ErrRecordNotFound` everywhere, since that is
how stores report not-found, while forbidding `gorm.DB` outside the
storage layer.

## `//nolint` needs a reason

```go
//nolint:forbidigo // CLI output, not a log line
func Shout() { fmt.Println("hello") }
```

Name the specific linter, and write the reason after `//`. A bare
`//nolint` disables everything on that line, which hides the next bug
too.

You can require this:

```yaml
linters:
  settings:
    nolintlint:
      require-explanation: true
      require-specific: true
      allow-unused: false
```

`allow-unused: false` also flags directives that no longer suppress
anything, so they get deleted rather than accumulating.

## Generated code and paths

```yaml
linters:
  exclusions:
    generated: lax
    paths:
      - ".*_templ\\.go$"
```

`generated: lax` skips files carrying the `// Code generated ... DO NOT
EDIT.` marker from
[build and codegen](../11-architecture-and-conventions/05-build-codegen-and-cgo.md).
There is no point reporting style issues in output nobody edits.

`paths` excludes files by name instead of by marker. Each entry is a
regular expression matched against the file path; matching files are
still analysed, but their issues are not reported. Here it names
templ's `_templ.go` output.

## Formatting

```yaml
formatters:
  enable:
    - gofmt
    - goimports
```

```bash
golangci-lint fmt ./...
```

In v2 formatters are a separate section from linters, and `fmt` applies
them. Let the tool do this; formatting is not worth a code review
comment.

## In CI

```yaml
- name: Lint
  run: golangci-lint run --timeout=5m ./...
```

Raise the timeout on a large codebase — the default is short enough to
fail a cold cache. Cache `~/.cache/golangci-lint` between runs.

Introducing the tool to an existing codebase produces thousands of
findings. `--new-from-rev=origin/main` reports only what your change
introduced, which lets you adopt it without a cleanup sprint first.

## What it cannot do

It cannot check the conventions in
[project conventions](../11-architecture-and-conventions/06-project-conventions.md):
whether an error message is useful, whether a comment explains why,
whether a name is honest. Those stay with reviewers. The point of
configuring the linter well is that reviewers get to spend their
attention on them.

> **From Python:** this is `ruff` — one fast runner over many checks —
> with the addition that `depguard` and `forbidigo` let you encode
> architecture rules, which `import-linter` does separately in Python.

## Quick reference

| Task | Form |
|---|---|
| config | `.golangci.yml`, `version: "2"` |
| run | `golangci-lint run ./...`, pinned version in CI |
| restrict imports | `depguard`, with `desc` on every rule |
| cover new packages automatically | scope with `"$all"` minus exceptions |
| ban an identifier | `forbidigo` with **`analyze-types: true`** |
| suppress | `//nolint:linter // reason` |
| enforce that | `nolintlint` with `require-explanation` |
| skip generated files | `exclusions.generated: lax` |
| format | `formatters` section, `golangci-lint fmt ./...` |
| adopt gradually | `--new-from-rev=origin/main` |

## Sources

- [golangci-lint — golangci-lint.run](https://golangci-lint.run/)
- [Configuration reference — golangci-lint.run/docs/configuration/file/](https://golangci-lint.run/docs/configuration/file/)
- [depguard — github.com/OpenPeeDeeP/depguard](https://github.com/OpenPeeDeeP/depguard)
- [forbidigo — github.com/ashanbrown/forbidigo](https://github.com/ashanbrown/forbidigo)
- [`go vet` — pkg.go.dev/cmd/vet](https://pkg.go.dev/cmd/vet)
