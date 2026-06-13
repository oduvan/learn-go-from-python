# Керування потоком виконання

Три ключових слова покривають майже все: `if`, `for`, `switch`.

## `if`

Фігурні дужки **обов'язкові**. Дужок навколо умови немає.

```go
if x > 0 {
    fmt.Println("positive")
} else if x < 0 {
    fmt.Println("negative")
} else {
    fmt.Println("zero")
}
```

Умова **має бути типу `bool`** — немає «правдивих» та «хибних» значень. Дивіться [02-basic-types.md](02-basic-types.md).

### `if` з ініціалізатором

Поширена ідіома Go: оголошення змінної, видимої лише в ланцюжку `if`/`else if`/`else`.

```go
if n, err := strconv.Atoi(s); err == nil {
    fmt.Println("parsed:", n)
} else {
    fmt.Println("bad input:", err)
}
// n та err тут НЕ видимі.
```

Це тримає короткоживучі імена поза зовнішньою областю видимості. Канонічний спосіб обробляти помилки від одного виклику функції.

## `for` — єдиний цикл

У Go немає `while` і немає `do…while`. Ключове слово `for` охоплює все.

### Трикомпонентна форма (класичний стиль C)

```go
for i := 0; i < 5; i++ {
    fmt.Println(i)
}
```

### Форма «лише умова» (`while` у Go)

```go
n := 100
for n > 1 {
    n /= 2
}
```

### Нескінченний цикл

```go
for {
    if shouldStop() {
        break
    }
}
```

### Форма `range` — перебір

`for ... range` перебирає зрізи, масиви, рядки, мапи, канали.

```go
nums := []int{10, 20, 30}

for i, v := range nums {
    fmt.Println(i, v)
}
// 0 10
// 1 20
// 2 30

for _, v := range nums {        // ігноруємо індекс
    fmt.Println(v)
}

for i := range nums {           // ігноруємо значення
    fmt.Println(i)
}

for range nums {                // просто рахуємо ітерації
    fmt.Println("tick")
}
```

Мапи повертають `(ключ, значення)` — але **порядок випадковий** при кожному переборі:

```go
m := map[string]int{"a": 1, "b": 2, "c": 3}
for k, v := range m {
    fmt.Println(k, v)        // порядок може відрізнятися від запуску до запуску
}
```

Рядки повертають `(byteIndex, rune)`:

```go
for i, r := range "hi世" {
    fmt.Printf("%d %c\n", i, r)
}
// 0 h
// 1 i
// 2 世
```

### Range за цілим числом

```go
for i := range 5 {
    fmt.Println(i)       // 0 1 2 3 4
}
```

### Range за функцією

Деякі функції стандартної бібліотеки повертають *функцію-ітератор* (технічно `iter.Seq[T]` з пакету `iter`), по якій можна безпосередньо робити `range`. Наприклад: `strings.Lines` повертає кожен рядок рядкового значення, включно з завершальним символом нового рядка.

```go
import "strings"

s := "alpha\nbeta\ngamma\n"
for line := range strings.Lines(s) {
    fmt.Printf("%q\n", line)
}
// output:
// "alpha\n"
// "beta\n"
// "gamma\n"
```

Власні функції-ітератори ви можете писати самостійно — це розглядається в наступних темах.

> **З досвіду Python:** `for ... range` охоплює `for x in seq:`, `for i, x in enumerate(seq):`, `for k, v in dict.items():`. Цілочисельна форма `for i := range 5` відповідає `for i in range(5):`.

### `break` та `continue`

```go
for i, v := range data {
    if v < 0 {
        continue            // пропустити цю ітерацію
    }
    if v > 1000 {
        break               // вийти з циклу
    }
    process(i, v)
}
```

### Мітки — вихід із вкладених циклів

```go
Outer:
for i := 0; i < 10; i++ {
    for j := 0; j < 10; j++ {
        if data[i][j] == target {
            fmt.Println("found at", i, j)
            break Outer        // виходимо з обох циклів
        }
    }
}
```

`continue Outer` переходить до наступної ітерації зовнішнього циклу.

## `switch`

### Switch за виразом

```go
switch x {
case 1:
    fmt.Println("one")
case 2, 3:                  // кілька значень в одному case
    fmt.Println("two or three")
case 4:
    fmt.Println("four")
default:
    fmt.Println("other")
}
```

**Неявного fallthrough немає.** Кожна гілка `case` завершується неявним `break`. Якщо потрібен fallthrough у стилі C, його пишуть явно:

```go
switch x {
case 1:
    fmt.Println("one")
    fallthrough
case 2:
    fmt.Println("one or two")
}
```

### «Безтегова» форма switch — замінює ланцюжки `if/else if`

```go
switch {
case x < 0:
    fmt.Println("negative")
case x == 0:
    fmt.Println("zero")
case x > 0:
    fmt.Println("positive")
}
```

Ідіоматично для Go — краще за довгі ланцюжки `if/else if/else if`.

### `switch` з ініціалізатором

Та сама схема, що й у `if`:

```go
switch n := len(s); {
case n == 0:
    fmt.Println("empty")
case n < 10:
    fmt.Println("short")
default:
    fmt.Println("long")
}
```

### Type switch

Для значень інтерфейсу — детально розглядається пізніше, але варто побачити вже зараз. `any` — вбудований псевдонім для «будь-якого типу» (так функція каже «прийму що завгодно»):

```go
func describe(x any) {
    switch v := x.(type) {
    case int:
        fmt.Println("int:", v)
    case string:
        fmt.Println("string of length", len(v))
    case nil:
        fmt.Println("nil")
    default:
        fmt.Printf("unknown type %T\n", v)
    }
}
```

## `goto`

Існує, має звичайні обмеження (не можна перестрибувати через оголошення змінних, не можна переходити всередину блоку). Майже ніколи не використовується поза згенерованим кодом або вузькоспеціалізованими реалізаціями скінченних автоматів. Не вживайте його безпідставно.

## Чого немає порівняно з Python

| Python | Еквівалент у Go |
|---|---|
| `while cond:` | `for cond { ... }` |
| `else` на циклі `for` | немає аналога — потрібно переструктурувати |
| `match`/`case` (PEP 634) | `switch` (схоже, але простіше) |
| `try`/`except`/`finally` | виключень немає — дивіться обробку помилок далі; `defer` покриває cleanup у стилі `finally` |
| генератори списків | немає — пишіть цикл `for` |
| `pass` | не потрібен — порожній блок `{}` цілком допустимий |

## Джерела

- [Оператори if — go.dev/ref/spec#If_statements](https://go.dev/ref/spec#If_statements)
- [Оператори for — go.dev/ref/spec#For_statements](https://go.dev/ref/spec#For_statements)
- [Оператори switch — go.dev/ref/spec#Switch_statements](https://go.dev/ref/spec#Switch_statements)
- [Break/continue/мічені оператори — go.dev/ref/spec#Break_statements](https://go.dev/ref/spec#Break_statements)
- [Range за цілим числом (нотатки до релізу Go 1.22) — go.dev/doc/go1.22#language](https://go.dev/doc/go1.22#language)
- [Range за функцією (нотатки до релізу Go 1.23) — go.dev/doc/go1.23#language](https://go.dev/doc/go1.23#language)
