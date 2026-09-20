# Dependency direction

Which package may import which is the one architectural decision that
is hard to reverse. Go makes part of it mechanical — import cycles are a
compile error — but the rest is a rule you have to hold deliberately.

```
web  →  services  ←  storage
         (contracts)
```

Both the web layer and the storage layer depend on the contracts. The
contracts depend on neither.

## Point dependencies at abstractions

The naive layering has the handler import the store package directly.
That couples your HTTP layer to `database/sql`, and every test of a
handler needs a database.

Inverting it: a middle package declares the interfaces and the domain
types, and both sides depend on *it*.

```go
// package services — the contract. No database imports at all.
type UserStore interface {
    ByID(ctx context.Context, id int64) (User, error)
}

type User struct {
    ID   int64
    Name string
}

var ErrUserNotFound = errors.New("user not found")
```

```go
// package storage — the implementation
type userStore struct{ db *sql.DB }

func (s userStore) ByID(ctx context.Context, id int64) (services.User, error) { ... }
```

```go
// package web — the consumer
type Handler struct{ users services.UserStore }
```

`web` never imports `storage`. Only the startup code, which already
knows everything, wires one to the other.

## Where the interface lives

Go's usual advice is "define the interface where it is consumed", and
for a one-off collaborator that is right — a small unexported interface
next to the handler that uses it.

A central contracts package is the variant that scales: one place
listing every storage operation the application has, so the boundary is
visible rather than scattered. The trade is that the package grows, and
an interface there can accumulate methods no single consumer needs.

Both are legitimate. What is not legitimate is defining the interface in
the *implementation* package — that points the dependency the wrong way
and defeats the exercise.

## Keep the database handle out of the upper layers

The concrete rule that follows: `*sql.DB` should appear in exactly two
places — the code that opens it, and the code that runs queries.

If a handler can name `*sql.DB`, it can run a query, and eventually one
will. Then the store is no longer the only path to the data, and
"where does this table get written?" has no answer.

The same goes for the contracts package. If `services` imports
`database/sql` to describe a return type, the abstraction has leaked
and every consumer inherits the dependency.

## Errors cross the boundary too

A contract is not just method signatures. Leaking `sql.ErrNoRows` upward
couples callers to the storage engine just as surely as leaking the
handle. Translate at the boundary, as
[the repository pattern](../09-database-sql/04-the-repository-pattern.md)
shows:

```go
if errors.Is(err, sql.ErrNoRows) {
    return services.User{}, fmt.Errorf("id %d: %w", id, services.ErrUserNotFound)
}
```

The sentinel belongs to the contract package, so callers depend on the
contract for both the happy and the unhappy path.

## The direction matters more than the names

Folders called `handlers`, `services` and `repositories` are not an
architecture. A `services` package importing `storage` has the same
coupling as a handler doing it, with extra indirection.

Ask instead:

1. If I deleted the storage package, would the rest still compile
   against the contracts? It should.
2. Can I test the business logic with no database? You should be able
   to.
3. Does anything above the store import `database/sql`? It should not.

Three yes-no questions beat any folder convention.

## Cycles are the symptom, not the disease

An import cycle is Go telling you two packages are really one, or that
something shared needs extracting. The fixes, in order of preference:

1. **Invert with an interface.** If `A` needs a function from `B` and
   `B` needs a type from `A`, have `A` declare the interface it needs
   and let `B` satisfy it. The dependency now points one way.
2. **Extract the shared thing** into a small third package with no
   dependencies of its own — a type, a key, a sentinel error.
3. **Merge them**, if they are genuinely one concern that was split for
   cosmetic reasons.

What not to do is add an `interface{}` or a callback purely to dodge the
compiler. The cycle is information; take it.

## Enforcing it

Prose in a README does not survive contact with a deadline. The rules
here — no `*sql.DB` above the store, no database imports in the
contracts package — are mechanically checkable, and a linter can fail
the build on them. That is configuration of a third-party tool, so it
lives with the other external libraries. The design decision is this
article; the enforcement is a config file.

> **From Python:** the same dependency-inversion idea, minus abstract
> base classes and registration. The compiler enforces acyclicity, which
> Python does not, so a cycle is a hard error rather than a subtle
> import-order bug.

## Quick reference

| Question | Answer |
|---|---|
| where do interfaces go | the consumer's side, or a central contracts package |
| where they must not go | the implementation package |
| what may name `*sql.DB` | the code that opens it, and the stores |
| what the contracts package imports | not `database/sql`, not the stores |
| storage errors | translate to domain sentinels at the boundary |
| an import cycle means | invert with an interface, or extract a small package |
| the real test | can the business logic compile and be tested without a database |

## Sources

- [Effective Go: interfaces — go.dev/doc/effective_go#interfaces](https://go.dev/doc/effective_go#interfaces)
- [Go wiki: code review comments — go.dev/wiki/CodeReviewComments#interfaces](https://go.dev/wiki/CodeReviewComments#interfaces)
- [Import declarations — go.dev/ref/spec#Import_declarations](https://go.dev/ref/spec#Import_declarations)
- [Organizing a Go module — go.dev/doc/modules/layout](https://go.dev/doc/modules/layout)
