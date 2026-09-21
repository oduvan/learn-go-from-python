# Wiring and package structure

Go has no dependency-injection framework in common use, and needs none.
Dependencies are struct fields, set by constructors, assembled in one
place at startup. The discipline is in where you put things, not in a
container.

```go
type App struct {
    Config Config
    DB     *sql.DB
    Stores *Stores
}
```

## `cmd/` and `internal/`

Two directories carry almost all of the convention:

```
myservice/
  cmd/
    api/main.go        ← one binary
    migrate/main.go    ← another
  internal/
    app/               ← the root object
    store/             ← implementations
    services/          ← contracts
    web/               ← handlers
  go.mod
```

`internal/` is enforced by the compiler: nothing outside the module can
import it, as [special folders](../01-ecosystem-and-installation/05-special-folders.md)
covers. Put everything there unless you intend it to be a public
library. It costs nothing and means you can refactor freely without
breaking someone.

Each directory under `cmd/` is one `package main` producing one binary.
Keep those files tiny.

## Keep `main` thin

`main` should parse flags, build the application, run it, and handle
shutdown. Nothing else:

```go
func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    cfg, err := LoadConfig()
    if err != nil {
        return err
    }

    app, err := NewApp(cfg)
    if err != nil {
        return err
    }
    defer app.Close()

    return app.Run()
}
```

The `run() error` split matters for a reason covered in
[flags and environment](../07-operating-system/04-flags-and-environment.md):
`os.Exit` skips deferred functions, so anything in `main` that needs
cleanup must live in a function that *returns*.

## Constructor injection

A type declares what it needs as fields, and a constructor takes them:

```go
type Handler struct {
    users  UserStore
    logger *slog.Logger
}

func NewHandler(users UserStore, logger *slog.Logger) *Handler {
    return &Handler{users: users, logger: logger}
}
```

Unexported fields, exported constructor. There is no reflection, no
tags, no registration — the compiler checks the wiring, and reading the
signature tells you the whole dependency list.

The rule that keeps this honest: **take the narrowest interface you
use**. A handler needing two methods takes an interface with two
methods, not the whole store bag.

## A root object

One struct owns the long-lived resources:

```go
type App struct {
    Config Config
    DB     *sql.DB
    Stores *Stores
}

func NewApp(cfg Config) (*App, error) {
    db, err := sql.Open("pgx", cfg.DSN)
    if err != nil {
        return nil, fmt.Errorf("opening database: %w", err)
    }
    if err := db.PingContext(context.Background()); err != nil {
        return nil, fmt.Errorf("database unreachable: %w", err)
    }
    return &App{Config: cfg, DB: db, Stores: NewStores(db)}, nil
}

func (a *App) Close() error { return a.DB.Close() }
```

Everything that must be created once and shut down cleanly lives here.
Construction order is just the order of the statements — which is the
advantage over a framework: when it fails, the stack trace points at
the line.

## A container is fine; passing it everywhere is not

Grouping the stores is reasonable:

```go
type Stores struct {
    Users UserStore
    Posts PostStore
}

func NewStores(db *sql.DB) *Stores {
    return &Stores{Users: &userStore{db: db}, Posts: &postStore{db: db}}
}
```

The mistake is handing that struct to every component:

```go
NewHandler(stores)        // what does it actually use?
NewHandler(stores.Users)  // two methods, stated
```

Once a type takes the container, its real dependencies are invisible,
every test must build the whole thing, and nothing stops a handler
reaching for a store it has no business touching.

## Options for the optional

Required dependencies are parameters. Genuinely optional ones — a
cache, a metrics recorder, a hook — are better as functional options,
the pattern from [functions](../02-language-basics/06-functions.md):

```go
type Option func(*Server)

func WithCache(c Cache) Option { return func(s *Server) { s.cache = c } }

func NewServer(store Store, opts ...Option) *Server {
    s := &Server{store: store}
    for _, o := range opts {
        o(s)
    }
    return s
}
```

You will also meet chainable `With*` setters that mutate and return the
receiver. They read fine at startup, but they allow a half-built object
to escape — prefer options when the type must be valid the moment it is
returned.

## Breaking an import cycle

Go forbids import cycles outright. When `auth` needs something from
`scheduler` and `scheduler` needs something from `auth`, the fix is
almost always a third package holding the shared thing:

```
internal/auth       →  internal/principal  ←  internal/scheduler
```

Make it small — a type, a key, an interface — with no dependencies of
its own. Resist the urge to create a `common` or `util` package for
this; name it after what it holds, or it becomes a dumping ground with
its own cycles.

The other fix is often better: if `A` needs a function from `B`, have
`A` declare an interface and let `B`'s type satisfy it. The dependency
then points one way only.

## Flat beats deep

A single level of packages under `internal/` is easier to navigate than
a hierarchy. Deep nesting tends to produce cycles and packages named
after layers rather than things.

Name packages after what they contain — `store`, `billing`, `indexer` —
not what they are — `models`, `helpers`, `utils`. A package called
`utils` has no boundary, so everything ends up in it.

Remember the name is part of every call site: `store.New`, not
`store.NewStore`, because `store.NewStore` stutters.

> **From Python:** there is no `__init__.py`, no import-time
> registration, no `settings` module imported everywhere. Wiring is
> explicit and checked at compile time, and the equivalent of a DI
> container is a struct literal you can read.

## Quick reference

| Concern | Convention |
|---|---|
| binaries | one directory per binary under `cmd/` |
| everything private | `internal/` — compiler-enforced |
| `main` | flags, build, run, shut down; delegate to `run() error` |
| dependencies | constructor parameters, unexported fields |
| how much to take | the narrowest interface you use |
| long-lived resources | one root `App` struct with a `Close` |
| a container struct | fine to build, do not pass it around |
| optional collaborators | functional options |
| import cycle | extract a small shared package, or invert with an interface |
| package names | after the thing, never `utils` |

## Sources

- [Effective Go: package names — go.dev/doc/effective_go#package-names](https://go.dev/doc/effective_go#package-names)
- [Go wiki: code review comments — go.dev/wiki/CodeReviewComments](https://go.dev/wiki/CodeReviewComments)
- [Internal packages — go.dev/doc/go1.4#internalpackages](https://go.dev/doc/go1.4#internalpackages)
- [Organizing a Go module — go.dev/doc/modules/layout](https://go.dev/doc/modules/layout)
- [Go blog: package names — go.dev/blog/package-names](https://go.dev/blog/package-names)
