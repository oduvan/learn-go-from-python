# Тестування HTTP

`net/http/httptest` охоплює обидва напрямки: виклик обробника без
мережі і піднімання справжнього сервера, з яким може говорити ваш
клієнт.

```go
rec := httptest.NewRecorder()
req := httptest.NewRequest(http.MethodGet, "/users/1", nil)

api.Handler().ServeHTTP(rec, req)
```

## Тестування обробника: `NewRecorder`

Обробник — це просто функція, що приймає писача (writer) і запит, тож
ви можете викликати її напряму. `httptest.NewRecorder` — це
`ResponseWriter`, що запам'ятовує все записане, а `httptest.NewRequest`
будує запит, не парсячи URL через мережу:

```go
api := API{Users: stubUsers{ByIDFn: func(_ context.Context, id string) (User, error) {
    return User{Name: "Ada"}, nil
}}}

rec := httptest.NewRecorder()
api.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/users/1", nil))

// status=200 ct="application/json" body="{\"name\":\"Ada\"}"
```

Без порту, без слухача (listener), без прибирання. Читайте результат
із `rec.Code`, `rec.Body` та `rec.Result().Header`.

Викликайте `ServeHTTP` на **mux**, а не на самій функції-обробнику,
коли хочете перевірити ще й маршрутизацію та збіг методу — інакше
неправильний шлях ніколи не буде задіяний.

`httptest.NewRequest` панікує на неправильно сформованій цілі, замість
того щоб повертати помилку — саме це й потрібно в тесті.

Шляхи помилок — це те місце, де заглушка себе виправдовує:

```go
api := API{Users: stubUsers{ByIDFn: func(context.Context, string) (User, error) {
    return User{}, ErrNotFound
}}}
// status=404 body="not found"
```

## Тестування клієнта: `NewServer`

Для протилежного напрямку запустіть справжній HTTP-сервер на
справжньому loopback-порту. Клієнт під тестом тоді працює через
реальне з'єднання:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(User{Name: "Bo"})
}))
t.Cleanup(srv.Close)

c := Client{BaseURL: srv.URL, HTTP: srv.Client()}
```

`srv.URL` — це призначена адреса — порт обирає ОС, тож паралельні
тести ніколи не конфліктують. `srv.Client()` повертає клієнт,
налаштований саме під цей сервер, що важливо для `NewTLSServer`, де він
несе тестовий сертифікат.

`t.Cleanup(srv.Close)`, а не `defer srv.Close()`: працює з `t.Parallel`
і всередині хелперів.

Саме тому тип клієнта має приймати свою базову URL-адресу й свій
`*http.Client` як поля. Клієнт, що зашиває URL прямо в код, так
протестувати взагалі неможливо.

## Скриптинг відповідей

Фейковий сервер — це звичайний обробник, тож може поводитись як завгодно
потрібно тесту. Підрахунок викликів через atomic робить логіку повторів
(retry) тестованою:

```go
var calls atomic.Int32

srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if calls.Add(1) < 3 {
        w.WriteHeader(http.StatusServiceUnavailable)
        return
    }
    json.NewEncoder(w).Encode(User{Name: "Bo"})
}))
```

```go
u, err := c.Fetch(context.Background(), "1")
// calls=3 user={Name:Bo} err=<nil>
```

Це перевіряє саме цікаву річ: клієнт двічі повторив спробу й досяг
успіху на третій. Використовуйте atomic, а не простий `int` — обробник
виконується на горутині сервера, тож звичайний лічильник — це гонитва,
яку детектор позначить.

Та сама схема покриває `500`, який ніколи не відновлюється, таймаут
(`time.Sleep` довший за дедлайн клієнта), спотворений JSON і
з'єднання, закрите посеред тіла відповіді.

## Перевірка того, що надіслав клієнт

Захопіть це в обробнику:

```go
var gotPath, gotToken string

srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    gotPath = r.URL.Path
    gotToken = r.Header.Get("X-Token")
    json.NewEncoder(w).Encode(User{Name: "x"})
}))
// path="/users/7" token="abc"
```

Так ви перевіряєте, що заголовки автентифікації, параметри запиту й
тіла запитів формуються правильно — не заглядаючи у внутрішню будову
клієнта.

Якщо клієнта викликають конкурентно, захистіть ці змінні або
використовуйте канали; присвоєння з горутини обробника й читання з
горутини тесту — це гонитва.

## Що обирати

| Тестуємо | Використовуємо |
|---|---|
| обробник або middleware | `NewRecorder` + `NewRequest` |
| клієнт, або поведінку повторів і таймаутів | `NewServer` |
| поведінку, специфічну для TLS | `NewTLSServer` + `srv.Client()` |
| увесь стек наскрізь | `NewServer`, що обгортає ваш справжній роутер |

`NewRecorder` швидший і простіший, тож надавайте йому перевагу для
логіки обробника. `NewServer` — чесний вибір щоразу, коли те, що ви
тестуєте, зачіпає транспорт: таймаути, повтори, повторне використання
з'єднання, стрімінг.

## Чого рекордер не робить

`httptest.NewRecorder` — це заглушка, а не справжній сервер. Він не
виконує стан-машину HTTP, тож не скаже вам про обробник, що записує
заголовки після тіла відповіді, а його `Flush` не робить нічого
корисного для стрімінгу. Для
[подій від сервера](../08-http-with-net-http/05-server-sent-events.md)
чи будь-чого іншого, де скидання буфера важливе, використовуйте
`NewServer` і читайте тіло відповіді як потік.

> **З досвіду Python:** `NewRecorder` — це тестовий клієнт, що обходить
> мережу, а `NewServer` — це `responses`/`httpretty` навиворіт: замість
> перехоплення викликів клієнта ви запускаєте справжній сервер і
> спрямовуєте клієнта на нього. Менше несподіванок, бо нічого не
> пропатчено на льоту (monkey-patched).

## Швидка довідка

| Задача | Виклик |
|---|---|
| фейковий writer для відповіді | `httptest.NewRecorder()` |
| побудувати запит | `httptest.NewRequest(method, target, body)` |
| викликати обробник | `h.ServeHTTP(rec, req)` — для маршрутизації беріть mux |
| прочитати результат | `rec.Code`, `rec.Body`, `rec.Result().Header` |
| справжній сервер на вільному порту | `httptest.NewServer(handler)` |
| його адреса / клієнт | `srv.URL`, `srv.Client()` |
| вимкнути | `t.Cleanup(srv.Close)` |
| рахувати виклики | `atomic.Int32` в обробнику |
| перевірити, що надіслано | захопити зсередини обробника |
| стрімінг чи TLS | `NewServer` / `NewTLSServer`, не рекордер |

## Джерела

- [`net/http/httptest` — pkg.go.dev/net/http/httptest](https://pkg.go.dev/net/http/httptest)
- [`httptest.NewRecorder` — pkg.go.dev/net/http/httptest#NewRecorder](https://pkg.go.dev/net/http/httptest#NewRecorder)
- [`httptest.NewServer` — pkg.go.dev/net/http/httptest#NewServer](https://pkg.go.dev/net/http/httptest#NewServer)
- [`httptest.NewRequest` — pkg.go.dev/net/http/httptest#NewRequest](https://pkg.go.dev/net/http/httptest#NewRequest)
- [`sync/atomic` — pkg.go.dev/sync/atomic](https://pkg.go.dev/sync/atomic)
