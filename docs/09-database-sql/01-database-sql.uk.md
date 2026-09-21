# `database/sql`

`database/sql` — стандартний інтерфейс до SQL-бази даних. Це не ORM, і
він не генерує запити — ви пишете SQL, а він керує з'єднаннями та
перетворює результати у значення Go.

```go
var name string
err := db.QueryRowContext(ctx, `SELECT name FROM users WHERE id = $1`, id).Scan(&name)
```

> **Одна примітка щодо цієї теми.** Щоб взаємодіяти зі справжньою базою
> даних, `database/sql` потребує драйвер, а кожен драйвер — сторонній
> модуль. Приклади тут запускались на PostgreSQL, але в цій статті
> немає нічого специфічного для драйвера, окрім синтаксису
> плейсхолдерів. Вибір і імпорт драйвера розглянуто разом з іншими
> зовнішніми бібліотеками.

## `Open` дає вам пул, а не з'єднання

```go
db, err := sql.Open("pgx", dsn)
defer db.Close()
```

Тут два сюрпризи. `sql.Open` **не з'єднується** — вона перевіряє
аргументи й одразу повертає результат, тож неправильний хост тут не
дасть жодної помилки. І `*sql.DB` — це *пул*, а не з'єднання: він
безпечний для конкурентного використання, відкриває з'єднання за
потреби, і його варто створити один раз при старті й передавати далі.
Відкривати новий пул на кожен запит — серйозна помилка.

Щоб дізнатись, чи база даних насправді досяжна, запитайте:

```go
if err := db.PingContext(ctx); err != nil {
    return fmt.Errorf("database unreachable: %w", err)
}
```

Робіть це при старті, щоб неправильно налаштований DSN одразу давав
збій, а не при першому запиті.

### Ліміти пулу

Типові налаштування — необмежена кількість відкритих з'єднань і лише
два вільні, що для сервера неправильно в обидва боки:

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(25)
db.SetConnMaxLifetime(5 * time.Minute)
```

- `SetMaxOpenConns` обмежує загальну кількість з'єднань. Без обмеження
  сплеск трафіку вичерпає ліміт з'єднань бази даних замість того, щоб
  запити стали в чергу. Тримайте значення з запасом нижче того, що
  дозволяє база даних, поділеного на кількість реплік.
- `SetMaxIdleConns` зазвичай варто зробити таким самим, інакше
  з'єднання постійно закриваються й відкриваються знову.
- `SetConnMaxLifetime` періодично «списує» з'єднання, що дозволяє
  балансувальнику навантаження перебалансувати трафік і уникнути
  таймаутів на боці сервера для неактивних з'єднань.

`db.Stats()` показує стан пулу — варто винести це на дашборд:
зростання `WaitCount` означає, що запити стоять у черзі за з'єднанням.

## Три виклики для запитів

| Виклик | Для |
|---|---|
| `ExecContext` | `INSERT`/`UPDATE`/`DELETE` — без рядків у відповідь |
| `QueryRowContext` | рівно один рядок |
| `QueryContext` | багато рядків |

Завжди використовуйте варіанти з `Context`. Прості `Exec`, `Query` і
`QueryRow` існують заради сумісності й не дають скасування — повільний
запит переживе той запит (request), що його викликав.

```go
res, err := db.ExecContext(ctx, `INSERT INTO users (name, age) VALUES ($1, $2)`, "Ada", 36)
n, _ := res.RowsAffected()
fmt.Println(n, err)   // output: 1 <nil>
```

`LastInsertId` підтримують не всі драйвери — наприклад, у PostgreSQL
його немає. Замість цього використовуйте `RETURNING`: це лише одне
звернення до бази даних:

```go
var id int64
err := db.QueryRowContext(ctx,
    `INSERT INTO users (name, age) VALUES ($1, $2) RETURNING id`, "Bo", 7).Scan(&id)
```

## Плейсхолдери — це не форматування рядка

Аргументи надсилаються окремо від SQL. База даних ніколи не розбирає
їх як код, тож немає нічого екранувати, і ін'єкція неможлива:

```go
evil := "Ada'; DROP TABLE users; --"

var cnt int
db.QueryRowContext(ctx, `SELECT count(*) FROM users WHERE name = $1`, evil).Scan(&cnt)
fmt.Println(cnt)   // output: 0
// таблиця досі на місці
```

Він шукав користувача, буквально названого
`Ada'; DROP TABLE users; --`, і нікого не знайшов.

Синтаксис плейсхолдерів залежить від драйвера: `$1, $2` для
PostgreSQL, `?` для MySQL і SQLite. **Ніколи** не збирайте запит через
`fmt.Sprintf` і користувацький ввід. Там, де мусить змінюватись
ідентифікатор — наприклад, колонка для сортування — перевіряйте його
за списком дозволених назв (allow-list), бо ідентифікатори не можна
параметризувати.

## `Scan` копіює колонки у ваші змінні

Передайте по вказівнику на кожну колонку, у тому порядку, в якому
запит їх вибирає:

```go
var u User
err := db.QueryRowContext(ctx,
    `SELECT id, name, email, age FROM users WHERE name = $1`, "Ada").
    Scan(&u.ID, &u.Name, &u.Email, &u.Age)
```

Перелічуйте назви колонок явно, а не використовуйте `SELECT *`. З `*`
додавання колонки до таблиці ламає кожен `Scan` під час виконання.

Запит, що нічому не відповідає, повертає сигнальну помилку (sentinel
error) — це очікуваний результат, а не збій:

```go
err := db.QueryRowContext(ctx, `... WHERE name = $1`, "Nobody").Scan(&u.Name)
fmt.Println(errors.Is(err, sql.ErrNoRows))   // output: true
// err: sql: no rows in result set
```

Обробляйте це явно — зазвичай перетворюючи на власну помилку «не
знайдено», щоб виклики (callers) не були прив'язані до `database/sql`.

## NULL потребує змінної, що допускає NULL

Спроба сканувати `NULL` у звичайний `string` завершується помилкою:

```go
var email string
err := db.QueryRowContext(ctx, `SELECT email FROM users WHERE name = $1`, "Ada").Scan(&email)
fmt.Println(err)
// output: sql: Scan error on column index 0, name "email": converting NULL to string is unsupported
```

Є два способи це виправити. **Вказівник**, де `nil` означає NULL:

```go
var email *string
// ... Scan(&email)
fmt.Println(email == nil)   // output: true
```

Або `sql.NullString`, який явно несе прапорець дійсності:

```go
var ns sql.NullString
// ... Scan(&ns)
fmt.Println(ns.Valid, ns.String)   // output: false ""
```

Для кожного базового типу є свій `sql.Null*`, а ще є узагальнений
(generic) `sql.Null[T]`. Вказівники виглядають природніше у
структурах; типи `Null` зрозуміліші там, де не можна плутати NULL з
нульовим значенням. Третій варіант — виправити це в SQL за допомогою
`COALESCE(email, '')`.

## Ітерація по рядках

```go
rows, err := db.QueryContext(ctx, `SELECT id, name, age FROM users ORDER BY name`)
if err != nil {
    return err
}
defer rows.Close()

var users []User
for rows.Next() {
    var u User
    if err := rows.Scan(&u.ID, &u.Name, &u.Age); err != nil {
        return err
    }
    users = append(users, u)
}
return rows.Err()
```

Чотири правила, і останнє — те, яке найчастіше пропускають:

1. `defer rows.Close()` — незакритий `Rows` утримує з'єднання з пулу.
   Якщо таких витоків достатньо, пул вичерпується, і це виглядає як
   зависання.
2. Перевіряйте помилку від `QueryContext`, перш ніж торкатись `rows`.
3. Перевіряйте помилку від кожного `Scan`.
4. **Перевіряйте `rows.Err()` після циклу.** `Next` повертає false і в
   кінці результатів, і при збої. Без цієї перевірки з'єднання, що
   обірвалось посеред ітерації, виглядає точнісінько як короткий
   набір результатів — ви повертаєте часткові дані без жодної
   помилки.

`rows.Close` ідемпотентний і безпечний поруч із `defer`.

## Помилки від бази даних

Порушення обмеження (constraint) повертається як помилка драйвера з
власним повідомленням бази даних:

```go
_, err := db.ExecContext(ctx, `INSERT INTO users (name, age) VALUES ($1, $2)`, "Ada", 1)
fmt.Println(err)
// output: ERROR: duplicate key value violates unique constraint "users_name_key" (SQLSTATE 23505)
```

Звірятись із текстом повідомлення — крихкий підхід. Драйвери надають
типізовану помилку, яку можна перевірити через `errors.As`, щоб
прочитати код SQLSTATE — `23505` означає порушення унікальності,
`23503` — порушення зовнішнього ключа. Так дублікат перетворюється на
охайну відповідь «вже існує». Цей тип специфічний для драйвера, тож
розглядається разом із ним.

## Підготовлені запити (prepared statements)

`database/sql` сам готує (prepare) і кешує запити, коли ви передаєте
аргументи, тож явний виклик `Prepare` рідко того вартий. Він має
значення для запиту, що виконується багато разів у щільному циклі:

```go
stmt, err := db.PrepareContext(ctx, `INSERT INTO users (name, age) VALUES ($1, $2)`)
defer stmt.Close()

for _, u := range users {
    if _, err := stmt.ExecContext(ctx, u.Name, u.Age); err != nil {
        return err
    }
}
```

`*sql.Stmt` прив'язаний до пулу і за потреби сам перепідготується на
іншому з'єднанні.

## Коли потрібне одне з'єднання

Усе, що має стан сесії — змінна сесії, консультативне блокування
(advisory lock), тимчасова таблиця — мусить виконуватись на одному й
тому самому з'єднанні. Виклики через пул щоразу дають довільне
з'єднання, тож зарезервуйте одне:

```go
conn, err := db.Conn(ctx)
defer conn.Close()   // повертає його в пул
```

Це важливіше, ніж здається: блокування, взяте на одному з'єднанні з
пулу і «звільнене» на іншому, насправді не звільняється.

> **З досвіду Python:** `*sql.DB` — це *пул* з'єднань, а не з'єднання
> з DB-API — ближче до engine у SQLAlchemy. Тут немає курсора; ви
> отримуєте `Rows`. А `Scan` працює у протилежному напрямку порівняно
> з `fetchone()`: ви передаєте йому вказівники замість того, щоб
> отримувати кортеж (tuple), і саме це дозволяє йому бути
> типобезпечним без рефлексії.

## Швидка довідка

| Завдання | Виклик |
|---|---|
| відкрити пул | `sql.Open(driver, dsn)` один раз — не з'єднується |
| перевірити доступність | `db.PingContext(ctx)` при старті |
| ліміти пулу | `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime` |
| запис | `db.ExecContext(ctx, q, args...)` → `RowsAffected` |
| новий id | `RETURNING id` + `QueryRowContext(...).Scan(&id)` |
| один рядок | `db.QueryRowContext(ctx, q, args...).Scan(&a, &b)` |
| немає збігу | `errors.Is(err, sql.ErrNoRows)` |
| багато рядків | `QueryContext`, `defer rows.Close()`, **`rows.Err()`** |
| NULL | `*string`, або `sql.NullString` |
| користувацький ввід | завжди плейсхолдер, ніколи `fmt.Sprintf` |
| стан сесії | `db.Conn(ctx)` |
| здоров'я пулу | `db.Stats()` |

## Джерела

- [`database/sql` package reference — pkg.go.dev/database/sql](https://pkg.go.dev/database/sql)
- [`sql.DB` — pkg.go.dev/database/sql#DB](https://pkg.go.dev/database/sql#DB)
- [`sql.Rows` — pkg.go.dev/database/sql#Rows](https://pkg.go.dev/database/sql#Rows)
- [`sql.ErrNoRows` — pkg.go.dev/database/sql#pkg-variables](https://pkg.go.dev/database/sql#pkg-variables)
- [Go wiki: SQL database drivers — go.dev/wiki/SQLDrivers](https://go.dev/wiki/SQLDrivers)
- [Accessing databases — go.dev/doc/database/](https://go.dev/doc/database/)
