# Fiber

Вебфреймворк, побудований на fasthttp, а не на `net/http`. Він міняє
сумісність зі стандартною бібліотекою на швидкість і компактніший API
обробників. Усе зі статті про [HTTP-сервер](../08-http-with-net-http/01-http-server.md)
все ще застосовується концептуально — просто форми змінюються.

> **Модуль:** `github.com/gofiber/fiber/v3`.

```go
app := fiber.New()
app.Get("/users/:id", func(c fiber.Ctx) error {
    return c.JSON(fiber.Map{"id": c.Params("id")})
})
app.Listen(":8080")
```

## v3 змінила сигнатуру обробника

У v2 обробник брав `*fiber.Ctx`. У **v3 `fiber.Ctx` — інтерфейс**, що
передається за значенням:

```go
func(c fiber.Ctx) error
```

Більшість прикладів v2 в інтернеті не скомпілюються. Якщо в фрагменті
написано `*fiber.Ctx` — це v2.

Справжня перевага цієї сигнатури над `net/http` — повернена `error`.
Замість того, щоб писати відповідь і пам'ятати про `return`, ви
повертаєте помилку, а центральний обробник розбирається з нею — що
прибирає найпоширенішу помилку обробників `net/http`.

## Маршрутизація

```go
app.Get("/users/:id", h.get)
app.Post("/users", h.create)
app.Get("/search", h.search)
```

Функції, названі за методом, а не патерни, і `:id` для параметрів.
Читайте їх через `Params`, `Query` і `FormValue`:

```go
c.Params("id")           // /users/:id
c.Query("q")             // ?q=go
c.Get("Content-Type")    // a request header
```

`c.Get` читає заголовок *запиту*; `c.Set` пише заголовок *відповіді*.
Ця асиметрія плутає людей.

Групи тримають префікси й middleware разом:

```go
api := app.Group("/api/v1")
api.Use(requireAuth)
api.Get("/users/:id", h.get)
```

Незбіжні маршрути й неправильні методи поводяться розумно без жодних
зусиль:

```
GET    /nope      404 {"error":"Not Found"}
DELETE /users/7   405 {"error":"Method Not Allowed"}
```

## Обробники як методи

Та сама дисципліна залежностей, що й будь-де ще — структура, що тримає
потрібні їй вузькі інтерфейси:

```go
type Handler struct{ users Store }

func (h Handler) get(c fiber.Ctx) error {
    u, err := h.users.ByID(c.Params("id"))
    if err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
    }
    return c.JSON(u)
}
```

```
GET /users/7    200 {"name":"Ada:7","age":36}
GET /users/404  404 {"error":"not found"}
```

`c.JSON` встановлює тип вмісту й кодує. `fiber.Map` — це
`map[string]any` з коротшим ім'ям.

## Прив'язка

```go
var u User
if err := c.Bind().Body(&u); err != nil {
    return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bad body"})
}
```

`Bind().Body` обирає декодер за типом вмісту. Ще є `Bind().Query`,
`Bind().Params` і `Bind().Header`.

```
POST /users {"name":"Bo","age":7}   201 {"name":"Bo","age":7}
POST /users {                       400 {"error":"bad body"}
```

Прив'язка — не валідація. Вона заповнює структуру; вона не перевіряє,
що `Age` правдоподібний, а `Name` непорожній. Валідуйте після
прив'язки — вручну або з бібліотекою валідації.

## Одне місце для помилок

Повернена помилка направляється до `ErrorHandler` застосунку:

```go
app := fiber.New(fiber.Config{
    ErrorHandler: func(c fiber.Ctx, err error) error {
        code := fiber.StatusInternalServerError
        if e, ok := err.(*fiber.Error); ok {
            code = e.Code
        }
        return c.Status(code).JSON(fiber.Map{"error": err.Error()})
    },
})
```

```go
return fiber.NewError(fiber.StatusTeapot, "teapot")
// 418 {"error":"teapot"}
```

Саме тут окупається повернення `error`: одне місце вирішує форму
відповіді, а обробники просто провалюються. Розширте його, щоб
розпізнавати ваші власні доменні помилки — `errors.Is(err, ErrNotFound)`,
що стає `404` — і обробники взагалі перестають конструювати відповіді.

Будьте обережні з тим, що доходить до клієнта. Стандартний обробник
вище відлунює `err.Error()`, що охоче витече внутрішнє повідомлення.
Зіставляйте відомі помилки з безпечним текстом і повертайте загальне
повідомлення для решти.

## Middleware

```go
app.Use(requestid.New())
app.Use(recover.New())
app.Use(compress.New(compress.Config{
    Next: func(c fiber.Ctx) bool {
        return strings.HasSuffix(c.Path(), "/stream")   // never compress SSE
    },
}))
```

Middleware — це `func(c fiber.Ctx) error`, та сама форма, що й
обробник. Викличте `c.Next()`, щоб продовжити; поверніть без нього,
щоб зупинитися.

`recover.New()` перетворює паніку на `500` через ваш обробник помилок:

```
GET /boom   500 {"error":"kaboom"}
```

Більшість вбудованих middleware бере предикат `Next`, щоб пропускати
конкретні маршрути — приклад зі стисненням вище — саме той, що має
значення, якщо ви стрімите, бо буферизація для стиснення зводить нанівець
flush.

## Чого вам коштує fasthttp

Fiber не використовує `net/http`, і це має наслідки:

- **Middleware `net/http` не працює.** Жодного `otelhttp`, жодних
  сторонніх обгорток обробників, без адаптера.
- **`httptest` не працює.** Використовуйте `app.Test(req)`, що бере
  `*http.Request` і повертає `*http.Response` — зручно, і саме це
  використовують приклади тут.
- **Жоден контекст не завершується, коли клієнт відключається.**
  `c.Context()` повертає `context.Context`, який ви зберегли через
  `c.SetContext`, або порожній `context.Background()`, якщо ви нічого
  не зберігали. `c.RequestCtx()` повертає власний об'єкт fasthttp —
  `*fasthttp.RequestCtx`. Він теж задовольняє `context.Context`, але
  його канал `Done` закривається лише тоді, коли сервер завершує
  роботу. У `net/http` `r.Context()` завершується разом із запитом;
  у Fiber додавайте власний дедлайн через `context.WithTimeout`.
- **Значення запиту й відповіді повторно використовуються між запитами.**
  `[]byte` або рядок з `c.Params` чи `c.Body` дійсні лише під час
  обробника. Зберегти щось за межами повернення — у горутині, кеші,
  полі структури — дасть вам дані з непов'язаного запиту пізніше.
  Скопіюйте: `string(append([]byte(nil), b...))`, або просто рядок від
  `c.Params`, якщо ваша версія вже копіює. Це найважче знайти помилку,
  бо вона проявляється лише під конкурентністю.

Останній пункт — справжній компроміс. Fiber швидкий частково тому, що
повторно використовує буфери, а повторно використовувані буфери вимагають дисципліни.

## Стрімінг: SSE — це fasthttp, а не `http.Flusher`

Стаття про [події, надіслані сервером](../08-http-with-net-http/05-server-sent-events.md)
навчає механізму `net/http`: отримати `http.ResponseController` чи
`http.Flusher` і викликати `Flush` після кожного повідомлення. У
Fiber цього інтерфейсу просто немає:

```go
_, isFlusher := any(c.Response().BodyWriter()).(http.Flusher)
// false
```

fasthttp натомість стрімить через body-stream writer. `c.RequestCtx()`
дає вам об'єкт запиту fasthttp, і його метод `SetBodyStreamWriter`
бере `fasthttp.StreamWriter`: функцію, що отримує `*bufio.Writer` і
скидає його. Обидві назви належать fasthttp, а не Fiber, тож
документацію варто шукати саме в fasthttp:

```go
app.Get("/stream", func(c fiber.Ctx) error {
    c.Set("Content-Type", "text/event-stream")
    c.Set("Cache-Control", "no-cache")

    c.RequestCtx().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
        for i := 1; i <= 3; i++ {
            fmt.Fprintf(w, "event: tick\ndata: %d\n\n", i)
            if err := w.Flush(); err != nil {
                return          // client went away
            }
            time.Sleep(10 * time.Millisecond)
        }
    }))
    return nil
})
```

```
ct: text/event-stream
  "event: tick"
  "data: 1"
  "event: tick"
  "data: 2"
  "event: tick"
  "data: 3"
```

Усе інше з базової статті все ще діє: формат передачі, порожній рядок,
що завершує кожне повідомлення, keepalive-коментарі, і скидання
повільних підписників неблокувальним надсиланням.

Три моменти, специфічні для Fiber. Обробник **повертається негайно**
— `SetBodyStreamWriter` реєструє колбек, що виконується після вашого
повернення, тож усе потрібне має бути захоплене до того, а значення,
взяте з `c`, має бути скопійоване з причини вище. Провалений
`w.Flush()` — це ваш сигнал відключення, оскільки скасування
з'єднання fasthttp не поводиться як `r.Context()`. І виключіть
маршрут зі стиснення, інакше буферизація зведе flush нанівець.

## Чи варто його використовувати

Якщо вам потрібна екосистема `net/http` — інструментація OpenTelemetry,
наявні middleware, `httptest` — стандартна бібліотека з роутером
спокійніший вибір, і достатньо швидка майже для всього.

Fiber має сенс, коли його ергономіка підходить команді, або коли
частота запитів справді виправдовує fasthttp. Вирішуйте один раз,
бо змішувати непрактично.

> **З досвіду Python:** Fiber схожий формою на FastAPI — маршрутизація
> в стилі декораторів, прив'язка в типізовану структуру, центральний
> обробник винятків — що працює на нестандартному сервері, так само як
> `uvloop` заміняє цикл asyncio. Правило про повторне використання буферів
> не має аналога в Python, і саме його варто пам'ятати.

## Швидка довідка

| Задача | Форма |
|---|---|
| застосунок | `fiber.New(fiber.Config{...})` |
| обробник | `func(c fiber.Ctx) error` — **значення, не вказівник, у v3** |
| маршрути | `app.Get("/users/:id", h)`, `app.Group("/api")` |
| шлях / query | `c.Params("id")`, `c.Query("q")` |
| заголовок запиту чи відповіді | `c.Get(...)` проти `c.Set(...)` |
| декодувати тіло | `c.Bind().Body(&v)` — потім валідувати |
| відповісти | `c.JSON(v)`, `c.Status(code).JSON(...)` |
| провалитися | `return fiber.NewError(code, msg)` |
| одне місце для помилок | `Config.ErrorHandler` |
| middleware | `app.Use(...)`, з `Next` для пропуску маршрутів |
| тестування | `app.Test(req)` — не `httptest` |
| **повторне використання буферів** | копіюйте все, що зберігаєте поза обробником |

## Джерела

- [Fiber documentation — docs.gofiber.io](https://docs.gofiber.io/)
- [`fiber/v3` — pkg.go.dev/github.com/gofiber/fiber/v3](https://pkg.go.dev/github.com/gofiber/fiber/v3)
- [What's new in v3 — docs.gofiber.io/next/whats_new](https://docs.gofiber.io/next/whats_new)
- [fasthttp — pkg.go.dev/github.com/valyala/fasthttp](https://pkg.go.dev/github.com/valyala/fasthttp)
