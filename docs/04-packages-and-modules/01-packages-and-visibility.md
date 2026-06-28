# Packages and visibility

A Go program is organised into **packages**. A package is a directory of
`.go` files that are compiled together and share one namespace. Every
source file begins with a `package` clause naming the package it belongs
to.

```go
package store
```

Two rules anchor everything else:

- **One directory = one package.** All `.go` files in a directory must
  declare the same package name; together they form that package.
- **`package main` is special** — it's the entry point of an executable,
  and it must contain a `func main()`. Every other package is a library,
  imported by others.

## Import paths vs. package names

You *import* a package by its **import path** (its location), but you
*refer* to it by its **package name** (the identifier in the `package`
clause). They're usually the last path element, but not always:

```go
import "example.com/shop/store"   // import path

store.Open()                       // used by package name
```

The package name is what appears in code, so it's chosen for how it reads
at the call site.

## Exported vs. unexported: capitalisation is the access control

An identifier whose name starts with an **uppercase letter is exported** —
visible to code in other packages. Lowercase is **unexported** — visible
only within its own package. This applies to every top-level name: types,
functions, variables, constants, and struct fields.

```go
package store

type Item struct {
    Name  string   // exported field
    price int      // unexported field
}

func New() *Item    { return &Item{} }   // exported
func reset(i *Item) { i.price = 0 }       // unexported
```

The privacy boundary is the **package**, not the type or the file. Files
in the same package see each other's unexported names freely; another
package sees only the exported ones. From a *different* package the
compiler enforces it:

```go
// in package main, importing the store package above
i := store.New()
fmt.Println(i.Name)    // ok — Name is exported
fmt.Println(i.price)   // compile error: i.price undefined
                       // (cannot refer to unexported field price)
```

> **From Python:** there's no `_private` convention that's merely
> advisory, and no `__name` mangling — visibility is enforced by the
> compiler, and the unit of privacy is the package, not the class.

## Files in one package

Splitting a package across several files is purely organisational — the
files share one namespace, and declaration order doesn't matter (a
function may call another declared later, in any file of the package).

```go
// item.go
package store
func New() *Item { return &Item{price: basePrice} }

// price.go
package store
const basePrice = 100      // visible to item.go without any import
```

## `init` functions

A package may declare one or more `func init()` — no parameters, no
results. They run **automatically** when the package is initialised,
after all package-level variables are set up and before `main` starts. A
package's imports are initialised first, so by the time your `init` runs,
everything you depend on is ready.

```go
package config

var settings map[string]string

func init() {
    settings = map[string]string{"env": "dev"}
}
```

The ordering — package-level variables first, then `init`, then `main` — is
observable:

```go
var x = setup()

func setup() int { fmt.Println("var init"); return 1 }
func init()      { fmt.Println("init func") }
func main()      { fmt.Println("main") }
// output:
// var init
// init func
// main
```

Use `init` sparingly — for setup that genuinely can't be expressed as a
plain variable initialiser. Multiple `init`s (even across files) run in
the order the files are presented to the compiler.

## Naming conventions

- Package names are **short, lowercase, single words** — `http`, `bytes`,
  `strconv`. No under_scores or camelCase.
- **Avoid stutter.** The package qualifier is already there at the call
  site, so name `bytes.Buffer`, not `bytes.BytesBuffer`; `store.New`, not
  `store.NewItem` when the package makes it obvious.
- A **doc comment** on the package — `// Package store ...` — goes above
  the `package` clause in one file and documents the whole package.

```go
// Package store manages the product catalogue.
package store
```

## No import cycles

Package imports must form an acyclic graph: if `a` imports `b`, then `b`
may not import `a`, directly or transitively. The compiler rejects cycles
outright. When two packages seem to need each other, it's a sign a shared
type should move to a third package they both import.

```go
// compile error: import cycle not allowed
// package a imports b; package b imports a
```

## Quick reference

| Concept | Rule |
|---|---|
| package clause | every `.go` file starts with `package X` |
| one directory | exactly one package |
| `package main` + `func main()` | builds an executable |
| import path | *where* a package is; used in `import` |
| package name | *what* you call it in code |
| `Uppercase` | exported (visible to other packages) |
| `lowercase` | unexported (package-private) |
| `func init()` | runs at package init, before `main` |
| import cycle | forbidden |

## Sources

- [Packages — go.dev/ref/spec#Packages](https://go.dev/ref/spec#Packages)
- [Exported identifiers — go.dev/ref/spec#Exported_identifiers](https://go.dev/ref/spec#Exported_identifiers)
- [Package initialization — go.dev/ref/spec#Package_initialization](https://go.dev/ref/spec#Package_initialization)
- [Effective Go: names — go.dev/doc/effective_go#names](https://go.dev/doc/effective_go#names)
- [Effective Go: package names — go.dev/blog/package-names](https://go.dev/blog/package-names)
