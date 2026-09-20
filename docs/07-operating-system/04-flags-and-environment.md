# Flags and environment

The standard library's `flag` package is deliberately small. It handles
the common case well, has one surprising rule about argument order, and
is enough for most programs.

```go
port := flag.Int("port", 8080, "port to listen on")
flag.Parse()

fmt.Println(*port)
```

## Defining and parsing

Each `flag.X` call returns a **pointer**, because the value does not
exist until `Parse` runs:

```go
verbose := flag.Bool("verbose", false, "enable verbose output")
port    := flag.Int("port", 8080, "port to listen on")
name    := flag.String("name", "app", "service name")
timeout := flag.Duration("timeout", 5*time.Second, "request timeout")

flag.Parse()

fmt.Println(*verbose, *port, *name, *timeout)
// with no arguments: false 8080 app 5s
```

Reading `*port` before `Parse` gives the default, silently. Call `Parse`
once, at the top of `main`, before anything reads a flag.

`flag.Duration` parses the same strings as `time.ParseDuration` from the
[time article](../06-text-time-and-data/04-time.md), so `--timeout 2m`
works. There is also the `flag.XVar` form, which writes into a variable
you already have — handy for filling a config struct.

Single and double dashes are equivalent, and `=` is optional:

```
-verbose  -port=9090  --timeout 2m
```

## Flags stop at the first non-flag argument

This is the rule that catches everyone:

```go
fs.Parse([]string{"-f", "a", "b"})
fmt.Println(*f, fs.Args())   // output: true [a b]

fs.Parse([]string{"a", "-f", "b"})
fmt.Println(*f, fs.Args())   // output: false [a -f b]
```

In the second case `-f` was never parsed — it is just another positional
argument. Go has no GNU-style permutation, so **flags must come before
positional arguments**. Whatever remains is `flag.Args()`, with
`flag.NArg()` and `flag.Arg(i)` as accessors.

## Boolean flags need `=`

A bool flag is set by its presence, so it never consumes the next
argument. To pass `false` explicitly you must use `=`:

```go
fs.Parse([]string{"-d=false"})
fmt.Println(*d)   // output: false
```

`-d false` sets `d` to true and leaves `"false"` as a positional
argument — which is why a default-true flag needs this form.

## Usage and errors

`flag.PrintDefaults` writes the generated help:

```
  -name string
    	service name (default "app")
  -port int
    	port to listen on (default 8080)
  -timeout duration
    	request timeout (default 5s)
  -verbose
    	enable verbose output
```

The third argument of each definition is that help text, so write it for
a reader.

On a bad value the default `flag.CommandLine` prints the error plus
usage and calls `os.Exit(2)`:

```
invalid value "abc" for flag -n: parse error
```

## Subcommands with `flag.NewFlagSet`

There is no built-in subcommand support. `flag.NewFlagSet` gives each
subcommand its own set, and you dispatch on `os.Args[1]`:

```go
serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
servePort := serveCmd.Int("port", 8080, "port")

migrateCmd := flag.NewFlagSet("migrate", flag.ExitOnError)
migrateDown := migrateCmd.Bool("down", false, "roll back")

if len(os.Args) < 2 {
    fmt.Fprintln(os.Stderr, "expected 'serve' or 'migrate'")
    os.Exit(2)
}

switch os.Args[1] {
case "serve":
    serveCmd.Parse(os.Args[2:])
    fmt.Println("serving on", *servePort)
case "migrate":
    migrateCmd.Parse(os.Args[2:])
    fmt.Println("rolling back:", *migrateDown)
default:
    fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
    os.Exit(2)
}
```

The error-handling mode matters. `flag.ExitOnError` exits on a bad flag;
`flag.ContinueOnError` returns the error so you can decide, which is
what makes a flag set testable.

## Environment variables

```go
fmt.Printf("%q\n", os.Getenv("APP_PORT"))          // output: "3000"
fmt.Printf("%q\n", os.Getenv("DEFINITELY_UNSET"))  // output: ""
```

`Getenv` returns `""` for both "unset" and "set to empty". When that
difference matters — an empty value meaning "explicitly disabled" —
`LookupEnv` reports it:

```go
v, ok := os.LookupEnv("DEFINITELY_UNSET")
fmt.Printf("%q %v\n", v, ok)   // output: "" false

os.Setenv("EMPTY", "")
v, ok = os.LookupEnv("EMPTY")
fmt.Printf("%q %v\n", v, ok)   // output: "" true
```

Everything is a string, so anything else needs `strconv`:

```go
n, err := strconv.Atoi(os.Getenv("APP_PORT"))
fmt.Println(n, err)   // output: 3000 <nil>
```

Parse configuration **once at startup and fail loudly** rather than
calling `Getenv` deep in the code. Scattered lookups make a program's
inputs impossible to find, and a typo becomes a zero value instead of an
error. Gathering them into a single config struct, validated once, is a
pattern the architecture topic returns to.

The standard library does not read `.env` files — that is a third-party
convenience.

## Exit codes

`os.Exit` stops immediately. It does **not** run deferred functions:

```go
func main() {
    defer fmt.Println("cleanup")   // never printed
    os.Exit(1)
}
```

So the idiom is to keep `main` thin and let a `run` function return an
error:

```go
func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    defer cleanup()   // does run
    // ...
    return nil
}
```

Errors go to `os.Stderr`, output to `os.Stdout`, so a caller can pipe one
without the other. `log.Fatal` writes to stderr and exits with 1, with
the same defer problem.

By convention `0` is success, `1` is a general failure, and `2` is a
usage error — which is what `flag` uses.

> **From Python:** `flag` is `argparse` with far fewer features: no
> `nargs`, no subparsers, no mutually exclusive groups, no automatic
> `--help` epilogue beyond the generated list. The two real behavioural
> differences are that flags must precede positional arguments, and that
> `os.Exit` skips `defer` the way `os._exit` skips `finally`.

## Quick reference

| Task | Call |
|---|---|
| define | `flag.Int("port", 8080, "help")` → `*int` |
| into an existing variable | `flag.IntVar(&cfg.Port, "port", 8080, "help")` |
| parse | `flag.Parse()` once, at the top of `main` |
| leftovers | `flag.Args()`, `flag.NArg()`, `flag.Arg(i)` |
| durations | `flag.Duration` — accepts `2m`, `1h30m` |
| explicit false | `-d=false`, never `-d false` |
| subcommands | `flag.NewFlagSet(name, flag.ExitOnError)` + `os.Args[1]` |
| testable parsing | `flag.ContinueOnError` |
| env, may be empty | `os.Getenv(k)` |
| distinguish unset | `os.LookupEnv(k)` → `(value, ok)` |
| env to a number | `strconv.Atoi` |
| fail | write to `os.Stderr`, then `os.Exit(1)` — skips `defer` |

## Sources

- [`flag` package reference — pkg.go.dev/flag](https://pkg.go.dev/flag)
- [`flag.NewFlagSet` — pkg.go.dev/flag#NewFlagSet](https://pkg.go.dev/flag#NewFlagSet)
- [`os.LookupEnv` — pkg.go.dev/os#LookupEnv](https://pkg.go.dev/os#LookupEnv)
- [`os.Exit` — pkg.go.dev/os#Exit](https://pkg.go.dev/os#Exit)
- [`strconv` package reference — pkg.go.dev/strconv](https://pkg.go.dev/strconv)
