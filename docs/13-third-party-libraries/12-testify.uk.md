# testify

Твердження (assertions) з читабельними повідомленнями про провал. Пакет
[`testing`](../10-testing/01-the-testing-package.md) в Go дає вам `if`
і `t.Errorf`; testify дає один рядок на перевірку і diff, коли вона
провалюється.

> **Модуль:** `github.com/stretchr/testify`.

```go
require.NoError(t, err)
assert.Equal(t, "Ada", u.Name)
```

## `assert` не зупиняє нічого, `require` зупиняє все

Два пакети, однакові назви функцій, одна відмінність:

| Пакет | При провалі |
|---|---|
| `assert` | записує й **продовжує** — як `t.Errorf` |
| `require` | записує й **зупиняє тест** — як `t.Fatalf` |

Звідси й правило: `require` для всього, від чого залежить решта тесту,
`assert` — для незалежних перевірок.

```go
u, err := Load("1")
require.NoError(t, err)          // stop here if it failed
require.Equal(t, "Ada", u.Name)

assert.Len(t, u.Tags, 2)         // independent checks
assert.Contains(t, u.Tags, "a")
```

Використання `assert.NoError` там, де належить `require.NoError` —
поширена помилка: тест продовжується до розіменування nil, а паніка
ховає справжнє повідомлення.

## Порядок аргументів

```go
assert.Equal(t, expected, actual)
```

Спочатку `t`, потім **очікуване, потім фактичне**. Переплутайте
порядок — і тест усе одно пройде чи провалиться правильно, але diff
перевернутий і читається як брехня.

## Як виглядає провал

```
Error Trace:	lib_test.go:27
Error:      	Not equal:
            	expected: "Bob"
            	actual  : "Ada"

            	Diff:
            	--- Expected
            	+++ Actual
            	@@ -1 +1 @@
            	-Bob
            	+Ada
Test:       	TestFailureOutput
Messages:   	name should match the seeded value
```

Ось причина її використовувати. На структурі чи зрізі diff показує
поле, що відрізняється, замість того, щоб виводити обидва значення й
залишати порівняння вам.

Кожне твердження бере опціональне завершальне повідомлення:

```go
assert.Equal(t, "Ada", u.Name, "name should match the seeded value")
```

Варто додавати, коли саме твердження не каже, *чому* це важливо.

## Твердження, якими ви користуватиметесь насправді

```go
require.NoError(t, err)
require.Error(t, err)
assert.ErrorIs(t, err, ErrEmpty)      // unwraps, like errors.Is
assert.ErrorAs(t, err, &target)

assert.Equal(t, want, got)
assert.NotEqual(t, a, b)
assert.Nil(t, v)
assert.True(t, ok)

assert.Len(t, u.Tags, 2)
assert.Empty(t, list)
assert.Contains(t, u.Tags, "a")
assert.ElementsMatch(t, []string{"b", "a"}, u.Tags)   // ignores order

assert.InDelta(t, 3.14159, 3.1416, 0.001)             // floats
assert.JSONEq(t, `{"a":1,"b":2}`, `{"b":2,"a":1}`)    // ignores key order
```

`ErrorIs` — те, до чого тягнутися для помилок — він розгортає, тож
працює й через `fmt.Errorf("...: %w", err)`.

`ElementsMatch` і `JSONEq` прибирають цілих дві категорії нестабільних
тестів: порядок у зрізі й порядок ключів у JSON — жодне з яких ви
зазвичай не мали намір перевіряти.

`InDelta` існує тому, що `assert.Equal` на числах з плаваючою комою
порівнює точно, а точне порівняння чисел з плаваючою комою — це баг.

## `Equal` розрізняє nil і порожнє

`assert.Equal` використовує семантику `reflect.DeepEqual`, тому:

```go
var a []string
b := []string{}
assert.Equal(t, a, b)
```

```
Error: Not equal:
       expected: []string(nil)
       actual  : []string{}
```

nil-зріз і порожній зріз поводяться однаково в кожній операції Go —
`len`, `range`, `append` — і відрізняються тут. Коли ви маєте на увазі
"жодних елементів", використовуйте `assert.Empty`, який приймає обидва
варіанти.

Те саме стосується мап. Це постійно ловить людей при порівнянні
декодованої структури JSON з літералом.

## Не викликайте `require` з горутини

`require` провалюється через `t.FailNow`, що викликає `runtime.Goexit`.
З горутини (goroutine), яку запустив ваш тест, це вбиває лише цю
горутину — тест не провалюється правильно й може зависнути в очікуванні
на неї.

Надішліть результат назад у горутину тесту й перевіряйте там, або
використовуйте `assert` (яка лише записує) всередині горутини.

## Решта бібліотеки

`testify/mock` генерує моки на основі очікувань, а `testify/suite`
додає класи налаштування й завершення в стилі xUnit. Багато кодових баз
на Go використовують лише `assert` і `require` і нічого більше, з
причин зі статті про [фейки та заглушки](../10-testing/04-fakes-and-stubs.md):
написані вручну дублікати зрозуміліші за `EXPECT().Times(1)`, а
`t.Cleanup` покриває те, що робило б завершення в suite.

Спершу засвойте два пакети тверджень. Додавайте решту, лише якщо
натрапите на щось, що вони справді розв'язують.

## Чи варта вона залежності

Чесний аргумент проти: стандартна бібліотека може все це, а бібліотеки
тверджень мають звичку розростатись, доки тести не пишуться DSL-ем
замість Go.

Аргумент за: вивід diff, і те, що `require.NoError(t, err)` — один
рядок там, де звичайна версія — три. На великому наборі тестів це
реальна відмінність у тому, скільки сигналу несе кожен тест.

Це також майже загальноприйнято в кодових базах на Go, тож ідіому
знає більшість читачів.

> **З досвіду Python:** це переписування тверджень `pytest`, зроблене
> явним — ви викликаєте `assert.Equal` замість `assert x == y` й
> отримуєте порівнянний diff. `require` проти `assert` — відмінність,
> якої в `pytest` немає, оскільки там кожне провалене твердження
> зупиняє тест.

## Швидка довідка

| Потреба | Виклик |
|---|---|
| провалитись і зупинитись | `require.X(t, ...)` |
| провалитись і продовжити | `assert.X(t, ...)` |
| порядок | `(t, expected, actual)` |
| без помилки | `require.NoError(t, err)` |
| конкретна помилка | `assert.ErrorIs(t, err, ErrX)` — розгортає |
| довжина / належність | `assert.Len`, `assert.Contains` |
| зріз без урахування порядку | `assert.ElementsMatch` |
| JSON | `assert.JSONEq` |
| числа з плаваючою комою | `assert.InDelta` |
| nil *або* порожнє | `assert.Empty` — **не** `Equal` |
| усередині горутини | ніколи `require`; надсилайте результат назад |

## Джерела

- [testify — pkg.go.dev/github.com/stretchr/testify](https://pkg.go.dev/github.com/stretchr/testify)
- [`assert` — pkg.go.dev/github.com/stretchr/testify/assert](https://pkg.go.dev/github.com/stretchr/testify/assert)
- [`require` — pkg.go.dev/github.com/stretchr/testify/require](https://pkg.go.dev/github.com/stretchr/testify/require)
- [`testing.T.FailNow` — pkg.go.dev/testing#T.FailNow](https://pkg.go.dev/testing#T.FailNow)
