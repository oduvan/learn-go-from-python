# Патерни конкурентності

Будівельні блоки — горутини, канали, `select`, `sync` та `context` —
складаються в кілька патернів, по які ви тягнетеся знову й знову. Ця стаття
показує канонічні три: **пули робітників**, **fan-out/fan-in** та
**конвеєри**.

## Пул робітників

Коли у вас багато незалежних завдань і ви хочете обмежити паралельність,
запустіть *фіксовану* кількість робітників, які тягнуть зі спільного каналу
`jobs` і пишуть у канал `results`. Розмір пулу обмежує, скільки виконується
водночас.

```go
jobs := make(chan int, 100)
results := make(chan int, 100)
var wg sync.WaitGroup

for w := 0; w < 3; w++ {           // 3 робітники
    wg.Add(1)
    go func() {
        defer wg.Done()
        for j := range jobs {       // кожен робітник вичерпує jobs
            results <- j * j
        }
    }()
}

for i := 1; i <= 5; i++ {
    jobs <- i
}
close(jobs)                         // більше завдань немає; цикли range робітників завершаться

go func() { wg.Wait(); close(results) }()   // закрити results, щойно всі робітники завершаться

sum := 0
for r := range results {            // зібрати (порядок недетермінований)
    sum += r
}
fmt.Println(sum)                    // output: 55
```

Дві ідіоми роблять це надійним: **закрийте `jobs`**, щоб цикли `range`
робітників завершилися, і **закрийте `results` в окремій горутині після
`wg.Wait()`**, щоб цикл `range` збирача завершився. Оскільки робітники
завершуються в довільному порядку, агрегуйте незалежно від порядку (тут —
сума).

## Fan-out / fan-in

*Fan-out* = кілька горутин читають з одного каналу (пул робітників вище — це
fan-out). *Fan-in* = злиття кількох каналів в один. Ось половина-злиття, що
використовує `WaitGroup`, аби закрити злитий канал, щойно вичерпано кожне
джерело:

```go
func merge(cs ...<-chan int) <-chan int {
    out := make(chan int)
    var wg sync.WaitGroup
    for _, c := range cs {
        wg.Add(1)
        go func(c <-chan int) {
            defer wg.Done()
            for v := range c {
                out <- v
            }
        }(c)
    }
    go func() { wg.Wait(); close(out) }()
    return out
}
```

Як це запустити — два джерела, злиті в один потік (результати надходять у
будь-якому порядку, тож сортуємо перед друком для сталого результату):

```go
var got []int
for v := range merge(gen(1, 2), gen(3, 4)) {
    got = append(got, v)
}
sort.Ints(got)
fmt.Println(got)   // output: [1 2 3 4]
```

## Конвеєр

Конвеєр (pipeline) — це ланцюг стадій, кожна з яких є функцією, що **бере
канал лише-на-отримання й повертає такий самий**, виконуючи свою роботу в
горутині. Значення течуть від стадії до стадії; кожна стадія закриває свій
вихід, коли її вхід вичерпано. Один ланцюг зберігає порядок.

```go
func gen(nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        for _, n := range nums {
            out <- n
        }
        close(out)
    }()
    return out
}

func sq(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        for n := range in {
            out <- n * n
        }
        close(out)
    }()
    return out
}

for v := range sq(gen(2, 3, 4)) {
    fmt.Println(v)
}
// output:
// 4
// 9
// 16
```

Кожна стадія незалежна й композиційна: `sq(sq(gen(...)))` просто працює, і
стадії виконуються конкурентно, поки дані течуть крізь них.

## Робіть горутини зупинними

Кожна довговічна горутина потребує шляху виходу, інакше вона **протікає** —
живе до кінця програми, утримуючи пам'ять і, можливо, блокуючись назавжди.
Дайте кожній спосіб вийти: закрийте її вхідний канал або передайте
`context.Context` і робіть `select` на `ctx.Done()`. Горутина, яку не можна
зупинити, — це баг.

```go
func worker(ctx context.Context, jobs <-chan int, done chan<- struct{}) {
    for {
        select {
        case <-ctx.Done():       // скасування перемагає
            fmt.Println("stopped")
            close(done)
            return
        case j := <-jobs:
            fmt.Println("did", j)
        }
    }
}

ctx, cancel := context.WithCancel(context.Background())
jobs := make(chan int)
done := make(chan struct{})
go worker(ctx, jobs, done)

jobs <- 1                        // робітник обробляє одне завдання
cancel()                         // потім ми кажемо йому зупинитися
<-done                           // і чекаємо, доки він справді вийде
// output:
// did 1
// stopped
```

## Емпіричні правила

- **Не запускайте горутину, не знаючи, як вона зупиниться.**
- **Канал закриває відправник, ніколи отримувач** — і лише раз.
- **Агрегуйте результати незалежно від порядку**, якщо конвеєр не гарантує
  порядок.
- Надавайте перевагу **обмеженому** пулу робітників перед породженням однієї
  горутини на завдання, коли завдань необмежено.
- Протягуйте **`context`** через довгі операції, щоб викликачі могли
  скасувати.

## Швидка довідка

| Патерн | Форма |
|---|---|
| Пул робітників | N горутин роблять `range` по спільному каналу `jobs` |
| Fan-out | кілька горутин читають один канал |
| Fan-in (merge) | багато каналів → один, `WaitGroup`, потім close |
| Конвеєр | стадії: `func(<-chan T) <-chan U`, кожна закриває свій вихід |
| Зупинити горутину | закрити її вхід або `select` на `ctx.Done()` |
| Дисципліна закриття | закриває відправник, раз |

## Джерела

- [Go blog: pipelines and cancellation — go.dev/blog/pipelines](https://go.dev/blog/pipelines)
- [Go Concurrency Patterns (talk) — go.dev/talks/2012/concurrency.slide](https://go.dev/talks/2012/concurrency.slide)
- [Effective Go: concurrency — go.dev/doc/effective_go#concurrency](https://go.dev/doc/effective_go#concurrency)
- [sync.WaitGroup — pkg.go.dev/sync#WaitGroup](https://pkg.go.dev/sync#WaitGroup)
