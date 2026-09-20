# Довгоживучі горутини

Більшість горутин у попередніх статтях короткі: запустилася, зробила одну
справу, завершилася до того, як повернеться `Wait`. Але в сервера є й
горутини, що живуть увесь час роботи процесу, — опитувач (poller),
споживач черги, фоновий записувач. Їм потрібні три речі, яких не
потребують короткі горутини: власна обробка паніки, спосіб зупинитися і
спосіб, яким завершення роботи може їх дочекатися.

```go
go func() {
    panic("worker exploded")
}()

time.Sleep(time.Second)
fmt.Println("never reached")
// panic: worker exploded
// exit status 2
```

## Паніка не може перетнути межу горутини

`recover` працює лише у функції, відкладеній (deferred) тією ж горутиною,
яка панікує. Горутина, що запустила роботу, — це інша горутина, тож її
`defer`/`recover` ніколи не побачить цю паніку, — а невідновлена паніка
валить увесь процес:

```go
defer func() {
    if r := recover(); r != nil {
        fmt.Println("caller recovered:", r)   // never printed
    }
}()

go func() {
    panic("worker exploded")
}()
```

Це прямо випливає зі статті про [panic та
recover](../02-language-basics/15-panic-and-recover.md), але наслідок
вартий того, щоб сказати його прямо: **кожна горутина, яку ви запускаєте
і яка переживає один запит, потребує власного recover**, інакше одні
погані вхідні дані будь-де завалять увесь сервер.

Виправлення — обгортка, через яку ви запускаєте горутини:

```go
func safeGo(name string, fn func()) {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                fmt.Printf("goroutine %s panicked: %v\n", name, r)
            }
        }()
        fn()
    }()
}
```

```go
safeGo("worker", func() { panic("boom") })

time.Sleep(50 * time.Millisecond)
fmt.Println("main is still running")
// output:
// goroutine worker panicked: boom
// main is still running
```

Сон тут лише для того, щоб приклад мав що показати: без нього `main`
повернувся б і завершив процес раніше, ніж горутина взагалі встигла б
надрукувати щось. Це властивість прикладу, а не `safeGo`.

У реальному коді логуйте стек поряд зі значенням. Повідомлення про паніку
саме собою майже нічого не каже про те, звідки вона взялася:

```go
if r := recover(); r != nil {
    fmt.Printf("goroutine %s panicked: %v\n%s", name, r, debug.Stack())
}
```

`debug.Stack()` походить із `runtime/debug` і повертає стек тієї горутини,
яка її викликає, — тож викликати її треба саме всередині відновлювального
`defer`, поки ця горутина ще на стеку.

## Зупинка: `select` на `ctx.Done()`

Довгоживучому циклу потрібен вимикач, і цей вимикач — це
[контекст](05-context.md), який йому передали. Цикл із тікером (ticker) —
це форма, яку ви писатимете найчастіше:

```go
func poll(ctx context.Context, every time.Duration) {
    t := time.NewTicker(every)
    defer t.Stop()

    for {
        select {
        case <-ctx.Done():
            fmt.Println("poller stopping:", ctx.Err())
            return
        case <-t.C:
            fmt.Println("tick")
        }
    }
}
```

```go
ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
defer cancel()
poll(ctx, 100*time.Millisecond)
// output:
// tick
// tick
// poller stopping: context deadline exceeded
```

`defer t.Stop()` тут важливий. `time.Ticker` тримає таймер часу виконання,
який продовжує спрацьовувати, поки його не зупинять, тож цикл, який
повертається, не зупинивши свій тікер, протікає — по одному таймеру на
кожен виклик.

Ставте гілку `ctx.Done()` першою за звичкою. `select` обирає випадково
серед готових гілок, тож порядок насправді не надає їй пріоритету, — але
коли вона читається першою, шлях виходу — це перше, що бачить будь-хто,
хто читає код.

## Дренаж: очікування завершення роботи

Скасування каже горутинам зупинитися. Воно не чекає на них і не заважає
передати їм нову роботу вже за мікросекунду. Компонент, що володіє
фоновою роботою, має спершу відмовляти в новій роботі, а тоді чекати — з
дедлайном, бо чекати вічно — це не завершення роботи:

```go
type Pool struct {
    mu     sync.Mutex
    closed bool
    wg     sync.WaitGroup
}

func (p *Pool) Go(fn func()) bool {
    p.mu.Lock()
    defer p.mu.Unlock()
    if p.closed {
        return false
    }
    p.wg.Add(1)
    go func() {
        defer p.wg.Done()
        fn()
    }()
    return true
}
```

Блокування тут не опційне. `wg.Add` не повинен виконуватися після того,
як почав працювати `wg.Wait`, і саме прапорець `closed` під тим самим
м'ютексом це гарантує.

```go
func (p *Pool) Shutdown(ctx context.Context) {
    p.mu.Lock()
    p.closed = true
    p.mu.Unlock()

    drained := make(chan struct{})
    go func() {
        p.wg.Wait()
        close(drained)
    }()

    select {
    case <-drained:
        fmt.Println("all work finished")
    case <-ctx.Done():
        fmt.Println("drain budget expired")
    }
}
```

У `wg.Wait()` немає таймауту, тож його переносять в окрему горутину й
перетворюють на канал, що закривається, коли `Wait` повертається. Тепер
він може змагатися з контекстом у `select`. Це перетворення «чекати, але
не вічно» варто запам'ятати — воно працює для будь-чого блокувального, що
не можна скасувати напряму.

```go
var p Pool
p.Go(func() { time.Sleep(50 * time.Millisecond) })
p.Shutdown(context.Background())
fmt.Println("accepted after close:", p.Go(func() {}))
// output:
// all work finished
// accepted after close: false
```

А коли робота переживає відведений бюджет часу:

```go
var p Pool
p.Go(func() { time.Sleep(2 * time.Second) })

ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()
p.Shutdown(ctx)
// output: drain budget expired
```

Вичерпання бюджету — це реальний результат, а не баг. Скажіть про це в
лозі — мовчазне відкидання роботи, що виконується, це якраз те, як
«чистий» деплой втрачає дані.

> **З погляду Python:** `safeGo` — це приблизно те, що `Thread` дає вам
> безкоштовно, адже виключення в потоці друкується й гине, не вбиваючи
> процес. У Go все навпаки: за замовчуванням це фатально, а виживання
> доводиться підключати самому.

## Швидка довідка

| Питання | Форма |
|---|---|
| паніка у фоновій горутині | власний `defer`/`recover` усередині цієї горутини |
| зупинка циклу | `select` із гілкою `case <-ctx.Done(): return` |
| повторювана робота | `time.NewTicker` разом із `defer t.Stop()` |
| відмова в новій роботі | прапорець `closed` під тим самим м'ютексом, що й `wg.Add` |
| очікування з дедлайном | `wg.Wait()` у горутині, `close(ch)`, `select` на ньому |

## Джерела

- [Handling panics — go.dev/ref/spec#Handling_panics](https://go.dev/ref/spec#Handling_panics)
- [`sync.WaitGroup` — pkg.go.dev/sync#WaitGroup](https://pkg.go.dev/sync#WaitGroup)
- [`time.Ticker` — pkg.go.dev/time#Ticker](https://pkg.go.dev/time#Ticker)
- [Довідник пакету `context` — pkg.go.dev/context](https://pkg.go.dev/context)
- [`runtime/debug.Stack` — pkg.go.dev/runtime/debug#Stack](https://pkg.go.dev/runtime/debug#Stack)
- [Go blog: pipelines and cancellation — go.dev/blog/pipelines](https://go.dev/blog/pipelines)
