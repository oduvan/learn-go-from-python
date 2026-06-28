# select

Оператор `select` чекає на **кілька операцій з каналами одночасно** й
продовжує з тією, яка готова першою. Це керівна структура, що робить канали
композиційними — таймаути, скасування та мультиплексування будуються на
ній.

```go
select {
case v := <-ch1:
    fmt.Println("from ch1:", v)
case v := <-ch2:
    fmt.Println("from ch2:", v)
}
```

Кожен `case` — це надсилання чи отримання. `select` блокує, доки одна з них
не зможе виконатися, виконує цей випадок і продовжує. Якщо **кілька**
готові водночас, він обирає одну **випадково** — тож не покладайтеся на
порядок випадків як на пріоритет.

```go
c1 := make(chan string, 1)
c2 := make(chan string, 1)
c1 <- "hello"            // лише c1 має значення

select {
case m := <-c1:
    fmt.Println(m)       // output: hello
case m := <-c2:
    fmt.Println(m)
}
```

## `default`: не блокувати

Випадок `default` виконується негайно, якщо жоден інший не готовий,
перетворюючи `select` на **неблокувальну** операцію. Так опитують канал чи
роблять неблокувальне надсилання.

```go
ch := make(chan int)     // порожній, без відправника

select {
case v := <-ch:
    fmt.Println("got", v)
default:
    fmt.Println("nothing ready")   // output: nothing ready
}
```

## Таймаути через `time.After`

`time.After(d)` повертає канал, що видає значення через тривалість `d`.
Покладіть його в `select` — і отримаєте таймаут безкоштовно:

```go
ch := make(chan int)     // ніхто не надішле

select {
case v := <-ch:
    fmt.Println("got", v)
case <-time.After(10 * time.Millisecond):
    fmt.Println("timeout")          // output: timeout
}
```

## Цикл `for`-`select` із каналом done

Робоча конячка для довготривалої горутини: цикл на `select`, який обробляє
роботу на одному каналі та **сигнал зупинки** на іншому. Закритий канал
змушує кожне отримання повертатися негайно, тож він транслює «стоп» усім
слухачам.

```go
nums := make(chan int)
done := make(chan struct{})
finished := make(chan struct{})

go func() {
    for {
        select {
        case n := <-nums:
            fmt.Println("got", n)
        case <-done:
            fmt.Println("stopping")
            close(finished)
            return
        }
    }
}()

nums <- 1
nums <- 2
close(done)              // сигнал горутині зупинитися
<-finished               // чекати, доки вона справді завершиться
// output:
// got 1
// got 2
// stopping
```

## Вимкнення випадку через nil-канал

Отримання (чи надсилання) на `nil`-каналі блокується назавжди, тож
присвоєння змінній каналу `nil` **прибирає** його випадок із `select`. Це
ідіоматичний спосіб припинити слухати канал посеред циклу, не
перебудовуючи `select`.

```go
var ch chan int          // nil
select {
case <-ch:               // ніколи не спрацює — ch є nil
    fmt.Println("unreachable")
case <-time.After(time.Millisecond):
    fmt.Println("only this can happen")   // output: only this can happen
}
```

## Порожній `select` блокується назавжди

`select {}` без випадків блокує горутину назавжди — інколи вживають, щоб
«припаркувати» `main`, доки працюють фонові горутини, але в `main`, коли
більше нічого не може виконатися, це запускає детектор взаємних блокувань:

```go
func main() {
    select {}   // без випадків — блокування назавжди
}
// fatal error: all goroutines are asleep - deadlock!
```

## Швидка довідка

| Форма | Значення |
|---|---|
| `select { case ...: }` | чекати, доки одна операція з каналом готова |
| кілька готові | одну обрано **випадково** |
| `default:` | виконати, якщо нічого не готове (неблокувально) |
| `case <-time.After(d):` | гілка таймауту |
| `for { select { ... } }` | довготривалий мультиплексований цикл |
| `case <-done:` | сигнал зупинки (закритий канал транслює) |
| випадок із nil-каналом | вимкнено (блокується назавжди) |

## Джерела

- [Select statements — go.dev/ref/spec#Select_statements](https://go.dev/ref/spec#Select_statements)
- [time.After — pkg.go.dev/time#After](https://pkg.go.dev/time#After)
- [Go blog: pipelines and cancellation — go.dev/blog/pipelines](https://go.dev/blog/pipelines)
- [Effective Go: concurrency — go.dev/doc/effective_go#concurrency](https://go.dev/doc/effective_go#concurrency)
