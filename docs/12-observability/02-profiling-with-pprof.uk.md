# Профілювання через pprof

Коли щось працює повільно чи використовує забагато пам'яті, pprof
показує, де саме, а не там, де ви здогадалися. Він вбудований у час
виконання, майже нічого не коштує в стані спокою й працює на живому
продакшн-процесі.

```bash
go tool pprof -top -nodecount=5 cpu.prof
```

[Трейсер виконання](../01-ecosystem-and-installation/03-go-tool-trace.md)
відповідає на інше питання. pprof семплює *куди йдуть час і пам'ять*;
трейсер показує *коли щось відбувалося* — планування, блокування,
паузи GC. Тягніться спершу до pprof.

## Профілі

| Профіль | Відповідає на |
|---|---|
| `cpu` | які функції спалюють CPU |
| `heap` | що виділяє пам'ять і що досі живе |
| `goroutine` | скільки горутин (goroutine) існує й де вони застрягли |
| `allocs` | кожне виділення пам'яті від старту |
| `mutex` | які блокування конкурують |
| `block` | що блокується на каналах і системних викликах |

`goroutine` — той, що знаходить витоки: кількість, що стабільно
зростає, — це горутина, яка ніколи не повертається.

`mutex` і `block` вимкнені за замовчуванням і потребують увімкнення:

```go
runtime.SetMutexProfileFraction(5)
runtime.SetBlockProfileRate(10000)
```

## З тесту чи бенчмарку

Найпростіший спосіб, і той, до якого варто тягнутися під час
розробки:

```bash
go test -run XXX -bench . -cpuprofile cpu.prof -memprofile mem.prof ./...
```

Тепер у вас профіль відтворюваного навантаження, що краще за
профілювання цілого сервера, коли ви вже підозрюєте одну функцію.

## З коду

Для пакетного завдання чи CLI обгорніть роботу:

```go
f, err := os.Create("cpu.prof")
if err != nil {
    return err
}
if err := pprof.StartCPUProfile(f); err != nil {
    return err
}
defer pprof.StopCPUProfile()
```

Профіль купи — це знімок, тож робіть його в цікавий момент — і
викличте `runtime.GC()` спершу, інакше ви вимірюєте сміття, яке ще не
зібрали:

```go
runtime.GC()
pprof.WriteHeapProfile(mf)
```

`pprof.Profiles()` перелічує доступне:

```
allocs (20)
block (0)
goroutine (1)
heap (20)
mutex (0)
threadcreate (9)
```

## З живого сервера

`net/http/pprof` реєструє обробники як побічний ефект імпорту — сліпий
імпорт зі статті про [імпорти](../02-language-basics/16-imports.md):

```go
import _ "net/http/pprof"
```

Це прикріплюється до **типового мультиплексора** (default mux), тож
якщо ви побудували власний `ServeMux`, реєструйте явно:

```go
mux.HandleFunc("/debug/pprof/", pprof.Index)
mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
mux.HandleFunc("/debug/pprof/heap", pprof.Index)
```

Тоді спрямуйте інструмент на живий процес:

```bash
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
go tool pprof http://localhost:8080/debug/pprof/heap
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

**Ніколи не виставляйте ці ендпоінти публічно.** Вони розкривають
імена функцій, дозволяють будь-кому запустити 30-секундний
CPU-профіль, а дамп горутин показує вашу внутрішню структуру.
Прив'яжіть їх до окремого внутрішнього порту, поставте за
автентифікацією або вмикайте через змінну середовища.

## Читання виводу

```
Duration: 204.38ms, Total samples = 30ms (14.68%)
Showing nodes accounting for 30ms, 100% of 30ms total
      flat  flat%   sum%        cum   cum%
      10ms 33.33% 33.33%       10ms 33.33%  runtime.kevent
         0     0%   100%       10ms 33.33%  main.main
```

Дві колонки мають значення:

- **flat** — час у власному коді *цієї* функції.
- **cum** — час у цій функції *і в усьому, що вона викликала*.

Функція з високим `cum` і майже нульовим `flat` — це виклик; ідіть за
ним далі. Високий `flat` — це там, де робота насправді відбувається.

Зверніть увагу на `Total samples = 30ms (14.68%)` — ця програма
здебільшого простоювала, тож кількість семплів низька, а результати —
шум. **Профілю потрібно достатньо семплів, щоб щось означати.**
Профілюйте під реальним навантаженням або досить довго, інакше верхні
записи будуть функціями планування часу виконання на кшталт `kevent` і
`usleep`, а не вашим кодом.

Профілі пам'яті чіткіші на короткому запуску:

```
   37.76MB 90.87% 90.87%    37.76MB 90.87%  strings.(*Builder).WriteString
```

Це однозначно: одна точка виклику — це 91% виділень.

Профіль купи має чотири погляди, обрані через `-sample_index`:

| Індекс | Показує |
|---|---|
| `inuse_space` | байти, живі зараз — **типовий**, для витоків |
| `inuse_objects` | кількість об'єктів, живих зараз |
| `alloc_space` | байти, коли-небудь виділені — для тиску на GC |
| `alloc_objects` | кількість виділень коли-небудь |

`inuse_*` знаходить те, що утримується; `alloc_*` знаходить те, що
обертається. Програма без витоку може все ще проводити весь час у GC.

## Інтерактивний та веб-перегляд

```bash
go tool pprof cpu.prof
(pprof) top
(pprof) list work        # анотоване джерело, рядок за рядком
(pprof) web              # SVG-граф викликів, потребує graphviz
```

`list` — найкорисніша команда: вона друкує вихідний код функції з
вартістю на кожен рядок, що зазвичай завершує розслідування.

```bash
go tool pprof -http=:8081 cpu.prof
```

Це відкриває веб-інтерфейс із flame-графом. Читайте його так: ширина —
це час, і кожен брусок сидить на своєму виклику. Шукайте широкі бруски,
а не глибокі стеки — глибина — це лише вкладеність викликів.

Ви також можете порівняти два профілі, і саме так доводиться, що
оптимізація спрацювала:

```bash
go tool pprof -base before.prof after.prof
```

## Порядок дій

1. **Спершу вимірюйте.** Інтуїція щодо продуктивності Go зазвичай
   хибна; вузьким місцем часто виявляється виділення пам'яті чи
   блокування, а не арифметика, на яку ви дивилися.
2. **Забенчмарчте підозрюваного**, щоб мати число, яке рухається.
3. **Профілюйте його** й знайдіть справжній рядок.
4. **Змініть одну річ**, перебенчмарчте, порівняйте через `benchstat`.

Профілювання програми, яка й так достатньо швидка, — це спосіб
згаяти вечір.

> **З досвіду Python:** `cProfile` плюс `memory_profiler`, але через
> семплування, а не інструментування, тож накладні витрати достатньо
> низькі, щоб лишити ввімкненим у продакшні — і це працює на живому
> процесі через HTTP, чому в Python немає стандартного еквівалента.

## Швидка довідка

| Задача | Команда |
|---|---|
| з бенчмарку | `go test -bench . -cpuprofile cpu.prof` |
| з коду | `pprof.StartCPUProfile(f)` / `defer StopCPUProfile()` |
| знімок купи | `runtime.GC()`, потім `pprof.WriteHeapProfile(f)` |
| з сервера | `_ "net/http/pprof"`, потім `go tool pprof <url>` |
| живий CPU-профіль | `.../debug/pprof/profile?seconds=30` |
| витік горутин | `.../debug/pprof/goroutine` |
| увімкнути mutex/block | `SetMutexProfileFraction`, `SetBlockProfileRate` |
| топ функцій | `go tool pprof -top -nodecount=10 cpu.prof` |
| вартість на рядок | `(pprof) list FuncName` |
| flame-граф | `go tool pprof -http=:8081 cpu.prof` |
| порівняти | `go tool pprof -base before.prof after.prof` |
| витік проти обороту | `-sample_index=inuse_space` проти `alloc_space` |

## Джерела

- [`runtime/pprof` — pkg.go.dev/runtime/pprof](https://pkg.go.dev/runtime/pprof)
- [`net/http/pprof` — pkg.go.dev/net/http/pprof](https://pkg.go.dev/net/http/pprof)
- [Go blog: profiling Go programs — go.dev/blog/pprof](https://go.dev/blog/pprof)
- [Diagnostics — go.dev/doc/diagnostics](https://go.dev/doc/diagnostics)
- [`go tool pprof` — github.com/google/pprof/blob/main/doc/README.md](https://github.com/google/pprof/blob/main/doc/README.md)
