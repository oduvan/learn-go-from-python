# Структуроване логування через `log/slog`

`log/slog` — це структурований логер стандартної бібліотеки. Кожен
запис — це повідомлення плюс типізовані атрибути ключ-значення, а це
означає, що логи можна опитувати, а не грепати.

```go
slog.Info("server started", "port", 8080, "tls", false)
// level=INFO msg="server started" port=8080 tls=false
```

## Два обробники

Сам логер нічого не форматує — це робить `Handler`. Два постачаються зі
стандартною бібліотекою:

```go
slog.New(slog.NewTextHandler(os.Stdout, nil))   // для читання людиною
slog.New(slog.NewJSONHandler(os.Stdout, nil))   // для читання машиною
```

```go
jl.Info("request", "method", "GET", "path", "/users", "ms", 12)
// {"level":"INFO","msg":"request","method":"GET","path":"/users","ms":12}
```

Звичне влаштування — текст локально й JSON у продакшні, вирішене один
раз при старті:

```go
func init() {
    if os.Getenv("ENVIRONMENT") != "local" {
        opts := &slog.HandlerOptions{Level: parseLevel(os.Getenv("LOG_LEVEL"))}
        slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, opts)))
    }
}
```

`slog.SetDefault` робить так, що `slog.Info` на рівні пакета й подібні
функції використовують ваш обробник, тож код бібліотек логує через
нього, навіть не отримуючи логер напряму.

Логуйте у **stdout**, а не у файл. У контейнері щось інше збирає цей
вивід; запис у файл означає, що ротація логів стає вашою проблемою.

## Атрибути

Вільна форма чергує ключі й значення:

```go
slog.Info("cache miss", "key", "u:1")
```

Це стисло й неперевірено — непарна кількість аргументів дає запис
`!BADKEY`, а не помилку компіляції. Типізована форма цього уникає й
швидша, бо нічого не доводиться боксувати:

```go
slog.Info("started",
    slog.Int("workers", 4),
    slog.Duration("timeout", 5*time.Second),
)
// {"level":"INFO","msg":"started","workers":4,"timeout":5000000000}
```

Зверніть увагу, `Duration` серіалізується в наносекундах у JSON. Якщо
ваша платформа логування хоче мілісекунди, передайте натомість
`slog.Int64("timeout_ms", d.Milliseconds())`.

`slog.Any` покриває типи без спеціального конструктора.

## Рівні

Чотири рівні: `Debug`, `Info`, `Warn`, `Error`. Це впорядковані цілі
числа, тож фільтрація — це порівняння:

```go
fmt.Println(slog.LevelWarn > slog.LevelInfo)   // output: true
fmt.Println(int(slog.LevelError))              // output: 8
```

Встановіть поріг на обробнику:

```go
slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
```

Усе нижче нього відкидається дешево — аргументи навіть не
форматуються. Зробіть рівень настроюваним через змінну середовища, щоб
продакшн можна було підняти без деплою.

`Error` означає, що людина має діяти. Дивіться статтю про
[домовленості проєкту](../11-architecture-and-conventions/06-project-conventions.md)
щодо того, для чого призначений кожен рівень; коротка версія — логуйте
збій один раз, на тому рівні, що його обробляє.

## `With` для контексту, що повторюється

`With` повертає логер, що несе атрибути, тож ви встановлюєте їх один
раз:

```go
log := slog.Default().With(
    slog.String("service", "api"),
    slog.String("version", "1.2.3"),
)
log.Info("started", slog.Int("workers", 4))
// {"level":"INFO","msg":"started","service":"api","version":"1.2.3","workers":4}
```

Це головний інструмент для зіставлення записів між собою. Логер,
створений на кожен запит із прикріпленим id запиту, означає, що кожен
рядок з цього запиту можна знайти, не протягуючи id крізь кожен виклик.

`WithGroup` і `slog.Group` вкладають атрибути, що не дає іменам
зіштовхуватися:

```go
slog.Info("db query",
    slog.Group("db", slog.String("table", "users"), slog.Int("rows", 3)),
)
// {"level":"INFO","msg":"db query","db":{"table":"users","rows":3}}
```

## `LogValuer` тримає секрети подалі

Тип може контролювати власне представлення в логах. Це надійний спосіб
не пустити пароль у логи — надійний тому, що працює всюди, де значення
логується, а не лише там, де хтось про це згадав:

```go
type User struct {
    Name     string
    Password string
}

func (u User) LogValue() slog.Value {
    return slog.GroupValue(slog.String("name", u.Name))
}
```

```go
slog.Info("login", "user", User{Name: "ada", Password: "hunter2"})
// {"level":"INFO","msg":"login","user":{"name":"ada"}}
```

Пароль зник. Реалізуйте `LogValue` на кожному типі, що тримає облікові
дані, токен чи персональні дані, і редагування слідує за типом усюди.

Це також відкладає роботу: `LogValue` викликається, лише якщо запис
справді видається, тож дороге представлення нічого не коштує на
відфільтрованому рівні.

## Варіанти з `Context`

`InfoContext`, `ErrorContext` та подібні передають контекст обробнику.
Самі собою вони нічого видимого не роблять — суть у тому, що власний
обробник може дістати з нього значення:

```go
type ctxHandler struct{ slog.Handler }

func (h ctxHandler) Handle(ctx context.Context, r slog.Record) error {
    if id, ok := ctx.Value(ctxKey{}).(string); ok {
        r.AddAttrs(slog.String("request_id", id))
    }
    return h.Handler.Handle(ctx, r)
}
```

```go
cl.InfoContext(ctx, "handled")
// {"level":"INFO","msg":"handled","request_id":"req-42"}
```

`slog.Handler` — це інтерфейс із чотирма методами:

```go
type Handler interface {
    Enabled(context.Context, Level) bool
    Handle(context.Context, Record) error
    WithAttrs(attrs []Attr) Handler
    WithGroup(name string) Handler
}
```

Вбудовування `slog.Handler` означає, що ви успадковуєте всі чотири й
перевизначаєте лише те, що потрібно — той самий трюк із вбудовуванням,
що й рекордер статусу з
[middleware](../08-http-with-net-http/03-middleware.md). Є одна пастка.
Успадковані `WithAttrs` і `WithGroup` повертають *внутрішній* обробник,
тож `cl.With("user", "ada")` дає вам логер без вашої обгортки, і
`request_id` тихо зникає. Перевизначте і ці два методи, щоб вони знову
обгортали свій результат:

```go
func (h ctxHandler) WithAttrs(as []slog.Attr) slog.Handler {
    return ctxHandler{h.Handler.WithAttrs(as)}
}

func (h ctxHandler) WithGroup(name string) slog.Handler {
    return ctxHandler{h.Handler.WithGroup(name)}
}
```

```go
cl.With("user", "ada").InfoContext(ctx, "handled")
// {"level":"INFO","msg":"handled","user":"ada","request_id":"req-42"}
```

Саме так id трейсу автоматично потрапляє в кожен рядок логу.
Використовуйте варіанти з `Context` за замовчуванням; вони нічого не
коштують і роблять це можливим пізніше.

## Перевірка логів у тесті

Підмініть стандартний обробник на той, що пише в буфер:

```go
var buf bytes.Buffer
old := slog.Default()
slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
defer slog.SetDefault(old)

slog.Warn("careful", "n", 1)
// time=2026-09-23T10:15:02.000+02:00 level=WARN msg=careful n=1
```

З JSON-обробником ви можете декодувати кожен рядок і перевіряти поля,
а не зіставляти текст. Відновіть попередній стандартний обробник —
`t.Cleanup` тут доречне місце — інакше ви залишите буфер витікати в
кожен наступний тест.

`HandlerOptions.ReplaceAttr` — це те, що робить вивід детермінованим.
Обробник викликає його для кожного атрибута перед записом, і якщо
повернути порожній `slog.Attr`, цей атрибут зникає. Приберіть час, і
рядки стають порівнюваними:

```go
opts := &slog.HandlerOptions{
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        if a.Key == slog.TimeKey && len(groups) == 0 {
            return slog.Attr{} // прибрати атрибут часу
        }
        return a
    },
}
slog.SetDefault(slog.New(slog.NewTextHandler(&buf, opts)))

slog.Warn("careful", "n", 1)
// level=WARN msg=careful n=1
```

## Старіший пакет `log`

`log.Printf` і `log.Fatal` досі існують і досі трапляються в невеликих
програмах. Дві речі варто знати: `log.Fatal` викликає `os.Exit`, тож
відкладені функції не виконуються; і його вивід неструктурований, тож
його не можна опитувати. Для сервісу використовуйте `slog`.

> **З досвіду Python:** це `structlog`, у стандартній бібліотеці, і
> без жодної ієрархії логерів — без `getLogger(__name__)`, без
> поширення, без `dictConfig`. Ви будуєте обробник, встановлюєте
> стандартний і передаєте логери явно. `LogValuer` — це `__repr__` для
> логів, і саме цій частині в логуванні Python немає гідного
> еквівалента.

## Швидка довідка

| Задача | Форма |
|---|---|
| обробник | `slog.NewJSONHandler(os.Stdout, opts)` / `NewTextHandler` |
| зробити стандартним | `slog.SetDefault(slog.New(h))` |
| логувати | `slog.Info("msg", "key", value)` |
| типізовані, перевірені атрибути | `slog.Int("n", 4)`, `slog.String(...)` |
| поріг | `&slog.HandlerOptions{Level: slog.LevelInfo}` |
| повторювані поля | `logger.With(slog.String("service", "api"))` |
| вкладеність | `slog.Group("db", ...)` |
| редагувати тип | реалізувати `LogValue() slog.Value` |
| значення з контексту | `InfoContext` плюс обробник, що його читає |
| перевірити тестом | підмінити стандартний обробник на буферний, відновити після |
| детермінований вивід | `HandlerOptions.ReplaceAttr`, щоб прибрати `time` |

## Джерела

- [`log/slog` package reference — pkg.go.dev/log/slog](https://pkg.go.dev/log/slog)
- [`slog.Handler` — pkg.go.dev/log/slog#Handler](https://pkg.go.dev/log/slog#Handler)
- [`slog.LogValuer` — pkg.go.dev/log/slog#LogValuer](https://pkg.go.dev/log/slog#LogValuer)
- [`slog.HandlerOptions` — pkg.go.dev/log/slog#HandlerOptions](https://pkg.go.dev/log/slog#HandlerOptions)
- [Go blog: structured logging with slog — go.dev/blog/slog](https://go.dev/blog/slog)
