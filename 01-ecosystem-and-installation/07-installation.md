# Installing Go

These materials target **Go 1.26.4** (current stable). Check [go.dev/dl](https://go.dev/dl/) for the absolute latest at any moment.

## macOS

Three reasonable paths. Pick one — don't mix them.

### Option 1 — Homebrew (recommended for daily use on macOS)

If you already use Homebrew:

```bash
brew install go
```

- Easy upgrades: `brew upgrade go`.
- Clean uninstall: `brew uninstall go`.
- Install path is Homebrew's prefix (`/opt/homebrew/Cellar/go/...` on Apple Silicon, `/usr/local/Cellar/go/...` on Intel), with a symlink at `$(brew --prefix)/bin/go`.
- Lags official releases by a few days — usually fine.

### Option 2 — Official `.pkg` installer

1. Download from [go.dev/dl](https://go.dev/dl/) — pick `darwin-arm64.pkg` (Apple Silicon) or `darwin-amd64.pkg` (Intel).
2. Double-click and follow the prompts (requires admin password).
3. The installer puts the toolchain at `/usr/local/go` and adds `/usr/local/go/bin` to your `PATH` via `/etc/paths.d/go`.
4. Restart your terminal.

### Option 3 — Manual tarball

For full control over the install location:

```bash
# download go1.26.4.darwin-arm64.tar.gz from https://go.dev/dl/

sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf ~/Downloads/go1.26.4.darwin-arm64.tar.gz
```

Add `/usr/local/go/bin` to your `PATH` in `~/.zshrc` (or `~/.bash_profile`):

```bash
export PATH=$PATH:/usr/local/go/bin
```

Reload your shell and verify (see "Verifying the install" below).

## Linux

The official path is the tarball. Distribution packages (`apt install golang-go`, `dnf install golang`, etc.) often lag the official release by several months — fine for casual use, not recommended if you want current features.

1. Download the matching `.tar.gz` from [go.dev/dl](https://go.dev/dl/) for your architecture.
2. Remove any previous install and extract:

```bash
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.4.linux-amd64.tar.gz
```

3. Add `/usr/local/go/bin` to your `PATH` by appending this line to `~/.profile` (or `~/.bashrc`):

```bash
export PATH=$PATH:/usr/local/go/bin
```

4. Apply and verify:

```bash
source ~/.profile
go version
```

## Windows

1. Download the `.msi` from [go.dev/dl](https://go.dev/dl/).
2. Double-click; the installer puts Go in `Program Files` and adds it to `PATH`.
3. Close and reopen any open command prompts so they pick up the new `PATH`.
4. Verify in a new prompt:

```cmd
go version
```

## Verifying the install

After installation, run three quick checks:

```bash
go version
# example: go version go1.26.4 darwin/arm64

go env GOROOT
# wherever the toolchain landed (e.g. /usr/local/go)

go env GOPATH
# defaults to $HOME/go — holds module cache and installed tools
```

Then run a hello-world to confirm the whole pipeline works end-to-end:

```bash
mkdir -p /tmp/hello && cd /tmp/hello
go mod init example.com/hello
cat > hello.go <<'EOF'
package main

import "fmt"

func main() {
    fmt.Println("hello, go")
}
EOF
go run .
# hello, go
```

If `go version` works but `go run .` does not, the toolchain is installed correctly and the failure is in your code or in `PATH` for the module cache.

## Installing additional versions side-by-side

Once any Go is installed, you can fetch others through the official `dl/` mechanism — no version manager required:

```bash
go install golang.org/dl/go1.22.3@latest
go1.22.3 download
go1.22.3 version
go1.22.3 build ./...
```

Each version becomes its own command (`go1.22.3`, `go1.23.0`, etc.). They install under `~/sdk/<version>/` and don't disturb your primary `go` binary.

Find each one's `GOROOT`:

```bash
go1.22.3 env GOROOT
```

Uninstall a side-by-side version by removing its `GOROOT` directory and the `goX.Y.Z` binary from `$GOBIN`.

See [06-multiple-go-versions.md](06-multiple-go-versions.md) for the bigger picture — `GOTOOLCHAIN=auto` handles the per-project version selection automatically once you have any Go installed.

## Uninstalling Go

### macOS (`.pkg` install)

```bash
sudo rm -rf /usr/local/go
sudo rm /etc/paths.d/go
```

Optionally also remove `~/go/` to wipe the module cache and installed tools.

### macOS (Homebrew install)

```bash
brew uninstall go
```

### Linux

```bash
sudo rm -rf /usr/local/go
# then remove the PATH line from ~/.profile or ~/.bashrc
```

### Windows

Control Panel → **Add/Remove Programs** → **Go Programming Language** → **Uninstall**. Environment variables are cleaned up automatically.

## From Python

A few things to recalibrate after years of Python:

- **One toolchain ships everything.** Compiler, linker, test runner, formatter, dependency manager — all the `go` binary. There is no analog to needing `pip`, `venv`, `pytest`, `black`, and `flake8` as separate installs.
- **No per-project virtual environments.** Module isolation comes from `go.mod` files at each project's root; dependencies are cached globally under `$GOPATH/pkg/mod/` but resolved per-module. You don't `activate` anything.
- **`$GOBIN` ≈ `pipx`'s install location.** After `go install <tool>@latest`, the binary lives in `~/go/bin/` — a single global place for CLI tools, no virtualenvs involved.
- **No "system Python" trap.** macOS doesn't ship Go, so there's no risk of accidentally upgrading the OS's Go. Whatever you install is the only Go.

## Sources

- [Download and install — go.dev/doc/install](https://go.dev/doc/install)
- [All Go releases — go.dev/dl](https://go.dev/dl/)
- [Managing Go installations — go.dev/doc/manage-install](https://go.dev/doc/manage-install)
- [Tutorial: Get started with Go — go.dev/doc/tutorial/getting-started](https://go.dev/doc/tutorial/getting-started)
