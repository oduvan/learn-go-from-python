# HTTP-сервер

`net/http` — це вебсервер, придатний для продакшену, прямо в стандартній
бібліотеці. Щоб обслуговувати справжній трафік, фреймворк не потрібен, а
кожен фреймворк, який ви зустрінете, побудований на інтерфейсах з цієї
статті.

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "hello")
})
http.ListenAndServe(":8080", mux)
```

## Обробники

Усе, що обслуговує запит, задовольняє один інтерфейс:

```go
type Handler interface {
    ServeHTTP(w http.ResponseWriter, r *http.Request)
}
```

Написати структуру з цим методом — один із варіантів. Зазвичай пишуть
функцію й дозволяють `http.HandlerFunc` адаптувати її — це тип-адаптер,
чий `ServeHTTP` просто викликає функцію:

```go
func hello(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "hello")
}

var h http.Handler = http.HandlerFunc(hello)
```

`mux.HandleFunc` робить це перетворення за вас. `w` — це `io.Writer`,
тож усе зі статті про
[читачі та письменники](../07-operating-system/02-readers-and-writers.md)
застосовується й тут: `fmt.Fprintf`, `io.Copy`, `json.NewEncoder(w)`.

Коли обробнику потрібні залежності, зробіть його методом структури.
Саме так упроваджують базу даних чи логер, не вдаючись до глобальних
змінних:

```go
type API struct{ store Store }

func (a *API) getUser(w http.ResponseWriter, r *http.Request) {
    // a.store доступний тут
}

mux.HandleFunc("GET /users/{id}", a.getUser)
```

## Маршрутизація за методом і шаблонами з підстановкою

Патерни `ServeMux` можуть містити необов'язковий метод і захоплювати
сегменти шляху:

```go
mux.HandleFunc("GET /hello", helloHandler)
mux.HandleFunc("GET /users/{id}", getUser)
mux.HandleFunc("POST /users", createUser)
```

```go
func getUser(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "user %s", r.PathValue("id"))
}
// GET /users/42  →  200 "user 42"
```

`r.PathValue("id")` читає захоплений сегмент. `{rest...}` наприкінці
патерну збігається з усім, що залишилось.

Патерн, що закінчується на `/`, збігається з піддеревом; без слеша —
лише точно. Перемагає найконкретніший патерн, тож `/users/{id}` бере
гору над `/users/`.

Назвавши метод, ви безкоштовно отримуєте правильну поведінку:

```
GET  /users   →  405 Method Not Allowed
GET  /nope    →  404 page not found
```

`405` без жодних зусиль з вашого боку — ось причина писати метод прямо в
патерні, а не перевіряти `r.Method` усередині обробника.

## Читання запиту

```go
q := r.URL.Query().Get("q")          // ?q=go
id := r.PathValue("id")              // /users/{id}
ct := r.Header.Get("Content-Type")   // заголовок, регістр не важливий
```

`Query()` парсить дані при кожному виклику, тож зберігайте результат,
якщо потрібно кілька значень. Відсутній ключ дає `""` — щоб відрізнити
відсутність від порожнього значення, використовуйте `r.URL.Query().Has("k")`.

Декодуйте тіло JSON прямо з потоку, а не читайте його повністю заздалегідь:

```go
func createUser(w http.ResponseWriter, r *http.Request) {
    var u User
    if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
        http.Error(w, "bad json", http.StatusBadRequest)
        return
    }
    // ...
}
```

Зверніть увагу на `return` після `http.Error`. Запис відповіді не
зупиняє обробник, і забути про `return` — найпоширеніша помилка в такому
коді: ви зрештою пишете другу відповідь поверх першої, з помилкою.

Сервер сам закриває `r.Body`; тіло відповіді *клієнта* — ваш обов'язок,
про це [наступна стаття](02-http-client.md).

Обмежте, скільки читатимете, інакше велике завантаження стане аварією
через нестачу пам'яті:

```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20)   // 1 МіБ
```

## Запис відповіді у правильному порядку

Три кроки, і вони мають відбуватись саме в такій послідовності:

```go
w.Header().Set("Content-Type", "application/json")   // 1. заголовки
w.WriteHeader(http.StatusCreated)                    // 2. статус
json.NewEncoder(w).Encode(u)                         // 3. тіло
```

Перший `Write` надсилає `200` і заморожує заголовки. Усе, що
встановлюється після цього, ігнорується, а сервер лише пише скаргу в
лог, а не падає:

```go
w.Write([]byte("body first"))
w.WriteHeader(http.StatusTeapot)   // запізно
w.Header().Set("X-Late", "yes")    // запізно
// log: http: superfluous response.WriteHeader call from ...
// клієнт усе одно отримує 200
```

Для `200` просто пропустіть `WriteHeader`. Використовуйте іменовані
константи — `http.StatusCreated`, `http.StatusBadRequest` — а не голі
числа.

Якщо `Content-Type` не встановлено, сервер визначає його за першими
байтами, тому звичайний текст перетворюється на
`text/plain; charset=utf-8`. Для будь-чого структурованого встановлюйте
його самостійно.

`http.Error(w, msg, code)` одним викликом пише текстову помилку й
статус.

## Налаштуйте сервер, не покладайтесь на типовий

`http.ListenAndServe` годиться для прикладу. Справжній код будує
`http.Server`, бо в типових налаштуваннях **взагалі немає таймаутів** —
повільний клієнт може тримати з'єднання відкритим нескінченно:

```go
srv := &http.Server{
    Addr:              ":8080",
    Handler:           mux,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       60 * time.Second,
}

if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
    return err
}
```

`ListenAndServe` блокує, поки сервер не зупиниться, і повертає
`http.ErrServerClosed` після коректної зупинки — це нормальний
результат, а не збій, тому й `errors.Is`.

Уникайте `http.HandleFunc` і `http.ListenAndServe(addr, nil)`. Вони
використовують типовий мультиплексор пакетного рівня — глобальний
змінний стан, у якому може реєструватись будь-який імпортований пакет.
Створюйте власний `ServeMux`.

## Зупинка

`Shutdown` припиняє приймати з'єднання й чекає на запити, що вже
виконуються, у межах контексту — той самий підхід, що й у статті про
[сигнали та коректну зупинку](../07-operating-system/06-signals-and-graceful-shutdown.md).
Код записує помилки через `slog` — стандартний структурований логер Go,
про який пізніше розповідає стаття
[Структуроване логування через slog](../12-observability/01-structured-logging-with-slog.md):

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

go func() {
    if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        slog.Error("listen", "error", err)
    }
}()

<-ctx.Done()
stop()

shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()
return srv.Shutdown(shutdownCtx)
```

`Shutdown` не чекає на перехоплені (hijacked) з'єднання чи довгоживучі
потокові з'єднання. Їм потрібне власне скасування.

## Кожен запит несе контекст

`r.Context()` скасовується, коли клієнт від'єднується, тож передавайте
його в усе повільне — робота зупиниться, коли слухати вже нема кому:

```go
func (a *API) getUser(w http.ResponseWriter, r *http.Request) {
    u, err := a.store.User(r.Context(), r.PathValue("id"))
    // ...
}
```

## Статичні файли

```go
mux.Handle("GET /static/", http.StripPrefix("/static/",
    http.FileServer(http.Dir("assets"))))
```

`StripPrefix` потрібен, бо `FileServer` розв'язує весь шлях запиту.
Роздача з `embed.FS` замість диска використовує `http.FS` і лишає
бінарник самодостатнім.

> **З досвіду Python:** `ServeMux` — це приблизно маршрутизація Flask,
> тільки з набагато меншою кількістю можливостей, але тут немає шару
> WSGI й окремого сервера для розгортання позаду — це *і є* продакшн-
> сервер. Обробник отримує writer як параметр, а не повертає відповідь,
> тож "повернутися одразу після запису" — це дисципліна, яку система
> типів не примусить дотримуватись.

## Швидка довідка

| Задача | Виклик |
|---|---|
| маршрутизатор | `http.NewServeMux()` — не типовий мультиплексор |
| зареєструвати | `mux.HandleFunc("GET /users/{id}", fn)` |
| сегмент шляху | `r.PathValue("id")` |
| запит | `r.URL.Query().Get("q")`, `.Has("q")` |
| JSON на вхід | `json.NewDecoder(r.Body).Decode(&v)` |
| обмежити тіло | `http.MaxBytesReader(w, r.Body, n)` |
| заголовки, потім статус, потім тіло | `w.Header().Set` → `w.WriteHeader` → `w.Write` |
| відповідь-помилка | `http.Error(w, msg, code)` — потім **`return`** |
| таймаути | будуйте `http.Server`, ніколи не типові налаштування |
| коректна зупинка | `srv.Shutdown(ctx)`; очікуйте `http.ErrServerClosed` |
| скасування на рівні запиту | `r.Context()` |
| статичні файли | `http.StripPrefix` + `http.FileServer` |

## Джерела

- [`net/http` package reference — pkg.go.dev/net/http](https://pkg.go.dev/net/http)
- [`http.ServeMux` — pkg.go.dev/net/http#ServeMux](https://pkg.go.dev/net/http#ServeMux)
- [`http.Server` — pkg.go.dev/net/http#Server](https://pkg.go.dev/net/http#Server)
- [`http.Server.Shutdown` — pkg.go.dev/net/http#Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)
- [Routing enhancements — go.dev/blog/routing-enhancements](https://go.dev/blog/routing-enhancements)
