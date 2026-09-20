# Project conventions

`gofmt` settles formatting and `go vet` catches a class of mistakes.
What remains is the set of habits a team agrees on — the things no tool
checks, which are exactly the things worth writing down.

```go
if err != nil {
    return fmt.Errorf("loading user %d: %w", id, err)
}
```

## Wrap every error with context

An error that reaches a log as `sql: no rows in result set` tells you
nothing about which query, which id, or which request. Wrap at every
level that adds information:

```go
return fmt.Errorf("loading user %d: %w", id, err)
```

The conventions that make wrapped errors readable:

- **Lower case, no trailing punctuation.** Errors get concatenated;
  `Failed to load user.` inside another message reads badly.
- **No "failed to".** The fact that it is an error already says that.
  `loading user 42: connection refused` beats `failed to load user 42:
  failed to connect: connection refused`.
- **`%w`, not `%v`**, unless you deliberately want to break the chain.
  `%v` flattens the error to text and `errors.Is` stops working.
- **Add something.** Wrapping with no new information is noise — pass
  the error up unchanged instead.

Never discard an error silently. If it genuinely does not matter, say
so:

```go
_ = resp.Body.Close()   // best effort
```

The explicit `_` tells a reviewer it was a decision.

## `ctx` first, always

Every function that performs I/O or can block takes a context as its
first parameter, named `ctx`:

```go
func (s *store) ByID(ctx context.Context, id int64) (User, error)
```

Consistency matters more than the individual case. If half the
functions take one, the other half become the ones you have to look up.

## Log levels that mean something

Levels are only useful if they are used the same way everywhere:

| Level | Meaning |
|---|---|
| `Error` | a human needs to act; something is broken |
| `Warn` | degraded but handled — a fallback fired, a retry happened |
| `Info` | a significant event: started, stopped, job completed |
| `Debug` | detail for diagnosis, off in production |

Two rules that save more pain than the table. **Log an error once**, at
the level that handles it — logging and returning at every level
produces five entries for one failure. And **do not log at `Error`
something you also return**: the caller will log it, and now you have
the same failure twice with different context.

Never log secrets, tokens or personal data. That includes `%+v` on a
struct that happens to contain a password field.

## Comment the why, not the what

```go
// PreferSimpleProtocol avoids the prepared-statement cache, which
// the connection pooler in front of this database does not support.
PreferSimpleProtocol: true,
```

The code says what. A comment earns its place by explaining what the
code cannot: a workaround, a non-obvious constraint, a decision that
looks wrong and is not. Comments restating the line below rot the
moment the line changes.

Exported identifiers get a doc comment starting with the name:

```go
// ByID returns the user with the given id, or ErrUserNotFound.
func (s *store) ByID(ctx context.Context, id int64) (User, error)
```

That form is what `go doc` and pkg.go.dev render.

## Justify raw SQL

Where a project has a standard way of reaching the database and you
step outside it, leave a reason:

```go
// Raw: generated column — fts_content is generated, so the INSERT must
// name its columns explicitly rather than let the ORM derive them.
```

The value is not the individual comment; it is that a reviewer can see
whether the exception is one of the handful the team accepts, or a
shortcut. The same habit applies to any escape hatch: a `//nolint`
directive, an `unsafe` call, a sleep in a test. Escape hatches without
reasons multiply.

## Naming

- **Short names for short scopes.** `i`, `r`, `w`, `db` are good inside
  a five-line function and bad as package-level identifiers.
- **No stuttering.** `store.New`, not `store.NewStore`; `http.Client`,
  not `http.HTTPClient`.
- **Interfaces describe behaviour.** `Reader`, `UserStore` — not
  `IUserStore` or `UserStoreInterface`.
- **Receivers are one or two letters**, consistent across every method
  on the type.
- **Acronyms keep their case**: `userID`, `ServeHTTP`, `parseURL` —
  never `userId` or `parseUrl`.

## Accept interfaces, return structs

Take the narrowest interface you need as a parameter; return the
concrete type. Callers keep full information, and you are free to add
methods later without breaking anyone.

The corollary: do not create an interface until there is a second
implementation or a test that needs one. A one-implementation interface
is indirection with no payoff.

## Zero values should work

A type whose zero value is usable removes a whole class of constructor:

```go
var buf bytes.Buffer   // ready
var mu sync.Mutex      // ready
```

Design for it where you can. Where you cannot, make that obvious — an
unexported field that a constructor must set, so a zero value fails
loudly rather than behaving subtly wrong.

## Where to write this down

A `CONTRIBUTING.md` nobody reads is worse than nothing. What works:

- **Make it mechanical where possible.** Anything a linter can check
  should be checked by a linter rather than by reviewers.
- **Explain the why.** A rule with a reason survives; a rule without
  one gets argued about every six months.
- **Keep it short.** Ten rules people follow beat fifty they skim.

> **From Python:** `gofmt` ends the formatting discussion that `black`
> ended, but earlier and with no configuration at all. Error wrapping
> is the equivalent of `raise ... from err`, except you must do it by
> hand every time — which is the cost of errors being values.

## Quick reference

| Convention | Form |
|---|---|
| wrap errors | `fmt.Errorf("doing thing: %w", err)` — lower case, no "failed to" |
| ignore an error | `_ =`, deliberately |
| context | `ctx context.Context` first, always |
| log a failure | once, where it is handled |
| comments | why, not what; doc comments start with the name |
| escape hatches | leave a written reason |
| names | short scope short name; no stutter; `ID`, `URL`, `HTTP` |
| interfaces | accept narrow ones, return concrete types |
| zero values | make them usable where you can |

## Sources

- [Effective Go — go.dev/doc/effective_go](https://go.dev/doc/effective_go)
- [Go wiki: code review comments — go.dev/wiki/CodeReviewComments](https://go.dev/wiki/CodeReviewComments)
- [Go blog: working with errors in Go 1.13 — go.dev/blog/go1.13-errors](https://go.dev/blog/go1.13-errors)
- [Go Doc Comments — go.dev/doc/comment](https://go.dev/doc/comment)
- [Go blog: package names — go.dev/blog/package-names](https://go.dev/blog/package-names)
