# Читачі та письменники

`io.Reader` та `io.Writer` — два найважливіших інтерфейси в Go.
[Інтерфейси](../03-object-oriented-go/02-interfaces.md) представили їх
як приклади маленьких інтерфейсів; тут показано, як ними користуватись
насправді.

```go
b, err := io.ReadAll(strings.NewReader("abc"))
fmt.Println(err)                 // output: <nil>
fmt.Printf("%q\n", string(b))   // output: "abc"
```

## По одному методу

```go
type Reader interface { Read(p []byte) (n int, err error) }
type Writer interface { Write(p []byte) (n int, err error) }
```

Це весь контракт, і саме тому файл, мережеве з'єднання, тіло HTTP-
запиту, gzip-потік, `strings.Reader` і `bytes.Buffer` — усі
взаємозамінні. Функція, що приймає `io.Reader`, працює з кожним із них,
включно з тими, яких не існувало, коли її написали.

Беріть інтерфейс, а не `*os.File`. Це нічого не коштує й робить функцію
тестованою через `strings.NewReader`.

## `io.Copy` — робочий кінь

Копіювання переміщує дані фіксованими шматками, тож пам'ять залишається
рівною незалежно від розміру джерела:

```go
var dst bytes.Buffer
n, err := io.Copy(&dst, strings.NewReader("stream me"))
fmt.Println(n, err, dst.String())   // output: 9 <nil> stream me
```

Копіювання файлу — це два відкриття й один `Copy` — функції `os.Copy`
немає:

```go
in, err := os.Open(src)
if err != nil {
    return err
}
defer in.Close()

out, err := os.Create(dst)
if err != nil {
    return err
}
if _, err := io.Copy(out, in); err != nil {
    out.Close()
    return err
}
return out.Close()   // закриття, яке має значення: повідомте його помилку
```

`io.Discard` — це письменник, який усе викидає, корисний для
спорожнення тіла запиту, яке вам байдуже:

```go
n, _ := io.Copy(io.Discard, strings.NewReader("thrown away"))
fmt.Println(n)   // output: 11
```

## `io.EOF` — це значення, а не збій

`Read` повертає кількість отриманих байтів і помилку. У кінці вводу ця
помилка — `io.EOF`, що є звичайним результатом:

```go
r := strings.NewReader("hello")
buf := make([]byte, 2)
for {
    n, err := r.Read(buf)
    if n > 0 {
        fmt.Printf("%q ", string(buf[:n]))
    }
    if err == io.EOF {
        break
    }
}
// output: "he" "ll" "o"
```

Два правила, на яких люди спотикаються. **Обробляйте байти до перевірки
помилки** — `Read` може повернути дані *й* `io.EOF` одночасно. І завжди
зрізайте до `buf[:n]`; решта буфера містить те, що там було раніше.

Ви рідко пишете цей цикл самі. `io.ReadAll`, `io.Copy` і
`bufio.Scanner` роблять це за вас.

`io.ReadFull` — виняток, що трактує короткий зчитаний фрагмент як
помилку:

```go
_, err := io.ReadFull(strings.NewReader("abc"), make([]byte, 10))
fmt.Println(errors.Is(err, io.ErrUnexpectedEOF), err)
// output: true unexpected EOF
```

## `bufio.Scanner` для рядків

Читання рядок за рядком — достатньо поширена задача, щоб мати
спеціальний тип:

```go
sc := bufio.NewScanner(r)
for sc.Scan() {
    fmt.Println(sc.Text())
}
if err := sc.Err(); err != nil {
    return err
}
```

`Scan` повертає false і в кінці вводу, *і* при помилці, тож
**перевірка `sc.Err()` після циклу не опційна** — пропустіть її, і збій
читання буде виглядати точнісінько як чистий кінець файлу.

`Text()` повертає рядок без символу нового рядка. `Bytes()` уникає
виділення пам'яті, але дійсний лише до наступного `Scan`.

Змінити те, що вважається токеном, можна через `Split`:

```go
sc := bufio.NewScanner(strings.NewReader("a bb  ccc"))
sc.Split(bufio.ScanWords)
for sc.Scan() {
    fmt.Printf("%q ", sc.Text())
}
// output: "a" "bb" "ccc"
```

`bufio.ScanLines` — типовий варіант; `ScanWords`, `ScanRunes` і
`ScanBytes` — інші.

### Ліміт у 64 КБ

`Scanner` відмовляється від будь-якого токена, довшого за 64 КБ, і
повідомляє про це як про помилку, а не обрізає:

```go
long := strings.Repeat("x", 100000)

sc := bufio.NewScanner(strings.NewReader(long))
fmt.Println(sc.Scan(), sc.Err())
// output: false bufio.Scanner: token too long
```

Це класичний баг обробника логів: він працює, поки не прийде один
величезний рядок, а тоді зупиняється, і без перевірки `sc.Err()`
зупиняється *мовчки*. Піднімайте межу, коли рядки можуть бути
довгими:

```go
sc := bufio.NewScanner(strings.NewReader(long))
sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
fmt.Println(sc.Scan(), sc.Err(), len(sc.Text()))
// output: true <nil> 100000
```

Для дійсно необмежених рядків використовуйте
`bufio.Reader.ReadString('\n')`, який росте за потреби.

## `bufio.Writer` треба скидати

Обгортання письменника пакує багато дрібних записів у кілька великих.
Буферизовані дані не записуються, поки ви не скажете:

```go
var out bytes.Buffer
w := bufio.NewWriter(&out)
w.WriteString("buffered")

fmt.Printf("%q ", out.String())   // output: ""
w.Flush()
fmt.Printf("%q\n", out.String())  // output: "buffered"
```

`defer w.Flush()` відкидає помилку. Якщо дані важливі, скидайте явно й
перевіряйте.

## Комбінування читачів і письменників

```go
var a, b bytes.Buffer
mw := io.MultiWriter(&a, &b)
fmt.Fprint(mw, "both")            // пише в обидва
```

```go
lb, _ := io.ReadAll(io.LimitReader(strings.NewReader("abcdefgh"), 3))
fmt.Printf("%q\n", string(lb))    // output: "abc"
```

`io.LimitReader` обмежує, скільки читається — стандартний захист від
тіла запиту, яке заявляє, що воно нескінченне. `io.TeeReader` пропускає
дані крізь себе, копіюючи їх ще кудись, а `io.MultiReader` з'єднує
читачів один за одним.

## Написання власного

Реалізуйте один метод — і все вище працює з вашим типом:

```go
type upperReader struct{ r io.Reader }

func (u upperReader) Read(p []byte) (int, error) {
    n, err := u.r.Read(p)
    for i := range p[:n] {
        if p[i] >= 'a' && p[i] <= 'z' {
            p[i] -= 32
        }
    }
    return n, err
}
```

```go
b, _ := io.ReadAll(upperReader{strings.NewReader("shout")})
fmt.Printf("%q\n", string(b))   // output: "SHOUT"
```

Зверніть увагу: перетворюється лише `p[:n]`, а помилка передається без
змін.

## Звідки береться кожне джерело

| У вас є | Обгорніть через |
|---|---|
| `string` | `strings.NewReader(s)` |
| `[]byte` | `bytes.NewReader(b)` |
| байти, що ростуть | `bytes.Buffer` — і читач, *і* письменник |
| рядок, який будується | `strings.Builder` — лише письменник |
| файл | `os.Open` / `os.Create` |
| стандартні потоки | `os.Stdin`, `os.Stdout`, `os.Stderr` |
| нічого | `io.Discard` |

`strings.NewReader` — причина, чому кожен приклад у цій статті можна
запустити без файлу, і саме так варто тестувати будь-що у формі читача.

> **З досвіду Python:** `io.Reader` — це кінець файлового об'єкта для
> читання, але звужений до одного методу, тож будь-що може ним бути.
> `io.Copy` — це `shutil.copyfileobj`. Дві відмінності, які кусаються:
> `EOF` — це повернуте значення, а не порожній рядок, а `bufio.Scanner`
> мовчки обмежує рядки 64 КБ там, де ітерація в Python такого ліміту
> не має.

## Швидка довідка

| Завдання | Виклик |
|---|---|
| прочитати все | `io.ReadAll(r)` |
| перенести з одного в інше потоково | `io.Copy(w, r)` |
| скопіювати файл | `os.Open` + `os.Create` + `io.Copy` |
| рядок за рядком | `bufio.NewScanner(r)`, потім **перевірте `sc.Err()`** |
| довгі рядки | `sc.Buffer(...)` або `bufio.Reader.ReadString('\n')` |
| пакувати дрібні записи | `bufio.NewWriter(w)` + `Flush()` |
| обмежити ввід | `io.LimitReader(r, n)` |
| розгалузити | `io.MultiWriter(a, b)` |
| спостерігати проходячи повз | `io.TeeReader(r, w)` |
| викинути | `io.Discard` |
| читач із рядка | `strings.NewReader(s)` |

## Джерела

- [`io` package reference — pkg.go.dev/io](https://pkg.go.dev/io)
- [`bufio` package reference — pkg.go.dev/bufio](https://pkg.go.dev/bufio)
- [`bufio.Scanner` — pkg.go.dev/bufio#Scanner](https://pkg.go.dev/bufio#Scanner)
- [`io.Copy` — pkg.go.dev/io#Copy](https://pkg.go.dev/io#Copy)
- [`io.Reader` — pkg.go.dev/io#Reader](https://pkg.go.dev/io#Reader)
- [Effective Go: interfaces — go.dev/doc/effective_go#interfaces](https://go.dev/doc/effective_go#interfaces)
