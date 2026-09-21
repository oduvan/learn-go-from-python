# Прапорці та середовище

Пакет `flag` зі стандартної бібліотеки навмисно малий. Він добре
обробляє звичайний випадок, має одне неочікуване правило про порядок
аргументів, і цього достатньо для більшості програм.

```go
port := flag.Int("port", 8080, "port to listen on")
flag.Parse()

fmt.Println(*port)
```

## Визначення та розбір

Кожен виклик `flag.X` повертає **вказівник**, бо значення не існує,
поки не виконається `Parse`:

```go
verbose := flag.Bool("verbose", false, "enable verbose output")
port    := flag.Int("port", 8080, "port to listen on")
name    := flag.String("name", "app", "service name")
timeout := flag.Duration("timeout", 5*time.Second, "request timeout")

flag.Parse()

fmt.Println(*verbose, *port, *name, *timeout)
// без аргументів: false 8080 app 5s
```

Читання `*port` до `Parse` мовчки дає значення за замовчуванням.
Викликайте `Parse` один раз, на самому початку `main`, перш ніж
щось читає прапорець.

`flag.Duration` розбирає ті самі рядки, що й `time.ParseDuration` зі
[статті про час](../06-text-time-and-data/04-time.md), тож
`--timeout 2m` працює. Є також форма `flag.XVar`, яка пише в змінну, що
у вас уже є, — зручно для заповнення структури конфігурації.

Одинарний і подвійний дефіс рівнозначні, а `=` опційний:

```
-verbose  -port=9090  --timeout 2m
```

## Прапорці зупиняються на першому аргументі, що не є прапорцем

Це правило, на якому спотикаються всі:

```go
fs.Parse([]string{"-f", "a", "b"})
fmt.Println(*f, fs.Args())   // output: true [a b]

fs.Parse([]string{"a", "-f", "b"})
fmt.Println(*f, fs.Args())   // output: false [a -f b]
```

У другому випадку `-f` взагалі не розібрано — це просто ще один
позиційний аргумент. У Go немає GNU-стилю перестановки, тож
**прапорці мають стояти перед позиційними аргументами**. Усе, що
залишається, — це `flag.Args()`, з `flag.NArg()` і `flag.Arg(i)` як
методами доступу.

## Булеві прапорці потребують `=`

Булевий прапорець встановлюється своєю присутністю, тож ніколи не
споживає наступний аргумент. Щоб явно передати `false`, треба
використати `=`:

```go
fs.Parse([]string{"-d=false"})
fmt.Println(*d)   // output: false
```

`-d false` встановлює `d` у true й лишає `"false"` позиційним
аргументом — саме тому прапорцю зі значенням true за замовчуванням
потрібна ця форма.

## Довідка й помилки

`flag.PrintDefaults` виводить згенеровану довідку:

```
  -name string
    	service name (default "app")
  -port int
    	port to listen on (default 8080)
  -timeout duration
    	request timeout (default 5s)
  -verbose
    	enable verbose output
```

Третій аргумент кожного визначення — це той текст довідки, тож пишіть
його для читача.

При неправильному значенні типовий `flag.CommandLine` виводить помилку
разом із довідкою й викликає `os.Exit(2)`:

```
invalid value "abc" for flag -n: parse error
```

## Підкоманди через `flag.NewFlagSet`

Вбудованої підтримки підкоманд немає. `flag.NewFlagSet` дає кожній
підкоманді власний набір, а ви розподіляєте за `os.Args[1]`:

```go
serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
servePort := serveCmd.Int("port", 8080, "port")

migrateCmd := flag.NewFlagSet("migrate", flag.ExitOnError)
migrateDown := migrateCmd.Bool("down", false, "roll back")

if len(os.Args) < 2 {
    fmt.Fprintln(os.Stderr, "expected 'serve' or 'migrate'")
    os.Exit(2)
}

switch os.Args[1] {
case "serve":
    serveCmd.Parse(os.Args[2:])
    fmt.Println("serving on", *servePort)
case "migrate":
    migrateCmd.Parse(os.Args[2:])
    fmt.Println("rolling back:", *migrateDown)
default:
    fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
    os.Exit(2)
}
```

Режим обробки помилок має значення. `flag.ExitOnError` завершує роботу
при неправильному прапорці; `flag.ContinueOnError` повертає помилку,
щоб ви самі вирішили, — саме це робить набір прапорців тестованим.

## Змінні середовища

```go
fmt.Printf("%q\n", os.Getenv("APP_PORT"))          // output: "3000"
fmt.Printf("%q\n", os.Getenv("DEFINITELY_UNSET"))  // output: ""
```

`Getenv` повертає `""` і для "не встановлено", і для "встановлено як
порожнє". Коли ця різниця важлива — порожнє значення означає "явно
вимкнено" — `LookupEnv` про неї повідомляє:

```go
v, ok := os.LookupEnv("DEFINITELY_UNSET")
fmt.Printf("%q %v\n", v, ok)   // output: "" false

os.Setenv("EMPTY", "")
v, ok = os.LookupEnv("EMPTY")
fmt.Printf("%q %v\n", v, ok)   // output: "" true
```

Усе — рядок, тож для чогось іншого потрібен `strconv`:

```go
n, err := strconv.Atoi(os.Getenv("APP_PORT"))
fmt.Println(n, err)   // output: 3000 <nil>
```

Розбирайте конфігурацію **один раз під час запуску й голосно
провалюйтесь**, а не викликайте `Getenv` глибоко в коді. Розкидані
пошуки роблять вхідні дані програми неможливими знайти, а описка стає
нульовим значенням замість помилки. Збирання їх в одну структуру
конфігурації, перевірену один раз, — патерн, до якого повертається
тема архітектури.

Стандартна бібліотека не читає файли `.env` — це зручність від
сторонніх бібліотек.

## Коди виходу

`os.Exit` зупиняє негайно. Він **не** виконує відкладені функції:

```go
func main() {
    defer fmt.Println("cleanup")   // ніколи не друкується
    os.Exit(1)
}
```

Тож ідіома — тримати `main` тонким і дозволити функції `run` повертати
помилку:

```go
func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    defer cleanup()   // виконується
    // ...
    return nil
}
```

Помилки йдуть у `os.Stderr`, вивід — у `os.Stdout`, тож викликач може
перенаправити один без іншого. `log.Fatal` пише в stderr і завершує
роботу з кодом 1, з тією самою проблемою `defer`.

За домовленістю `0` означає успіх, `1` — загальний збій, а `2` —
помилку використання — саме те, що використовує `flag`.

> **З досвіду Python:** `flag` — це `argparse` зі значно меншим
> набором можливостей: немає `nargs`, немає підпарсерів, немає
> взаємовиключних груп, немає автоматичного епілогу `--help` понад
> згенерований список. Дві реальні поведінкові відмінності — прапорці
> мають передувати позиційним аргументам, і `os.Exit` пропускає `defer`
> так само, як `os._exit` пропускає `finally`.

## Швидка довідка

| Завдання | Виклик |
|---|---|
| визначити | `flag.Int("port", 8080, "help")` → `*int` |
| у наявну змінну | `flag.IntVar(&cfg.Port, "port", 8080, "help")` |
| розібрати | `flag.Parse()` один раз, на початку `main` |
| залишок | `flag.Args()`, `flag.NArg()`, `flag.Arg(i)` |
| тривалості | `flag.Duration` — приймає `2m`, `1h30m` |
| явний false | `-d=false`, ніколи `-d false` |
| підкоманди | `flag.NewFlagSet(name, flag.ExitOnError)` + `os.Args[1]` |
| тестований розбір | `flag.ContinueOnError` |
| середовище, може бути порожнім | `os.Getenv(k)` |
| відрізнити невстановлене | `os.LookupEnv(k)` → `(value, ok)` |
| середовище в число | `strconv.Atoi` |
| провалитись | пишіть в `os.Stderr`, тоді `os.Exit(1)` — пропускає `defer` |

## Джерела

- [`flag` package reference — pkg.go.dev/flag](https://pkg.go.dev/flag)
- [`flag.NewFlagSet` — pkg.go.dev/flag#NewFlagSet](https://pkg.go.dev/flag#NewFlagSet)
- [`os.LookupEnv` — pkg.go.dev/os#LookupEnv](https://pkg.go.dev/os#LookupEnv)
- [`os.Exit` — pkg.go.dev/os#Exit](https://pkg.go.dev/os#Exit)
- [`strconv` package reference — pkg.go.dev/strconv](https://pkg.go.dev/strconv)
