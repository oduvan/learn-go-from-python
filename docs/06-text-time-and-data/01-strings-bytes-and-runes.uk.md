# Рядки, байти та руни

Стаття про [базові типи](../02-language-basics/02-basic-types.md)
встановила, що таке рядок: незмінна послідовність байтів, зазвичай у
кодуванні UTF-8. Ця стаття — про пакети, до яких ви насправді звертаєтесь —
`strings`, `bytes` і `unicode/utf8` — і про місця, де різниця між байтом і
символом дає про себе знати.

```go
s := "  Name: Ada Lovelace  "
fmt.Printf("%q\n", strings.TrimSpace(s))   // output: "Name: Ada Lovelace"
```

## Пошук

```go
fmt.Println(strings.Contains("chicken", "ken"))    // output: true
fmt.Println(strings.HasPrefix("chicken", "chi"))   // output: true
fmt.Println(strings.Index("chicken", "ken"))       // output: 4
fmt.Println(strings.Index("chicken", "zz"))        // output: -1
```

`Index` повертає зміщення в **байтах** і `-1`, якщо збігу немає. Вона
ніколи не завершується помилкою, тож обробляти нічого не потрібно.

## `Cut` — зазвичай те, що потрібно

Розбити "key: value" на дві частини — настільки поширена задача, що для
неї є окрема функція, яка ще й повідомляє, чи роздільник узагалі був:

```go
key, val, found := strings.Cut("Name: Ada", ": ")
fmt.Printf("%q %q %v\n", key, val, found)   // output: "Name" "Ada" true

_, _, found = strings.Cut("noseparator", ": ")
fmt.Println(found)                          // output: false
```

Без результату `found` неможливо відрізнити «роздільника немає» від
«значення було порожнім» — а саме цей баг `Cut` і покликана запобігти.

## `Split` і `Fields` — це не одне й те саме

`Split` розрізає за точним роздільником і зберігає порожні шматки.
`Fields` розбиває за послідовностями пробільних символів і відкидає їх:

```go
fmt.Printf("%q\n", strings.Split("a,b,,c", ","))   // output: ["a" "b" "" "c"]
fmt.Printf("%q\n", strings.Fields("  a   b \t c\n"))
// output: ["a" "b" "c"]
```

Сягайте по `Split`, коли роздільник — це дані (поле CSV, шлях), і по
`Fields`, коли це просто відступи. Використання `Split(s, " ")` на тексті,
відформатованому для людини, дає зріз, повний порожніх рядків:

```go
fmt.Printf("%q\n", strings.Split("  a   b ", " "))
// output: ["" "" "a" "" "" "b" ""]
```

## Побудова та заміна

```go
fmt.Println(strings.Join([]string{"a", "b", "c"}, "-"))  // output: a-b-c
fmt.Println(strings.Repeat("ab", 3))                     // output: ababab
fmt.Println(strings.ReplaceAll("a.b.c", ".", "/"))       // output: a/b/c
fmt.Println(strings.Replace("a.b.c", ".", "/", 1))       // output: a/b.c
```

Для кількох замін за один прохід `strings.NewReplacer` кращий за ланцюжок
викликів `ReplaceAll` — він проходить вхідні дані один раз, і значення
безпечно повторно використовувати та ділити між горутинами:

```go
r := strings.NewReplacer("<", "&lt;", ">", "&gt;")
fmt.Println(r.Replace("<b>hi</b>"))   // output: &lt;b&gt;hi&lt;/b&gt;
```

## `strings.Builder`

Стаття про [оператори](../02-language-basics/04-operators.md) показала,
чому `s += x` у циклі — це O(n²). `Builder` — це виправлення, і оскільки
він реалізує `io.Writer`, у нього можна писати напряму через `Printf`:

```go
var b strings.Builder
for i := range 3 {
    fmt.Fprintf(&b, "%d,", i)
}
fmt.Println(b.String())   // output: 0,1,2,
```

Зверніть увагу на `&b` — методи `Write` мають отримувачі-вказівники, тож
`Builder` не можна копіювати після першого використання.

## Пакет `bytes` дзеркалить `strings`

Майже кожна функція в `strings` має двійника в `bytes` з такою самою
назвою, що працює з `[]byte`. Використовуйте його, коли дані приходять у вигляді
байтів — з файлу, сокета, тіла запиту — так ви уникаєте конвертації в
`string` і назад, оскільки кожна конвертація копіює дані:

```go
bb := []byte("hello")
fmt.Println(bytes.Contains(bb, []byte("ell")))   // output: true
fmt.Println(string(bytes.ToUpper(bb)))           // output: HELLO
```

`string(...)` тут — не прикраса. `bytes.ToUpper` повертає `[]byte`, і
друк такого значення показує числа:

```go
fmt.Println(bytes.ToUpper(bb))   // output: [72 69 76 76 79]
```

Єдина прогалина — це `strings.NewReplacer`: аналога `bytes.NewReplacer`
не існує. Це єдина експортована функція в `strings`, у якої немає
відповідника, тож у цьому випадку конвертуйте або з'єднуйте виклики
`bytes.ReplaceAll` в ланцюжок.

`bytes.Buffer` — це аналог `strings.Builder` для `[]byte`, і водночас це
і `io.Writer`, і `io.Reader`:

```go
var buf bytes.Buffer
buf.WriteString("abc")
buf.WriteByte('!')
fmt.Println(buf.String(), buf.Len())   // output: abc! 4
```

## Байти — це не символи

`len` рахує байти. Руна — одна кодова точка Unicode — займає від одного
до чотирьох байтів:

```go
g := "héllo, 世界"
fmt.Println(len(g), utf8.RuneCountInString(g))   // output: 14 9
```

Перебір рядка через `range` декодує руни й видає **зміщення в байтах**
для кожної з них, тож індекс стрибає:

```go
for i, r := range "gö" {
    fmt.Printf("%d:%c(%d) ", i, r, r)
}
// output: 0:g(103) 1:ö(246)
```

Індексу `2` тут немає: `ö` займає байти 1 і 2. Саме тому зрізання рядка
зрізає *байти*, а конвертація в `[]rune` — це те, що зрізає символи:

```go
fmt.Println(g[:5])                    // output: héll
fmt.Println(string([]rune(g)[:5]))    // output: héllo
```

Конвертація в `[]rune` виділяє копію, тож робіть це тоді, коли позиції
символів справді потрібні, а не за звичкою.

`unicode/utf8` бере на себе незручні краї — декодування по одній руні за
раз і перевірку, чи байти, що прийшли ззовні вашої програми, взагалі є
коректним UTF-8:

```go
r, size := utf8.DecodeRuneInString("世界")
fmt.Println(string(r), size)          // output: 世 3

fmt.Println(utf8.ValidString("ok"))                    // output: true
fmt.Println(utf8.ValidString(string([]byte{0xff})))    // output: false
```

## Зміна регістру — не проста річ

`ToUpper` і `ToLower` перетворюють руну за руною, а це не те саме, що
правила регістру якоїсь конкретної мови:

```go
fmt.Println(strings.ToUpper("größe"))   // output: GRÖßE
fmt.Println(strings.ToLower("ÄPFEL"))   // output: äpfel
```

У німецькій `ß` немає однорунного варіанту у верхньому регістрі, тож вона
лишається без змін. Щоб порівняти два рядки без урахування регістру, не
переводьте обидва в нижній регістр — використовуйте функцію, створену
саме для цього:

```go
fmt.Println(strings.EqualFold("Go", "GO"))   // output: true
```

> **З досвіду Python:** Go `string` — це радше Python `bytes` з угодою
> про UTF-8, а не Python `str`. Саме тому `len()` рахує інакше, `s[0]`
> дає байт, а не рядок з одного символу, а `[]rune(s)` — найближчий
> аналог того, що Python дає вам при індексуванні `str`.

## Швидка довідка

| Задача | Виклик |
|---|---|
| перевірка підрядка / позиція | `strings.Contains`, `strings.Index` (зміщення в байтах, `-1`, якщо немає) |
| розбити на "key: value" | `strings.Cut` — також повертає прапорець `found` |
| розбити за роздільником | `strings.Split` (зберігає порожні) |
| розбити за пробілами | `strings.Fields` (відкидає порожні) |
| об'єднати / повторити | `strings.Join`, `strings.Repeat` |
| багато замін за раз | `strings.NewReplacer` |
| будувати рядок у циклі | `strings.Builder`, передається як `&b` |
| ті самі операції над `[]byte` | пакет `bytes`, а також `bytes.Buffer` |
| порахувати символи | `utf8.RuneCountInString` (`len` рахує байти) |
| зрізати за символами | `[]rune(s)[a:b]` (виділяє пам'ять) |
| порівняння без урахування регістру | `strings.EqualFold` |
| перевірити зовнішні байти | `utf8.ValidString` |

## Джерела

- [`strings` package reference — pkg.go.dev/strings](https://pkg.go.dev/strings)
- [`bytes` package reference — pkg.go.dev/bytes](https://pkg.go.dev/bytes)
- [`unicode/utf8` package reference — pkg.go.dev/unicode/utf8](https://pkg.go.dev/unicode/utf8)
- [Go blog: strings, bytes, runes and characters — go.dev/blog/strings](https://go.dev/blog/strings)
- [String types — go.dev/ref/spec#String_types](https://go.dev/ref/spec#String_types)
- [For statements with range clause — go.dev/ref/spec#For_range](https://go.dev/ref/spec#For_range)
