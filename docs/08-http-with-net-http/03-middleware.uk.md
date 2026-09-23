# Проміжні обробники

Проміжний обробник (middleware) — це одна ідея: функція, що приймає
обробник і повертає обробник. Логування, відновлення після паніки,
автентифікація й ідентифікатори запитів мають одну й ту саму форму, і
ця форма не специфічна для HTTP.

```go
func logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Printf("-> %s %s\n", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}
```

Це працює, бо `http.Handler` — інтерфейс з одним методом, тож обгортку
не відрізнити від того, що вона обгортає. Це патерн "декоратор",
побудований на [замиканнях](../02-language-basics/06-functions.md) та
[інтерфейсах](../03-object-oriented-go/02-interfaces.md).

## Складання ланцюжка

Застосувати кілька обробників вручну — незручно, вкладення виходить
громіздким:

```go
h := logging(recoverMW(auth(mux)))
```

Невеликий допоміжний код читається краще й робить порядок явним:

```go
func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
    for i := len(mws) - 1; i >= 0; i-- {
        h = mws[i](h)
    }
    return h
}

h := chain(mux, logging, recoverMW, auth)
```

Цикл виконується у зворотному порядку, тож **перший аргумент — це
найзовнішніша обгортка**. Запит проходить через `logging`, потім
`recoverMW`, потім `auth`, і лише тоді дістається мультиплексора;
відповідь повертається у зворотному напрямку.

Порядок — питання коректності, а не смаку:

- `logging` — найзовніший, тож бачить кожен запит, включно з
  відхиленими.
- `recoverMW` — зовні всього, що може панікувати, але всередині
  `logging`, щоб відновлений `500` усе одно потрапив у лог.
- `auth` — останній, тож неавтентифіковані запити ніколи не торкаються
  ваших обробників.

## Відновлення після паніки

Без цього паніка в обробнику вбила б процес. У `net/http` є власне
відновлення, але воно закриває з'єднання без відповіді — власне
відновлення дає клієнтові справжній `500` і записує паніку в лог через
[`slog`](../12-observability/01-structured-logging-with-slog.md):

```go
func recoverMW(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                slog.Error("panic", "err", rec, "stack", string(debug.Stack()))
                http.Error(w, "internal error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

З `chain(mux, logging, recoverMW)` та обробником, що панікує з
`"kaboom"`, запит на `/boom` виводить на сервері таке:

```
-> GET /boom
2026/09/23 10:15:02 ERROR panic err=kaboom stack="goroutine 38 [running]:\n..."
```

Клієнт отримує `500` із тілом `internal error`.

Два обмеження. Це покриває лише паніки у *власній горутині запиту* —
усе, що обробник породжує сам, потребує власного `recover`, як
пояснює стаття про
[довгоживучі горутини](../05-concurrency/08-long-running-goroutines.md).
І якщо обробник уже записав відповідь, `500` приходить надто пізно,
щоб змінити статус.

## Передавання значень вниз по ланцюжку

Проміжний обробник спілкується з обробниками через контекст запиту.
Ключ має бути **неекспортованого типу**, щоб жоден інший пакет не міг
випадково зіткнутися з ним:

```go
type ctxKey int

const userKey ctxKey = 0

func auth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tok := r.Header.Get("X-Token")
        if tok == "" {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return   // не викликати next
        }
        ctx := context.WithValue(r.Context(), userKey, "user:"+tok)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

```go
func me(w http.ResponseWriter, r *http.Request) {
    u, _ := r.Context().Value(userKey).(string)
    fmt.Fprintf(w, "hello %s", u)
}
// output: hello user:abc
```

Три деталі. Запит незмінний, тож новий контекст приєднують через
`r.WithContext(ctx)` і передають далі *саме його*. Відхилення означає
записати відповідь і повернутись **не викликаючи `next`**. І `Value`
повертає `any`, тож читання цього значення — це твердження про тип
(type assertion) — використовуйте форму comma-ok, оскільки відсутнє
значення дає `nil`. Саме форма з двома значеннями запобігає краху: якщо
в контексті немає користувача, `u` — це просто `""`, навіть попри те,
що `me` відкидає `ok`.

Тримайте у значеннях контексту лише факти в межах запиту: особу
викликача, ідентифікатор запиту, span трейсу. Справжні залежності
повинні бути полями структури обробника, де компілятор може їх
перевірити.

## Захоплення коду статусу

`http.ResponseWriter` не скаже вам, який статус був записаний, що
незручно для логера. Обгорніть його, вбудувавши інтерфейс, щоб
успадкувати всі методи й перевизначити лише один:

```go
type statusRecorder struct {
    http.ResponseWriter
    status int
}

func (s *statusRecorder) WriteHeader(code int) {
    s.status = code
    s.ResponseWriter.WriteHeader(code)
}
```

```go
rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
next.ServeHTTP(rec, r)
// rec.status тепер містить те, що записав обробник
```

Типове значення `200` важливе: обробник, який лише викликає `Write`,
ніколи не викликає `WriteHeader`, і статус залишається `200`.

Це [вбудовування](../02-language-basics/10-structs.md), що виконує
реальну роботу. Одна засторога — обгортання приховує необов'язкові
інтерфейси, які реалізує справжній writer, наприклад `http.Flusher`.
Якщо щось далі за ланцюжком стрімить дані, обгортка мусить прокидати
й ці інтерфейси теж — саме на це наштовхується стаття про
[події від сервера](05-server-sent-events.md).

## Проміжні обробники для того, що не є HTTP

Патерн стосується *форми*, а не протоколу. Усе, що має єдину сигнатуру
"обробника", можна обгорнути так само:

```go
type Job func(ctx context.Context) error

func timed(name string, next Job) Job {
    return func(ctx context.Context) error {
        start := time.Now()
        err := next(ctx)
        slog.Info("job", "name", name, "ms", time.Since(start).Milliseconds())
        return err
    }
}
```

Один і той самий декоратор дає заплановане завдання, споживач черги чи
обробник виклику інструменту з логуванням, метриками й відновленням,
написаними лише раз.

> **З досвіду Python:** це декоратор, але явний — тут немає `@`, ви
> самі складаєте функції докупи, тому порядок видно прямо в коді.
> Найближчий загальний аналог — проміжний обробник WSGI/ASGI:
> застосунок, що обгортає інший застосунок.

## Швидка довідка

| Аспект | Форма |
|---|---|
| форма | `func(http.Handler) http.Handler` |
| адаптувати функцію | `http.HandlerFunc(fn)` |
| скласти докупи | допоміжна функція `chain`; перший аргумент — найзовніший |
| відновитись | `defer` + `recover` усередині обгортки, логувати стек |
| зупинити ланцюжок | записати відповідь і `return` без виклику `next` |
| передати значення вниз | `r.WithContext(context.WithValue(...))`, неекспортований тип ключа |
| прочитати його | `v, ok := r.Context().Value(key).(T)` |
| спостерігати за статусом | вбудувати `http.ResponseWriter`, перевизначити `WriteHeader`, типово 200 |
| обробники поза HTTP | та сама форма обгортки над власним типом функції |

## Джерела

- [`http.Handler` — pkg.go.dev/net/http#Handler](https://pkg.go.dev/net/http#Handler)
- [`http.HandlerFunc` — pkg.go.dev/net/http#HandlerFunc](https://pkg.go.dev/net/http#HandlerFunc)
- [`context.WithValue` — pkg.go.dev/context#WithValue](https://pkg.go.dev/context#WithValue)
- [`http.Request.WithContext` — pkg.go.dev/net/http#Request.WithContext](https://pkg.go.dev/net/http#Request.WithContext)
- [Go blog: context — go.dev/blog/context](https://go.dev/blog/context)
