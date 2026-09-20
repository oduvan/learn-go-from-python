# Запуск зовнішніх команд

`os/exec` запускає інші програми. Він **не** запускає оболонку (shell),
і це найважливіша річ, яку варто про нього розуміти, — і причина, чому
цілі категорії багів впровадження (injection) просто не виникають.

```go
out, err := exec.CommandContext(ctx, "echo", "hello world").Output()
fmt.Printf("%q %v\n", string(out), err)
// output: "hello world\n" <nil>
```

## Завжди беріть контекст

Надавайте перевагу `exec.CommandContext` над `exec.Command`. Контекст
дає вам вимикач, а підпроцес без таймауту — це підпроцес, який може
навічно повісити ваш сервіс:

```go
ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()

err := exec.CommandContext(ctx, "sleep", "5").Run()
fmt.Println(err, ctx.Err())
// output: signal: killed context deadline exceeded
```

Зверніть увагу, дві помилки кажуть різне. `err` повідомляє, *як*
процес загинув; `ctx.Err()` каже, *чому*. Перевіряйте контекст, коли
треба відрізнити таймаут від справжнього збою.

## Аргументи — це список, а не командний рядок

Кожен аргумент передається програмі точно так, як написаний. Нічого не
розбивається на пробіли, не розкриває глоби, не інтерпретує лапки:

```go
out, _ := exec.CommandContext(ctx, "echo", "a b", "c").Output()
fmt.Printf("%q\n", string(out))   // output: "a b c\n"

out, _ = exec.CommandContext(ctx, "echo", "*").Output()
fmt.Printf("%q\n", string(out))   // output: "*\n"
```

`*` лишився буквальною зірочкою. Немає оболонки, яка б його розкрила.

Ось чому передавати ім'я файлу з користувацького вводу як аргумент
безпечно: воно ніколи не може стати другою командою. Ця безпека
зникає в ту мить, коли ви пишете `sh -c`:

```go
exec.CommandContext(ctx, "sh", "-c", "grep "+pattern+" file.txt")   // ін'єкція
```

Тут `pattern` — це джерело коду оболонки. Використовуйте `sh -c` лише
для фіксованого рядка, який ви самі написали, ніколи з
інтерпольованим вводом. Якщо потрібен конвеєр (pipeline), або зберіть
його в Go, підключивши `cmd.Stdout` до `cmd.Stdin` наступної команди,
або виконайте роботу в Go й обійдіться без підпроцесу.

## Який виклик використовувати

| Виклик | Повертає | Використовуйте, коли |
|---|---|---|
| `Output()` | stdout | потрібен результат, stderr окремо при збої |
| `CombinedOutput()` | stdout + stderr впереміш | логування чи налагодження |
| `Run()` | лише помилка | ви самі підключили потоки |
| `Start()` + `Wait()` | лише помилка | потрібно робити роботу, поки воно виконується |

```go
co, err := exec.CommandContext(ctx, "sh", "-c", "echo out; echo err >&2; exit 1").CombinedOutput()
fmt.Printf("%q %v\n", string(co), err)
// output: "out\nerr\n" exit status 1
```

## Збій: `*exec.ExitError` несе stderr

Ненульовий код виходу — це помилка. З `Output()` перехоплений stderr
прикріплюється до неї, а це те, що робить збій діагностованим:

```go
_, err := exec.CommandContext(ctx, "sh", "-c", "echo oops >&2; exit 3").Output()

var ee *exec.ExitError
if errors.As(err, &ee) {
    fmt.Println("code:", ee.ExitCode(), "stderr:", strings.TrimSpace(string(ee.Stderr)))
}
// output: code: 3 stderr: oops
```

`ExitCode()` має значення, коли інструмент використовує код виходу як
дані — `grep` повертає 1 для "немає збігу", `git diff --quiet` повертає
1 для "є зміни". Це відповіді, а не збої, тож перевіряйте код, а не
трактуйте будь-яку помилку як фатальну.

`ee.Stderr` заповнюється лише через `Output()`. З `Run()` ви самі
перехопили stderr; з `CombinedOutput()` він у повернених байтах.

Відсутній виконуваний файл — інший тип збою, ще до того, як процес
взагалі запустився:

```go
_, err := exec.CommandContext(ctx, "definitely-not-a-command-xyz").Output()
fmt.Println(err)
// output: exec: "definitely-not-a-command-xyz": executable file not found in $PATH

fmt.Println(errors.Is(err, exec.ErrNotFound))   // output: true
```

Перевіряйте залежність заздалегідь через `exec.LookPath`, щоб
відсутній інструмент повідомлявся під час запуску, а не посеред
запиту.

## Підключення потоків

Встановіть поля перед запуском. Будь-який `io.Reader` працює як stdin,
а будь-який `io.Writer` — як stdout чи stderr — інтерфейси зі статті
[читачі та письменники](02-readers-and-writers.md):

```go
cmd := exec.CommandContext(ctx, "sh", "-c", "cat; echo done >&2")
cmd.Stdin = strings.NewReader("piped in\n")

var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout
cmd.Stderr = &stderr

err := cmd.Run()
fmt.Printf("%q %q %v\n", stdout.String(), stderr.String(), err)
// output: "piped in\n" "done\n" <nil>
```

Призначте `os.Stdout`, щоб пропустити вивід прямо у власний, — саме
це вам потрібно для інструмента збірки.

Не встановлюйте `cmd.Stdout`, а потім не викликайте `Output()` — це
помилка, оскільки `Output` потребує сам володіти stdout.

## Робочий каталог і середовище

```go
cmd := exec.CommandContext(ctx, "sh", "-c", "echo $MYVAR; pwd")
cmd.Dir = dir
cmd.Env = append(os.Environ(), "MYVAR=set")
```

`cmd.Dir` запускає команду в іншому місці без зміни каталогу вашим
процесом — важливо, бо `chdir` на весь процес небезпечний, коли
працюють інші горутини.

`cmd.Env` **повністю замінює** середовище. Залишити його nil означає
успадкувати ваше; встановити його як просто `[]string{"MYVAR=set"}`
дає дочірньому процесу майже порожнє середовище без `PATH`.
`append(os.Environ(), ...)` — звична форма. Пізніші записи перемагають
при дублікатах.

## Довготривалі процеси

`Start` повертається одразу, щойно процес запускається, а `Wait`
блокується, чекаючи на нього:

```go
cmd := exec.CommandContext(ctx, "long-task")
if err := cmd.Start(); err != nil {
    return err
}
// ... виконати іншу роботу ...
return cmd.Wait()
```

Кожен `Start` потребує рівно одного `Wait`, інакше ви залишите
процес-зомбі й будь-які канали, які створили. Якщо ви підключили
канали через `StdoutPipe`, прочитайте їх до кінця *перед* викликом
`Wait`.

> **З досвіду Python:** це `subprocess`, а `Output()` — це
> `check_output`. Відмінність, яку варто засвоїти, — немає
> `shell=True`, якщо ви не пропишете `sh -c` явно, тож типовий варіант
> — безпечний, а небезпечна форма видна прямо в коді.

## Швидка довідка

| Завдання | Виклик |
|---|---|
| запустити й перехопити stdout | `exec.CommandContext(ctx, name, args...).Output()` |
| stdout і stderr разом | `.CombinedOutput()` |
| потоки, підключені вручну | встановіть `Stdin`/`Stdout`/`Stderr`, тоді `.Run()` |
| запустити у фоні | `.Start()`, тоді `.Wait()` — завжди обидва |
| код виходу | `errors.As(err, &ee)`, `ee.ExitCode()`, `ee.Stderr` |
| відсутній бінарник | `errors.Is(err, exec.ErrNotFound)`, або `exec.LookPath` заздалегідь |
| в іншому місці | `cmd.Dir = path` |
| додаткове середовище | `cmd.Env = append(os.Environ(), "K=V")` |
| таймаут | контекст — він вбиває процес |

## Джерела

- [`os/exec` package reference — pkg.go.dev/os/exec](https://pkg.go.dev/os/exec)
- [`exec.CommandContext` — pkg.go.dev/os/exec#CommandContext](https://pkg.go.dev/os/exec#CommandContext)
- [`exec.ExitError` — pkg.go.dev/os/exec#ExitError](https://pkg.go.dev/os/exec#ExitError)
- [`exec.Cmd` — pkg.go.dev/os/exec#Cmd](https://pkg.go.dev/os/exec#Cmd)
- [`exec.LookPath` — pkg.go.dev/os/exec#LookPath](https://pkg.go.dev/os/exec#LookPath)
