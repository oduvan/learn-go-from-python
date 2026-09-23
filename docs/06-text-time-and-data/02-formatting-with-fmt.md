# Formatting with `fmt`

`fmt.Println` has carried every example so far. The rest of the package
is a small formatting language, and two of its verbs — `%+v` for
debugging and `%w` for wrapping errors — come up constantly.

```go
type Point struct {
    X, Y int
    name string
}

p := Point{X: 1, Y: 2, name: "origin"}

fmt.Printf("%v\n", p)    // output: {1 2 origin}
fmt.Printf("%+v\n", p)   // output: {X:1 Y:2 name:origin}
```

## The four functions

Every formatting function is one of these, with a prefix saying where the
result goes:

| Prefix | Destination | Example |
|---|---|---|
| *(none)* | standard output | `fmt.Printf("%d\n", n)` |
| `S` | returned as a `string` | `s := fmt.Sprintf("%d", n)` |
| `F` | an `io.Writer` you pass | `fmt.Fprintf(w, "%d", n)` |
| `Errorf` | returned as an `error` | `fmt.Errorf("bad %d", n)` |

`Print` writes its arguments, `Println` adds spaces and a newline,
`Printf` takes a format string. `Fprintf` is the one that makes the
package composable — it writes to anything implementing `io.Writer`,
which includes `os.Stdout`, a file, an HTTP response, and
`strings.Builder`.

## `%v`, `%+v`, `%#v` and `%T`

`%v` is the default rendering of any value. The other three are for when
you are trying to work out what you actually have:

```go
fmt.Printf("%v\n", p)    // output: {1 2 origin}
fmt.Printf("%+v\n", p)   // output: {X:1 Y:2 name:origin}
fmt.Printf("%#v\n", p)   // output: main.Point{X:1, Y:2, name:"origin"}
fmt.Printf("%T\n", p)    // output: main.Point
```

`%+v` adds field names and is the one to reach for in a log line or a
quick debug print. `%#v` prints Go syntax you could paste back into
source. `%T` prints the type, which is how you find out what is really
inside an `any`.

Composite values nest sensibly, and **map keys are printed in sorted
order** so output is reproducible:

```go
fmt.Printf("%v\n", []int{1, 2})                    // output: [1 2]
fmt.Printf("%v\n", map[string]int{"b": 2, "a": 1}) // output: map[a:1 b:2]
fmt.Printf("%v\n", &p)                             // output: &{1 2 origin}
```

A pointer to a struct prints as `&{...}`. A pointer to anything else
prints as a hexadecimal address, which differs every run.

## Strings, quoting and runes

```go
s := "hi\tthere"
fmt.Printf("%s|%q\n", s, s)   // output: hi	there|"hi\tthere"
```

`%q` adds quotes and escapes — invaluable when the question is "does this
string have trailing whitespace or a tab in it?", which `%s` cannot
answer. It is why the strings article prints with `%q`.

A `rune` is an integer, so `%v` shows the number. `%q` shows the
character literal, `%c` the character, `%U` the code point:

```go
fmt.Printf("%v %q\n", 'A', 'A')        // output: 65 'A'
fmt.Printf("%c %U\n", 0x4e16, 0x4e16)  // output: 世 U+4E16
```

## Numbers

```go
fmt.Printf("%d %b %o %x %X\n", 255, 255, 255, 255, 255)
// output: 255 11111111 377 ff FF

fmt.Printf("%f %.2f %e %g\n", 3.14159, 3.14159, 3.14159, 3.14159)
// output: 3.141590 3.14 3.141590e+00 3.14159

fmt.Printf("%t\n", true)   // output: true
```

`%f` always prints six decimal places. `%g` picks the shortest
representation that round-trips, which is usually what you want for a
number whose magnitude you do not know in advance.

## Width and precision

A number between `%` and the verb sets a minimum width; a `.n` sets
precision. A minus sign left-aligns, a leading zero pads with zeros:

```go
fmt.Printf("[%5d][%-5d][%05d]\n", 42, 42, 42)
// output: [   42][42   ][00042]

fmt.Printf("[%8.3f][%-8.3f]\n", 3.14159, 3.14159)
// output: [   3.142][3.142   ]
```

On a string, precision *truncates*:

```go
fmt.Printf("[%6s][%-6s][%.3s]\n", "go", "go", "golang")
// output: [    go][go    ][gol]
```

A `*` takes the width from the argument list, for when it is computed:

```go
fmt.Printf("%*d\n", 6, 42)   // output:     42
```

## `fmt.Stringer`

A type with a `String() string` method controls its own `%v` and
`%s` rendering, and `Println` picks it up too:

```go
type Temp float64

func (t Temp) String() string { return fmt.Sprintf("%.1f°C", float64(t)) }

fmt.Printf("%v\n", Temp(21.456))   // output: 21.5°C
fmt.Println(Temp(21.456))          // output: 21.5°C
```

The trap in a `String` method is formatting the receiver with a
string-like verb. `fmt` only consults `String` for `%v`, `%s` and their
relatives, so this recurses until the stack is gone:

```go
func (t Temp) String() string { return fmt.Sprintf("%v°C", t) }
// go vet: fmt.Sprintf format %v with arg t causes recursive
//         (main.Temp).String method call
```

Numeric verbs are safe — `%.1f` reads the number and never calls
`String` — so the `float64(t)` conversion above is for clarity rather
than necessity. When in doubt, convert to the underlying type: it is
correct under every verb, and `go vet` flags the case where it matters.

## `%w` belongs to `Errorf` alone

`%w` does not format anything — it records the error for
`errors.Is` and `errors.As`, as the [errors article](../02-language-basics/07-errors.md)
covers. Visually `%w` and `%v` are identical:

```go
err := errors.New("disk full")
wrapped := fmt.Errorf("saving: %w", err)

fmt.Printf("%v | %s\n", wrapped, wrapped)
// output: saving: disk full | saving: disk full

fmt.Println(errors.Is(wrapped, err))   // output: true
```

Only `Errorf` understands `%w`. Passing it to `Printf` produces the
badness below.

## When a verb and an argument disagree

`fmt` never panics on a mismatch. It writes the problem into the output,
which means a broken format string shows up in your logs rather than
taking down the process:

```go
fmt.Printf("%d %s\n", 1)      // output: 1 %!s(MISSING)
fmt.Printf("%d\n", "nope")    // output: %!d(string=nope)
```

`go vet` catches both at build time, and it runs as part of `go test`.

> **From Python:** `%v` is `str()` and `%#v` is close to `repr()`.
> `Sprintf` is `%`-formatting or `f"..."`, but the verbs are typed — a
> Python f-string does not care what you hand it, whereas `%d` with a
> string is an error `go vet` will point at.

## Quick reference

| Verb | Use |
|---|---|
| `%v` | default form of any value |
| `%+v` | struct with field names — the debugging default |
| `%#v` | Go syntax |
| `%T` | the type |
| `%s` / `%q` | string / quoted and escaped string |
| `%d` `%b` `%o` `%x` | integer in base 10, 2, 8, 16 |
| `%f` `%.2f` `%e` `%g` | float: fixed, 2 places, scientific, shortest |
| `%c` / `%U` | rune as a character / as `U+XXXX` |
| `%t` | bool |
| `%w` | wrap an error — **`fmt.Errorf` only** |
| `%5d` `%-5d` `%05d` | width, left-aligned, zero-padded |
| `%.3s` | truncate a string |

## Sources

- [`fmt` package reference — pkg.go.dev/fmt](https://pkg.go.dev/fmt)
- [`fmt.Stringer` — pkg.go.dev/fmt#Stringer](https://pkg.go.dev/fmt#Stringer)
- [`fmt.Errorf` — pkg.go.dev/fmt#Errorf](https://pkg.go.dev/fmt#Errorf)
- [Effective Go: printing — go.dev/doc/effective_go#printing](https://go.dev/doc/effective_go#printing)
- [Go blog: working with errors in Go 1.13 — go.dev/blog/go1.13-errors](https://go.dev/blog/go1.13-errors)
