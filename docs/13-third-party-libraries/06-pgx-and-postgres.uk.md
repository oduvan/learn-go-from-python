# pgx та PostgreSQL

`database/sql` не прив'язаний до конкретного драйвера, а це означає,
що він говорить лише тим підмножинним SQL, який спільний для всіх баз
даних. pgx — драйвер, специфічний саме для PostgreSQL, і використання
його напряму дає вам типи, які Postgres справді має.

> **Модулі:** `github.com/jackc/pgx/v5` та
> `github.com/google/uuid`.

Спирається на [`database/sql`](../09-database-sql/01-database-sql.md) —
API, який цей матеріал заміняє, або в який вбудовується, якщо хочете.

```go
pool, err := pgxpool.New(ctx, dsn)
defer pool.Close()
```

## Два способи використання

pgx може бути драйвером *позаду* `database/sql`:

```go
import _ "github.com/jackc/pgx/v5/stdlib"

db, err := sql.Open("pgx", dsn)
```

Тоді все з базової статті застосовується без змін. Оберіть цей варіант,
коли хочете портативність або коли якась бібліотека очікує `*sql.DB`.

Або використовуйте його нативний API, про який йде решта цієї статті:

```go
pool, err := pgxpool.New(ctx, dsn)
```

Нативний API дає вам типи Postgres, пакетну обробку й `COPY`. Шлях
через `database/sql` дає стандартний інтерфейс. Обидва прийнятні;
змішувати їх в одній кодовій базі — ні.

## Пул

```go
cfg, err := pgxpool.ParseConfig(dsn)
cfg.MaxConns = 10
cfg.MaxConnLifetime = time.Hour

pool, err := pgxpool.NewWithConfig(ctx, cfg)
defer pool.Close()

if err := pool.Ping(ctx); err != nil {
    return fmt.Errorf("database unreachable: %w", err)
}
```

`ParseConfig` читає DSN і дає вам структуру для налаштування — саме так
ви встановлюєте ліміти пулу, не кодуючи їх у рядку підключення. Він
також читає стандартні змінні середовища `PG*`.

`*pgxpool.Pool` — те, що безпечно передавати між горутинами, еквівалент
`*sql.DB`. `pool.Stat()` показує `MaxConns`, `AcquiredConns` і
`EmptyAcquireCount` — варто винести на дашборд.

## Типи Postgres просто працюють

Ось причина використовувати нативний API. Жодних `pq.Array`, жодних
типів-обгорток — масиви, `jsonb`, `uuid` і `timestamptz` зіставляються
зі звичайними значеннями Go:

```go
_, err = pool.Exec(ctx,
    `INSERT INTO items (id, name, tags, meta) VALUES ($1,$2,$3,$4)`,
    id, "first", []string{"a", "b"}, map[string]any{"k": 1})

var (
    gotID   uuid.UUID
    tags    []string
    meta    map[string]any
    created time.Time
)
err = pool.QueryRow(ctx, `SELECT id,name,tags,meta,created_at FROM items WHERE id=$1`, id).
    Scan(&gotID, &name, &tags, &meta, &created)
// [a b]  map[k:1]
```

Колонка `text[]` сканується напряму в `[]string`, а `jsonb` — у
`map[string]any`. Через `database/sql` обом потрібен був би власний
[`sql.Scanner`](../09-database-sql/02-custom-column-types.md).

## UUID, і чому v7

```go
id, err := uuid.NewV7()
fmt.Println(id.Version())   // output: VERSION_7
```

Версія 4 повністю випадкова. Версія 7 кладе часову мітку у старші
біти, тож id, згенеровані пізніше, сортуються пізніше:

```go
id1, _ := uuid.NewV7()
id2, _ := uuid.NewV7()
fmt.Println(id1.String() < id2.String())   // output: true
```

Це має значення для первинного ключа. Випадкові ключі v4 розсіюють
вставки по всьому B-дереву, фрагментуючи індекс; впорядковані за часом
ключі v7 дописуються в кінець, що тримає вставки дешевими, а свіжі
рядки — фізично поруч.

`uuid.Parse` валідує:

```go
_, err := uuid.Parse("not-a-uuid")
fmt.Println(err)   // output: invalid UUID length: 10
```

У Postgres 18 є вбудований `uuidv7()`, тож `DEFAULT uuidv7()` у схемі —
альтернатива генерації id у коді Go.

## Помилки

Відсутність рядків має власний сентинел:

```go
err := pool.QueryRow(ctx, `SELECT name FROM items WHERE name='nope'`).Scan(&name)
fmt.Println(errors.Is(err, pgx.ErrNoRows))   // output: true
// no rows in result set
```

Зауважте — це `pgx.ErrNoRows`, а не `sql.ErrNoRows`. Через адаптер
`database/sql` ви отримаєте останній.

Усе інше, що відхиляє сервер, повертається як `*pgconn.PgError` з кодом
SQLSTATE:

```go
var pgErr *pgconn.PgError
if errors.As(err, &pgErr) {
    fmt.Println("code:", pgErr.Code, "constraint:", pgErr.ConstraintName)
}
// code: 23505 constraint: items_name_key
```

Саме це перетворює дублікат на чисту відповідь "вже існує" замість
`500`:

| SQLSTATE | Значення |
|---|---|
| `23505` | порушення унікальності |
| `23503` | порушення зовнішнього ключа |
| `23502` | порушення not-null |
| `23514` | порушення check-обмеження |
| `40001` | помилка серіалізації — повторити |
| `57014` | запит скасовано (тайм-аут стейтменту) |

Порівнюйте за кодом, ніколи за текстом повідомлення. І перекладайте на
межі сховища, як доводить
[патерн репозиторію](../09-database-sql/04-the-repository-pattern.md) —
`pgErr` не має витікати у ваші обробники.

## Збір рядків

`pgx.CollectRows` прибирає цикл сканування:

```go
rows, _ := pool.Query(ctx, `SELECT name FROM items ORDER BY name`)
names, err := pgx.CollectRows(rows, pgx.RowTo[string])
// [first second]
```

І у структури:

```go
type Item struct {
    ID   uuid.UUID
    Name string
}

rows, _ := pool.Query(ctx, `SELECT id, name FROM items ORDER BY name`)
items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Item])
```

`RowToStructByPos` зіставляє за **позицією**, тож порядок у `SELECT`
має збігатися з порядком полів — зміна порядку в структурі мовчки
ламає це. `RowToStructByName` зіставляє за тегами `db` натомість, що
безпечніше, коли будь-яка сторона змінюється.

`CollectRows` сам закриває рядки й повертає будь-яку помилку, включно з
тією, яку ніс би `rows.Err()` — прибираючи найлегшу помилку з циклу
базової статті.

## Пакетна обробка

Кілька стейтментів за один раунд-тріп:

```go
b := &pgx.Batch{}
for _, item := range items {
    b.Queue(`INSERT INTO items (id,name) VALUES ($1,$2)`, item.ID, item.Name)
}

br := pool.SendBatch(ctx, b)
if err := br.Close(); err != nil {
    return err
}
```

`Close` повідомляє першу помилку, і його **обов'язково** треба
викликати. Для масового завантаження `pool.CopyFrom` реалізує протокол
Postgres `COPY` і ще швидший — тисячі рядків замість десятків. `COPY` —
це команда Postgres для масового завантаження: замість одного `INSERT`
на кожен рядок, клієнт надсилає всі рядки на сервер за одну операцію.

## Транзакції та listen/notify

```go
tx, err := pool.Begin(ctx)
defer tx.Rollback(ctx)   // no-op after commit
// ...
return tx.Commit(ctx)
```

Та сама форма, що й у базовій статті, тільки методи беруть контекст.
`pgx.BeginFunc` обгортає патерн замикання за вас.

pgx також надає `LISTEN`/`NOTIFY` через `conn.WaitForNotification`, що
дає push-сповіщення від бази даних без опитування — те, чого
`database/sql` взагалі не вміє висловити.

## pgvector

Для ембедингів `pgvector` додає тип колонки `vector` і пакет
`github.com/pgvector/pgvector-go`, що реєструється з pgx. Колонка
`vector(1536)` тоді сканується в `[]float32`, а
`ORDER BY embedding <=> $1 LIMIT 10` виконує пошук найближчих сусідів
прямо в SQL.

> **З досвіду Python:** pgx — це psycopg3, нативний драйвер Postgres із
> справжньою адаптацією типів, тоді як обгортка `stdlib` ближча до
> використання його через SQLAlchemy Core. `CollectRows` — це
> `fetchall()` з row factories, а `pgErr.Code` — `e.sqlstate` з
> psycopg.

## Швидка довідка

| Задача | Форма |
|---|---|
| нативний пул | `pgxpool.New(ctx, dsn)` / `NewWithConfig` |
| налаштувати | `pgxpool.ParseConfig`, потім встановити `MaxConns` |
| позаду `database/sql` | `_ "github.com/jackc/pgx/v5/stdlib"`, `sql.Open("pgx", …)` |
| масиви / jsonb / uuid | звичайні `[]string`, `map[string]any`, `uuid.UUID` |
| немає рядків | `errors.Is(err, pgx.ErrNoRows)` |
| порушення обмеження | `errors.As(err, &pgErr)`, перевірити `pgErr.Code` |
| зібрати результати | `pgx.CollectRows(rows, pgx.RowToStructByName[T])` |
| багато записів | `pgx.Batch` + `SendBatch`, або `CopyFrom` для масового завантаження |
| транзакції | `pool.Begin(ctx)`, або `pgx.BeginFunc` |
| первинні ключі | `uuid.NewV7()` — впорядкований за часом, дружній до індексу |

## Джерела

- [pgx — pkg.go.dev/github.com/jackc/pgx/v5](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [`pgxpool` — pkg.go.dev/github.com/jackc/pgx/v5/pgxpool](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool)
- [`pgconn.PgError` — pkg.go.dev/github.com/jackc/pgx/v5/pgconn#PgError](https://pkg.go.dev/github.com/jackc/pgx/v5/pgconn#PgError)
- [`google/uuid` — pkg.go.dev/github.com/google/uuid](https://pkg.go.dev/github.com/google/uuid)
- [PostgreSQL error codes — postgresql.org/docs/current/errcodes-appendix.html](https://www.postgresql.org/docs/current/errcodes-appendix.html)
- [pgvector — github.com/pgvector/pgvector-go](https://github.com/pgvector/pgvector-go)
