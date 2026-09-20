# Регулярні вирази

Пакет `regexp` у Go реалізує RE2, а не Perl-сумісний діалект, який
використовує Python. Більшість патернів, які ви вже знаєте, працюють без
змін; дві можливості відсутні навмисно, і саме про це варто знати
заздалегідь.

```go
var semver = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

fmt.Println(semver.MatchString("v1.22.3"))   // output: true
fmt.Println(semver.MatchString("1.22.3"))    // output: false
```

## Компілюйте один раз, на рівні пакета

Компіляція патерна коштовна; зіставлення з ним — дешеве. Скомпільований
`*regexp.Regexp` безпечний для конкурентного використання, тож ідіома —
змінна на рівні пакета, побудована через `MustCompile`, яка панікує на
поганому патерні:

```go
var semver = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)
```

Паніка — правильна поведінка тут: патерн — це константа у вашому коді,
тож зламаний патерн — це баг, який має зупинити програму при старті, а не
при першому запиті. Використовуйте `regexp.Compile` й обробляйте помилку
лише тоді, коли патерн приходить ззовні програми:

```go
_, err := regexp.Compile(`[a-`)
fmt.Println(err)
// output: error parsing regexp: missing closing ]: `[a-`
```

Зверніть увагу на рядок у **зворотних лапках**. У звичайному літералі
`"..."` довелося б писати `\\d` для кожного `\d`, тож сирі рядки практично
обов'язкові для патернів.

## Зіставлення та вилучення

`MatchString` відповідає так чи ні. `FindStringSubmatch` повертає весь
збіг під індексом 0 і кожну групу захоплення після нього — або `nil`,
якщо збігу немає:

```go
m := semver.FindStringSubmatch("v1.22.3")
fmt.Printf("%q\n", m)   // output: ["v1.22.3" "1" "22" "3"]

fmt.Println(semver.FindStringSubmatch("nope") == nil)   // output: true
```

Другого результату `ok` тут немає, тож перевірка на `nil` *і є* перевіркою
на збіг. Індексація `m[1]` без цієї перевірки панікує, якщо збігу немає.

Іменовані групи використовують `(?P<name>...)`, а `SubexpIndex` повертає
ім'я назад у позицію:

```go
named := regexp.MustCompile(`(?P<key>\w+)=(?P<val>\w+)`)
m := named.FindStringSubmatch("mode=fast")

fmt.Println(named.SubexpIndex("val"), m[named.SubexpIndex("val")])
// output: 2 fast
```

Це незграбніше за `m.group("val")` у Python, і саме це — головна
ергономічна вада пакета.

## Пошук усіх збігів

Варіанти з `All` приймають лічильник, де `-1` означає «без обмеження»:

```go
word := regexp.MustCompile(`\w+`)

fmt.Printf("%q\n", word.FindAllString("a bb ccc", -1))  // output: ["a" "bb" "ccc"]
fmt.Printf("%q\n", word.FindAllString("a bb ccc", 2))   // output: ["a" "bb"]

fmt.Println(word.FindString("  hi there"))              // output: hi
fmt.Println(word.FindStringIndex("  hi there"))         // output: [2 4]
```

`FindString` повертає `""`, якщо збігу немає, а це неоднозначно, коли
патерн може збігтися з порожнім рядком — `FindStringIndex`, що повертає
`nil`, дає однозначну форму.

## Заміна

У рядку заміни `$1` і `${name}` посилаються на групи захоплення:

```go
date := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)

fmt.Println(date.ReplaceAllString("on 2026-09-20 ok", "$3/$2/$1"))
// output: on 20/09/2026 ok
```

Використовуйте фігурні дужки щоразу, коли цифра може «зіллятися» з
наступним текстом. `$3x` буде прочитано як група з іменем `3x`, якої не
існує, і вона розгорнеться в нічого:

```go
fmt.Println(date.ReplaceAllString("on 2026-09-20 ok", "${3}x"))
// output: on 20x ok
```

Коли заміна потребує справжньої логіки, `ReplaceAllStringFunc` передає
вам кожен збіг:

```go
fmt.Println(word.ReplaceAllStringFunc("go rocks", func(s string) string {
    return "<" + s + ">"
}))
// output: <go> <rocks>
```

Розбиття приймає лічильник з тієї ж причини, що й функції з `All`:

```go
sp := regexp.MustCompile(`\s*,\s*`)
fmt.Printf("%q\n", sp.Split("a ,b,  c", -1))   // output: ["a" "b" "c"]
```

## Прапорці йдуть усередині патерна

Аргументу для прапорців немає. `(?i)`, `(?m)` і `(?s)` розміщуються на
початку патерна:

```go
fmt.Println(regexp.MustCompile(`(?i)^go$`).MatchString("GO"))   // output: true

fmt.Printf("%q\n", regexp.MustCompile(`(?m)^\w+`).FindAllString("one\ntwo", -1))
// output: ["one" "two"]

fmt.Println(regexp.MustCompile(`(?s)a.b`).MatchString("a\nb"))  // output: true
```

`(?i)` — без урахування регістру, `(?m)` змушує `^`/`$` збігатися на межах
рядків, `(?s)` дозволяє `.` збігатися з символом нового рядка.

## Чого RE2 не робить

Зворотні посилання (backreferences) і lookaround відсутні, і відсутні
навмисно. RE2 гарантує зіставлення за час, лінійний відносно довжини
вхідних даних, а це виключає конструкції, через які регулярний вираз
може вибухнути експоненційно. Патерн, який отримує ворожі вхідні дані й
вішає процес, тут неможливий.

```go
_, err := regexp.Compile(`(\w)\1`)
fmt.Println(err)
// output: error parsing regexp: invalid escape sequence: `\1`

_, err = regexp.Compile(`(?=foo)`)
fmt.Println(err)
// output: error parsing regexp: invalid or unsupported Perl syntax: `(?=`
```

Якщо вони вам потрібні, відповідь зазвичай — зіставити щось ширше, а
решту перевірити звичайним кодом Go, що зазвичай навіть зрозуміліше, ніж
був би lookahead.

## Екранування літерала

Коли частина патерна приходить з даних, екрануйте її:

```go
fmt.Println(regexp.QuoteMeta("a.b*c"))   // output: a\.b\*c
```

## Спершу сягайте по `strings`

Регулярний вираз повільніший і важче читається за прямий виклик. Якщо
`strings.Contains`, `HasPrefix`, `Cut` чи `Fields` впораються — вживайте
їх, дивіться [рядки, байти та руни](01-strings-bytes-and-runes.md).

> **З досвіду Python:** `MustCompile` — це `re.compile` під час імпорту,
> `MatchString` — це `re.search`, що повертає bool, а
> `FindStringSubmatch` — це `m.groups()` з повним збігом на початку.
> Відмінності, на яких можна впіймати: прапорці живуть усередині патерна,
> об'єкта збігу немає, тож перевіряють `nil`, іменованим групам потрібен
> `SubexpIndex`, а `\1` і `(?=...)` просто не існують.

## Швидка довідка

| Задача | Виклик |
|---|---|
| скомпілювати сталий патерн | ``regexp.MustCompile(`...`)`` на рівні пакета |
| скомпілювати недовірені вхідні дані | `regexp.Compile`, обробити помилку |
| так/ні | `MatchString` |
| групи захоплення | `FindStringSubmatch` — `nil` означає відсутність збігу |
| іменована група | `SubexpIndex("name")` |
| кожен збіг | `FindAllString(s, -1)` |
| заміна з групами | `ReplaceAllString(s, "${1}")` |
| заміна з логікою | `ReplaceAllStringFunc` |
| розбиття | `Split(s, -1)` |
| без регістру / багаторядковий / dotall | `(?i)` / `(?m)` / `(?s)` |
| екранувати літеральний текст | `regexp.QuoteMeta` |

## Джерела

- [`regexp` package reference — pkg.go.dev/regexp](https://pkg.go.dev/regexp)
- [`regexp/syntax` — pkg.go.dev/regexp/syntax](https://pkg.go.dev/regexp/syntax)
- [RE2 syntax — github.com/google/re2/wiki/Syntax](https://github.com/google/re2/wiki/Syntax)
- [Go blog: regular expression matching can be simple and fast — swtch.com/~rsc/regexp/regexp1.html](https://swtch.com/~rsc/regexp/regexp1.html)
