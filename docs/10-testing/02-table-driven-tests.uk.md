# Табличні тести

Домінантний стиль тестів у Go: зріз (slice) випадків, один цикл, один
підтест на кожен. Саме так тестується стандартна бібліотека, і
додавання нового випадку коштує один рядок.

```go
tests := []struct {
    name string
    in   string
    want string
}{
    {"trims and lowers", "  Go  ", "go"},
    {"already fine", "go", "go"},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        if got := Normalize(tt.in); got != tt.want {
            t.Errorf("got %q, want %q", got, tt.want)
        }
    })
}
```

## Анонімна структура

Тип випадку оголошується прямо в коді, бо існує лише для цього тесту.
Імена полів роблять кожен випадок читабельним у місці виклику, і поле
`name` за домовленістю йде першим.

Тримайте одну поведінку на таблицю. Якщо половині випадків потрібен
інший набір полів — це два тести, замасковані під один; розділіть їх.

## `t.Run` дає кожному випадку власний тест

Підтести — не косметика. Кожен із них:

- **має ім'я**, тож провал повідомляє, який саме випадок зламався, а
  не який рядок;
- **провалюється незалежно**, тож `t.Fatalf` в одному випадку не
  ховає решту;
- **можна запустити окремо** через `-run`.

```
=== RUN   TestNormalizeTable
=== RUN   TestNormalizeTable/trims_and_lowers
=== RUN   TestNormalizeTable/already_fine
--- PASS: TestNormalizeTable (0.00s)
    --- PASS: TestNormalizeTable/trims_and_lowers (0.00s)
    --- PASS: TestNormalizeTable/already_fine (0.00s)
```

Зауважте, що **пробіли в імені стають підкресленнями**. Саме таке ім'я
ви передаєте в `-run`:

```bash
go test -run 'TestNormalizeTable/trims_and_lowers' ./...
```

Патерн — це regex для кожного сегмента шляху, тож `-run 'TestX/.*empty'`
теж працює.

## Тестування помилок у таблиці

Покладіть очікувану помилку в кейс і порівнюйте через `errors.Is`:

```go
tests := []struct {
    name    string
    in      string
    want    string
    wantErr error
}{
    {"trims and lowers", "  Go  ", "go", nil},
    {"empty", "   ", "", ErrEmpty},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := Normalize(tt.in)
        if !errors.Is(err, tt.wantErr) {
            t.Fatalf("err = %v, want %v", err, tt.wantErr)
        }
        if got != tt.want {
            t.Errorf("got %q, want %q", got, tt.want)
        }
    })
}
```

`errors.Is(nil, nil)` дорівнює `true`, тож успішні випадки працюють із
тим самим порівнянням — окреме поле `wantErr bool` не потрібне.

Не порівнюйте *тексти* помилок. Формулювання повідомлення не є API;
API — це сигнальна помилка (sentinel) або тип, у чому й полягає суть
[власних типів помилок](../03-object-oriented-go/06-custom-error-types.md).

## `t.Parallel`

Додавання одного рядка запускає випадки одночасно:

```go
t.Run(tt.name, func(t *testing.T) {
    t.Parallel()
    // ...
})
```

```
=== RUN   TestParallelSubtests/lower
=== PAUSE TestParallelSubtests/lower
=== RUN   TestParallelSubtests/already
=== PAUSE TestParallelSubtests/already
=== CONT  TestParallelSubtests/lower
=== CONT  TestParallelSubtests/already
```

Пари `PAUSE`/`CONT` показують механізм: паралельний підтест ставиться
на паузу, батьківський тест дочікується запуску всіх підтестів, а
потім вони відновлюються разом.

Звідси випливають два наслідки, і обидва важливі:

**Батьківський тест завершується раніше за дочірні.** Код після циклу
виконується, поки підтести ще тривають. Усе, що потрібне підтестам,
має прибиратися через `t.Cleanup`, а не через `defer` у батьківському
тесті.

**Випадки не повинні ділити змінний стан.** Паралельні випадки, що
торкаються однієї й тієї самої мапи чи лічильника — це гонитва даних
(data race). Запустіть `-race` у CI, і вона виявиться; без нього тест
проходить, аж поки не перестане.

Паралелізм окупається, коли випадки роблять I/O. Для чистих функцій
він лише додає шум від планувальника без жодної користі — не додавайте
його рефлекторно.

## Тримайте цикл тупим

Тіло циклу має робити те саме для кожного випадку. Щойно в ньому
з'являється `if tt.special { ... }`, таблиця перестає описувати дані й
починає кодувати керування потоком виконання, і окремий тест буде
зрозумілішим.

Поле-функція — чесний спосіб урізноманітнити поведінку:

```go
tests := []struct {
    name  string
    setup func(*testing.T) *Store
    want  int
}{
    {"empty", func(t *testing.T) *Store { return NewStore() }, 0},
    {"seeded", seededStore, 3},
}
```

Це зберігає цикл однорідним, дозволяючи кожному випадку самостійно
себе налаштувати.

> **З досвіду Python:** це `@pytest.mark.parametrize`, написаний
> вручну. Багатослівніше, але випадки — звичайні значення, які можна
> будувати, фільтрувати чи генерувати в Go, а поле `name` робить те
> саме, що згенеровані ідентифікатори `pytest` — тільки ви обираєте
> його самі.

## Швидка довідка

| Питання | Форма |
|---|---|
| таблиця | зріз вбудованої анонімної структури, `name` першим |
| один підтест на випадок | `t.Run(tt.name, func(t *testing.T) { ... })` |
| запустити один випадок | `-run 'TestX/case_name'` — пробіли стають підкресленнями |
| очікувані помилки | поле `wantErr error` плюс `errors.Is` |
| паралельність | `t.Parallel()` усередині підтесту |
| прибирання з паралельними випадками | `t.Cleanup`, ніколи не `defer` у батьківському тесті |
| випадок, що відрізняється | поле `setup func(*testing.T)`, а не `if` |

## Джерела

- [`testing.T.Run` — pkg.go.dev/testing#T.Run](https://pkg.go.dev/testing#T.Run)
- [`testing.T.Parallel` — pkg.go.dev/testing#T.Parallel](https://pkg.go.dev/testing#T.Parallel)
- [Go blog: subtests and sub-benchmarks — go.dev/blog/subtests](https://go.dev/blog/subtests)
- [Go wiki: table-driven tests — go.dev/wiki/TableDrivenTests](https://go.dev/wiki/TableDrivenTests)
- [`errors.Is` — pkg.go.dev/errors#Is](https://pkg.go.dev/errors#Is)
