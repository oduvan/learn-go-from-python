# `go:embed`

`//go:embed` копіює файли в бінарник під час збірки. Програма на Go вже
є єдиним статичним виконуваним файлом; вбудовування — те, що зберігає
цю властивість, коли їй потрібні шаблони, міграції чи статичні
ресурси.

```go
import _ "embed"

//go:embed version.txt
var version string

fmt.Printf("%q\n", version)   // output: "v1.4.2\n"
```

Немає файлу для постачання, немає шляху для налаштування, нічому
губитись у образі контейнера.

## Три типи цілей

Директива стоїть одразу над `var` на рівні пакета, без порожнього
рядка між ними, і тип змінної визначає, що ви отримаєте:

| Тип | Результат |
|---|---|
| `string` | вміст файлу як текст |
| `[]byte` | вміст файлу як байти |
| `embed.FS` | файлова система лише для читання, один або багато файлів |

```go
//go:embed version.txt
var version string

//go:embed version.txt
var versionBytes []byte

//go:embed static
var staticFS embed.FS
```

Для `string` і `[]byte` директива має називати рівно один файл. Лише
`embed.FS` приймає каталоги чи шаблони (patterns).

## Порожній імпорт

Якщо файл використовує `//go:embed` лише для змінних `string` чи
`[]byte`, він ніколи не згадує пакет `embed` за іменем — але компілятор
усе одно вимагає його імпортування. Звідси порожній імпорт:

```go
import _ "embed"
```

Використання `embed.FS` означає, що ви називаєте пакет, тож звичайний
`import "embed"` це покриває. Забути про це дає чітку помилку
компіляції, тож це одноразовий сюрприз. Це той самий механізм, що й
реєстрація драйвера в статті про [імпорти](../02-language-basics/16-imports.md).

## Читання з `embed.FS`

`embed.FS` реалізує `fs.FS`, тож усе з `io/fs` працює на ньому —
включно з `fs.WalkDir` зі статті про [файли та
шляхи](01-files-and-paths.md):

```go
b, err := staticFS.ReadFile("static/css/app.css")
fmt.Println(err)                // output: <nil>
fmt.Printf("%q\n", string(b))   // output: "body{color:red}"
```

**Шлях зберігає каталог, який ви вбудували.** Вбудовування `static`
означає, що запис — це `static/css/app.css`, а не `css/app.css`.
Шляхи завжди використовують прямий слеш, на будь-якій платформі, і
мають бути чистими — без `./`, без `..`:

```go
fmt.Println(fs.ValidPath("static/css/app.css"))   // output: true
fmt.Println(fs.ValidPath("./static"))             // output: false
```

Відсутній файл дає помилку, обгорнуту `fs.ErrNotExist`, яку можна
перевірити звичним способом:

```go
_, err := staticFS.ReadFile("nope")
fmt.Println(err)   // output: open nope: file does not exist
```

## `fs.Sub` прибирає префікс

Носити `static/` через кожен пошук — це шум. `fs.Sub` повертає файлову
систему, вкорінену в підкаталозі:

```go
sub, _ := fs.Sub(staticFS, "static")
b, _ := fs.ReadFile(sub, "css/app.css")
fmt.Printf("%q\n", string(b))   // output: "body{color:red}"
```

Це те, що ви передаєте будь-чому, що очікує обслуговувати корінь
каталогу.

## `all:` і те, що пропускається

За замовчуванням вбудовування каталогу **пропускає файли, що
починаються з `.` чи `_`**. Це правило успадковане від того, як
інструмент `go` взагалі ігнорує такі файли, і воно мовчки відкидає
речі, які ви мали намір включити:

```go
//go:embed migrations
var plain embed.FS
// migrations/001_init.sql, migrations/002_b.sql

//go:embed all:migrations
var withAll embed.FS
// migrations/001_init.sql, migrations/002_b.sql, migrations/_skipme.sql
```

Префікс `all:` вимикає це правило. Використовуйте його для всього, де
відсутній файл — проблема коректності, — особливо для міграцій, де
пропущений файл означає схему, яка мовчки розходиться.

Шаблони — інша форма, і вони не рекурсивні:

```go
//go:embed tpl/*.tmpl
var tplFS embed.FS
```

## Розбір шаблонів прямо з неї

`ParseFS` читає шаблони з будь-якої `fs.FS`, тож бінарник несе їх із
собою. Самі шаблони — тема наступної статті; тут важливо лише те, що
вбудована файлова система підключається до всього, що приймає `fs.FS`:

```go
t := template.Must(template.ParseFS(tplFS, "tpl/*.tmpl"))
t.ExecuteTemplate(os.Stdout, "greet.tmpl", map[string]string{"Name": "Ada"})
// output: Hello Ada
```

`template.Must` панікує на поганому шаблоні, а це саме те, що потрібно
під час запуску: шаблон, який не парситься, — це помилка часу збірки,
тож провалюйтесь одразу, а не при першому запиті.

## Правила, які варто пам'ятати

- **Кожен спосіб неправильно розмістити директиву — це помилка
  збірки**, і це добра новина — тут немає режиму мовчазного збою.
  Директива має стояти безпосередньо перед `var` на рівні пакета
  потрібного типу; порожні рядки та коментарі `//` між ними явно
  дозволені:

```go
//go:embed version.txt
func nope() {}
// build error: misplaced go:embed directive

//go:embed version.txt
var wrongType int
// build error: go:embed cannot apply to var of type int

func f() {
    //go:embed version.txt
    var local string
    // build error: go:embed cannot apply to var inside func
}
```

- Вбудувати можна лише файли **всередині каталогу пакета**. Батьківський
  шлях відхиляється одразу:

```go
//go:embed ../outside.txt
// build error: pattern ../outside.txt: invalid pattern syntax
```

- Шаблон без збігів — це **помилка збірки**, а не порожня FS, тож
  описка в імені файлу дійсно провалюється голосно:

```go
//go:embed assets/*.zzz
// build error: pattern assets/*.zzz: no matching files found
```

- Вбудований вміст лише для читання й фіксований під час збірки.
  Змініть файл — і доведеться перезібрати.
- Файли враховуються в розмір бінарника. Вбудовування кількох сотень
  кілобайтів ресурсів — нормально; вбудовування великого набору даних —
  це рішення.

## Коли не варто

Усе, що має змінюватись без перезбірки, — конфігурація, секрети,
завантажені користувачем файли — має лишатись на диску чи в сховищі.
Вбудовування — для речей, що належать коду: SQL-міграції, HTML-шаблони,
CSS і JS, типова конфігурація, файл довідкових даних.

> **З досвіду Python:** це замінює `importlib.resources` і всю проблему
> `package_data`/`MANIFEST.in`. Немає режиму збою "чи встановився файл
> даних?", бо дані всередині виконуваного файлу.

## Швидка довідка

| Потреба | Пишіть |
|---|---|
| один файл як текст | `//go:embed f.txt` над `var s string` |
| один файл як байти | те саме, над `var b []byte` |
| дерево | `//go:embed dir` над `var f embed.FS` |
| включити файли з крапкою й `_` | `//go:embed all:dir` |
| шаблон | `//go:embed tpl/*.tmpl` (без рекурсії) |
| лише `string`/`[]byte` у файлі | додайте `import _ "embed"` |
| прибрати префікс | `fs.Sub(f, "dir")` |
| прочитати | `f.ReadFile("dir/x")` — прямий слеш, зберігає префікс |
| обійти | `fs.WalkDir(f, "dir", fn)` |
| шаблони | `template.ParseFS(f, "tpl/*.tmpl")` |

## Джерела

- [`embed` package reference — pkg.go.dev/embed](https://pkg.go.dev/embed)
- [`io/fs` package reference — pkg.go.dev/io/fs](https://pkg.go.dev/io/fs)
- [`fs.Sub` — pkg.go.dev/io/fs#Sub](https://pkg.go.dev/io/fs#Sub)
- [`template.ParseFS` — pkg.go.dev/text/template#ParseFS](https://pkg.go.dev/text/template#ParseFS)
- [Go 1.16 embed announcement — go.dev/doc/go1.16#embed](https://go.dev/doc/go1.16#embed)
