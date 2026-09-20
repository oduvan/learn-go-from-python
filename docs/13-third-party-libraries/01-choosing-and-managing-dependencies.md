# Choosing and managing dependencies

Everything from here on lands in `go.mod`. The rest of this book is the
language and its standard library, which stay true whatever you build;
this section is one stack's worth of choices, and choices need
maintaining.

```bash
go get github.com/google/uuid
go mod tidy
```

## Judge a module before adding it

A dependency is a permanent commitment to somebody else's judgement.
Worth two minutes:

- **Is it maintained?** Recent commits, issues being answered, releases
  that are not all from one weekend three years ago.
- **How big is its own dependency tree?** `go mod graph` after adding
  it. A library pulling in forty transitive modules brings forty more
  things that can break.
- **Is it v1?** A `v0` module makes no compatibility promise. That is
  fine for a tool, riskier for something in your request path.
- **Could you write it?** A twenty-line helper is not worth a module.
  Go's standard library is broad enough that many popular packages in
  other ecosystems have no equivalent here because they are not needed.
- **What does it cost to remove?** A library behind your own interface
  is replaceable. One whose types appear in every signature is not.

The standard library is the default. Reach outside it when the problem
is genuinely large — a web framework, a database driver, an ORM — not
to save ten lines.

## `go.mod` and `go.sum`

`go.mod` records the module path, the Go version, and direct
requirements. `go.sum` records cryptographic hashes of every module in
the graph, direct and indirect.

**Commit both.** `go.sum` is what makes builds reproducible and what
detects a tampered dependency: a hash mismatch fails the build rather
than silently running different code.

`// indirect` marks a requirement nothing in your code imports
directly — it is there because a dependency needs it.

```bash
go mod tidy      # add what is used, drop what is not
go mod graph     # the whole dependency graph
go mod why <pkg> # why is this here at all
```

`go mod why` is the one to reach for when `tidy` adds something
unexpected. Run `tidy` before every commit that touches imports;
CI should check it produces no diff.

## Versions and upgrading

Modules use semantic versioning, and the tooling takes it literally.

```bash
go get github.com/x/y@v1.4.2     # exact
go get github.com/x/y@latest     # newest release
go get -u ./...                  # upgrade minor and patch
go get github.com/x/y@none       # remove
```

Go uses **minimal version selection**: the build picks the *lowest*
version satisfying every requirement, not the highest available. Builds
are therefore reproducible without a lockfile — upgrades happen when
you ask, never on someone else's schedule.

`go get -u` bumps minor and patch versions but never major, because a
major version is a different module path.

## The `/vN` rule

From v2 onward the major version is part of the import path:

```go
import "github.com/pelletier/go-toml/v2"
import "github.com/jackc/pgx/v5"
```

Awkward at first, and the reason it exists is good: two major versions
can coexist in one build, so a transitive dependency on v1 does not
block your upgrade to v2. Migration can be gradual rather than a
flag day.

## The proxy, the checksum database, and private code

By default `go get` fetches through `proxy.golang.org` and verifies
against `sum.golang.org`. That gives you availability when an upstream
repository disappears, and tamper detection.

Neither should see your private code. One variable covers it:

```bash
export GOPRIVATE=gitlab.example.com/*,github.com/myorg/*
```

`GOPRIVATE` sets both `GONOPROXY` and `GONOSUMDB`, so matching modules
are fetched directly over git and skipped for checksum verification.
Without it, a private fetch fails confusingly — and worse, the module
path leaks to a public service.

## Vulnerability scanning

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

`govulncheck` is better than a generic scanner because it uses call
graph analysis: it reports a vulnerability only if your code actually
reaches the affected function. Far fewer false positives, so the output
stays worth reading. Run it in CI.

## Automated upgrades

Dependabot and Renovate open pull requests for new versions. They are
worth enabling, with two conditions: your test suite has to be good
enough that a green build means something, and someone has to actually
merge them. A repository with ninety open dependency PRs is worse off
than one with none, because the real security update is buried.

Group patch updates, keep major versions manual.

## Vendoring

```bash
go mod vendor
```

Copies every dependency into `vendor/`, and the build then uses it
exclusively. Occasionally justified — an air-gapped build, a hard
requirement to review every byte you ship. Mostly the module cache and
proxy already solve the problems vendoring solved, and it adds a large
directory to every diff.

## Replacing a module

```go
replace github.com/x/y => ../y
replace github.com/x/y => github.com/me/y v1.4.3-fork
```

For local development against two modules at once, prefer a
[workspace](../04-packages-and-modules/03-project-layout-and-workspaces.md) —
`go.work` is not committed, so it cannot leak into a release. A
`replace` pointing at a local path will break everyone else's build the
moment it is merged.

## Keep the count honest

Every dependency is code you did not write, running with your
privileges, that someone else can change. Periodically:

```bash
go mod graph | wc -l
```

If that number only grows, nobody is reading it. Removing a dependency
you outgrew is as much maintenance as adding one.

> **From Python:** `go.mod` is `pyproject.toml` and `go.sum` is the
> lockfile, except there is no virtualenv — dependencies live in a
> shared module cache, versioned by path, so two projects never
> conflict. Minimal version selection is the opposite of pip's
> resolver: you get the lowest version that works, and upgrades are
> always deliberate.

## Quick reference

| Task | Command |
|---|---|
| add | `go get module@version` |
| sync `go.mod` | `go mod tidy` — in CI too |
| why is this here | `go mod why <pkg>` |
| upgrade minor/patch | `go get -u ./...` |
| remove | `go get module@none` |
| major versions | `/v2`, `/v5` in the import path |
| private modules | `GOPRIVATE=host/org/*` |
| vulnerabilities | `govulncheck ./...` |
| local development | a `go.work` workspace, not `replace` |
| commit | both `go.mod` and `go.sum` |

## Sources

- [Go modules reference — go.dev/ref/mod](https://go.dev/ref/mod)
- [Managing dependencies — go.dev/doc/modules/managing-dependencies](https://go.dev/doc/modules/managing-dependencies)
- [Module version numbering — go.dev/doc/modules/version-numbers](https://go.dev/doc/modules/version-numbers)
- [`govulncheck` — pkg.go.dev/golang.org/x/vuln/cmd/govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)
- [Go module proxy and privacy — go.dev/ref/mod#private-modules](https://go.dev/ref/mod#private-modules)
