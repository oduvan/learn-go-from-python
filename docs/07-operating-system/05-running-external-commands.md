# Running external commands

`os/exec` runs other programs. It does **not** run a shell, which is the
single most important thing to understand about it — and the reason
whole categories of injection bug simply do not arise.

```go
out, err := exec.CommandContext(ctx, "echo", "hello world").Output()
fmt.Printf("%q %v\n", string(out), err)
// output: "hello world\n" <nil>
```

## Always take a context

Prefer `exec.CommandContext` over `exec.Command`. The context gives you
a kill switch, and a subprocess with no timeout is a subprocess that can
hang your service forever:

```go
ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()

err := exec.CommandContext(ctx, "sleep", "5").Run()
fmt.Println(err, ctx.Err())
// output: signal: killed context deadline exceeded
```

Note the two errors say different things. `err` reports *how* the
process died; `ctx.Err()` tells you *why*. Check the context when you
need to distinguish a timeout from a genuine failure.

## Arguments are a list, not a command line

Each argument is passed to the program exactly as written. Nothing
splits on spaces, expands globs, or interprets quotes:

```go
out, _ := exec.CommandContext(ctx, "echo", "a b", "c").Output()
fmt.Printf("%q\n", string(out))   // output: "a b c\n"

out, _ = exec.CommandContext(ctx, "echo", "*").Output()
fmt.Printf("%q\n", string(out))   // output: "*\n"
```

The `*` stayed a literal asterisk. There is no shell to expand it.

This is why passing a filename from user input as an argument is safe:
it can never become a second command. That safety disappears the moment
you write `sh -c`:

```go
exec.CommandContext(ctx, "sh", "-c", "grep "+pattern+" file.txt")   // injection
```

Here `pattern` is shell source. Use `sh -c` only for a fixed string you
wrote, never with interpolated input. If you need a pipeline, either
build it in Go with `cmd.Stdout` wired to the next command's `cmd.Stdin`,
or do the work in Go and skip the subprocess.

## Which call to use

| Call | Returns | Use when |
|---|---|---|
| `Output()` | stdout | you want the result, stderr separately on failure |
| `CombinedOutput()` | stdout + stderr interleaved | logging or debugging |
| `Run()` | error only | you wired the streams yourself |
| `Start()` + `Wait()` | error only | you need to do work while it runs |

```go
co, err := exec.CommandContext(ctx, "sh", "-c", "echo out; echo err >&2; exit 1").CombinedOutput()
fmt.Printf("%q %v\n", string(co), err)
// output: "out\nerr\n" exit status 1
```

## Failure: `*exec.ExitError` carries stderr

A non-zero exit is an error. With `Output()`, the captured stderr is
attached to it, which is what makes the failure diagnosable:

```go
_, err := exec.CommandContext(ctx, "sh", "-c", "echo oops >&2; exit 3").Output()

var ee *exec.ExitError
if errors.As(err, &ee) {
    fmt.Println("code:", ee.ExitCode(), "stderr:", strings.TrimSpace(string(ee.Stderr)))
}
// output: code: 3 stderr: oops
```

`ExitCode()` matters when a tool uses exit status as data — `grep`
returns 1 for "no match", `git diff --quiet` returns 1 for "there are
changes". Those are answers, not failures, so check the code rather than
treating any error as fatal.

`ee.Stderr` is only populated by `Output()`. With `Run()` you captured
stderr yourself; with `CombinedOutput()` it is in the returned bytes.

A missing executable is a different failure, before the process ever
starts:

```go
_, err := exec.CommandContext(ctx, "definitely-not-a-command-xyz").Output()
fmt.Println(err)
// output: exec: "definitely-not-a-command-xyz": executable file not found in $PATH

fmt.Println(errors.Is(err, exec.ErrNotFound))   // output: true
```

Check for a dependency up front with `exec.LookPath`, so a missing tool
is reported at startup rather than mid-request.

## Wiring the streams

Set the fields before running. Any `io.Reader` works as stdin and any
`io.Writer` as stdout or stderr — the interfaces from
[readers and writers](02-readers-and-writers.md):

```go
cmd := exec.CommandContext(ctx, "sh", "-c", "cat; echo done >&2")
cmd.Stdin = strings.NewReader("piped in\n")

var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout
cmd.Stderr = &stderr

err := cmd.Run()
fmt.Printf("%q %q %v\n", stdout.String(), stderr.String(), err)
// output: "piped in\n" "done\n" <nil>
```

Assign `os.Stdout` to pass output straight through to your own, which is
what you want for a build tool.

Do not set `cmd.Stdout` and then call `Output()` — that is an error,
since `Output` needs to own stdout.

## Working directory and environment

```go
cmd := exec.CommandContext(ctx, "sh", "-c", "echo $MYVAR; pwd")
cmd.Dir = dir
cmd.Env = append(os.Environ(), "MYVAR=set")
```

`cmd.Dir` runs the command elsewhere without your process changing
directory — important, since a process-wide `chdir` is not safe when
other goroutines are running.

`cmd.Env` **replaces** the environment entirely. Leaving it nil inherits
yours; setting it to just `[]string{"MYVAR=set"}` gives the child a
nearly empty environment with no `PATH`. `append(os.Environ(), ...)` is
the usual form. Later entries win on duplicates.

## Long-running processes

`Start` returns as soon as the process launches, and `Wait` blocks for
it:

```go
cmd := exec.CommandContext(ctx, "long-task")
if err := cmd.Start(); err != nil {
    return err
}
// ... do other work ...
return cmd.Wait()
```

Every `Start` needs exactly one `Wait`, or you leak a zombie process and
any pipes you created. If you attached pipes with `StdoutPipe`, read
them to completion *before* calling `Wait`.

> **From Python:** this is `subprocess`, and `Output()` is
> `check_output`. The difference to internalise is that there is no
> `shell=True` unless you spell out `sh -c` — so the default is the safe
> one, and the dangerous form is visible in the code.

## Quick reference

| Task | Call |
|---|---|
| run and capture stdout | `exec.CommandContext(ctx, name, args...).Output()` |
| stdout and stderr together | `.CombinedOutput()` |
| streams wired by hand | set `Stdin`/`Stdout`/`Stderr`, then `.Run()` |
| run in the background | `.Start()` then `.Wait()` — always both |
| exit status | `errors.As(err, &ee)`, `ee.ExitCode()`, `ee.Stderr` |
| missing binary | `errors.Is(err, exec.ErrNotFound)`, or `exec.LookPath` up front |
| elsewhere | `cmd.Dir = path` |
| extra env | `cmd.Env = append(os.Environ(), "K=V")` |
| a timeout | the context — it kills the process |

## Sources

- [`os/exec` package reference — pkg.go.dev/os/exec](https://pkg.go.dev/os/exec)
- [`exec.CommandContext` — pkg.go.dev/os/exec#CommandContext](https://pkg.go.dev/os/exec#CommandContext)
- [`exec.ExitError` — pkg.go.dev/os/exec#ExitError](https://pkg.go.dev/os/exec#ExitError)
- [`exec.Cmd` — pkg.go.dev/os/exec#Cmd](https://pkg.go.dev/os/exec#Cmd)
- [`exec.LookPath` — pkg.go.dev/os/exec#LookPath](https://pkg.go.dev/os/exec#LookPath)
