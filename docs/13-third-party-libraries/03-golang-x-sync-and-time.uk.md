# golang.org/x/sync та x/time

Два модулі команди Go, що стоять якраз поза межами стандартної
бібліотеки. `errgroup` — той, до якого ви тягнутиметеся найчастіше:
обмежена конкурентність з поширенням помилок приблизно у шість рядків.

> **Модулі:** `golang.org/x/sync` та `golang.org/x/time`. Підтримуються
> командою Go, версіюються окремо від toolchain — саме тому вони тут, а
> не в темі 05.

Спирається на [обмежену конкурентність](../05-concurrency/07-bounded-concurrency.md),
де показано варіант, написаний вручну.

## `errgroup` заміняє WaitGroup плюс обробку помилок

Версія зі стандартної бібліотеки потребує `WaitGroup`, м'ютекс і
змінну для першої помилки. `errgroup` — те саме, запаковане разом:

```go
g, ctx := errgroup.WithContext(context.Background())

for _, job := range jobs {
    g.Go(func() error {
        return process(ctx, job)
    })
}

if err := g.Wait(); err != nil {
    return err
}
```

`Wait` блокується, доки кожна горутина (goroutine) не повернеться, і
видає **першу ненульову помилку**. Пізніші помилки відкидаються — якщо
вам потрібні всі, зберіть їх самостійно через `errors.Join`.

```go
// four jobs, job 2 fails
// err: job 2 failed
```

## `WithContext` скасовує решту

Контекст, який вона повертає, скасовується, щойно будь-яка горутина
повертає помилку. Робота, що ще триває, бачить `ctx.Done()` і може
зупинитися:

```go
g, ctx := errgroup.WithContext(context.Background())

g.Go(func() error { return errors.New("boom") })
g.Go(func() error {
    <-ctx.Done()
    return nil
})

err := g.Wait()
// boom, and ctx.Err() is context canceled
```

Це головна причина її використовувати. Без скасування один провал
залишає кожного "сусіда" працювати до кінця заради результату, який
нікому не потрібен.

**Горутини мусять справді стежити за контекстом.** `time.Sleep` або
блокувальний виклик без контексту повністю ігнорує скасування, і
`Wait` усе одно на нього чекає.

Зауважте: використовуйте `ctx`, який повертає `WithContext`, а не той,
що ви передали. Передача зовнішнього контексту в горутини зводить
нанівець весь механізм.

## `SetLimit` обмежує конкурентність

```go
g := new(errgroup.Group)
g.SetLimit(3)

for _, job := range jobs {
    g.Go(func() error { return process(job) })
}
// peak concurrency: 3
```

`g.Go` блокується, коли ліміт досягнутий, тож сотня завдань виконується
по три за раз. Це повністю заміняє семафор на буферизованому каналі.

Викликайте `SetLimit` **до** першого `Go`; зміна його, поки горутини
працюють, панікує. `SetLimit(-1)` означає необмежено.

`TryGo` запускає горутину, лише якщо є вільне місце, повертаючи, чи
вдалося це:

```go
g.SetLimit(1)
fmt.Println(g.TryGo(slowJob))   // output: true
fmt.Println(g.TryGo(quickJob))  // output: false
```

Корисно для "зроби це, якщо є запасна потужність, інакше пропусти".

## Збір результатів

Різні індекси зрізу не потребують блокування — точнісінько як раніше:

```go
g := new(errgroup.Group)
g.SetLimit(2)

out := make([]int, len(jobs))
for i, job := range jobs {
    g.Go(func() error {
        v, err := process(job)
        out[i] = v
        return err
    })
}
err := g.Wait()
// [0 1 4 9 16]
```

Результати зберігають порядок вхідних даних. Усе, що справді спільне,
все ще потребує м'ютекса.

## Чого вона не робить

- **Вона не відновлюється після паніки.** Паніка у функції `g.Go`
  забирає весь процес, точнісінько як у звичайній горутині — саме про
  це стаття про
  [довгоживучі горутини](../05-concurrency/08-long-running-goroutines.md).
  Якщо робота може панікувати, робіть `recover` всередині функції.
- **`Wait` можна викликати лише раз**, і група потім не придатна для
  повторного використання.
- **Вона нічого не обмежує за часом.** Це робота контексту.

`golang.org/x/sync` також має `semaphore` для зважених лімітів і
`singleflight` для згортання дублікатних одночасних викликів з тим
самим ключем — варто знати про їхнє існування, хоча вони рідко перший
інструмент, до якого тягнешся.

## `x/time/rate`: token bucket

Обмежувач швидкості для виклику чогось із опублікованим лімітом:

```go
lim := rate.NewLimiter(rate.Limit(100), 1)   // 100/second, burst 1

for _, req := range requests {
    if err := lim.Wait(ctx); err != nil {
        return err   // context cancelled
    }
    send(req)
}
```

`Wait` блокується, доки токен не стане доступним або контекст не
завершиться. Другий аргумент — **сплеск (burst)**: скільки може пройти
одночасно після періоду простою. Сплеск 1 дає рівномірно розподілені
запити; більший сплеск дозволяє пік.

Квота на хвилину переводиться напряму:

```go
rpm := 600
lim := rate.NewLimiter(rate.Limit(float64(rpm)/60.0), 1)
fmt.Println(float64(lim.Limit()))   // output: 10
```

`Allow` — неблокувальна форма: вона відповідає одразу й забирає токен,
лише якщо той був вільний:

```go
l := rate.NewLimiter(rate.Limit(1), 1)
fmt.Println(l.Allow(), l.Allow())   // output: true false
```

Використовуйте `Wait` для вихідних викликів, які хочете рівномірно
розподілити, `Allow` — для вхідних запитів, які хочете відхиляти з
`429`. Обмежувач безпечний для конкурентного використання, тож — один
на кожен зовнішній сервіс, створений при старті.

`Reserve` стоїть між ними: він каже, скільки б ви *чекали*, тож можна
вирішити, чи варто.

## Об'єднуючи їх разом

Ці два компонуються у форму, потрібну при виклику API з обмеженою
швидкістю конкурентно:

```go
// ctx приходить ззовні — наприклад, це контекст вхідного запиту
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(8)                                   // 8 in flight
lim := rate.NewLimiter(rate.Limit(20), 5)       // 20/s, burst 5

for _, item := range items {
    g.Go(func() error {
        if err := lim.Wait(ctx); err != nil {
            return err
        }
        return fetch(ctx, item)
    })
}
return g.Wait()
```

Конкурентність і швидкість — різні ліміти, і обидва мають значення:
вісім запитів у польоті все одно можуть перевищити двадцять на секунду,
якщо кожен швидкий.

> **З досвіду Python:** `errgroup` — це `asyncio.gather` з обмеженням
> конкурентності й автоматичним скасуванням при першій помилці, або
> `ThreadPoolExecutor`, що коректно поширює винятки. `rate.Limiter` —
> той самий token bucket, який ви б інакше писали вручну.

## Швидка довідка

| Задача | Форма |
|---|---|
| група | `g, ctx := errgroup.WithContext(ctx)` |
| почати роботу | `g.Go(func() error { ... })` |
| чекати | `g.Wait()` → перша помилка |
| обмежити конкурентність | `g.SetLimit(n)` до першого `Go` |
| лише якщо вільно | `g.TryGo(fn)` |
| який контекст використовувати | той, що повернув `WithContext` |
| паніки | не відновлюються — робіть це самі |
| розподілити вихідні виклики | `rate.NewLimiter(rate.Limit(n), burst)` + `Wait(ctx)` |
| відхилити вхідні | `lim.Allow()` |
| квота на хвилину | `rate.Limit(float64(rpm)/60.0)` |

## Джерела

- [`errgroup` — pkg.go.dev/golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup)
- [`semaphore` — pkg.go.dev/golang.org/x/sync/semaphore](https://pkg.go.dev/golang.org/x/sync/semaphore)
- [`singleflight` — pkg.go.dev/golang.org/x/sync/singleflight](https://pkg.go.dev/golang.org/x/sync/singleflight)
- [`rate` — pkg.go.dev/golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate)
- [Token bucket — en.wikipedia.org/wiki/Token_bucket](https://en.wikipedia.org/wiki/Token_bucket)
