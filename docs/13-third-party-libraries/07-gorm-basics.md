# GORM basics

GORM maps structs to tables and generates SQL. It removes most of the
`Scan` boilerplate from
[`database/sql`](../09-database-sql/01-database-sql.md), and it
introduces behaviours you have to know about — one of which writes the
wrong value to your database without telling you.

> **Modules:** `gorm.io/gorm` and `gorm.io/driver/postgres`.

```go
db, err := gorm.Open(postgres.New(postgres.Config{DSN: dsn}), &gorm.Config{})
```

## Opening

```go
db, err := gorm.Open(postgres.New(postgres.Config{
    DSN: dsn,
}), &gorm.Config{
    Logger:                 logger.Default.LogMode(logger.Warn),
    SkipDefaultTransaction: true,
})
```

`*gorm.DB` wraps a `database/sql` pool, so pool tuning still happens
there:

```go
sqlDB, err := db.DB()
sqlDB.SetMaxOpenConns(25)
sqlDB.SetConnMaxLifetime(time.Hour)
```

Two config options worth setting deliberately. `SkipDefaultTransaction`
turns off the implicit transaction GORM wraps around every single
write — a measurable saving when you are managing transactions
yourself. And set the `Logger` level explicitly, because the default
logs every slow query to stdout in a format nothing parses.

## Models

```go
type Base struct {
    ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
    UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type Book struct {
    Base
    Title    string    `gorm:"not null"`
    AuthorID uuid.UUID `gorm:"type:uuid;index"`
    Draft    bool      `gorm:"not null;default:true"`
    Notes    *string
    Ignored  string    `gorm:"-"`
}
```

Embedding a `Base` gives every table its id and timestamps — struct
embedding from [structs](../02-language-basics/10-structs.md), with the
tags promoted along with the fields.

| Tag | Effect |
|---|---|
| `primaryKey` | the primary key |
| `type:uuid` | the SQL column type |
| `not null`, `unique`, `index` | constraints |
| `default:expr` | a column default — **see the trap below** |
| `column:name` | override the derived column name |
| `autoCreateTime` / `autoUpdateTime` | maintained by GORM |
| `-` | not persisted |

Names are derived by convention: `Book` → `books`, `AuthorID` →
`author_id`. A typo in a `column:` tag compiles fine and silently maps
to the wrong column, which is why a schema-drift check is worth having.

A `*string` is a nullable column; a plain `string` is `NOT NULL` with
`''` as its zero. The distinction from
[encoding JSON](../06-text-time-and-data/07-encoding-json.md) applies
here too.

## The `default:` zero-value trap

This is the one to internalise. GORM omits zero-valued fields from the
generated `INSERT`, so the *database* default applies instead of the
value you set:

```go
b := Book{Title: "Explicitly not a draft", Draft: false}
db.Create(&b)

var back Book
db.First(&back, "id = ?", b.ID)
fmt.Println(back.Draft)   // output: true
```

You wrote `false`. The database holds `true`. No error anywhere.

The commonly-repeated fix is `Select`. **It does not work** — all three
forms still produce `true`:

```go
db.Select("Name", "Draft").Create(&b)   // still true
db.Select("*").Create(&b)               // still true
```

Two things actually fix it. A **pointer field**, where `nil` and
`&false` are distinguishable:

```go
type Book struct {
    Draft *bool `gorm:"not null;default:true"`
}

f := false
db.Create(&Book{Title: "x", Draft: &f})   // stored: false
```

Or **create from a map**, which has no zero values to skip:

```go
db.Model(&Book{}).Create(map[string]any{"name": "map", "draft": false})
// stored: false
```

The same trap applies to `Updates` with a struct:

```go
db.Model(&b).Updates(Book{Title: "Renamed", Draft: false})
// Title changes; Draft does not
```

`Updates` with a map updates exactly what you list:

```go
db.Model(&b).Updates(map[string]any{"draft": false})   // works
```

The simplest defence is to avoid `default:` on booleans and numbers
entirely, and set the value in Go. If you need the database default,
make the field a pointer.

## Reading

```go
var book Book
err := db.WithContext(ctx).First(&book, "title = ?", title).Error
```

Pass a context with `WithContext` on every call. Without it the query
has no cancellation, exactly as with the plain driver.

`First` returns a sentinel when nothing matches:

```go
err := db.First(&book, "title = ?", "nope").Error
fmt.Println(errors.Is(err, gorm.ErrRecordNotFound))   // output: true
```

**`Find` does not:**

```go
var books []Book
r := db.Where("title = ?", "nope").Find(&books)
fmt.Println(r.Error, r.RowsAffected, len(books))
// output: <nil> 0 0
```

An empty result is not an error for a list query. Check
`RowsAffected` or `len`, not `Error`.

Every call returns a `*gorm.DB` carrying `Error` and `RowsAffected`.
Checking `.Error` is the equivalent of `if err != nil`, and forgetting
it is the easiest mistake to make here — nothing in the type system
requires it.

## Writing

```go
res := db.WithContext(ctx).Create(&a)
fmt.Println(res.Error, res.RowsAffected)   // output: <nil> 1
```

`Create` fills the primary key and timestamps back into your struct.

Constraint violations surface as the driver's error, so
[`errors.As` on `*pgconn.PgError`](06-pgx-and-postgres.md) still works:

```
ERROR: duplicate key value violates unique constraint "uni_authors_name" (SQLSTATE 23505)
```

## Hooks

Methods with reserved names run around operations:

```go
func (b *Book) BeforeCreate(tx *gorm.DB) error {
    if b.Title == "" {
        return errors.New("title is required")
    }
    return nil
}
```

```go
err := db.Create(&Book{}).Error
fmt.Println(err)   // output: title is required
```

Returning an error aborts the write. Also available: `AfterCreate`,
`BeforeUpdate`, `BeforeDelete` and others.

Use them sparingly. A hook is behaviour that fires invisibly from the
call site, which makes it hard to trace — validation is usually clearer
in the service layer.

## `AutoMigrate` is for development

```go
db.AutoMigrate(&Author{}, &Book{})
```

It creates tables and adds missing columns. It will **not** drop
columns, change types safely, or produce a reviewable diff, and it has
no notion of running once. Use it in tests and local development; use
[goose](09-goose-migrations.md) for anything you deploy.

## Associations

```go
type Author struct {
    Base
    Name  string `gorm:"not null;unique"`
    Books []Book `gorm:"foreignKey:AuthorID"`
}
```

```go
var a Author
db.Preload("Books").First(&a, "id = ?", id)
```

Without `Preload`, `a.Books` is empty — GORM does not lazy-load. Be
deliberate: preloading a list endpoint's associations is how you get
an N+1 query pattern, and `Joins` is often the better answer.

> **From Python:** GORM is roughly SQLAlchemy's ORM with Django-style
> tags instead of a declarative schema. The zero-value trap has no
> Python equivalent, because `None` and `False` are distinct there —
> in Go they are the same `false` unless you use a pointer.

## Quick reference

| Task | Form |
|---|---|
| open | `gorm.Open(postgres.New(...), &gorm.Config{})` |
| pool tuning | `db.DB()` then the `database/sql` setters |
| context | `db.WithContext(ctx)` on every call |
| check for failure | `.Error` on the returned `*gorm.DB` |
| not found | `errors.Is(err, gorm.ErrRecordNotFound)` — `First` only |
| empty list | `Find` returns no error; check `RowsAffected` |
| **zero value + `default:`** | use a `*bool`, or `Create`/`Updates` with a map |
| associations | `Preload("Books")` — never lazy |
| schema | `AutoMigrate` in dev, real migrations in production |

## Sources

- [GORM documentation — gorm.io/docs/](https://gorm.io/docs/)
- [`gorm.io/gorm` — pkg.go.dev/gorm.io/gorm](https://pkg.go.dev/gorm.io/gorm)
- [Model declaration — gorm.io/docs/models.html](https://gorm.io/docs/models.html)
- [Create — gorm.io/docs/create.html](https://gorm.io/docs/create.html)
- [Hooks — gorm.io/docs/hooks.html](https://gorm.io/docs/hooks.html)
