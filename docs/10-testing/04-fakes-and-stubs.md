# Fakes and stubs

A unit test should not need a database, a network, or a clock that
actually ticks. In Go the substitution mechanism is the interface, and
the substitute is usually a struct you write in ten lines.

```go
type stubUsers struct {
    ByIDFn func(ctx context.Context, id string) (User, error)
}

func (s stubUsers) ByID(ctx context.Context, id string) (User, error) {
    return s.ByIDFn(ctx, id)
}
```

## Design for it: depend on an interface

None of this works if a type reaches for a global or constructs its own
dependency. The prerequisite is that the thing under test *receives*
what it needs:

```go
type API struct{ Users UserStore }
```

`UserStore` is the narrow interface from
[the repository pattern](../09-database-sql/04-the-repository-pattern.md).
Because satisfaction is implicit, the test double needs no base class,
no registration, and no relationship to the real implementation beyond
having the same methods.

## Function fields make each test say what it needs

The stub above holds a function per method. A test fills in only the
behaviour it cares about:

```go
api := API{Users: stubUsers{ByIDFn: func(_ context.Context, id string) (User, error) {
    return User{Name: "Ada"}, nil
}}}
```

Failure cases are one line each, which is the point — these are the
paths that are painful to trigger against a real dependency:

```go
// missing row
return User{}, ErrNotFound

// the database fell over
return User{}, errors.New("connection reset")
```

Add a compile-time check so the stub cannot drift from the interface:

```go
var _ UserStore = stubUsers{}
```

For an interface with several methods, give the unused ones a trivial
body returning zero values. If that gets tedious, the interface is
probably too wide for its consumer.

### Embedding the interface, for a deliberately partial fake

The other common idiom embeds the interface instead of listing every
method:

```go
type partialUsers struct {
    UserStore                      // embedded: satisfies the interface
    ByIDFn func(context.Context, string) (User, error)
}

func (p partialUsers) ByID(ctx context.Context, id string) (User, error) {
    return p.ByIDFn(ctx, id)
}
```

The embedded interface is nil, so the struct satisfies `UserStore`
while implementing exactly one method. Calling any other **panics with
a nil pointer dereference**.

That sounds like a defect and is the point. The test asserts not only
what the code does call but what it does not: if someone later makes
the code under test call `Create`, the test fails loudly instead of
quietly accepting a zero value. It also means a new method on the
interface does not break every existing double.

The trade against function fields:

| | Function fields | Embedded interface |
|---|---|---|
| unimplemented method | returns a zero value | panics |
| new interface method | must add a stub | nothing to change |
| reads as | explicit, verbose | terse, implicit |

Use function fields when several methods matter, embedding when the
test is about one method and you want the rest to be an error.

## Recording what happened

When the assertion is "it called the thing", have the stub record:

```go
type spyNotifier struct {
    sent []string
}

func (s *spyNotifier) Notify(_ context.Context, msg string) error {
    s.sent = append(s.sent, msg)
    return nil
}
```

Note the **pointer receiver** — a value receiver would append to a copy
and your recording would vanish. Pass `&spyNotifier{}`.

If the code under test is concurrent, guard the slice with a mutex or
the race detector will rightly complain.

## Three kinds of double, and when each fits

| Kind | What it is | Use when |
|---|---|---|
| **stub** | returns canned answers | you need a specific return, including errors |
| **spy** | records calls | the behaviour *is* the call |
| **fake** | a real, simple implementation | many tests need working behaviour |

A fake is worth the effort once several tests need the dependency to
actually behave — an in-memory store backed by a map, for instance:

```go
type memUsers struct {
    mu sync.Mutex
    m  map[string]User
}

func (f *memUsers) ByID(_ context.Context, id string) (User, error) {
    f.mu.Lock()
    defer f.mu.Unlock()
    u, ok := f.m[id]
    if !ok {
        return User{}, ErrNotFound
    }
    return u, nil
}
```

One fake serves a whole package's tests, and it stays honest because it
implements the same interface the real code uses.

## Why no mocking library

Go codebases often have none, and it is worth understanding why rather
than assuming it is an oversight.

Hand-written doubles are ordinary Go: they compile, they refactor with
your IDE, and reading one tells you exactly what it does. Generated
mocks add a build step, a dependency, and a layer of `EXPECT().Times(1)`
indirection that describes the *implementation* rather than the
behaviour — tests that break when you reorder two calls that were never
ordered in the first place.

Libraries earn their place with very wide interfaces, or when you need
strict call-order assertions. Both are usually signs the design could be
narrower. Start by hand.

## Faking time

Anything calling `time.Now()` internally cannot be tested deterministically.
Inject it:

```go
type Service struct {
    now func() time.Time
}

func New() *Service { return &Service{now: time.Now} }
```

```go
fixed := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
s := &Service{now: func() time.Time { return fixed }}
```

The same trick works for randomness and for id generation. A function
field is lighter than an interface when there is only one method.

Do not fake time with `time.Sleep` in tests. It makes suites slow and
flaky. If the code sleeps, that is usually a sign the delay should be a
parameter.

## What not to fake

Faking everything tests the fakes. Keep real:

- **Pure logic.** Nothing to inject.
- **The standard library.** Do not wrap `encoding/json` to fake it.
- **The thing you are actually testing.** Faking the database in the
  store's own tests means you are testing the fake. Those need a real
  database — which is what testcontainers is for, in the third-party
  topic.

The rule of thumb: fake across a boundary you own, at the point where
the test would otherwise need the network, the disk, or the clock.

> **From Python:** there is no `unittest.mock.patch`, because there is
> nothing to patch — you cannot reassign a function in another package
> at runtime. Substitution happens at construction, through a parameter.
> More verbose to set up, and it makes the seams explicit in the type
> signatures.

## Quick reference

| Task | Form |
|---|---|
| make it substitutable | take a narrow interface as a field or parameter |
| a stub | a struct of function fields, one per method |
| pin it to the interface | `var _ Iface = stub{}` |
| record calls | a spy with a **pointer** receiver |
| reusable behaviour | a fake — an in-memory implementation |
| fake the clock | a `now func() time.Time` field |
| mocking library | usually unnecessary; start by hand |
| do not fake | pure logic, the stdlib, or the thing under test |

## Sources

- [Effective Go: interfaces — go.dev/doc/effective_go#interfaces](https://go.dev/doc/effective_go#interfaces)
- [Go wiki: code review comments — go.dev/wiki/CodeReviewComments#interfaces](https://go.dev/wiki/CodeReviewComments#interfaces)
- [`testing` package reference — pkg.go.dev/testing](https://pkg.go.dev/testing)
- [Go blog: the race detector — go.dev/blog/race-detector](https://go.dev/blog/race-detector)
