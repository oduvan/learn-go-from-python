# GORM queries and transactions

Building queries with the chain API, dropping to SQL when it stops
paying, and running transactions. Builds on
[transactions](../09-database-sql/03-transactions.md) and
[GORM basics](07-gorm-basics.md).

```go
db.WithContext(ctx).
    Where("pages > ?", 80).
    Order("pages desc").
    Limit(2).
    Find(&books)
```

## Chaining

Each method returns a `*gorm.DB`, and nothing executes until a
**finisher** — `Find`, `First`, `Scan`, `Count`, `Create`, `Update`,
`Delete`:

```go
var books []Book
db.WithContext(ctx).
    Where("pages > ?", 80).
    Where("title <> ?", "Gamma").
    Order("pages desc").
    Limit(2).
    Find(&books)
// Beta 300
// Alpha 100
```

Repeated `Where` calls are combined with `AND`. `Or` and slice
arguments work as expected:

```go
db.Where("title = ?", "Alpha").Or("pages < ?", 60).Find(&b)
db.Where("title IN ?", []string{"Alpha", "Beta"}).Find(&b)
```

Note `IN ?` takes the slice directly — no placeholder expansion by
hand.

### Reusing a chain is a trap

A `*gorm.DB` mid-chain carries accumulated conditions. Storing one and
using it twice leaks conditions from the first query into the second:

```go
q := db.Where("pages > ?", 80)
q.Find(&a)                       // pages > 80
q.Where("title = ?", "x").Find(&b)  // pages > 80 AND title = 'x'
```

Start each query from `db`, or call `db.Session(&gorm.Session{})` to
get a clean one.

## Placeholders are still placeholders

The chain API parameterises everything, so injection is no more
possible than with the plain driver:

```go
evil := "Alpha'; DROP TABLE books; --"
db.Model(&Book{}).Where("title = ?", evil).Count(&n)
// matched: 0, table intact
```

What is *not* safe is interpolating into the condition string itself.
`Where(fmt.Sprintf("title = '%s'", input))` is an injection, exactly as
it would be anywhere else. The same goes for `Order` and `Select`,
where a user-supplied column name must be checked against an
allow-list — identifiers cannot be parameterised.

## Selecting into something that is not a model

Aggregates and joins rarely fit your entity types. Define a small
result struct and `Scan` into it:

```go
type titleCount struct {
    Name  string
    Total int
}

var out []titleCount
err := db.Model(&Author{}).
    Select("authors.name as name, count(books.id) as total").
    Joins("left join books on books.author_id = authors.id").
    Group("authors.name").
    Scan(&out).Error
// [{Ada 3}]
```

The `as name` / `as total` aliases are what let GORM match columns to
fields. `Scan` does not apply model logic — no hooks, no soft-delete
filtering — which for a read-only projection is what you want.

## Dropping to SQL

When a query is easier to read as SQL, write it as SQL:

```go
var rc []titleCount
err := db.Raw(`SELECT a.name, count(b.id) AS total FROM authors a
               LEFT JOIN books b ON b.author_id = a.id GROUP BY a.name`).
    Scan(&rc).Error
// [{Ada 3}]
```

`Raw` is for statements returning rows; `Exec` is for those that do
not:

```go
r := db.Exec(`UPDATE books SET pages = pages + 1 WHERE pages > ?`, 80)
fmt.Println(r.RowsAffected)   // output: 2
```

Both take placeholders, so both are safe with user input.

Reach for raw SQL when you need a window function, a CTE, a bulk
`UPDATE ... FROM`, full-text search, or anything vendor-specific — and
when the chain version would be longer than the SQL. A useful team
habit is to leave a one-line comment saying which of those it is, so a
reviewer can see whether the exception is a sanctioned one or a
shortcut.

## Transactions

```go
err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&a).Error; err != nil {
        return err
    }
    return tx.Create(&b).Error
})
```

Return `nil` to commit, an error to roll back. A panic rolls back and
re-panics — GORM handles that for you, unlike the hand-rolled version.

**Every statement inside must use `tx`.** A call on `db` runs on a
different connection and is not part of the transaction — the same rule
as `database/sql`, and just as easy to break because `db` is in scope.

## Carrying it in the context

Threading `tx` through every store method means two versions of each.
Putting it in the context keeps one signature, using the key discipline
from [context as a carrier](../11-architecture-and-conventions/02-context-as-a-carrier.md):

```go
type txKey struct{}

func WithTx(ctx context.Context, db *gorm.DB, fn func(context.Context) error) error {
    if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
        return fn(ctx)                       // already inside one: join it
    }
    return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        return fn(context.WithValue(ctx, txKey{}, tx))
    })
}

func resolve(ctx context.Context, db *gorm.DB) *gorm.DB {
    if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
        return tx.WithContext(ctx)
    }
    return db.WithContext(ctx)
}
```

Every store method begins with `resolve(ctx, db)` and works either way:

```go
err := WithTx(ctx, db, func(ctx context.Context) error {
    return resolve(ctx, db).Create(&Book{Title: "InTx"}).Error
})
```

Rollback works as expected:

```go
err := WithTx(ctx, db, func(ctx context.Context) error {
    resolve(ctx, db).Create(&Book{Title: "Doomed"})
    return errors.New("business rule")
})
// err: business rule, rows named Doomed: 0
```

And the early return means nesting joins the outer transaction instead
of deadlocking on a second connection:

```go
err := WithTx(ctx, db, func(ctx context.Context) error {
    return WithTx(ctx, db, func(ctx context.Context) error {
        return resolve(ctx, db).Create(&Book{Title: "Nested"}).Error
    })
})
// err: <nil>, rows named Nested: 1
```

The trade is the one the core article named: the signature no longer
says whether a function writes inside a transaction.

GORM also offers `SavePoint` and `RollbackTo` for partial rollback
within a transaction, and manual `db.Begin()`/`Commit()` when a
closure does not fit.

## Watching the SQL

When a chain does not produce what you expected, print it:

```go
sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
    return tx.Where("pages > ?", 80).Find(&[]Book{})
})
```

`ToSQL` builds the statement without running it. Turning the logger to
`logger.Info` in development shows every query with its timing, which
is usually how you notice a `Preload` has become N+1.

> **From Python:** the chain API is SQLAlchemy's query builder, and
> `Raw`/`Exec` are `session.execute(text(...))`. The reusability trap
> is the opposite of SQLAlchemy's: there, query objects are immutable
> and safe to reuse; here, a stored `*gorm.DB` accumulates state.

## Quick reference

| Task | Form |
|---|---|
| build | chain `Where`/`Order`/`Limit`, execute with a finisher |
| reuse a chain | don't — start from `db`, or `db.Session(...)` |
| slice condition | `Where("col IN ?", slice)` |
| aggregate | `Select("... as alias")` + `Scan(&dto)` |
| raw rows | `db.Raw(sql, args...).Scan(&v)` |
| raw statement | `db.Exec(sql, args...)` → `RowsAffected` |
| user-supplied column | allow-list it; only values can be placeholders |
| transaction | `db.Transaction(func(tx *gorm.DB) error { ... })` |
| inside one | **use `tx`**, never `db` |
| uniform signatures | carry the tx in the context, `resolve` per call |
| nesting | detect and join, or you deadlock |
| see the SQL | `db.ToSQL(...)`, or `logger.Info` |

## Sources

- [GORM query documentation — gorm.io/docs/query.html](https://gorm.io/docs/query.html)
- [Advanced query — gorm.io/docs/advanced_query.html](https://gorm.io/docs/advanced_query.html)
- [Raw SQL — gorm.io/docs/sql_builder.html](https://gorm.io/docs/sql_builder.html)
- [Transactions — gorm.io/docs/transactions.html](https://gorm.io/docs/transactions.html)
- [Session — gorm.io/docs/session.html](https://gorm.io/docs/session.html)
