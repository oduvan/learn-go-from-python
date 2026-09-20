# Основи GORM

GORM зіставляє структури з таблицями й генерує SQL. Він прибирає більшу
частину шаблонного коду `Scan` з
[`database/sql`](../09-database-sql/01-database-sql.md), і привносить
поведінку, про яку варто знати — одна з них мовчки записує неправильне
значення у вашу базу даних, нічого про це не повідомляючи.

> **Модулі:** `gorm.io/gorm` та `gorm.io/driver/postgres`.

```go
db, err := gorm.Open(postgres.New(postgres.Config{DSN: dsn}), &gorm.Config{})
```

## Відкриття

```go
db, err := gorm.Open(postgres.New(postgres.Config{
    DSN: dsn,
}), &gorm.Config{
    Logger:                 logger.Default.LogMode(logger.Warn),
    SkipDefaultTransaction: true,
})
```

`*gorm.DB` обгортає пул `database/sql`, тож налаштування пулу
відбувається саме там:

```go
sqlDB, err := db.DB()
sqlDB.SetMaxOpenConns(25)
sqlDB.SetConnMaxLifetime(time.Hour)
```

Дві опції конфігурації варті свідомого налаштування. `SkipDefaultTransaction`
вимикає неявну транзакцію, якою GORM обгортає кожен окремий запис —
відчутна економія, коли ви керуєте транзакціями самостійно. І
встановлюйте рівень `Logger` явно, бо стандартний логує кожен повільний
запит у stdout у форматі, який ніхто не парсить.

## Моделі

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

Вбудовування `Base` дає кожній таблиці її id і часові мітки — вбудовування
структур зі статті про [структури](../02-language-basics/10-structs.md),
з тегами, що просуваються разом із полями.

| Тег | Ефект |
|---|---|
| `primaryKey` | первинний ключ |
| `type:uuid` | тип колонки SQL |
| `not null`, `unique`, `index` | обмеження |
| `default:expr` | значення колонки за замовчуванням — **див. пастку нижче** |
| `column:name` | перевизначити виведену назву колонки |
| `autoCreateTime` / `autoUpdateTime` | підтримується GORM |
| `-` | не зберігається |

Імена виводяться за домовленістю: `Book` → `books`, `AuthorID` →
`author_id`. Одруківка в тегу `column:` компілюється без проблем і
мовчки зіставляється з неправильною колонкою — саме тому варта наявності
перевірка на розбіжність зі схемою.

`*string` — колонка, що допускає NULL; звичайний `string` — `NOT NULL`
з `''` як нульовим значенням. Ця відмінність зі статті про
[кодування JSON](../06-text-time-and-data/07-encoding-json.md)
застосовується й тут.

### Масиви-колонки Postgres потребують `lib/pq`

GORM, що працює на драйвері pgx, усе одно не зіставляє Postgres
`text[]` з `[]string` самостійно. Тип, що це робить — `pq.StringArray`
з `github.com/lib/pq`:

```go
type Tag struct {
    ID    int64          `gorm:"primaryKey"`
    Name  string         `gorm:"uniqueIndex"`
    Langs pq.StringArray `gorm:"type:text[];not null;default:'{}'"`
}
```

```go
t := Tag{Name: "go", Langs: pq.StringArray{"go", "python"}}
db.Create(&t)

var back Tag
db.First(&back, "name = ?", "go")
fmt.Println(back.Langs, len(back.Langs))   // output: [go python] 2
```

Це дивує людей, бо [pgx](06-pgx-and-postgres.md) зіставляє масиви з
`[]string` нативно — але це *власний* API pgx. Пройдіть через GORM, і
ви повернетесь до типу `driver.Valuer`/`sql.Scanner`, яким і є
`pq.StringArray`. Є `pq.Int64Array` та подібні для інших типів
елементів, і `pq.Array(&v)` обгортає зріз для одноразового запиту.

Тож кодова база може залежати від `lib/pq` суто заради цих типів,
використовуючи pgx як фактичний драйвер. Це не помилка; це звичайне
влаштування.

## Пастка нульового значення з `default:`

Це те, що варто засвоїти назавжди. GORM пропускає поля з нульовим
значенням у згенерованому `INSERT`, тож натомість застосовується
значення за замовчуванням *бази даних*, а не те, яке ви встановили:

```go
b := Book{Title: "Explicitly not a draft", Draft: false}
db.Create(&b)

var back Book
db.First(&back, "id = ?", b.ID)
fmt.Println(back.Draft)   // output: true
```

Ви написали `false`. У базі даних лежить `true`. Жодної помилки ніде.

Часто повторюване виправлення — `Select`. **Воно не працює** — усі три
форми все одно дають `true`:

```go
db.Select("Name", "Draft").Create(&b)   // still true
db.Select("*").Create(&b)               // still true
```

Насправді виправляють це дві речі. **Поле-вказівник**, де `nil` і
`&false` розрізняються:

```go
type Book struct {
    Draft *bool `gorm:"not null;default:true"`
}

f := false
db.Create(&Book{Title: "x", Draft: &f})   // stored: false
```

Або **створення з мапи**, в якій немає нульових значень, щоб їх
пропускати:

```go
db.Model(&Book{}).Create(map[string]any{"name": "map", "draft": false})
// stored: false
```

Та сама пастка застосовується до `Updates` зі структурою:

```go
db.Model(&b).Updates(Book{Title: "Renamed", Draft: false})
// Title changes; Draft does not
```

`Updates` з мапою оновлює точно те, що ви перелічили:

```go
db.Model(&b).Updates(map[string]any{"draft": false})   // works
```

Найпростіший захист — взагалі уникати `default:` на булевих і числових
полях і встановлювати значення в коді Go. Якщо вам потрібне значення за
замовчуванням бази даних, зробіть поле вказівником.

## Читання

```go
var book Book
err := db.WithContext(ctx).First(&book, "title = ?", title).Error
```

Передавайте контекст через `WithContext` у кожному виклику. Без нього
запит не має скасування, так само як і з простим драйвером.

`First` повертає сентинел, коли нічого не знайдено:

```go
err := db.First(&book, "title = ?", "nope").Error
fmt.Println(errors.Is(err, gorm.ErrRecordNotFound))   // output: true
```

**`Find` — ні:**

```go
var books []Book
r := db.Where("title = ?", "nope").Find(&books)
fmt.Println(r.Error, r.RowsAffected, len(books))
// output: <nil> 0 0
```

Порожній результат — не помилка для запиту списку. Перевіряйте
`RowsAffected` або `len`, а не `Error`.

Кожен виклик повертає `*gorm.DB`, що несе `Error` і `RowsAffected`.
Перевірка `.Error` — це еквівалент `if err != nil`, і забути про це —
найлегша помилка тут — нічого в системі типів цього не вимагає.

## Запис

```go
res := db.WithContext(ctx).Create(&a)
fmt.Println(res.Error, res.RowsAffected)   // output: <nil> 1
```

`Create` заповнює первинний ключ і часові мітки назад у вашу структуру.

Порушення обмежень виринають як помилка драйвера, тож
[`errors.As` на `*pgconn.PgError`](06-pgx-and-postgres.md) все ще
працює:

```
ERROR: duplicate key value violates unique constraint "uni_authors_name" (SQLSTATE 23505)
```

## Хуки

Методи з зарезервованими іменами запускаються навколо операцій:

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

Повернення помилки скасовує запис. Також доступні: `AfterCreate`,
`BeforeUpdate`, `BeforeDelete` та інші.

Використовуйте їх ощадливо. Хук — це поведінка, що спрацьовує непомітно
з місця виклику, що ускладнює трасування — валідація зазвичай зрозуміліша
в сервісному шарі.

## `AutoMigrate` — для розробки

```go
db.AutoMigrate(&Author{}, &Book{})
```

Він створює таблиці й додає відсутні колонки. Він **не** видалятиме
колонки, безпечно змінюватиме типи чи давати diff, придатний для
рецензування, і не має поняття про одноразовий запуск. Використовуйте
його в тестах і локальній розробці; використовуйте
[goose](09-goose-migrations.md) для всього, що ви деплоїте.

## Асоціації

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

Без `Preload` `a.Books` порожній — GORM не завантажує ліниво. Будьте
свідомі: попереднє завантаження асоціацій ендпоінта списку — це те, як
ви отримуєте патерн N+1 запитів, і `Joins` часто краща відповідь.

> **З досвіду Python:** GORM приблизно як ORM SQLAlchemy з тегами в
> стилі Django замість декларативної схеми. Пастка нульового значення
> не має аналогу в Python, бо там `None` і `False` — різні речі — у Go
> вони той самий `false`, якщо ви не використовуєте вказівник.

## Швидка довідка

| Задача | Форма |
|---|---|
| відкрити | `gorm.Open(postgres.New(...), &gorm.Config{})` |
| налаштування пулу | `db.DB()`, потім сетери `database/sql` |
| контекст | `db.WithContext(ctx)` у кожному виклику |
| перевірити провал | `.Error` на поверненому `*gorm.DB` |
| не знайдено | `errors.Is(err, gorm.ErrRecordNotFound)` — лише `First` |
| порожній список | `Find` не повертає помилку; перевірте `RowsAffected` |
| **нульове значення + `default:`** | використовуйте `*bool`, або `Create`/`Updates` з мапою |
| асоціації | `Preload("Books")` — ніколи не ліниво |
| схема | `AutoMigrate` у розробці, справжні міграції в продакшені |

## Джерела

- [GORM documentation — gorm.io/docs/](https://gorm.io/docs/)
- [`gorm.io/gorm` — pkg.go.dev/gorm.io/gorm](https://pkg.go.dev/gorm.io/gorm)
- [Model declaration — gorm.io/docs/models.html](https://gorm.io/docs/models.html)
- [Create — gorm.io/docs/create.html](https://gorm.io/docs/create.html)
- [Hooks — gorm.io/docs/hooks.html](https://gorm.io/docs/hooks.html)
