# Власні типи колонок

`Scan` уміє працювати з базовими типами. Для всього іншого — колонки
JSON, enum, списку через кому — ви навчаєте тип перетворювати себе
самостійно, реалізуючи два інтерфейси.

```go
type Status string

func (s Status) Value() (driver.Value, error) { return string(s), nil }
func (s *Status) Scan(src any) error          { /* ... */ }
```

## Два інтерфейси

```go
// database/sql/driver
type Valuer interface { Value() (driver.Value, error) }

// database/sql
type Scanner interface { Scan(src any) error }
```

`Valuer` перетворює ваш тип **у** щось, що драйвер може надіслати.
`Scanner` перетворює те, що прийшло назад, **у** ваш тип. Реалізуйте
обидва — і ваш тип працюватиме всюди, де підійшов би звичайний
`string`.

`driver.Value` мусить бути одним із невеликого набору типів: `int64`,
`float64`, `bool`, `[]byte`, `string`, `time.Time` або `nil`. Поверніть
щось інше — і драйвер це відхилить.

## Отримувачі не симетричні

```go
func (s Status) Value() (driver.Value, error)   // отримувач-значення
func (s *Status) Scan(src any) error            // отримувач-вказівник
```

`Scan` **мусить** приймати вказівник, бо він пише в отримувача. `Value`
лише читає, тож отримувач-значення тут правильний — а це означає, що й
`Status`, і `*Status` задовольняють `Valuer`, а це саме те, що потрібно,
коли ви передаєте значення як аргумент запиту.

Переплутайте це — і на етапі компіляції ніхто не поскаржиться. Тип
просто не розпізнається як `Scanner`, а під час виконання ви отримуєте
заплутану помилку перетворення. Варто додати
[перевірку на етапі компіляції](../03-object-oriented-go/02-interfaces.md):

```go
var (
    _ driver.Valuer = Status("")
    _ sql.Scanner   = (*Status)(nil)
)
```

## Колонка JSONB

Postgres-колонка `jsonb` приходить як байти. Зберігання її як сирих
байтів, а не розпакованої мапи, робить передачу дешевою і дозволяє
декодувати у справжню структуру лише там, де це справді потрібно:

```go
type JSONB []byte

func (j JSONB) Value() (driver.Value, error) {
    if len(j) == 0 {
        return nil, nil          // пусто означає SQL NULL
    }
    return string(j), nil
}

func (j *JSONB) Scan(src any) error {
    switch v := src.(type) {
    case nil:
        *j = nil
    case []byte:
        cp := make(JSONB, len(v))
        copy(cp, v)              // копія: src дійсний лише під час Scan
        *j = cp
    case string:
        *j = JSONB(v)
    default:
        return fmt.Errorf("jsonb: cannot scan %T", src)
    }
    return nil
}
```

**Копіювання тут не опційне.** `[]byte`, який передають у `Scan`,
належить драйверу і може бути повторно використаний для наступного
рядка. Якщо зберегти його без копіювання, значення почнуть змінюватись
самі собою — баг, який проявляється лише тоді, коли запит повертає
більше одного рядка.

Обробляйте `nil`, `[]byte` *і* `string`: який саме варіант ви
отримаєте, залежить від драйвера, тож приймання обох робить ваш тип
портативним.

Додавання `MarshalJSON` дозволяє колонці проходити прямо у відповідь
API без декодування й повторного кодування:

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

Без нього `JSONB` — це просто `[]byte`, і `encoding/json` закодував би
його в base64.

## Enum, що валідує дані на вході

`Scanner` — хороше місце, щоб відхиляти дані, яких не повинно існувати:

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

Рядок, записаний чимось іншим, або залишок після міграції, тепер
гучно провалюється, замість того щоб непомітно пройти як нерозпізнаний
рядок:

```go
err := db.QueryRowContext(ctx, `SELECT status FROM docs WHERE status='bogus'`).Scan(&st)
fmt.Println(err)
// output: sql: Scan error on column index 0, name "status": status: unknown value "bogus"
```

Зверніть увагу, як `database/sql` обгортає ваше повідомлення назвою
колонки — контекст ви отримуєте безкоштовно.

## Список в одній колонці

Той самий механізм «сплющує» зріз у скалярну колонку:

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

Читання рядка назад дає `tags=[go sql]`, а порожня колонка дає
порожній зріз. Це зручність, а не дизайн схеми — для справжнього
зв'язку «багато-до-багатьох» потрібна таблиця-з'єднання (join table), а
в Postgres є нативні масиви й типи `jsonb`, які можна опитувати
запитами.

## Часові мітки

`time.Time` уже є дійсним `driver.Value`, тож він працює без жодної
допомоги. Одне, що варто зробити правильно у схемі: використовуйте
`timestamptz`, а не `timestamp`. Без часового поясу те, що ви отримаєте
назад, залежатиме від сесії, і рано чи пізно ви помилитесь на кілька
годин.

Скануйте часову мітку, що допускає NULL, у `*time.Time` або
`sql.NullTime`, бо нульове значення `time.Time` — це 1-й рік, а не
відсутність значення. Саме на цьому наголошує
[стаття про час](../06-text-time-and-data/04-time.md).

## Коли не варто морочитись

Якщо колонка чисто відображається на базовий тип, використовуйте
базовий тип. Власні типи виправдовують себе тоді, коли перетворення
неочевидне, коли те саме зіставлення повторюється в багатьох запитах,
або коли варто валідувати дані при читанні. Один тип із `Value`/`Scan`
кращий за те саме перетворення, скопійоване у двадцять місць виклику.

> **З досвіду Python:** це аналог `TypeDecorator` із SQLAlchemy, тільки
> розділений на два методи вашого власного типу замість окремого
> класу-адаптера. Версія для Go перевіряється на етапі компіляції — але
> лише якщо ви додасте перевірку `var _ sql.Scanner = ...`, бо
> задоволення інтерфейсу відбувається неявно.

## Швидка довідка

| Питання | Форма |
|---|---|
| Go → база даних | `func (T) Value() (driver.Value, error)` — отримувач-значення |
| база даних → Go | `func (*T) Scan(src any) error` — отримувач-**вказівник** |
| дозволені типи повернення | `int64`, `float64`, `bool`, `[]byte`, `string`, `time.Time`, `nil` |
| NULL назовні | поверніть `nil, nil` із `Value` |
| NULL всередину | обробіть `case nil:` у `Scan` |
| відмінності драйверів | приймайте і `[]byte`, і `string` |
| зберігання байтів | **скопіюйте їх** — зріз буде використано повторно |
| довести, що компілюється | `var _ sql.Scanner = (*T)(nil)` |
| наскрізний прохід JSON | додайте `MarshalJSON` |
| валідація при читанні | поверніть помилку з `Scan` |

## Джерела

- [`driver.Valuer` — pkg.go.dev/database/sql/driver#Valuer](https://pkg.go.dev/database/sql/driver#Valuer)
- [`sql.Scanner` — pkg.go.dev/database/sql#Scanner](https://pkg.go.dev/database/sql#Scanner)
- [`driver.Value` — pkg.go.dev/database/sql/driver#Value](https://pkg.go.dev/database/sql/driver#Value)
- [`sql.NullTime` — pkg.go.dev/database/sql#NullTime](https://pkg.go.dev/database/sql#NullTime)
- [Accessing databases — go.dev/doc/database/](https://go.dev/doc/database/)
