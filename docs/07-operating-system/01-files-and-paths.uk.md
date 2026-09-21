# Файли та шляхи

Два пакети покривають майже все: `os` для файлової системи та
`path/filepath` для роботи з іменами. Виклики "весь файл за раз" —
ті, до яких ви звертаєтесь найчастіше.

```go
if err := os.WriteFile("notes.txt", []byte("hello\n"), 0o644); err != nil {
    return err
}

b, err := os.ReadFile("notes.txt")
if err != nil {
    return err
}
fmt.Printf("%q\n", string(b))   // output: "hello\n"
```

`ReadFile` повертає `[]byte`, а не рядок, і сам відкриває, читає та
закриває файл. Використовуйте її, коли файл спокійно вміщається в
пам'ять; інакше — потоково, про що розповідає [наступна
стаття](02-readers-and-writers.md).

## Права доступу — вісімкові

Це `0o644` — маска Unix-прав доступу: власник читає й пише, всі інші
лише читають. Префікс `0o` — це синтаксис вісімкового літерала в Go.
Він застосовується лише тоді, коли виклик *створює* файл; наявний файл
зберігає власний режим.

`0o644` для даних, `0o755` для каталогів і виконуваних файлів, `0o600`
коли вміст чутливий.

## Перевірка того, що є

```go
fi, err := os.Stat("notes.txt")
if err != nil {
    return err
}
fmt.Println(fi.Name(), fi.Size(), fi.IsDir(), fi.Mode().Perm())
// output: notes.txt 6 false -rw-r--r--
```

`Perm()` виводить символьну форму тих самих бітів: три групи `rwx` для
власника, групи та інших, з `-` там, де права відсутні. `-rw-r--r--` —
це `0o644`, записаний у зворотному порядку.

## Відсутні файли: перевіряйте помилку, а не шлях

Кожен виклик до файлової системи повертає `*fs.PathError`, що обгортає
конкретну причину, тож `errors.Is` відповідає на питання "чи існує":

```go
_, err := os.ReadFile("nope.txt")

fmt.Println(errors.Is(err, fs.ErrNotExist))   // output: true

var pe *fs.PathError
fmt.Println(errors.As(err, &pe), pe.Op)       // output: true open
```

`os.ErrNotExist` і `fs.ErrNotExist` — те саме значення, тож підходить
будь-яке з них. У старому коді є хелпер `os.IsNotExist(err)`; надавайте
перевагу `errors.Is`, яка бачить крізь обгортання.

Опирайтесь спокусі спочатку викликати `os.Stat`, а потім відкривати. Між
двома викликами відповідь може змінитися, а обробляти невдачу все одно
доведеться — просто відкрийте файл і перевірте помилку.

## Відкриття з контролем

`os.Create` обрізає або створює. `os.OpenFile` — загальна форма, і
дописування — найпоширеніша причина до неї звертатись:

```go
f, _ := os.Create("log.txt")
fmt.Fprintln(f, "line1")
f.Close()

af, _ := os.OpenFile("log.txt", os.O_APPEND|os.O_WRONLY, 0o644)
fmt.Fprintln(af, "line2")
af.Close()
```

```go
b, _ := os.ReadFile("log.txt")
fmt.Printf("%q\n", string(b))   // output: "line1\nline2\n"
```

`*os.File` — це `io.Writer`, тому `fmt.Fprintln` працює на ньому. У
реальному коді закривайте через `defer`:

```go
f, err := os.Open(path)
if err != nil {
    return err
}
defer f.Close()
```

Для файлу, у який ви *писали*, відкладений `Close`, чию помилку ви
відкидаєте, — реальний ризик. Виклики запису можуть завершитись успішно,
а файлова система повідомить про збій — переповнений диск, відпалий
мережевий диск — лише тоді, коли дескриптор закривається. Закривайте
явно й перевіряйте помилку, перш ніж повідомляти про успіх.

## Каталоги

```go
os.MkdirAll("a/b/c", 0o755)   // створює батьківські каталоги, без помилки, якщо вже існує
os.Remove(path)               // один файл або один порожній каталог
os.RemoveAll(dir)             // рекурсивно; без помилки, якщо відсутній
```

`MkdirAll` — те, до чого варто звертатись; звичайний `Mkdir` падає, якщо
батьківський каталог відсутній, і дає помилку, якщо каталог уже існує.
Видалення файлу, якого немає, — це *справді* помилка:

```go
fmt.Println(os.Remove("notes.txt"))        // output: <nil>
fmt.Println(os.Remove("notes.txt") != nil) // output: true
```

Для тимчасового простору дозвольте бібліотеці обрати місце й приберіть
за собою:

```go
dir, err := os.MkdirTemp("", "demo")
if err != nil {
    return err
}
defer os.RemoveAll(dir)
```

## Побудова шляхів через `filepath`

Ніколи не з'єднуйте шляхи через `+` чи `/`. `filepath.Join` використовує
правильний роздільник для платформи й очищає результат:

```go
fmt.Println(filepath.Join("a", "b", "..", "c"))   // output: a/c
fmt.Println(filepath.Join("a", "", "b"))          // output: a/b
```

Порожні сегменти зникають, тож безпечно з'єднувати змінну, яка може
бути порожньою.

```go
fmt.Println(filepath.Base("/x/y/z.txt"))   // output: z.txt
fmt.Println(filepath.Dir("/x/y/z.txt"))    // output: /x/y
fmt.Println(filepath.Ext("/x/y/z.txt"))    // output: .txt
```

`Ext` включає крапку. Прибрати її — робота для `strings`:

```go
name := "z.txt"
fmt.Println(strings.TrimSuffix(name, filepath.Ext(name)))   // output: z
```

`filepath.Rel` дає шлях з одного місця до іншого — так ви перетворюєте
абсолютні результати обходу назад у зручні для читання імена:

```go
rel, _ := filepath.Rel("/x/y", "/x/y/z/w.txt")
fmt.Println(rel)   // output: z/w.txt
```

Є також пакет `path`. Він для розділених слешем речей, які *не є*
шляхами файлової системи — URL, імена у вбудованій (embedded) FS. Для
файлів використовуйте `filepath`.

## Перелічення та обхід

`os.ReadDir` перелічує один каталог. Вона повертає значення
`fs.DirEntry`, які знають ім'я й те, чи це каталог, без додаткового
системного виклику:

```go
ents, err := os.ReadDir(dir)
if err != nil {
    return err
}
for _, e := range ents {
    fmt.Println(e.Name(), e.IsDir())
}
```

`filepath.WalkDir` рекурсивно обходить. Колбек отримує аргумент-помилку,
і перше, що варто зробити, — перевірити її, інакше нечитабельний
підкаталог мовчки обрізає ваш обхід:

```go
var found []string
walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
    if err != nil {
        return err
    }
    if !d.IsDir() && filepath.Ext(path) == ".go" {
        found = append(found, path)
    }
    return nil
})
if walkErr != nil {
    return walkErr
}
```

Повернення ненульової помилки зупиняє обхід, і `WalkDir` повертає її.
Повернення `fs.SkipDir` з каталогу пропускає натомість усе його
піддерево:

```go
filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
    if d.IsDir() && d.Name() == "vendor" {
        return fs.SkipDir
    }
    // ...
    return nil
})
```

Це ідіома для ігнорування `vendor`, `node_modules` чи `.git`.

## Перейменування та копіювання

`os.Rename` переміщує файл, атомарно, коли обидва шляхи на одній
файловій системі, — саме так ви безпечно записуєте файл: пишете під
тимчасовим іменем, а потім перейменовуєте поверх цілі, тож читач ніколи
не побачить наполовину записаний файл.

Функції `os.Copy` немає. Відкрийте обидва і використайте `io.Copy`, у
наступній статті.

> **З досвіду Python:** `os.ReadFile`/`WriteFile` — це
> `pathlib.read_bytes`/`write_bytes`, `filepath.Join` — це
> `os.path.join`, `WalkDir` — це `os.walk` з колбеком замість
> генератора. Звичку, яку варто відкинути, — `if os.path.exists(...)`:
> тут ви пробуєте виконати операцію й перевіряєте помилку через
> `errors.Is`.

## Швидка довідка

| Завдання | Виклик |
|---|---|
| прочитати весь файл | `os.ReadFile(p)` → `[]byte` |
| записати весь файл | `os.WriteFile(p, b, 0o644)` |
| відкрити для читання / запису | `os.Open` / `os.Create` / `os.OpenFile` |
| дописати | `os.OpenFile(p, os.O_APPEND\|os.O_WRONLY, 0o644)` |
| метадані | `os.Stat(p)` → `fs.FileInfo` |
| чи існує | `errors.Is(err, fs.ErrNotExist)` |
| створити каталоги | `os.MkdirAll(p, 0o755)` |
| видалити | `os.Remove` / `os.RemoveAll` |
| тимчасовий каталог | `os.MkdirTemp("", "prefix")` + `defer os.RemoveAll` |
| побудувати шлях | `filepath.Join(...)` |
| розбити шлях | `Base`, `Dir`, `Ext`, `Rel` |
| перелічити один каталог | `os.ReadDir(p)` |
| обійти дерево | `filepath.WalkDir(root, fn)`, `fs.SkipDir` для обрізання |
| перемістити / замінити атомарно | `os.Rename(tmp, target)` |

## Джерела

- [`os` package reference — pkg.go.dev/os](https://pkg.go.dev/os)
- [`path/filepath` package reference — pkg.go.dev/path/filepath](https://pkg.go.dev/path/filepath)
- [`io/fs` package reference — pkg.go.dev/io/fs](https://pkg.go.dev/io/fs)
- [`filepath.WalkDir` — pkg.go.dev/path/filepath#WalkDir](https://pkg.go.dev/path/filepath#WalkDir)
- [`fs.PathError` — pkg.go.dev/io/fs#PathError](https://pkg.go.dev/io/fs#PathError)
