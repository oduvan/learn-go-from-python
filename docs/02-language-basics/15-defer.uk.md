# `defer`

`defer` планує виклик функції на момент повернення з обгортаючої функції. Саме так у Go виконується cleanup (прибирання ресурсів): закриття файлів, зняття блокувань мʼютексів, завершення HTTP-відповідей, зупинка таймерів.

## Базова форма

```go
func read(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()           // виконується при поверненні з read
    return io.ReadAll(f)
}
```

Рядок `defer f.Close()` гарантує, що `f.Close()` буде викликано незалежно від того, як завершиться `read` — звичайне повернення, раннє повернення чи паніка. Більше не потрібно памʼятати «закрий f у кожній точці виходу».

> **З досвіду Python:** `defer` приблизно відповідає тому, що робить `try`/`finally` або оператор `with` — прив'язує прибирання до виходу з поточної області видимості. Відмінність: `defer` стосується *функції*, а не блоку.

## Три семантичні правила

### 1. Порядок LIFO

Відкладені виклики виконуються у зворотному порядку відносно їх планування.

```go
func main() {
    defer fmt.Println("third")
    defer fmt.Println("second")
    defer fmt.Println("first")
    fmt.Println("main")
}
// output:
// main
// first
// second
// third
```

Уявіть стек: кожен `defer` кладе елемент; при поверненні всі вони знімаються.

### 2. Аргументи обчислюються **у момент defer**

Це спантеличує кожного рівно один раз.

```go
func main() {
    x := 1
    defer fmt.Println(x)      // x захоплюється як 1 прямо зараз
    x = 2
}
// output: 1
```

Аргумент `x` обчислюється в момент виконання оператора `defer`, а не тоді, коли функція врешті-решт буде викликана. Якщо потрібно захопити *поточне* значення на момент повернення, відкладайте замикання (closure):

```go
func main() {
    x := 1
    defer func() {
        fmt.Println(x)        // замикається над x — зчитується у момент виклику
    }()
    x = 2
}
// output: 2
```

### 3. Виконується при будь-якому поверненні — включно з панікою

`panic` — це механізм Go для ситуацій «такого не повинно бути» — він зупиняє функцію та починає розкручувати стек, доки або `recover` не перехопить паніку, або програма не завершиться аварійно. Повна розповідь міститься в [16-panic-and-recover.md](16-panic-and-recover.md); важливий для `defer` момент: відкладені виклики **все одно виконуються** під час розкручування.

```go
func cleanup() {
    defer fmt.Println("running cleanup")
    panic("boom")
}
// output:
// running cleanup
// panic: boom
//   ... stack trace ...
```

Саме тому `defer` — природне місце для викликів `recover()` — `recover` робить щось корисне лише всередині відкладеної функції:

```go
func safe() {
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("recovered from:", r)
        }
    }()
    panic("oops")
}
```

## Ідіоматичне використання

### Закриття файлу

```go
f, err := os.Open(path)
if err != nil {
    return err
}
defer f.Close()
```

### Зняття блокування мʼютексу

`sync.Mutex` — це мʼютекс взаємного виключення зі стандартної бібліотеки — детально розглядається в темі конкурентності. Патерн нижче є канонічним використанням `defer`: взяти блокування, одразу запланувати його зняття, а потім виконати будь-яку критичну роботу, яку захищає блокування.

```go
var mu sync.Mutex

func update() {
    mu.Lock()
    defer mu.Unlock()
    // ... критична секція ...
}
```

### Відновлення стану

```go
func quiet() func() {
    oldLevel := log.Default().Flags()
    log.Default().SetFlags(0)
    return func() {
        log.Default().SetFlags(oldLevel)
    }
}

func main() {
    defer quiet()()           // зверніть увагу на подвійні () — quiet повертає функцію cleanup
    log.Println("hello")
}
```

### Вимірювання часу виконання функції

```go
func track(name string) func() {
    start := time.Now()
    return func() {
        fmt.Printf("%s took %v\n", name, time.Since(start))
    }
}

func work() {
    defer track("work")()
    time.Sleep(100 * time.Millisecond)
}
// output: work took 100.xxx ms
```

## Підводні камені

### defer у циклі накопичується

`defer` виконується при поверненні з **функції**, а не після ітерації циклу:

```go
func processAll(paths []string) error {
    for _, p := range paths {
        f, err := os.Open(p)
        if err != nil {
            return err
        }
        defer f.Close()       // !!! всі файли залишаються відкритими до повернення з processAll
        // ... робимо щось ...
    }
    return nil
}
```

Виправлення: виокремте роботу в окрему функцію, щоб `defer` виконувався на кожен виклик:

```go
func processAll(paths []string) error {
    for _, p := range paths {
        if err := processOne(p); err != nil {
            return err
        }
    }
    return nil
}

func processOne(p string) error {
    f, err := os.Open(p)
    if err != nil {
        return err
    }
    defer f.Close()           // закривається на кожній ітерації
    // ... робимо щось ...
    return nil
}
```

### defer має мізерну вартість

Вимірюється в наносекундах. У звичайному коді не варто про це думати. У гарячому внутрішньому циклі (мільйони викликів/с) можна вбудувати cleanup вручну.

### Не відкладайте повернення до перевірки помилки

```go
f, err := os.Open(path)
defer f.Close()               // !!! паніка, якщо err != nil і f дорівнює nil
if err != nil { ... }
```

Завжди спершу перевіряйте `err`, потім `defer`:

```go
f, err := os.Open(path)
if err != nil {
    return err
}
defer f.Close()
```

## Джерела

- [Оператори defer — go.dev/ref/spec#Defer_statements](https://go.dev/ref/spec#Defer_statements)
- [Defer, Panic та Recover — go.dev/blog/defer-panic-and-recover](https://go.dev/blog/defer-panic-and-recover)
- [Effective Go: Defer — go.dev/doc/effective_go#defer](https://go.dev/doc/effective_go#defer)
