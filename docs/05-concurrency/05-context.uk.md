# Context

`context.Context` несе **скасування, дедлайни та значення в межах запиту**
через межі API та горутини. Це те, як ви кажете дереву горутин «зупиніться
зараз» — коли запит скасовано, спрацював таймаут або сервер вимикається.

Основна ідея: `Context` виставляє канал `Done()`, який **закривається**,
коли контекст скасовано. Горутини роблять `select` на ньому й виходять.

## Корені: Background та TODO

Кожне дерево контекстів починається з кореня. `context.Background()` —
звичайний (вершина `main`, вхідні запити). `context.TODO()` — заповнювач
для «я ще не протягнув сюди контекст».

```go
ctx := context.Background()
```

Корінь ви ніколи не скасовуєте напряму — натомість ви *похідно створюєте*
дочірній контекст, який можна скасувати.

## WithCancel: явне скасування

`context.WithCancel` повертає дочірній контекст і функцію `cancel`. Виклик
`cancel` закриває канал `Done()` контексту, що бачить кожна горутина, яка за
ним стежить.

```go
ctx, cancel := context.WithCancel(context.Background())
done := make(chan struct{})

go func() {
    <-ctx.Done()                       // блокує, доки не скасовано
    fmt.Println("worker:", ctx.Err())  // worker: context canceled
    close(done)
}()

cancel()                               // запустити скасування
<-done
// output:
// worker: context canceled
```

`ctx.Err()` повідомляє, *чому* він завершився: `context.Canceled` після
`cancel` або `context.DeadlineExceeded` після таймауту. Завжди викликайте
`cancel` (зазвичай `defer cancel()`), щоб звільнити ресурси, навіть якщо
робота завершилася нормально.

## WithTimeout та WithDeadline

`WithTimeout` скасовує автоматично через тривалість; `WithDeadline` — у
фіксований час. Поєднайте з `select`, щоб обмежити будь-яку блокувальну
операцію:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
defer cancel()

select {
case <-time.After(time.Second):
    fmt.Println("work finished")
case <-ctx.Done():
    fmt.Println("gave up:", ctx.Err())   // gave up: context deadline exceeded
}
```

Таймаут 10 мс спрацьовує задовго до роботи на 1 с, тож `ctx.Done()` перемагає,
а `ctx.Err()` — це `context.DeadlineExceeded`.

## Поширення: передавайте вниз, не зберігайте

Домовленості тверді, і їх варто дотримуватися точно:

- **Передавайте `ctx` першим параметром** з ім'ям `ctx`:
  `func Fetch(ctx context.Context, url string) (...)`.
- **Не зберігайте `Context` у структурі** — протягуйте його через виклики.
- Похідно створюйте дочірні контексти, коли робота розгалужується;
  скасування батька скасовує всіх його дітей.
- Функція, що поважає контекст, **робить `select` на `ctx.Done()`** у своїх
  блокувальних циклах і повертає `ctx.Err()`, коли той спрацьовує.

```go
func work(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()      // оперативно зупинитися при скасуванні
        default:
            // ... одна одиниця роботи ...
            return nil
        }
    }
}
```

## Значення в межах запиту (помірно)

`context.WithValue` приєднує пару ключ/значення, що мандрує з контекстом, —
призначене для метаданих у межах запиту на кшталт ID запиту, **а не** для
передавання необов'язкових аргументів функції. Надмірне вживання приховує
залежності, тож надавайте перевагу явним параметрам, а до значень вдавайтеся
лише для наскрізних даних.

Використовуйте **неекспортований власний тип ключа**, а не звичайний рядок,
щоб ключі з різних пакетів не стикалися:

```go
type ctxKey string

ctx := context.WithValue(context.Background(), ctxKey("reqID"), "abc123")

fmt.Println(ctx.Value(ctxKey("reqID")))   // output: abc123
fmt.Println(ctx.Value(ctxKey("missing"))) // output: <nil>
```

`Value` повертає `any`, тож для відсутнього ключа це `nil`, і зазвичай ви
робите твердження типу результату назад до його конкретного типу перед
використанням.

## Швидка довідка

| Виклик | Значення |
|---|---|
| `context.Background()` | кореневий контекст |
| `context.TODO()` | корінь-заповнювач |
| `ctx, cancel := WithCancel(parent)` | ручне скасування |
| `WithTimeout(parent, d)` | авто-скасування через тривалість |
| `WithDeadline(parent, t)` | авто-скасування у час |
| `<-ctx.Done()` | закривається при скасуванні |
| `ctx.Err()` | `Canceled` чи `DeadlineExceeded` |
| `WithValue(parent, k, v)` | дані в межах запиту (помірно) |
| перший аргумент `ctx context.Context` | домовленість |

## Джерела

- [context — pkg.go.dev/context](https://pkg.go.dev/context)
- [Go blog: context and structs — go.dev/blog/context-and-structs](https://go.dev/blog/context-and-structs)
- [Go blog: pipelines and cancellation — go.dev/blog/pipelines](https://go.dev/blog/pipelines)
- [Go Concurrency Patterns: Context — go.dev/blog/context](https://go.dev/blog/context)
