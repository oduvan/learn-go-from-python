# Хелпери, фікстури та golden-файли

Усе навколо самої перевірки: спільне налаштування, що зрозуміло
повідомляє про провал, тимчасові файли, що самі за собою прибираються,
і порівняння з раніше записаним очікуваним виводом.

```go
func mustTempFile(t *testing.T, content string) string {
    t.Helper()
    p := filepath.Join(t.TempDir(), "f.txt")
    if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
        t.Fatalf("writing temp file: %v", err)
    }
    return p
}
```

## `t.Helper` виправляє повідомлений номер рядка

Без нього провал усередині хелпера повідомляється з рядка *в самому
хелпері*, який однаковий для кожного виклику й нічого не каже.
`t.Helper()` позначає функцію, тож провал приписується виклику замість
неї.

Викликайте його **першим оператором** — і в кожному хелпері, включно з
тими, що лише викликають інші хелпери.

Хелпер має приймати `*testing.T` першим параметром і провалюватися
через `t.Fatalf`, а не повертати помилку. Це те єдине місце, де
звичайне правило "повертайте помилки" перевертається: провал
допоміжної функції тесту означає, що тест не може продовжуватись, а
змушувати кожен виклик писати `if err != nil` лише ховає суть тесту.

## `t.TempDir`

Повертає свіжу директорію й автоматично реєструє її видалення:

```go
dir := t.TempDir()
```

Вона належить окремому тесту — підтест, що її викликає, отримує свою
власну, яка видаляється по завершенню *саме цього підтесту*:

```go
var saved string
t.Run("inner", func(t *testing.T) {
    saved = t.TempDir()   // exists here
})
// by now it is gone
```

Це робить її безпечною з `t.Parallel`, де спільна директорія призвела
б до того, що випадки заважали б один одному. Ніколи не
використовуйте фіксований шлях на кшталт `/tmp/mytest`; паралельні
прогони та `-count=2` спричинять конфлікт.

## `t.Cleanup` виконується в порядку "останній зареєстрований — перший виконаний"

```go
t.Cleanup(func() { order = append(order, "first registered") })
t.Cleanup(func() { order = append(order, "second registered") })
// cleanup order: [second registered first registered]
```

Той самий порядок, що й у `defer`, але прив'язаний до *тесту*, а не до
функції, що його оточує — тож хелпер може зареєструвати прибирання для
роботи, яку він сам розпочав, чого `defer` усередині цього хелпера
зробити не міг би.

Виконується і при провалі, і при паніці, що робить і `defer` у тілі
тесту. Причина віддавати перевагу `t.Cleanup` — це саме випадок
хелпера й сумісність із `t.Parallel`.

## `testdata/`

Інструмент `go` ігнорує будь-яку директорію з ім'ям `testdata`, тож
файли-фікстури живуть саме там і ніколи не компілюються й не
трактуються як пакет:

```
store/
  store.go
  store_test.go
  testdata/
    input.json
    expected.json
```

Шляхи в тесті відносні до **директорії пакета**, бо `go test` запускає
тести кожного пакета з нею як робочою директорією. Тож
`os.ReadFile("testdata/input.json")` працює незалежно від того, звідки
ви викликали `go test`.

## Golden-файли

Коли очікуваний вивід великий — відрендерений HTML, форматований
звіт, JSON-документ — вставлення його прямо в код робить тест
нечитабельним. Запишіть його у файл і порівнюйте:

```go
var update = flag.Bool("update", false, "rewrite golden files")

func assertGolden(t *testing.T, name, got string) {
    t.Helper()
    p := filepath.Join("testdata", name+".golden")

    if *update {
        if err := os.WriteFile(p, []byte(got), 0o644); err != nil {
            t.Fatalf("updating golden: %v", err)
        }
        return
    }

    want, err := os.ReadFile(p)
    if err != nil {
        t.Fatalf("reading golden (run with -update to create): %v", err)
    }
    if got != string(want) {
        t.Errorf("golden mismatch for %s\n got: %q\nwant: %q", name, got, string(want))
    }
}
```

```go
func TestGolden(t *testing.T) {
    assertGolden(t, "greeting", Render("GOLDEN"))
}
```

Перегенеруйте після навмисної зміни:

```bash
go test ./... -update
```

А потім **прочитайте diff перед тим, як його комітити**. У цьому й
полягає весь ризик golden-файлів: `-update` пропускає будь-яку зміну,
тож неперевірена регенерація мовчки благословляє баг. Файли мають бути
під контролем версій, а diff має бути частиною рев'ю.

`flag.Bool` на рівні пакета в тестовому файлі — це спосіб додати прапор
до `go test`; пакет `testing` парсить його разом зі своїми власними.

Ще два правила. Golden-файли мають бути **детермінованими** — жодних
позначок часу, жодного порядку ітерації мапи, жодних випадкових
ідентифікаторів. Нормалізуйте їх перед порівнянням, інакше отримаєте
тест, що провалюється по вівторках. І тримайте їх читабельними: diff
golden-файлу корисний, лише якщо людина бачить, що саме змінилося.

## Коли звичайного порівняння недостатньо

`==` працює для рядків і порівнюваних структур. Для мап і зрізів
використовуйте `slices.Equal` і `maps.Equal`; `reflect.DeepEqual` теж
працює, але трактує nil-зріз і порожній зріз як різні, що рідко є тим,
що мається на увазі в тесті — про це йшлося у статті
[XML, CSV та рефлексія](../06-text-time-and-data/08-xml-csv-and-reflection.md).

Для JSON порівнюйте *декодовані* значення, а не текст. Порядок ключів і
пробіли не є частиною того, що ви тестуєте.

> **З досвіду Python:** `t.TempDir` — це фікстура `tmp_path`, а
> `t.Cleanup` — `addfinalizer`, але без ін'єкції фікстур — хелпери є
> звичайними функціями, які ви викликаєте самі. `t.Helper` —
> еквівалент `__tracebackhide__`. Golden-файли — це `pytest-regressions`,
> написаний вручну за п'ятнадцять рядків.

## Швидка довідка

| Задача | Форма |
|---|---|
| спільне налаштування | хелпер, що приймає `*testing.T` першим |
| правильний рядок провалу | `t.Helper()` першим оператором |
| тимчасова директорія | `t.TempDir()` — на кожен тест, видаляється автоматично |
| прибирання | `t.Cleanup(fn)` — LIFO, переживає хелпери й `t.Parallel` |
| файли-фікстури | `testdata/`, ігнорується інструментом `go` |
| робоча директорія | завжди директорія пакета |
| великий очікуваний вивід | `.golden`-файл під `testdata/` |
| перегенерувати | пакетний `flag.Bool("update", ...)` |
| небезпека | рев'юйте golden diff; `-update` пропускає будь-що |
| порівняння колекцій | `slices.Equal` / `maps.Equal` |

## Джерела

- [`testing.T.Helper` — pkg.go.dev/testing#T.Helper](https://pkg.go.dev/testing#T.Helper)
- [`testing.T.TempDir` — pkg.go.dev/testing#T.TempDir](https://pkg.go.dev/testing#T.TempDir)
- [`testing.T.Cleanup` — pkg.go.dev/testing#T.Cleanup](https://pkg.go.dev/testing#T.Cleanup)
- [Package names and testdata — pkg.go.dev/cmd/go#hdr-Package_lists_and_patterns](https://pkg.go.dev/cmd/go#hdr-Package_lists_and_patterns)
- [`flag` package reference — pkg.go.dev/flag](https://pkg.go.dev/flag)
