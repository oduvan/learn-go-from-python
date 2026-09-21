# Шаблони

Два пакети мають один спільний синтаксис. `text/template` рендерить
будь-який текст; `html/template` рендерить HTML і автоматично екранує
його. Правильний вибір між ними — це рішення безпеки, а не стилю.

```go
t := template.Must(template.New("t").Parse("Hello {{.Name}}\n"))
t.Execute(os.Stdout, Item{Name: "Ada"})
// output: Hello Ada
```

## Парсинг і виконання

`Parse` компілює, `Execute` рендерить у `io.Writer`. `template.Must`
обгортає парсинг і панікує в разі невдачі:

```go
var tmpl = template.Must(template.New("page").Parse(src))
```

Парсьте **один раз**, під час запуску, у змінну рівня пакета. Парсинг
на кожен запит витрачає ресурси даремно й перетворює одруківку в
шаблоні на помилку часу виконання на одному невдалому шляху, а не на
падіння під час старту. `Must` тут доречний з тієї ж причини, що й
`regexp.MustCompile`: шаблон у вашому коді, який не парситься, — це
баг, який має зупинити програму.

`ParseFS` читає шаблони з `embed.FS` — так їх постачають прямо
всередині бінарника, див. [`go:embed`](../07-operating-system/03-go-embed.md).

## Синтаксис

`{{.}}` — це поточне значення; `{{.Field}}` заходить усередину нього.

```
{{range .Tags}}[{{.}}]{{else}}none{{end}}
{{if gt .Price 10.0}}expensive{{else}}cheap{{end}}
{{with .Name}}name={{.}}{{end}}
```

Маючи `Tags: ["x","y"]`, `Price: 5`, `Name: "Bo"`:

```
[x][y]
cheap
name=Bo
```

Без тегів (tags) і з `Price: 50`:

```
none
expensive
name=Cy
```

Усередині `range` і `with` `.` перев'язується до елемента — `$`
зберігає посилання на початкове значення верхнього рівня. `range`
приймає `{{else}}` для порожнього випадку, що охайніше за окремий
`if`. `with` повністю пропускає свій блок, коли значення порожнє.

Порівняння — це функції, а не оператори: `eq`, `ne`, `lt`, `le`, `gt`,
`ge`, а також `and`, `or`, `not`. Арифметики немає — обчислюйте в Go й
передавайте готовий результат.

Прибирайте зайві пробіли навколо за допомогою `{{-` і `-}}`:

```go
template.Must(template.New("t").Parse("a{{- if true}} b {{- end}}c\n"))
// output: a bc
```

## Власні функції

`Funcs` треба викликати **до** `Parse`, оскільки саме парсинг
розв'язує імена:

```go
fm := template.FuncMap{
    "upper": strings.ToUpper,
    "money": func(f float64) string { return fmt.Sprintf("$%.2f", f) },
}

t := template.Must(template.New("t").Funcs(fm).Parse("{{upper .Name}} {{money .Price}}\n"))
// output: ADA $3.50
```

Обмежуйте їх форматуванням. Логіка належить Go, де її можна тестувати.

## Відсутні дані поводяться по-різному залежно від типу

Відсутнє **поле структури** — це помилка виконання:

```go
t := template.Must(template.New("t").Parse("{{.Nope}}"))
err := t.Execute(os.Stdout, Item{})
// err: template: t:1:2: executing "t" at <.Nope>:
//      can't evaluate field Nope in type main.Item
```

Відсутній **ключ мапи** помилкою не є, і саме тут два пакети
розходяться. `text/template` рендерить його як `<no value>`:

```go
// text/template
t := template.Must(template.New("t").Parse("[{{.missing}}]\n"))
t.Execute(os.Stdout, map[string]string{})
// output: [<no value>]
```

`html/template` рендерить той самий відсутній ключ як цілковиту
порожнечу:

```go
// html/template
h := template.Must(template.New("h").Parse("[{{.missing}}]\n"))
h.Execute(os.Stdout, map[string]string{})
// output: []
```

Жоден із них не повідомляє, що ключа не було, — саме тому нижче й дано
пораду.

Для даних шаблону надавайте перевагу структурам. Тоді одруківку буде
виявлено, а обробка через `Option`
(`template.Option("missingkey=error")`) не знадобиться.

`Execute` може вже записати частину виводу перед тим, як зазнати
невдачі, тож спершу рендерте в `bytes.Buffer` і копіюйте у відповідь
лише в разі успіху — інакше помилка посеред шаблону лишить наполовину
записану сторінку з уже надісланим `200`.

## `html/template` екранує; `text/template` — ні

Той самий синтаксис, різна поведінка — і саме тому обидва існують:

```go
// html/template
h := template.Must(template.New("h").Parse("<p>{{.}}</p>\n"))
h.Execute(os.Stdout, `<script>alert("x")</script>`)
// output: <p>&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;</p>
```

```go
// text/template — однаковий код, без екранування
t := template.Must(template.New("t").Parse("<p>{{.}}</p>\n"))
t.Execute(os.Stdout, `<script>alert("x")</script>`)
// output: <p><script>alert("x")</script></p>
```

Єдина відмінність у місці виклику — шлях імпорту, тож **імпортувати не
той пакет — це діра для XSS, яка чисто компілюється**. Якщо вивід — це
HTML, імпортуйте `html/template`.

## Екранування залежить від контексту

`html/template` парсить HTML і екранує залежно від того, куди
потрапляє значення. В атрибуті він екранує лапки:

```go
h := template.Must(template.New("h").Parse(`<div x="{{.}}"></div>` + "\n"))
h.Execute(os.Stdout, `a" onclick="evil()`)
// output: <div x="a&#34; onclick=&#34;evil()"></div>
```

В `href` він розпізнає небезпечну схему й замінює все значення цілком:

```go
h := template.Must(template.New("h").Parse(`<a href="{{.}}">link</a>` + "\n"))
h.Execute(os.Stdout, `javascript:alert(1)`)
// output: <a href="#ZgotmplZ">link</a>
```

`ZgotmplZ` — це навмисний маркер, що означає "тут значення було
відхилено". Побачити його на сторінці означає, що небезпечний вміст
потрапив у позицію URL — виправляйте дані, а не шаблон.

Типи на кшталт `template.HTML` і `template.JS` виводять значення
з-під екранування. Вони означають "обіцяю, що це безпечно", тож у них
ніколи не можна класти щось, похідне від введення користувача.

## Композиція

`define` іменує блок; `template` викликає його. Саме так працюють
макети (layouts):

```go
layout := `{{define "page"}}<h1>{{.Name}}</h1>{{template "body" .}}{{end}}` +
          `{{define "body"}}<p>{{.Price}}</p>{{end}}`

h := template.Must(template.New("l").Parse(layout))
h.ExecuteTemplate(os.Stdout, "page", Item{Name: "Ada", Price: 9})
// output: <h1>Ada</h1><p>9</p>
```

Кінцева `.` у `{{template "body" .}}` передає дані далі — опустіть її,
і вкладений шаблон отримає `nil`. Коли визначено кілька шаблонів,
використовуйте `ExecuteTemplate`, щоб обрати потрібний за іменем.

`block` визначає типовий варіант, який пізніший шаблон може
перевизначити — це дає базовий макет із секціями, які можна замінити.

> **З досвіду Python:** це Jinja, тільки з набагато меншим набором
> можливостей — немає арифметики, немає фільтрів через `|` (функції
> тут префіксні), немає успадкування шаблонів понад `define`/`block`.
> Автоекранування тут сильніше за Jinja, бо воно знає, чи потрапляє
> значення в атрибут, URL чи скрипт. Пастка, якої немає в Jinja, —
> пакет без екранування лежить лише за один рядок імпорту.

## Швидка довідка

| Задача | Форма |
|---|---|
| вивід HTML | `html/template` — завжди |
| будь-який інший текст | `text/template` |
| парсити під час запуску | `template.Must(template.New(n).Parse(src))` |
| з вбудованої FS | `template.ParseFS(fsys, "tpl/*.tmpl")` |
| рендерити | `t.Execute(w, data)` / `t.ExecuteTemplate(w, name, data)` |
| поле / поточне значення | `{{.Field}}` / `{{.}}` |
| цикл | `{{range .Xs}}...{{else}}empty{{end}}` |
| умова | `{{if gt .N 3}}...{{end}}` — функції, а не оператори |
| прибрати пробіли | `{{-` і `-}}` |
| допоміжні функції | `.Funcs(FuncMap{...})` **до** `Parse` |
| макети | `{{define "x"}}`, `{{template "x" .}}`, `{{block}}` |
| передати дані далі | кінцева `.` у `{{template "x" .}}` |
| екранування відхилило значення | `ZgotmplZ` у виводі |

## Джерела

- [`text/template` package reference — pkg.go.dev/text/template](https://pkg.go.dev/text/template)
- [`html/template` package reference — pkg.go.dev/html/template](https://pkg.go.dev/html/template)
- [Template actions — pkg.go.dev/text/template#hdr-Actions](https://pkg.go.dev/text/template#hdr-Actions)
- [Contextual autoescaping — pkg.go.dev/html/template#hdr-Contexts](https://pkg.go.dev/html/template#hdr-Contexts)
- [`template.FuncMap` — pkg.go.dev/text/template#FuncMap](https://pkg.go.dev/text/template#FuncMap)
