# Події від сервера (SSE)

Іноді сервер мусить повідомити клієнта про щось раніше, ніж клієнт
здогадається запитати: прогрес завдання, живий лог, сповіщення. Події
від сервера (server-sent events) — найпростіший спосіб це зробити:
звичайна HTTP-відповідь, яка лишається відкритою й продовжує писати.

```go
w.Header().Set("Content-Type", "text/event-stream")
rc := http.NewResponseController(w)

fmt.Fprintf(w, "event: tick\ndata: %d\n\n", i)
rc.Flush()
```

Жодного нового протоколу, жодного рукостискання для оновлення
з'єднання (upgrade handshake), а клієнт — це кілька рядків JavaScript.
На відміну від WebSocket, це односпрямований канал — тільки від
сервера до клієнта, — а це все, що потрібно для більшості звітів про
прогрес.

## Формат на дроті

Звичайний текст. Кожне повідомлення — це набір рядків
`поле: значення`, що завершується **порожнім рядком**:

```
event: tick
data: 1

event: done
data: bye

```

| Поле | Значення |
|---|---|
| `data:` | корисне навантаження; повторюйте рядок для багаторядкового вмісту |
| `event:` | ім'я, яке клієнт може слухати; типово `message` |
| `id:` | дозволяє браузеру відновитись через `Last-Event-ID` після розриву |
| `retry:` | затримка перепідключення в мілісекундах |

Саме подвійний символ нового рядка завершує повідомлення. Надішліть
`data: x\n` з одним символом нового рядка, і клієнт сидітиме й
чекатиме на решту — це найпоширеніша помилка SSE.

Тримайте `data:` в одному рядку, кодуючи структуровані дані як JSON.

## Обробник

```go
func sseHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    rc := http.NewResponseController(w)

    for i := 1; i <= 3; i++ {
        select {
        case <-r.Context().Done():
            return
        default:
        }

        fmt.Fprintf(w, "event: tick\ndata: %d\n\n", i)
        if err := rc.Flush(); err != nil {
            return
        }
        time.Sleep(20 * time.Millisecond)
    }
}
```

Споживання цього показує форму:

```
content-type: text/event-stream
  "event: tick"
  "data: 1"
  "event: tick"
  "data: 2"
  "event: tick"
  "data: 3"
```

## Скидання буфера (flush) не є необов'язковим

Відповіді буферизуються. Без скидання (flush) нічого не дійде до
клієнта, поки обробник не завершиться, — а для потоку це ніколи, тож
клієнт висить з відкритим з'єднанням і без жодних даних.

`http.NewResponseController(w)` — це сучасний спосіб дістатись до
базового writer'а. Старіший підхід — це твердження про тип (type
assertion) до `http.Flusher`:

```go
flusher, ok := w.(http.Flusher)   // старіша форма
```

Надавайте перевагу response controller: він працює крізь проміжні
обробники, що обгортають `ResponseWriter`, а голе твердження про тип —
ні. Це та сама пастка, про яку йшлося в статті про
[проміжні обробники](03-middleware.md) — обгортка, що вбудовує
`http.ResponseWriter` без прокидання `Flush`, тихо ламає стрімінг.

Якщо `Flush` повертає помилку, клієнт зник. Сприймайте це як сигнал
зупинитись.

## Помічаємо, що клієнт пішов

Вкладка браузера закривається, і ніщо не повідомляє про це вашому
обробнику напряму — окрім контексту запиту, який скасовується:

```go
select {
case <-r.Context().Done():
    return
default:
}
```

```
  got "event: tick" then hanging up
server: client went away: context canceled
```

Без цієї перевірки цикл виконається до кінця, роблячи роботу для
нікого й утримуючи горутину. Для довгоживучого потоку зробіть
контекст справжнім варіантом (case) у `select`, а не опитуванням
через `default`:

```go
for {
    select {
    case <-r.Context().Done():
        return
    case msg := <-events:
        fmt.Fprintf(w, "data: %s\n\n", msg)
        if err := rc.Flush(); err != nil {
            return
        }
    case <-time.After(30 * time.Second):
        fmt.Fprint(w, ": keepalive\n\n")   // рядок-коментар
        rc.Flush()
    }
}
```

Рядок, що починається з `:`, — це коментар, який клієнт ігнорує.
Періодичне надсилання такого рядка не дає проксі закрити неактивне
з'єднання — а вони зазвичай роблять це через 30–60 секунд.

## Таймаути не повинні застосовуватись

`http.Server` із встановленим `WriteTimeout` обірве ваш потік рівно в
цю мить. На сервері, що стрімить дані, або лишіть це поле
невстановленим, або продовжуйте дедлайн для кожного запиту:

```go
rc := http.NewResponseController(w)
rc.SetWriteDeadline(time.Time{})   // без дедлайну для цієї відповіді
```

Те саме стосується проміжного обробника стиснення — буферизація
заради стиснення зводить нанівець скидання буфера, тож виключайте з
нього маршрути, що стрімлять дані.

## Ніколи не блокуйтесь на повільному підписнику

Кожен під'єднаний клієнт отримує горутину й зазвичай канал. Якщо один
клієнт перестає читати, блокувальне надсилання зупиняє горутину, що
його годує, — а якщо один виробник розсилає дані багатьом клієнтам,
один повільний читач зупиняє всіх.

Надсилайте без блокування й змиріться з утратою:

```go
select {
case ch <- msg:
default:
    slog.Warn("dropping event for slow subscriber")
}
```

```go
full := make(chan string, 1)
fmt.Println(dropSlow(full, "a"), dropSlow(full, "b"))
// output: true false
```

Перше надсилання вміщується в буфер; друге знаходить його заповненим
і здається, замість того щоб чекати. Відкинути подію для одного
повільного клієнта майже завжди краще, ніж деградувати всіх, — а якщо
клієнту потрібна повна історія, він має отримувати стан окремим
запитом, а не читати потік.

## Коли це не підходить

- **Двостороннє спілкування** — використовуйте WebSocket.
- **Більше, ніж жменька конкурентних потоків на репліку** — кожен
  тримає з'єднання й горутину протягом усього свого життя.
- **Кілька реплік** — клієнт під'єднаний до одного інстансу, тож
  подія, породжена на іншому, ніколи до нього не дійде. Для цього
  потрібен спільний шар pub/sub позаду обробників, а це вже турбота
  сторонніх рішень.

> **З досвіду Python:** це стрімінгова відповідь із `yield`, тільки
> написана як явний цикл над writer'ом, а `Flush` тут — те, що ви
> викликаєте самі, а не те, що фреймворк визначає за вас.

## Швидка довідка

| Аспект | Дія |
|---|---|
| тип вмісту | `text/event-stream`, плюс `Cache-Control: no-cache` |
| формат повідомлення | `data: ...\n\n` — завершує його **порожній рядок** |
| іменовані події | `event: name` перед рядком `data:` |
| виштовхнути дані | `http.NewResponseController(w).Flush()` на кожне повідомлення |
| клієнт зник | `<-r.Context().Done()` або помилка `Flush` |
| неактивні проксі | коментар `: keepalive\n\n` кожні ~30с |
| таймаут запису на сервері | зняти його, або `rc.SetWriteDeadline(time.Time{})` |
| повільний клієнт | неблокувальне надсилання через `select` із `default`, що відкидає |
| проміжний обробник | мусить прокидати `Flush` і не стискати цей маршрут |

## Джерела

- [`http.NewResponseController` — pkg.go.dev/net/http#NewResponseController](https://pkg.go.dev/net/http#NewResponseController)
- [`http.Flusher` — pkg.go.dev/net/http#Flusher](https://pkg.go.dev/net/http#Flusher)
- [`http.Request.Context` — pkg.go.dev/net/http#Request.Context](https://pkg.go.dev/net/http#Request.Context)
- [Server-sent events — developer.mozilla.org/en-US/docs/Web/API/Server-sent_events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)
- [WHATWG HTML: server-sent events — html.spec.whatwg.org/multipage/server-sent-events.html](https://html.spec.whatwg.org/multipage/server-sent-events.html)
