# Сигнали та коректна зупинка

Коли оркестратор зупиняє ваш сервіс, він надсилає `SIGTERM` і запускає
годинник. Обробіть його — і робота в польоті завершиться; ігноруйте
його — і вас уб'ють посеред запиту. Весь механізм — це одна функція
плюс патерни конкурентності зі статті про [довгоживучі
горутини](../05-concurrency/08-long-running-goroutines.md).

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

<-ctx.Done()
fmt.Println("signal received:", ctx.Err())
// output: signal received: context canceled
```

## `signal.NotifyContext`

Це сучасна форма, і саме до неї варто звертатись. Вона повертає
контекст, який скасовується, коли приходить будь-який із перелічених
сигналів, — це вмикається прямо в кожен `ctx.Done()`, який ви вже
написали, без окремого каналу, який треба прокладати.

Зауважте, що `ctx.Err()` — це `context.Canceled`, а не щось у формі
сигналу. Зупинка виглядає для коду, який зупиняють, точнісінько як
будь-яке інше скасування, — у цьому й суть.

Викликайте повернутий `stop`, коли закінчите. Він відновлює типову
поведінку сигналів, тож другий `Ctrl-C` під час повільної зупинки
вбиває процес, а не поглинається.

Старіша форма `signal.Notify` з каналом усе ще працює й усе ще
трапляється в наявному коді:

```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
<-sigCh
```

Канал **має бути буферизованим**. Доставка сигналу ніколи не
блокується, тож небуферизований канал, у якого в цю мить нема
отримувача, мовчки втрачає сигнал.

## Які сигнали

| Сигнал | Значення | Можна перехопити |
|---|---|---|
| `SIGTERM` | "будь ласка, зупинись" від оркестраторів | так — обробляйте цей |
| `SIGINT` | `Ctrl-C` | так |
| `SIGHUP` | термінал закрито; часто "перезавантаж конфігурацію" | так |
| `SIGKILL` | примусове вбивство | **ні** |

`SIGKILL` не можна перехопити, заблокувати чи відкласти. Саме тому
пільговий період має значення: середовище виконання контейнера надсилає
`SIGTERM`, чекає, а потім надсилає `SIGKILL`. Усе, що ви не встигли
завершити на той момент, втрачено.

## Форма зупинки

Три кроки, по порядку:

1. **Перестати приймати нову роботу.** Сервер перестає слухати;
   споживач перестає забирати повідомлення.
2. **Дренувати те, що в польоті**, з дедлайном.
3. **Звільнити ресурси** — скинути буфери, закрити пули.

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    srv := startServer()

    <-ctx.Done()
    stop()   // тепер другий сигнал вб'є нас, як і має бути

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    if err := srv.Shutdown(shutdownCtx); err != nil {
        fmt.Fprintln(os.Stderr, "shutdown:", err)
    }
}
```

`http.Server` та його метод `Shutdown` належать до наступної теми; тут
важлива лише форма — щось, що перестає приймати роботу й дренує її,
маючи дедлайн.

Контекст зупинки будується з `context.Background()`, **не** з `ctx`.
Побудова його з уже скасованого `ctx` дала б контекст, мертвий одразу
після народження, і нічого б не дреновано.

## Дренаж із дедлайном

У `wg.Wait()` немає таймауту. Перетворіть його на канал і змагайтесь із
ним — прийом із теми конкурентності:

```go
func drain(wg *sync.WaitGroup, budget time.Duration) string {
    done := make(chan struct{})
    go func() { wg.Wait(); close(done) }()

    select {
    case <-done:
        return "drained cleanly"
    case <-time.After(budget):
        return "budget expired"
    }
}
```

```go
// два воркери, 30мс і 40мс, бюджет 500мс
// output:
// worker 1 finished
// worker 2 finished
// drained cleanly
```

Коли бюджет вичерпується, скасуйте власний контекст воркерів, щоб вони
зупинились раніше, а не були вбиті посеред запису:

```go
// один воркер, якому треба 5с, бюджет 100мс
// output:
// budget expired
// worker 3 cancelled
```

Цей порядок — увесь дизайн: спершу попросіть ввічливо, зачекайте
обмежений час, а потім наполягайте.

## Арифметика бюджету

Розділіть пільговий період між фазами й переконайтесь, що сума
вміщується в нього. Якщо оркестратор дозволяє 30 секунд, дренаж
запитів на 25 секунд плюс фоновий дренаж на 10 секунд перевищує ліміт,
і решту буде вбито. Називайте бюджети константами, щоб сума була
видна:

```go
const (
    requestDrainBudget = 12 * time.Second
    edgeDrainBudget    = 8 * time.Second
)
```

## Речі, які тихо ламають це

- **`os.Exit` пропускає `defer`.** Будь-яке прибирання у відкладеній
  функції не виконується, як показала стаття [прапорці та
  середовище](04-flags-and-environment.md). Замість цього повертайтесь
  із `main`.
- **Самого відкладеного `stop()` недостатньо.** Викликайте `stop()`
  явно після `<-ctx.Done()`, щоб другий сигнал було враховано.
- **PID 1 у контейнері не має типових обробників сигналів.** Якщо ваш
  процес — PID 1 і не обробляє `SIGTERM`, типова дія не застосовується,
  і сигнал ігнорується — тоді контейнер щоразу вичерпує весь пільговий
  період і гине від `SIGKILL`.
- **Відокремлена робота втрачає свій контекст.** Усе, що запущено з
  запиту, але переживає його, потребує власного часу життя, інакше
  його скасовує та ж мить, коли запит завершується.

Розрізнення того, чому контекст завершився, використовує `errors.Is`:

```go
fmt.Println(errors.Is(ctx.Err(), context.DeadlineExceeded))   // output: true
fmt.Println(errors.Is(ctx.Err(), context.Canceled))           // output: false
```

> **З досвіду Python:** `signal.NotifyContext` замінює
> `signal.signal(SIGTERM, handler)`, і поводиться краще — немає
> окремого обробника, що виконується в довільний момент, лише контекст,
> який стає скасованим, і його видно там, де ви вже й так робите
> `select`.

## Швидка довідка

| Завдання | Виклик |
|---|---|
| перехопити сигнали як контекст | `signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)` |
| відновити типову поведінку | повернутий `stop()` — викликайте явно і його теж |
| старіша форма з каналом | `signal.Notify(ch, ...)` з **буферизованим** каналом |
| дедлайн зупинки | `context.WithTimeout(context.Background(), d)` — не з мертвого ctx |
| зупинити сервер | `srv.Shutdown(shutdownCtx)` |
| чекати з дедлайном | `wg.Wait()` у горутині, `close(done)`, `select` |
| після вичерпання бюджету | скасуйте контекст воркерів |
| чому завершився | `errors.Is(ctx.Err(), context.DeadlineExceeded)` |

## Джерела

- [`os/signal` package reference — pkg.go.dev/os/signal](https://pkg.go.dev/os/signal)
- [`signal.NotifyContext` — pkg.go.dev/os/signal#NotifyContext](https://pkg.go.dev/os/signal#NotifyContext)
- [`signal.Notify` — pkg.go.dev/os/signal#Notify](https://pkg.go.dev/os/signal#Notify)
- [`context` package reference — pkg.go.dev/context](https://pkg.go.dev/context)
- [`http.Server.Shutdown` — pkg.go.dev/net/http#Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)
