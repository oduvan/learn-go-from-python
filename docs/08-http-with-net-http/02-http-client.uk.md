# HTTP-клієнт

Виклик іншого сервісу — це друга половина `net/http`. Пакет проведе вас
дуже далеко, але в його типових налаштуваннях є одна небезпечна
прогалина й одна поведінка, яка дивує геть усіх.

```go
client := &http.Client{Timeout: 10 * time.Second}

req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
resp, err := client.Do(req)
```

## Ніколи не використовуйте `http.DefaultClient`

`http.Get`, `http.Post` і `http.DefaultClient` усі діляться одним
клієнтом **без таймауту**. Сервер, який приймає з'єднання й потім
ніколи не відповідає, підвісить цю горутину назавжди, а під
навантаженням процес вичерпає їх усі. Це найпоширеніший спосіб, яким
падає Go-сервіс.

Створіть власний, один раз, і перевикористовуйте його:

```go
var client = &http.Client{Timeout: 10 * time.Second}
```

`http.Client` безпечний для конкурентного використання й внутрішньо
об'єднує з'єднання в пул. Створення нового клієнта на кожен запит
викидає цей пул і призводить до витоку файлових дескрипторів —
створюйте його під час запуску й передавайте далі.

`Timeout` охоплює весь обмін: з'єднання, надсилання, очікування й
читання тіла. Встановлюйте його довшим за ваш найповільніший легітимний
виклик.

## Будуйте запити з контекстом

`client.Get(url)` годиться для одноразового виклику. Справжні виклики
використовують `NewRequestWithContext`, тож запит помирає разом із тим,
що його спричинило:

```go
req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
if err != nil {
    return err
}
req.Header.Set("Content-Type", "application/json")
req.Header.Set("X-Token", token)

resp, err := client.Do(req)
```

Передавання `r.Context()` із вхідного обробника означає, що
відключення клієнта скасовує й вихідний виклик, замість того щоб дати
йому завершитись, коли результат уже нікому не потрібен.

Тіло — це `io.Reader`, тож підходять `bytes.NewReader`,
`strings.NewReader` і навіть відкритий файл:

```go
body, _ := json.Marshal(User{Name: "Bo"})
req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
// сервер отримує POST із тілом {"name":"Bo","age":0}
```

## 404 — це не помилка

Це одного разу трапляється з кожним. `err` не дорівнює `nil` лише тоді,
коли сам обмін не відбувся — DNS, з'єднання, таймаут. **Будь-яка
відповідь, яку надіслав сервер, хоч би з яким статусом, — це успіх:**

```go
resp, err := client.Get(url + "/404")
fmt.Println("err:", err, "status:", resp.StatusCode)
// output: err: <nil> status: 404
```

Тож кожен виклик потребує двох перевірок:

```go
resp, err := client.Do(req)
if err != nil {
    return fmt.Errorf("calling %s: %w", url, err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
    return fmt.Errorf("%s: unexpected status %s", url, resp.Status)
}
```

## Завжди закривайте тіло, завжди спорожняйте його

`resp.Body` — це відкритий мережевий потік. Якщо не закрити його,
станеться витік з'єднання:

```go
defer resp.Body.Close()
```

Викликайте `defer` одразу після перевірки на помилку — до всього, що
могло б повернутись достроково.

Є ще друге, тихіше правило: з'єднання перевикористовується, лише якщо
тіло прочитане до кінця. На шляху, де ви ігноруєте тіло, спорожніть
його:

```go
io.Copy(io.Discard, resp.Body)
resp.Body.Close()
```

Пропустіть це, і кожен такий запит відкриватиме нове з'єднання — під
навантаженням це проявляється як загадкове вичерпання сокетів, а не
як помилка.

Декодування прямо з потоку спорожняє його само собою:

```go
var u User
if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
    return fmt.Errorf("decoding response: %w", err)
}
```

Використовуйте `io.LimitReader`, якщо іншій стороні не варто повністю
довіряти.

## Розпізнавання таймауту

Помилка клієнта загорнута в `*url.Error`, і `errors.Is` бачить причину
крізь неї:

```go
_, err := tc.Get(slowURL)
fmt.Println(errors.Is(err, context.DeadlineExceeded))   // output: true

var ue *url.Error
fmt.Println(errors.As(err, &ue), ue.Timeout())          // output: true true
```

І власний `Timeout` клієнта, і скасований контекст виявляються як
`context.DeadlineExceeded`, тож одна перевірка покриває обидва
випадки.

## Повторні спроби

Ніщо не повторює запит за вас. Повторюйте лише те, що безпечно
повторювати, — `GET` або запит із ключем ідемпотентності — і робіть
паузи між спробами:

```go
for attempt := range 5 {
    resp, err := client.Do(req)
    if err == nil && resp.StatusCode < 500 {
        return resp, nil
    }
    if resp != nil {
        io.Copy(io.Discard, resp.Body)
        resp.Body.Close()
    }
    time.Sleep(time.Duration(1<<attempt) * 100 * time.Millisecond)
}
// third attempt succeeds: attempt 2 status 200 recovered
```

`1<<attempt` щоразу подвоює час очікування. Важливі три деталі:
закривайте тіло й у невдалій спробі теж, не повторюйте `4xx` (він
провалиться так само знову), і перевіряйте `ctx.Err()` в циклі, щоб
скасований запит припиняв повторні спроби.

Тіло запиту типу `io.Reader` споживається першою ж спробою. Щоб
повторити `POST`, зберігайте байти й щоразу будуйте новий
`bytes.NewReader`.

Реальні системи додають джиттер (jitter), щоб багато клієнтів, які
відновлюються після одного й того самого збою, не повторювали спроби
синхронно.

## Побудова URL

Ніколи не конкатенуйте рядки запиту (query string). `url.Values`
екранує їх за вас:

```go
u, _ := url.Parse(base + "/search")
q := u.Query()
q.Set("q", "go & rust")
q.Set("page", "2")
u.RawQuery = q.Encode()

fmt.Println(u.String())
// output: http://127.0.0.1:8080/search?page=2&q=go+%26+rust
```

`Encode` сортує ключі, тож результат стабільний — це зручно для
кешування й тестів. `&` перетворився на `%26`, і саме в цьому суть.

```go
fmt.Println(url.QueryEscape("a b&c"))   // output: a+b%26c
```

Розбір (parsing) дає вам окремі частини:

```go
pu, _ := url.Parse("https://x.example/a/b?k=v#frag")
fmt.Println(pu.Scheme, pu.Host, pu.Path, pu.Query().Get("k"), pu.Fragment)
// output: https x.example /a/b v frag
```

## Тонке налаштування транспорту

`http.Client` відповідає за політику — таймаути, перенаправлення,
куки. `Transport` відповідає за з'єднання. Типові налаштування
годяться для жменьки хостів; сервісу, що інтенсивно б'є в один
бекенд, зазвичай потрібен більший пул на хост:

```go
transport := http.DefaultTransport.(*http.Transport).Clone()
transport.MaxIdleConnsPerHost = 100
transport.IdleConnTimeout = 90 * time.Second

client := &http.Client{Timeout: 10 * time.Second, Transport: transport}
```

`DefaultMaxIdleConnsPerHost` дорівнює 2, тож без цього кожен
додатковий конкурентний запит до того самого хоста відкриває й одразу
відкидає з'єднання. `Clone` тут важливий — зміна
`http.DefaultTransport` напряму вплине на все в процесі.

> **З досвіду Python:** це `requests`, тільки без зручностей. Немає
> `raise_for_status()`, тож код перевіряєте самі; немає `resp.json()`,
> тож декодуєте самі; і немає сесії за замовчуванням, тож доводиться
> тримати один клієнт, щоб отримати пул з'єднань. Перевага — контекст,
> який дає скасування, аналога якому в `requests` немає.

## Швидка довідка

| Задача | Виклик |
|---|---|
| клієнт | `&http.Client{Timeout: d}` під час запуску — ніколи не типовий |
| запит | `http.NewRequestWithContext(ctx, method, url, body)` |
| надіслати | `client.Do(req)` |
| перевірити | `err != nil` **і** `resp.StatusCode` — 404 не є помилкою |
| закрити | `defer resp.Body.Close()`, одразу після перевірки на помилку |
| перевикористати з'єднання | спорожнити через `io.Copy(io.Discard, resp.Body)` |
| декодувати | `json.NewDecoder(resp.Body).Decode(&v)` |
| виявити таймаут | `errors.Is(err, context.DeadlineExceeded)` |
| повторити спробу | лише ідемпотентні виклики, з паузами, тіло створювати заново |
| рядки запиту | `url.Values` + `q.Encode()` |
| більший пул з'єднань | `DefaultTransport.Clone()`, `MaxIdleConnsPerHost` |

## Джерела

- [`http.Client` — pkg.go.dev/net/http#Client](https://pkg.go.dev/net/http#Client)
- [`http.NewRequestWithContext` — pkg.go.dev/net/http#NewRequestWithContext](https://pkg.go.dev/net/http#NewRequestWithContext)
- [`http.Transport` — pkg.go.dev/net/http#Transport](https://pkg.go.dev/net/http#Transport)
- [`net/url` package reference — pkg.go.dev/net/url](https://pkg.go.dev/net/url)
- [`url.Values` — pkg.go.dev/net/url#Values](https://pkg.go.dev/net/url#Values)
