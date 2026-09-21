# The repository pattern

Spreading SQL through your handlers couples the whole program to one
database. The fix is a boundary: an interface describing what you need,
and one package that implements it.

```go
type UserStore interface {
    ByID(ctx context.Context, id int64) (User, error)
    Create(ctx context.Context, name string) (User, error)
}
```

Everything above that boundary works in terms of `User` and
`ErrUserNotFound`. Nothing above it imports `database/sql`.

## Two packages, one contract

The interface, the domain types and the sentinel errors go in a
**contract** package. It has no database imports at all:

```go
type User struct {
    ID   int64
    Name string
}

var ErrUserNotFound = errors.New("user not found")

type UserStore interface {
    ByID(ctx context.Context, id int64) (User, error)
    Create(ctx context.Context, name string) (User, error)
}
```

The SQL lives in a separate **implementation** package, in an
unexported struct holding the pool:

```go
type userStore struct{ db *sql.DB }

func (s userStore) ByID(ctx context.Context, id int64) (User, error) {
    var u User
    err := s.db.QueryRowContext(ctx,
        `SELECT id, name FROM users WHERE id=$1`, id).Scan(&u.ID, &u.Name)

    if errors.Is(err, sql.ErrNoRows) {
        return User{}, fmt.Errorf("id %d: %w", id, ErrUserNotFound)
    }
    if err != nil {
        return User{}, fmt.Errorf("selecting user %d: %w", id, err)
    }
    return u, nil
}
```

Unexported, because callers should hold the interface. A constructor
returns it:

```go
func NewUserStore(db *sql.DB) UserStore { return userStore{db: db} }
```

Prove the implementation satisfies the contract at compile time:

```go
var _ UserStore = userStore{}
```

## Translate storage errors at the boundary

This is the part that carries the pattern. `sql.ErrNoRows` is a
`database/sql` concept; leaking it upward means every caller imports
the SQL package and a change of storage engine breaks all of them.

The store converts it:

```go
_, err := store.ByID(ctx, 99999)

fmt.Println(errors.Is(err, ErrUserNotFound))   // output: true
fmt.Println(errors.Is(err, sql.ErrNoRows))     // output: false
fmt.Println(err)                               // output: id 99999: user not found
```

Wrapping with `%w` keeps `errors.Is` working while adding which id was
missing. Unexpected errors get wrapped too, but pass through as
themselves — the caller cannot mistake a dropped connection for a
missing row.

Now business logic reads in domain terms:

```go
func (g Greeter) Greet(ctx context.Context, id int64) (string, error) {
    u, err := g.users.ByID(ctx, id)
    if errors.Is(err, ErrUserNotFound) {
        return "hello, stranger", nil
    }
    if err != nil {
        return "", fmt.Errorf("greeting %d: %w", id, err)
    }
    return "hello, " + u.Name, nil
}
```

## Depend on the narrowest interface

A single `Store` interface with forty methods forces every consumer to
know about all of them, and every test double to implement all of them.
Declare the small set each consumer actually uses:

```go
type Greeter struct{ users UserStore }
```

`Greeter` needs two methods. Give it two. This is "accept interfaces,
return concrete types" from
[interfaces](../03-object-oriented-go/02-interfaces.md), applied to
storage.

A container struct is a reasonable way to assemble them at startup —
just do not pass the whole container into every component:

```go
type Stores struct {
    Users UserStore
    Posts PostStore
}

g := Greeter{users: stores.Users}   // not: Greeter{stores}
```

## Testing without a database

Because the consumer holds an interface, a hand-written double is
enough. Function fields make each test say exactly what it needs:

```go
type stubUsers struct {
    ByIDFn func(ctx context.Context, id int64) (User, error)
}

func (s stubUsers) ByID(ctx context.Context, id int64) (User, error) {
    return s.ByIDFn(ctx, id)
}
func (s stubUsers) Create(context.Context, string) (User, error) { return User{}, nil }
```

The same logic now runs with no database at all:

```go
g := Greeter{users: stubUsers{ByIDFn: func(_ context.Context, id int64) (User, error) {
    return User{ID: id, Name: "Stub"}, nil
}}}
fmt.Println(g.Greet(ctx, 1))   // output: hello, Stub <nil>
```

And the branches that are awkward to produce against a real database
become one line each:

```go
// not found
return User{}, ErrUserNotFound
// → hello, stranger <nil>

// connection failure
return User{}, errors.New("connection reset")
// → greeting 7: connection reset
```

That second case is the argument for the whole pattern. Simulating a
dropped connection against a live database is hard; against an
interface it is trivial.

## Where it stops being worth it

- **Do not add a method per query.** A store with `ByName`,
  `ByNameAndStatus`, `ByNameAndStatusAndCreatedAfter` is a query builder
  written badly. Take a filter struct instead.
- **Do not map everything.** A reporting query feeding one endpoint can
  be a single method returning a purpose-built result type. Forcing it
  through the domain model helps nobody.
- **Do not abstract for a swap that will not happen.** The real benefits
  are testability and keeping SQL in one place. "We might change
  database" almost never materialises, and designing for it makes the
  interface worse.
- **Transactions cross stores.** Two stores in one transaction need to
  share it — which is the context-carried transaction from the
  [previous article](03-transactions.md), precisely because the store
  method signatures do not mention `*sql.Tx`.

> **From Python:** this is the repository pattern you would build over
> SQLAlchemy, but the interface is defined by the *consumer* and
> satisfied implicitly, so there is no base class and no registration.
> A test double is any struct with the right methods.

## Quick reference

| Concern | Do |
|---|---|
| where the interface lives | the contract package, with the domain types |
| where the SQL lives | a separate package, unexported struct |
| constructor | `func NewUserStore(db *sql.DB) UserStore` |
| prove it satisfies | `var _ UserStore = userStore{}` |
| no rows | translate to your own `ErrNotFound`, wrapped with `%w` |
| other errors | wrap with context, pass through |
| what consumers take | the narrowest interface they use |
| test doubles | a struct with function fields |
| transactions across stores | carry it in the context |
| granularity | filter structs, not a method per query |

## Sources

- [Effective Go: interfaces — go.dev/doc/effective_go#interfaces](https://go.dev/doc/effective_go#interfaces)
- [`errors.Is` — pkg.go.dev/errors#Is](https://pkg.go.dev/errors#Is)
- [`sql.ErrNoRows` — pkg.go.dev/database/sql#pkg-variables](https://pkg.go.dev/database/sql#pkg-variables)
- [Go Code Review Comments: interfaces — go.dev/wiki/CodeReviewComments#interfaces](https://go.dev/wiki/CodeReviewComments#interfaces)
- [Accessing databases — go.dev/doc/database/](https://go.dev/doc/database/)
