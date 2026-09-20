# Custom column types

`Scan` handles the basic types. For anything else — a JSON column, an
enum, a comma-separated list — you teach the type to convert itself by
implementing two interfaces.

```go
type Status string

func (s Status) Value() (driver.Value, error) { return string(s), nil }
func (s *Status) Scan(src any) error          { /* ... */ }
```

## The two interfaces

```go
// database/sql/driver
type Valuer interface { Value() (driver.Value, error) }

// database/sql
type Scanner interface { Scan(src any) error }
```

`Valuer` converts your type **to** something the driver can send.
`Scanner` converts what came back **into** your type. Implement both and
your type works anywhere a plain `string` would.

`driver.Value` must be one of a small set: `int64`, `float64`, `bool`,
`[]byte`, `string`, `time.Time`, or `nil`. Return anything else and the
driver rejects it.

## The receivers are not symmetric

```go
func (s Status) Value() (driver.Value, error)   // value receiver
func (s *Status) Scan(src any) error            // pointer receiver
```

`Scan` **must** take a pointer, because it writes to the receiver. `Value`
only reads, so a value receiver is right — and it means both `Status`
and `*Status` satisfy `Valuer`, which is what you want when passing one
as a query argument.

Get this backwards and nothing complains at compile time. The type
simply is not recognised as a `Scanner`, and you get a confusing
conversion error at runtime. The
[compile-time assertion](../03-object-oriented-go/02-interfaces.md) is
worth adding:

```go
var (
    _ driver.Valuer = Status("")
    _ sql.Scanner   = (*Status)(nil)
)
```

## A JSONB column

Postgres `jsonb` arrives as bytes. Holding it as raw bytes rather than a
decoded map keeps it cheap to pass through, and lets you decode into a
real struct only where you need to:

```go
type JSONB []byte

func (j JSONB) Value() (driver.Value, error) {
    if len(j) == 0 {
        return nil, nil          // empty means SQL NULL
    }
    return string(j), nil
}

func (j *JSONB) Scan(src any) error {
    switch v := src.(type) {
    case nil:
        *j = nil
    case []byte:
        cp := make(JSONB, len(v))
        copy(cp, v)              // copy: src is only valid during Scan
        *j = cp
    case string:
        *j = JSONB(v)
    default:
        return fmt.Errorf("jsonb: cannot scan %T", src)
    }
    return nil
}
```

**The copy is not optional.** The `[]byte` handed to `Scan` belongs to
the driver and may be reused for the next row. Keeping it without
copying gives you values that mutate under you — a bug that only shows
up when a query returns more than one row.

Handle `nil`, `[]byte` *and* `string`: which one you get depends on the
driver, so accepting both makes your type portable.

Adding `MarshalJSON` lets the column pass straight through to an API
response without being decoded and re-encoded:

```go
func (j JSONB) MarshalJSON() ([]byte, error) {
    if j == nil {
        return []byte("null"), nil
    }
    return j, nil
}
```

```go
out, _ := json.Marshal(Doc{ID: 1, Meta: JSONB(`{"k":1}`)})
fmt.Println(string(out))   // output: {"id":1,"meta":{"k":1}}

out, _ = json.Marshal(Doc{ID: 2})
fmt.Println(string(out))   // output: {"id":2,"meta":null}
```

Without it, `JSONB` is a `[]byte` and `encoding/json` would base64-encode
it.

## An enum that validates on the way in

A `Scanner` is a good place to reject data that should not exist:

```go
type Status string

const (
    StatusActive Status = "active"
    StatusBanned Status = "banned"
)

func (s *Status) Scan(src any) error {
    var str string
    switch v := src.(type) {
    case string:
        str = v
    case []byte:
        str = string(v)
    default:
        return fmt.Errorf("status: cannot scan %T", src)
    }
    switch Status(str) {
    case StatusActive, StatusBanned:
        *s = Status(str)
        return nil
    }
    return fmt.Errorf("status: unknown value %q", str)
}
```

A row written by something else, or left over from a migration, now
fails loudly instead of flowing through as an unrecognised string:

```go
err := db.QueryRowContext(ctx, `SELECT status FROM docs WHERE status='bogus'`).Scan(&st)
fmt.Println(err)
// output: sql: Scan error on column index 0, name "status": status: unknown value "bogus"
```

Note how `database/sql` wraps your message with the column name — you
get the context for free.

## A list in one column

The same mechanism flattens a slice into a scalar column:

```go
type CSV []string

func (c CSV) Value() (driver.Value, error) { return strings.Join(c, ","), nil }

func (c *CSV) Scan(src any) error {
    s, _ := src.(string)
    if s == "" {
        *c = nil
        return nil
    }
    *c = strings.Split(s, ",")
    return nil
}
```

Reading a row back gives `tags=[go sql]`, and an empty column gives an
empty slice. This is a convenience, not a schema design — a real
many-to-many wants a join table, and Postgres has native array and
`jsonb` types that are queryable.

## Timestamps

`time.Time` is already a valid `driver.Value`, so it works without any
help. One thing to get right in the schema: use `timestamptz`, not
`timestamp`. Without the zone, what you get back depends on the session
and you will eventually be off by hours.

Scan a nullable timestamp into `*time.Time` or `sql.NullTime`, since the
zero `time.Time` is year 1 rather than an absence — the point the
[time article](../06-text-time-and-data/04-time.md) makes.

## When not to bother

If a column maps cleanly to a basic type, use the basic type. Custom
types earn their place when the conversion is non-obvious, when the same
mapping is repeated in many queries, or when validating on read is worth
doing. One type with `Value`/`Scan` beats the same conversion copied
across twenty call sites.

> **From Python:** this is a SQLAlchemy `TypeDecorator`, split into two
> methods on your own type instead of a separate adapter class. The Go
> version is checked at compile time — but only if you add the
> `var _ sql.Scanner = ...` assertion, because satisfying an interface
> is implicit.

## Quick reference

| Concern | Form |
|---|---|
| Go → database | `func (T) Value() (driver.Value, error)` — value receiver |
| database → Go | `func (*T) Scan(src any) error` — **pointer** receiver |
| allowed return types | `int64`, `float64`, `bool`, `[]byte`, `string`, `time.Time`, `nil` |
| NULL out | return `nil, nil` from `Value` |
| NULL in | handle `case nil:` in `Scan` |
| driver differences | accept both `[]byte` and `string` |
| keeping bytes | **copy them** — the slice is reused |
| prove it compiles | `var _ sql.Scanner = (*T)(nil)` |
| JSON passthrough | add `MarshalJSON` |
| validate on read | return an error from `Scan` |

## Sources

- [`driver.Valuer` — pkg.go.dev/database/sql/driver#Valuer](https://pkg.go.dev/database/sql/driver#Valuer)
- [`sql.Scanner` — pkg.go.dev/database/sql#Scanner](https://pkg.go.dev/database/sql#Scanner)
- [`driver.Value` — pkg.go.dev/database/sql/driver#Value](https://pkg.go.dev/database/sql/driver#Value)
- [`sql.NullTime` — pkg.go.dev/database/sql#NullTime](https://pkg.go.dev/database/sql#NullTime)
- [Accessing databases — go.dev/doc/database/](https://go.dev/doc/database/)
