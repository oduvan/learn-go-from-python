# Час

У пакеті `time` людей дивують три речі: макети (layouts) записуються як
приклад конкретної дати, а не як `%Y-%m-%d`, тривалість — це справжній
тип, над яким можна робити арифметику, і `==` — неправильний спосіб
порівняти два моменти часу.

```go
t := time.Date(2026, time.September, 20, 15, 4, 5, 0, time.UTC)

fmt.Println(t.Format("2006-01-02 15:04:05"))   // output: 2026-09-20 15:04:05
```

## Еталонний макет

`2006-01-02 15:04:05` — це не патерн з підстановочними символами. Це
конкретний момент — **Mon Jan 2 15:04:05 MST 2006** — і ви записуєте
макет у тій формі, яку хочете отримати. У кожного компонента одне число:

| Компонент | Значення | Чому |
|---|---|---|
| місяць | `1` або `01` або `Jan` або `January` | перший |
| день | `2` або `02` | другий |
| година | `15` (24-годинний) або `3`/`03` (12-годинний) | третій |
| хвилина | `4` або `04` | четвертий |
| секунда | `5` або `05` | п'ятий |
| рік | `2006` або `06` | шостий |
| часовий пояс | `-0700` або `Z07:00` | сьомий |

Мнемоніка — порядок 1, 2, 3, 4, 5, 6, 7. Отже:

```go
fmt.Println(t.Format("2006-01-02"))              // output: 2026-09-20
fmt.Println(t.Format("02/01/2006"))              // output: 20/09/2026
fmt.Println(t.Format("Jan 2, 2006 at 3:04PM"))   // output: Sep 20, 2026 at 3:04PM
```

Зверніть увагу на `15` проти `3`: `15` просить 24-годинний формат, `3` —
12-годинний. Окремого прапорця для цього немає.

Пакет постачає поширені макети як константи — надавайте їм перевагу:

```go
fmt.Println(t.Format(time.RFC3339))   // output: 2026-09-20T15:04:05Z
fmt.Println(t.Format(time.Kitchen))   // output: 3:04PM
```

`RFC3339` — той, що варто використовувати для всього, що потім читатиме
машина.

## Парсинг використовує той самий макет

```go
p, err := time.Parse("2006-01-02", "2026-09-20")
fmt.Println(p.Format(time.RFC3339), err)
// output: 2026-09-20T00:00:00Z <nil>
```

Компоненти, яких немає, за замовчуванням дорівнюють нулю, тож парсинг
лише дати дає опівніч за UTC. Коли вхідні дані не збігаються з макетом,
помилка називає обидва боки, тож баги в макеті легко помітити:

```go
_, err = time.Parse("2006-01-02", "20/09/2026")
fmt.Println(err)
// output: parsing time "20/09/2026" as "2006-01-02": cannot parse "20/09/2026" as "2006"
```

Використовуйте `time.ParseInLocation`, коли у вхідних даних немає
зміщення, але ви знаєте, у якому поясі їх записали — звичайний `Parse`
припускає UTC.

## Тривалості — це тип, а не число

`time.Duration` — це `int64`-лічильник наносекунд з методом `String`,
тож він друкується читабельно й підтримує арифметику напряму:

```go
d := 90 * time.Minute
fmt.Println(d)                         // output: 1h30m0s
fmt.Println(d.Hours(), d.Minutes())    // output: 1.5 90

fmt.Println(2*time.Hour + 30*time.Minute)   // output: 2h30m0s
```

`90 * time.Minute` працює, бо `90` — це нетипізована константа, як
пояснює стаття про [конвертацію
типів](../02-language-basics/03-type-conversions.md). Якби `90` було
*змінною*, це не скомпілювалося б — знадобиться
`time.Duration(n) * time.Minute`:

```go
fmt.Println(time.Duration(1500) * time.Millisecond)   // output: 1.5s
```

Тривалості також парсяться з рядків, і саме так вони приходять з
конфігурації:

```go
pd, _ := time.ParseDuration("1h30m")
fmt.Println(pd, pd == 90*time.Minute)   // output: 1h30m0s true

fmt.Println(d.Round(time.Hour), d.Truncate(time.Hour))
// output: 2h0m0s 1h0m0s
```

## Арифметика

`Add` приймає тривалість; `Sub` повертає тривалість; `AddDate` працює в
календарних одиницях:

```go
later := t.Add(48 * time.Hour)
fmt.Println(later.Format(time.RFC3339))       // output: 2026-09-22T15:04:05Z
fmt.Println(later.Sub(t))                     // output: 48h0m0s

fmt.Println(t.AddDate(0, 1, 0).Format("2006-01-02"))   // output: 2026-10-20
```

`time.Since(start)` — це скорочення для `time.Now().Sub(start)`, і саме
так зазвичай вимірюють, скільки часу щось зайняло.

## `Add` і `AddDate` розходяться на межі переходу на літній/зимовий час

Саме тому існують обидва методи. «Додати один день» і «додати 24 години»
— різні питання, і в ніч переведення годинників вони дають різні
відповіді:

```go
loc, _ := time.LoadLocation("America/New_York")
before := time.Date(2026, time.March, 7, 12, 0, 0, 0, loc)

fmt.Println(before.AddDate(0, 0, 1).Format(time.RFC3339))
// output: 2026-03-08T12:00:00-04:00

fmt.Println(before.Add(24 * time.Hour).Format(time.RFC3339))
// output: 2026-03-08T13:00:00-04:00
```

`AddDate` зберігає час на стінному годиннику — і далі опівдень. `Add`
просуває реальний минулий час, а оскільки та неділя коротша на годину,
опівдень плюс 24 години потрапляє на першу годину дня. Використовуйте
`AddDate` для «завтра», `Add` для «через 24 години».

## Часові пояси

`time.Time` несе в собі локацію. `In` змінює, як значення
*відображається*, не змінюючи сам момент:

```go
ny, _ := time.LoadLocation("America/New_York")

fmt.Println(t.In(ny).Format(time.RFC3339))   // output: 2026-09-20T11:04:05-04:00
fmt.Println(t.UTC().Format(time.RFC3339))    // output: 2026-09-20T15:04:05Z
```

`LoadLocation` читає системну базу часових поясів, тож може зазнати
невдачі на мінімальному образі контейнера. `time.UTC` і `time.Local`
завжди працюють.

## Порівнюйте через `Equal`, не через `==`

`time.Time` — це структура, що містить час на стінному годиннику,
монотонне значення та вказівник на локацію. `==` порівнює все це, тож
той самий момент у двох поясах не буде `==`:

```go
a := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
b := a.In(ny)

fmt.Println(a == b)        // output: false
fmt.Println(a.Equal(b))    // output: true
```

Завжди використовуйте `Equal`, `Before` і `After`. `==` для `time.Time`
компілюється й майже завжди є багом — та сама пастка, про яку попереджає
стаття про [структури](../02-language-basics/10-structs.md) щодо
порівнюваних структур узагалі.

Нульове значення — це рік 1, і `IsZero` — спосіб сказати «не встановлено»:

```go
var zero time.Time
fmt.Println(zero.IsZero(), zero.Format(time.RFC3339))
// output: true 0001-01-01T00:00:00Z
```

Саме тому «відсутню» часову позначку зазвичай оголошують як
`*time.Time`: нульове значення — це справжня дата, а не відсутність
значення.

## Монотонні показники

`time.Now()` також записує показник монотонного годинника, яким
автоматично користуються `Sub`, `Since`, `Before` і `After`. Це означає,
що вимірювання минулого часу лишається коректним, навіть якщо системний
годинник скоригували під час вимірювання. Цей показник відкидається
методами `Round`, `Truncate`, `UTC` та `In` — тож спершу вимірюйте, а вже
потім конвертуйте.

> **З досвіду Python:** `Format`/`Parse` замінюють `strftime`/`strptime`,
> але макет — це приклад, а не набір кодів із `%`. `time.Duration` — це
> `timedelta` з арифметикою, яку зручніше читати. Звичку, яку варто
> позбутися, — це `==` для часових позначок: `datetime` у Python
> порівнює за моментом часу, а порівняння структур у Go — ні.

## Швидка довідка

| Задача | Виклик |
|---|---|
| поточний момент | `time.Now()` |
| побудувати момент | `time.Date(y, time.Month, d, h, m, s, ns, loc)` |
| форматувати | `t.Format(time.RFC3339)` або `t.Format("2006-01-02")` |
| розпарсити | `time.Parse(layout, s)`, `time.ParseInLocation` |
| минулий час | `time.Since(start)` |
| додати години | `t.Add(24 * time.Hour)` |
| додати календарні дні | `t.AddDate(0, 0, 1)` |
| різниця | `b.Sub(a)` → `time.Duration` |
| порівняти | `Equal`, `Before`, `After` — **ніколи `==`** |
| тривалість з конфігурації | `time.ParseDuration("1h30m")` |
| тривалість зі змінної | `time.Duration(n) * time.Second` |
| змінити пояс відображення | `t.In(loc)`, `t.UTC()` |
| «не встановлено» | `t.IsZero()` або поле `*time.Time` |

## Джерела

- [`time` package reference — pkg.go.dev/time](https://pkg.go.dev/time)
- [`time.Time.Format` — pkg.go.dev/time#Time.Format](https://pkg.go.dev/time#Time.Format)
- [Layout constants — pkg.go.dev/time#pkg-constants](https://pkg.go.dev/time#pkg-constants)
- [`time.Duration` — pkg.go.dev/time#Duration](https://pkg.go.dev/time#Duration)
- [`time.Time.Equal` — pkg.go.dev/time#Time.Equal](https://pkg.go.dev/time#Time.Equal)
- [Monotonic clocks — pkg.go.dev/time#hdr-Monotonic_Clocks](https://pkg.go.dev/time#hdr-Monotonic_Clocks)
