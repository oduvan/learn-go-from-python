# `go:embed`

`//go:embed` copies files into the binary at build time. A Go program is
already a single static executable; embedding is what keeps it that way
once it needs templates, migrations, or static assets.

```go
import _ "embed"

//go:embed version.txt
var version string

fmt.Printf("%q\n", version)   // output: "v1.4.2\n"
```

No file to ship, no path to configure, nothing to go missing in a
container image.

## The three target types

The directive sits immediately above a package-level `var`, with no
blank line between them, and the variable's type decides what you get:

| Type | Result |
|---|---|
| `string` | the file's contents as text |
| `[]byte` | the file's contents as bytes |
| `embed.FS` | a read-only filesystem, one or many files |

```go
//go:embed version.txt
var version string

//go:embed version.txt
var versionBytes []byte

//go:embed static
var staticFS embed.FS
```

For `string` and `[]byte` the directive must name exactly one file. Only
`embed.FS` takes directories or patterns.

## The blank import

If a file uses `//go:embed` only for `string` or `[]byte` variables, it
never mentions the `embed` package by name — but the compiler still
requires it to be imported. Hence the blank import:

```go
import _ "embed"
```

Using `embed.FS` means you name the package, so a normal `import "embed"`
covers it. Forgetting this gives a clear compile error, so it is a
one-time surprise. It is the same mechanism as the driver registration
in [imports](../02-language-basics/16-imports.md).

## Reading from an `embed.FS`

`embed.FS` implements `fs.FS`, so everything in `io/fs` works on it —
including `fs.WalkDir` from the [files and paths](01-files-and-paths.md)
article:

```go
b, err := staticFS.ReadFile("static/css/app.css")
fmt.Println(err)                // output: <nil>
fmt.Printf("%q\n", string(b))   // output: "body{color:red}"
```

**The path keeps the directory you embedded.** Embedding `static` means
the entry is `static/css/app.css`, not `css/app.css`. Paths always use
forward slashes, on every platform, and must be clean — no `./`, no
`..`:

```go
fmt.Println(fs.ValidPath("static/css/app.css"))   // output: true
fmt.Println(fs.ValidPath("./static"))             // output: false
```

A missing file gives an `fs.ErrNotExist`-wrapped error, testable the
usual way:

```go
_, err := staticFS.ReadFile("nope")
fmt.Println(err)   // output: open nope: file does not exist
```

## `fs.Sub` strips the prefix

Carrying `static/` through every lookup is noise. `fs.Sub` returns a
filesystem rooted at a subdirectory:

```go
sub, _ := fs.Sub(staticFS, "static")
b, _ := fs.ReadFile(sub, "css/app.css")
fmt.Printf("%q\n", string(b))   // output: "body{color:red}"
```

This is what you hand to anything that expects to serve a directory
root.

## `all:` and what gets skipped

By default, embedding a directory **skips files beginning with `.` or
`_`**. That rule is inherited from how the `go` tool ignores such files
generally, and it silently drops things you meant to include:

```go
//go:embed migrations
var plain embed.FS
// migrations/001_init.sql, migrations/002_b.sql

//go:embed all:migrations
var withAll embed.FS
// migrations/001_init.sql, migrations/002_b.sql, migrations/_skipme.sql
```

The `all:` prefix disables the rule. Use it for anything where a missing
file is a correctness problem — migrations especially, where a skipped
file means a schema that silently diverges.

Patterns are the other form, and they do not recurse:

```go
//go:embed tpl/*.tmpl
var tplFS embed.FS
```

## Parsing templates straight from it

`ParseFS` reads templates out of any `fs.FS`, so the binary carries
them. Templates themselves are the next topic's subject — the point here
is only that an embedded filesystem plugs into anything taking an
`fs.FS`:

```go
t := template.Must(template.ParseFS(tplFS, "tpl/*.tmpl"))
t.ExecuteTemplate(os.Stdout, "greet.tmpl", map[string]string{"Name": "Ada"})
// output: Hello Ada
```

`template.Must` panics on a bad template, which is what you want at
startup: a template that does not parse is a build-time mistake, so fail
immediately rather than on the first request.

## The rules worth remembering

- **Every way of misplacing the directive is a build error**, which is
  the good news — there is no silent failure mode here. The directive
  must precede a package-level `var` of the right type; blank lines and
  `//` comments between the two are explicitly allowed:

```go
//go:embed version.txt
func nope() {}
// build error: misplaced go:embed directive

//go:embed version.txt
var wrongType int
// build error: go:embed cannot apply to var of type int

func f() {
    //go:embed version.txt
    var local string
    // build error: go:embed cannot apply to var inside func
}
```

- Only files **inside the package directory** can be embedded. A parent
  path is rejected outright:

```go
//go:embed ../outside.txt
// build error: pattern ../outside.txt: invalid pattern syntax
```

- An unmatched pattern is a **build error**, not an empty FS, so a typo
  in a filename does fail loudly:

```go
//go:embed assets/*.zzz
// build error: pattern assets/*.zzz: no matching files found
```

- Embedded content is read-only and fixed at build time. Change a file
  and you must rebuild.
- The files count toward binary size. Embedding a few hundred kilobytes
  of assets is normal; embedding a large dataset is a decision.

## When not to

Anything that must change without a rebuild — configuration, secrets,
user uploads — should stay on disk or in a store. Embedding is for
things that belong to the code: SQL migrations, HTML templates, CSS and
JS, a default config, a reference data file.

> **From Python:** this replaces `importlib.resources` and the whole
> `package_data`/`MANIFEST.in` problem. There is no "was the data file
> installed?" failure mode, because the data is inside the executable.

## Quick reference

| Need | Write |
|---|---|
| one file as text | `//go:embed f.txt` above `var s string` |
| one file as bytes | same, above `var b []byte` |
| a tree | `//go:embed dir` above `var f embed.FS` |
| include dotfiles and `_` files | `//go:embed all:dir` |
| a pattern | `//go:embed tpl/*.tmpl` (no recursion) |
| only `string`/`[]byte` in the file | add `import _ "embed"` |
| drop the prefix | `fs.Sub(f, "dir")` |
| read | `f.ReadFile("dir/x")` — forward slashes, keeps the prefix |
| walk | `fs.WalkDir(f, "dir", fn)` |
| templates | `template.ParseFS(f, "tpl/*.tmpl")` |

## Sources

- [`embed` package reference — pkg.go.dev/embed](https://pkg.go.dev/embed)
- [`io/fs` package reference — pkg.go.dev/io/fs](https://pkg.go.dev/io/fs)
- [`fs.Sub` — pkg.go.dev/io/fs#Sub](https://pkg.go.dev/io/fs#Sub)
- [`template.ParseFS` — pkg.go.dev/text/template#ParseFS](https://pkg.go.dev/text/template#ParseFS)
- [Go 1.16 embed announcement — go.dev/doc/go1.16#embed](https://go.dev/doc/go1.16#embed)
