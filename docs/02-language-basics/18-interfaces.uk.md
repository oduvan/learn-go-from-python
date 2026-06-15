# Інтерфейси

**Інтерфейс** — це тип, який перелічує набір сигнатур методів. Будь-яке
значення, тип якого має *всі* ці методи, задовольняє інтерфейс — і його
можна зберегти у змінній цього інтерфейсного типу. Інтерфейси — це те, як
Go реалізує поліморфізм: код залежить від того, *що значення вміє робити*
(його методів), а не від його конкретного типу.

```go
type Shape interface {
    Area() float64
}
```

`Shape` тепер є типом. Змінна типу `Shape` може містити будь-яке значення,
що має метод `Area() float64`.

## Задоволення є неявним

Ключового слова `implements` немає. Тип задовольняє інтерфейс просто тим,
що має потрібні методи — компілятор перевіряє це структурно. Ви ніколи не
оголошуєте цей зв'язок; він просто існує.

```go
type Rectangle struct{ W, H float64 }
func (r Rectangle) Area() float64 { return r.W * r.H }

type Circle struct{ R float64 }
func (c Circle) Area() float64 { return math.Pi * c.R * c.R }

var s Shape = Rectangle{W: 3, H: 4}   // Rectangle задовольняє Shape — оголошення не потрібне
fmt.Println(s.Area())                 // output: 12
```

І `Rectangle`, і `Circle` задовольняють `Shape`, жодного разу його не
згадавши.

> **З погляду Python:** це качина типізація — «якщо має методи, то
> підходить» — але перевірена на **етапі компіляції**. Тип, у якого бракує
> методу, просто не скомпілюється там, де очікується інтерфейс, замість
> того щоб впасти під час виконання.

## Поліморфізм: одна функція, багато типів

Оскільки будь-який `Shape` має `Area()`, функція може приймати інтерфейс і
працювати з кожним конкретним типом однаково:

```go
func totalArea(shapes []Shape) float64 {
    sum := 0.0
    for _, s := range shapes {
        sum += s.Area()
    }
    return sum
}

shapes := []Shape{Rectangle{3, 4}, Circle{1}}
fmt.Printf("%.2f\n", totalArea(shapes))   // output: 15.14
```

## Інтерфейсне значення — це пара (тип, значення)

Усередині інтерфейсне значення містить дві речі: **динамічний тип** того,
що збережено, і саме **значення**. Нульове значення інтерфейсу — `nil`:
ні типу, ні значення.

```go
var s Shape          // nil-інтерфейс
fmt.Println(s == nil)   // output: true
```

Виклик методу на `nil`-інтерфейсі спричиняє паніку, бо немає конкретного
методу, до якого можна було б диспетчеризувати виклик.

## Отримувачі за вказівником чи за значенням визначають задоволення

Це найпоширеніша пастка інтерфейсів. **Набір методів** типу визначає, які
інтерфейси він задовольняє:

- методи з **отримувачем за значенням** належать і `T`, і `*T`
- методи з **отримувачем за вказівником** належать лише `*T`

Тож якщо метод має отримувача за вказівником, інтерфейс задовольняє лише
**вказівник** — значення ні:

```go
type Counter struct{ n int }
func (c *Counter) Add()        { c.n++ }      // отримувач за вказівником
func (c Counter) Value() int   { return c.n }

type Adder interface{ Add() }

var a Adder = &Counter{}   // ок: *Counter має Add
// var a Adder = Counter{} // compile error: Counter does not implement Adder
//                         //   (method Add has pointer receiver)
a.Add()
```

Емпіричне правило: якщо хоч одному методу потрібен отримувач за
вказівником, передавайте вказівник, коли хочете, щоб значення
задовольняло інтерфейс.

## Порожній інтерфейс та `any`

Інтерфейс без методів задовольняє **кожен** тип. Його сучасне написання —
`any` (псевдонім для `interface{}`), і це спосіб тримати «значення
невідомого типу».

```go
var x any
x = 42
x = "hello"
fmt.Println(x)   // output: hello
```

`any` — це засіб останньої надії: ви втрачаєте всю інформацію про тип на
етапі компіляції. Щоб дістати конкретне значення назад, ви використовуєте
твердження типу або перемикач типів.

> **З погляду Python:** `any` — це найближче до звичайного посилання на
> `object`: воно може тримати будь-що, і перед конкретним використанням
> треба перевірити тип.

## Твердження типу: дістаємо конкретне значення назад

**Твердження типу** `x.(T)` витягає конкретне значення типу `T` з
інтерфейсу. Форма з одним результатом **панікує**, якщо тип не збігається;
форма **comma-ok** натомість повідомляє про успіх.

```go
var x any = "hello"

s := x.(string)        // ок — x справді тримає string
fmt.Println(s)         // output: hello

n, ok := x.(int)       // безпечна форма — без паніки
fmt.Println(n, ok)     // output: 0 false
```

Завжди надавайте перевагу формі comma-ok, якщо ви не впевнені в типі.

## Перемикач типів: розгалуження за динамічним типом

**Перемикач типів** (type switch) перевіряє інтерфейсне значення проти
кількох типів одразу. Форма `v := x.(type)` прив'язує `v` до конкретного
значення в кожному випадку.

```go
func describe(x any) string {
    switch v := x.(type) {
    case int:
        return fmt.Sprintf("int: %d", v)
    case string:
        return fmt.Sprintf("string of len %d", len(v))
    default:
        return "unknown"
    }
}

fmt.Println(describe(42))      // output: int: 42
fmt.Println(describe("hi"))    // output: string of len 2
```

> **З погляду Python:** перемикач типів — це ідіоматична заміна ланцюжка
> перевірок `isinstance(x, T)`.

## Пастка типізованого nil

Інтерфейс є `nil`, лише коли **і** його тип, **і** значення є nil. Якщо ви
збережете nil-*вказівник* в інтерфейсі, інтерфейс тримає тип, тож він
**не** nil — часте джерело багів.

```go
type T struct{}
func (t *T) Foo() {}

var p *T = nil
var i interface{ Foo() } = p
fmt.Println(p == nil)   // output: true
fmt.Println(i == nil)   // output: false  — i має тип *T, тож він не-nil
```

Висновок: повертайте буквальний `nil` для інтерфейсу, а не nil конкретний
вказівник, коли маєте на увазі «нічого».

## Маленькі інтерфейси — це ідіоматично

Go надає перевагу крихітним інтерфейсам — часто з одним методом — які
оголошують там, де їх *використовують*, а не там, де визначають типи.
Стандартна бібліотека ними рясніє:

| Інтерфейс | Метод | Призначення |
|---|---|---|
| `fmt.Stringer` | `String() string` | власна текстова форма |
| `error` | `Error() string` | тип помилки |
| `io.Reader` | `Read([]byte) (int, error)` | джерело байтів |
| `io.Writer` | `Write([]byte) (int, error)` | приймач байтів |
| `sort.Interface` | `Len`/`Less`/`Swap` | власне сортування |

Реалізуйте `fmt.Stringer`, і `fmt.Println` використає його автоматично:

```go
type Color struct{ R, G, B int }
func (c Color) String() string {
    return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}
fmt.Println(Color{255, 165, 0})   // output: #FFA500
```

Настанова: **приймайте інтерфейси, повертайте конкретні типи.** Беріть
найменший інтерфейс, який ваша функція справді потребує, як параметр, а
повертайте конкретну структуру, щоб виклику лишалася повна інформація.

## Швидка довідка

| Поняття | Синтаксис |
|---|---|
| оголосити інтерфейс | `type Reader interface { Read(p []byte) (int, error) }` |
| задовольнити його | просто визначте методи — без ключового слова |
| порожній інтерфейс | `any` (= `interface{}`), тримає будь-яке значення |
| твердження типу (панікує) | `s := x.(string)` |
| твердження типу (безпечне) | `s, ok := x.(string)` |
| перемикач типів | `switch v := x.(type) { ... }` |
| nil-інтерфейс | і тип, і значення є nil |

## Джерела

- [Interface types — go.dev/ref/spec#Interface_types](https://go.dev/ref/spec#Interface_types)
- [Type assertions — go.dev/ref/spec#Type_assertions](https://go.dev/ref/spec#Type_assertions)
- [Type switches — go.dev/ref/spec#Type_switches](https://go.dev/ref/spec#Type_switches)
- [Method sets — go.dev/ref/spec#Method_sets](https://go.dev/ref/spec#Method_sets)
- [Effective Go: interfaces — go.dev/doc/effective_go#interfaces](https://go.dev/doc/effective_go#interfaces)
- [Go blog: errors are values / typed nil — go.dev/doc/faq#nil_error](https://go.dev/doc/faq#nil_error)
- [fmt.Stringer — pkg.go.dev/fmt#Stringer](https://pkg.go.dev/fmt#Stringer)
