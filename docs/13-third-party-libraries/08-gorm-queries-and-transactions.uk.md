# Запити та транзакції в GORM

Побудова запитів через ланцюжковий API, перехід на SQL, коли він
перестає окупатись, і виконання транзакцій. Спирається на
[транзакції](../09-database-sql/03-transactions.md) та
[основи GORM](07-gorm-basics.md).

```go
db.WithContext(ctx).
    Where("pages > ?", 80).
    Order("pages desc").
    Limit(2).
    Find(&books)
```

## Ланцюжки

Кожен метод повертає `*gorm.DB`, і нічого не виконується, доки не
з'явиться **завершувач** — `Find`, `First`, `Scan`, `Count`, `Create`,
`Update`, `Delete`:

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

Повторні виклики `Where` об'єднуються через `AND`. `Or` і аргументи-зрізи
працюють очікувано:

```go
db.Where("title = ?", "Alpha").Or("pages < ?", 60).Find(&b)
db.Where("title IN ?", []string{"Alpha", "Beta"}).Find(&b)
```

Зауважте: `IN ?` бере зріз напряму — жодного ручного розгортання
плейсхолдерів.

### Повторне використання ланцюжка — це пастка

`*gorm.DB` у середині ланцюжка несе накопичені умови. Зберегти один
такий об'єкт і використати його двічі означає, що умови з першого
запиту протікають у другий:

```go
q := db.Where("pages > ?", 80)
q.Find(&a)                       // pages > 80
q.Where("title = ?", "x").Find(&b)  // pages > 80 AND title = 'x'
```

Починайте кожен запит з `db`, або викликайте `db.Session(&gorm.Session{})`,
щоб отримати чистий об'єкт.

## Плейсхолдери залишаються плейсхолдерами

Ланцюжковий API параметризує все, тож ін'єкція так само неможлива, як і
з простим драйвером:

```go
evil := "Alpha'; DROP TABLE books; --"
db.Model(&Book{}).Where("title = ?", evil).Count(&n)
// matched: 0, table intact
```

Що *не* безпечно — це інтерполяція прямо в рядок умови.
`Where(fmt.Sprintf("title = '%s'", input))` — це ін'єкція, точно так
само, як і будь-де ще. Те саме стосується `Order` і `Select`, де
ім'я колонки, надане користувачем, має перевірятися за allow-list —
ідентифікатори не можна параметризувати.

## Вибірка в те, що не є моделлю

Агрегати й з'єднання рідко вкладаються у ваші типи сутностей. Оголосіть
невелику структуру результату й `Scan` у неї:

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

Псевдоніми `as name` / `as total` — саме те, що дозволяє GORM зіставити
колонки з полями. `Scan` не застосовує логіку моделі — без хуків, без
фільтрації м'якого видалення — що для проєкції лише на читання є саме
тим, що потрібно.

## Перехід на SQL

Коли запит легше читати як SQL, пишіть його як SQL:

```go
var rc []titleCount
err := db.Raw(`SELECT a.name, count(b.id) AS total FROM authors a
               LEFT JOIN books b ON b.author_id = a.id GROUP BY a.name`).
    Scan(&rc).Error
// [{Ada 3}]
```

`Raw` — для стейтментів, що повертають рядки; `Exec` — для тих, що ні:

```go
r := db.Exec(`UPDATE books SET pages = pages + 1 WHERE pages > ?`, 80)
fmt.Println(r.RowsAffected)   // output: 2
```

Обидва беруть плейсхолдери, тож обидва безпечні з користувацьким
введенням.

Тягніться до сирого SQL, коли вам потрібна віконна функція, CTE,
масовий `UPDATE ... FROM`, повнотекстовий пошук чи щось специфічне для
конкретної СУБД — і коли ланцюжкова версія була б довшою за SQL.
Корисна командна звичка — залишити однорядковий коментар, який саме з
цих випадків це є, щоб рецензент бачив, чи це санкціонований виняток, а
чи просто скорочення шляху.

## Клаузи: upsert, returning, блокування

Ланцюжковий API не може виразити все, що вміє SQL. `gorm.io/gorm/clause`
заповнює цю прогалину, і три його клаузи трапляються постійно.

**Upsert** — вставити, або оновити наявний рядок при конфлікті:

```go
up := Tag{Name: "go", Hits: 5, Langs: pq.StringArray{"go"}}

err := db.Clauses(clause.OnConflict{
    Columns:   []clause.Column{{Name: "name"}},
    DoUpdates: clause.AssignmentColumns([]string{"hits", "langs"}),
}).Create(&up).Error
// one row, hits=5 — updated rather than duplicated
```

`Columns` називає ціль конфлікту, `DoUpdates` — колонки для
перезапису. Це один атомарний стейтмент, тож він безпечний проти
конкурентної вставки так, як "спершу select, потім insert, якщо
відсутнє" — ні.

`DoNothing` — варіант з ігноруванням дублікатів:

```go
db.Clauses(clause.OnConflict{DoNothing: true}).Create(&Tag{Name: "go"})
// no error, no new row
```

**Returning** повертає згенеровані значення в тому самому раунд-тріпі:

```go
r := Tag{Name: "rust"}
db.Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Create(&r)
// r.ID is populated
```

**Locking** бере блокування рядка всередині транзакції для
читання-модифікації-запису без гонитви:

```go
db.Transaction(func(tx *gorm.DB) error {
    var locked Tag
    return tx.Clauses(clause.Locking{Strength: "UPDATE"}).
        First(&locked, "name = ?", "go").Error
})
```

Це генерує `SELECT ... FOR UPDATE`. Інші рядки не зачіпаються; інша
транзакція, що хоче той самий рядок, чекає.

Усі три тримають свої значення параметризованими, тож залишаються
безпечними з користувацьким введенням.

## Транзакції

```go
err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&a).Error; err != nil {
        return err
    }
    return tx.Create(&b).Error
})
```

Поверніть `nil`, щоб зафіксувати, помилку — щоб відкотити. Паніка
відкочує й знову панікує — GORM робить це за вас, на відміну від
версії, написаної вручну.

**Кожен стейтмент усередині має використовувати `tx`.** Виклик на `db`
виконується на іншому з'єднанні й не є частиною транзакції — те саме
правило, що й у `database/sql`, і так само легко порушити, бо `db`
все ще у видимості.

## Перенесення її в контекст

Протягування `tx` через кожен метод сховища означає дві версії кожного.
Розміщення в контексті залишає одну сигнатуру, використовуючи дисципліну
ключів зі статті про
[контекст як носій](../11-architecture-and-conventions/02-context-as-a-carrier.md):

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

Кожен метод сховища починається з `resolve(ctx, db)` і працює в обох
випадках:

```go
err := WithTx(ctx, db, func(ctx context.Context) error {
    return resolve(ctx, db).Create(&Book{Title: "InTx"}).Error
})
```

Відкат працює очікувано:

```go
err := WithTx(ctx, db, func(ctx context.Context) error {
    resolve(ctx, db).Create(&Book{Title: "Doomed"})
    return errors.New("business rule")
})
// err: business rule, rows named Doomed: 0
```

А ранній вихід означає, що вкладеність приєднується до зовнішньої
транзакції замість того, щоб зайти в deadlock на другому з'єднанні:

```go
err := WithTx(ctx, db, func(ctx context.Context) error {
    return WithTx(ctx, db, func(ctx context.Context) error {
        return resolve(ctx, db).Create(&Book{Title: "Nested"}).Error
    })
})
// err: <nil>, rows named Nested: 1
```

Обмін той самий, що назвала базова стаття: сигнатура більше не каже,
чи пише функція всередині транзакції.

GORM також пропонує `SavePoint` і `RollbackTo` для часткового відкату
всередині транзакції, і ручні `db.Begin()`/`Commit()`, коли замикання
не підходить.

## Спостереження за SQL

Коли ланцюжок видає не те, що ви очікували, виведіть його:

```go
sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
    return tx.Where("pages > ?", 80).Find(&[]Book{})
})
```

`ToSQL` будує стейтмент, не виконуючи його. Перемикання логера в
`logger.Info` під час розробки показує кожен запит із його таймінгом,
що зазвичай і є способом помітити, що `Preload` перетворився на N+1.

### `.Scan()` записує значення аргументів у лог

`logger.Config{ParameterizedQueries: true}` зазвичай тримає значення
поза логом, тож запит виглядає як `WHERE name = $1`. **`Scan` —
виняток**, і не має значення, чи запит прийшов з `Raw`, чи з
ланцюжка:

```go
db.Raw(`SELECT count(*) AS n FROM tags WHERE name = ?`, secret).Scan(&out)
// the log line contains the secret

db.Model(&Tag{}).Where("name = ?", secret).Scan(&out)
// so does this one

db.Where("name = ?", secret).Take(&one)    // stays parameterised
db.Where("name = ?", secret).Find(&many)   // stays parameterised
```

Тож хеш токена, id сесії чи API-ключ, передані в запит `Scan`,
опиняються у ваших логах відкритим текстом, тоді як ідентична умова в
`Take` чи `Find` — ні. Якщо значення чутливе, або уникайте `Scan` для
цього запиту, або хешуйте його до того, як воно дістанеться бази
даних, щоб залоговане значення було непридатним.

> **З досвіду Python:** ланцюжковий API — це білдер запитів
> SQLAlchemy, а `Raw`/`Exec` — це `session.execute(text(...))`.
> Пастка з повторним використанням протилежна до SQLAlchemy: там
> об'єкти запитів незмінні й безпечні для повторного використання; тут
> збережений `*gorm.DB` накопичує стан.

## Швидка довідка

| Задача | Форма |
|---|---|
| побудувати | ланцюжок `Where`/`Order`/`Limit`, виконати завершувачем |
| повторне використання ланцюжка | не робіть цього — починайте з `db`, або `db.Session(...)` |
| умова зі зрізом | `Where("col IN ?", slice)` |
| агрегат | `Select("... as alias")` + `Scan(&dto)` |
| сирі рядки | `db.Raw(sql, args...).Scan(&v)` |
| сирий стейтмент | `db.Exec(sql, args...)` → `RowsAffected` |
| колонка від користувача | занесіть в allow-list; плейсхолдерами можуть бути лише значення |
| транзакція | `db.Transaction(func(tx *gorm.DB) error { ... })` |
| усередині неї | **використовуйте `tx`**, ніколи `db` |
| однакові сигнатури | несіть tx у контексті, `resolve` на кожен виклик |
| вкладеність | виявити й приєднатися, або deadlock |
| побачити SQL | `db.ToSQL(...)`, або `logger.Info` |

## Джерела

- [GORM query documentation — gorm.io/docs/query.html](https://gorm.io/docs/query.html)
- [Advanced query — gorm.io/docs/advanced_query.html](https://gorm.io/docs/advanced_query.html)
- [Raw SQL — gorm.io/docs/sql_builder.html](https://gorm.io/docs/sql_builder.html)
- [Transactions — gorm.io/docs/transactions.html](https://gorm.io/docs/transactions.html)
- [Session — gorm.io/docs/session.html](https://gorm.io/docs/session.html)
