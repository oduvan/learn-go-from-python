# Multiple Go versions and the minimum-version declaration

## The `go` directive — declares the minimum

In `go.mod`:

```
go 1.23.0
```

From the official toolchain documentation:

> "The `go` line declares the minimum required Go version for using the module or workspace."

What this actually does:

1. **Controls language features.** A `go 1.23.0` module can use language features added through Go 1.23 (e.g. range-over-func iterators introduced in 1.23). Earlier toolchains will refuse to build it.
2. **Acts as a refusal floor.** "Go 1.21.2 will refuse to load a module or workspace with a `go 1.21.3` or `go 1.22` line" (verbatim).
3. **Imposes a transitive lower bound.** Your `go` line must be ≥ the highest `go` line among your dependencies.

## The `toolchain` directive — the preferred build version

```
go 1.21.0
toolchain go1.23.4
```

The `go` line says what minimum the language requires; the `toolchain` line says which version should *actually* be used to build.

If a developer has Go 1.21.0 installed locally and your `go.mod` says `toolchain go1.23.4`, the `go` command will automatically download and switch to Go 1.23.4 for the build (assuming default `GOTOOLCHAIN=auto`).

If you don't write a `toolchain` line, it is implicitly `toolchain go<your-go-line>`. So:

```
go 1.21.0
```

behaves as if you had written:

```
go 1.21.0
toolchain go1.21.0
```

## Auto-switching to newer toolchains

The Go binary you install is also a **bootstrap** that can download other toolchain versions on demand.

> "When a command encounters a module requiring a newer Go version and `GOTOOLCHAIN` permits running different toolchains (it is one of the `auto` or `path` forms), the `go` command chooses and switches to an appropriate newer toolchain to continue executing the current command."

You'll see output like:

```
go: module example.com/widget@v1.2.3 requires go >= 1.24rc1; switching to go 1.27.9
```

Downloaded toolchains are packaged as ordinary modules under the path `golang.org/toolchain` and cached in `$GOMODCACHE` — same proxy, same checksum database as your other dependencies.

## The `GOTOOLCHAIN` environment variable

Set with `go env -w GOTOOLCHAIN=...` (persists across shells) or in the current shell environment.

| Value | Behavior |
|---|---|
| `auto` (default) | Use the bundled toolchain; auto-download a newer one if a module requires it. |
| `local` | Never switch. Always use whatever `go` binary you ran. Older modules still build; modules requiring a newer `go` line will error out. |
| `go1.21.3` (exact) | Always use that specific version. Downloads it if needed. |
| `go1.21.3+auto` | Start with 1.21.3, but still upgrade if a module requires more. |
| `go1.21.3+path` | Start with 1.21.3, allow upgrades, but only from `$PATH` — no downloads. For locked-down or air-gapped environments. |

## Editing the directives — use `go get`, not hand-edits

```bash
go get go@1.22.1                       # raise minimum to 1.22.1
go get go@1.22.1 toolchain@go1.24.0    # raise both
go get toolchain@none                  # remove the explicit toolchain line
```

`go get` handles the interaction between the two lines for you. Raising the `go` line will adjust or drop the `toolchain` line automatically when they would otherwise match.

## Running several Go versions side by side

Three common patterns.

### 1. Toolchain auto-download (the modern default)

With `GOTOOLCHAIN=auto`, you do nothing. Different projects on the same machine transparently use whatever Go versions their `go.mod` files specify.

### 2. The official `go install` pattern

For ad-hoc testing of a specific version without touching your default install:

```bash
go install golang.org/dl/go1.22.3@latest
go1.22.3 download
go1.22.3 version
go1.22.3 build ./...
```

Each version becomes a separate command. The full list is at [go.dev/dl/#stable](https://go.dev/dl/#stable).

### 3. A version manager

`asdf` with the Go plugin, or `gvm`. Useful if you want to globally swap your default Go between several versions outside `go.mod` control.

## A reasonable starting point for new projects

```
module example.com/yourthing

go 1.23
```

Skip the `toolchain` line until you have a concrete reason to pin (such as "every developer and CI must use exactly 1.23.4"). When you do, add it via `go get toolchain@go1.23.4` rather than editing by hand.

## Sources

- [Go toolchains — go.dev/doc/toolchain](https://go.dev/doc/toolchain)
- [`go` directive in `go.mod` — go.dev/ref/mod#go-mod-file-go](https://go.dev/ref/mod#go-mod-file-go)
- [`toolchain` directive — go.dev/ref/mod#go-mod-file-toolchain](https://go.dev/ref/mod#go-mod-file-toolchain)
- [Managing Go installations — go.dev/doc/manage-install](https://go.dev/doc/manage-install)
