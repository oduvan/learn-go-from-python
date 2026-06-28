# Твердження типу та перемикачі типів

Інтерфейсне значення ховає конкретний тип усередині себе. Коли вам
потрібно повернути цей конкретний тип — щоб викликати метод, якого
інтерфейс не виставляє, або щоб розгалузитися за тим, що насправді
збережено, — Go дає два інструменти: **твердження типу** (type assertion)
та **перемикач типів** (type switch).

## Твердження типу

Твердження типу `x.(T)` заявляє, що інтерфейсне значення `x` тримає
значення типу `T`, і витягає його. Воно має дві форми.

Форма **з одним результатом** повертає значення і **панікує**, якщо
динамічний тип не `T`:

```go
var x any = "hello"
s := x.(string)
fmt.Println(s)        // output: hello

n := x.(int)          // panic: interface conversion: interface {} is string, not int
```

Форма **comma-ok** ніколи не панікує — вона повертає значення плюс булеве,
яке повідомляє, чи спрацювало твердження:

```go
var x any = "hello"

s, ok := x.(string)
fmt.Println(s, ok)    // output: hello true

n, ok := x.(int)
fmt.Println(n, ok)    // output: 0 false   — n є нульовим значенням
```

Надавайте перевагу comma-ok, якщо ви не впевнені в типі; форма з панікою —
для випадків, коли неправильний тип є справжнім багом.

## Твердження до інтерфейсного типу

`T` не обов'язково має бути конкретним типом — це може бути **інший
інтерфейс**. Тоді твердження запитує: «чи задовольняє збережене значення
*цей* інтерфейс також?» Саме так перевіряють наявність необов'язкової
можливості.

```go
var w any = bytes.NewBufferString("hi")

if s, ok := w.(fmt.Stringer); ok {
    fmt.Println(s.String())   // output: hi
}
```

Тут `w` тримає `*bytes.Buffer`; твердження успішне, бо цей тип має метод
`String() string`, тож він задовольняє `fmt.Stringer`.

## Перемикачі типів

Коли треба розгалузитися між кількома можливими типами, ланцюжок тверджень
незграбний. **Перемикач типів** робить це однією конструкцією: особлива
форма `x.(type)` (дозволена лише всередині `switch`) перевіряє динамічний
тип, а `v := x.(type)` прив'язує `v` до значення з відповідним типом у
кожному випадку.

```go
func describe(x any) string {
    switch v := x.(type) {
    case nil:
        return "nil"
    case int:
        return fmt.Sprintf("int: %d", v)        // тут v має тип int
    case string:
        return fmt.Sprintf("string of len %d", len(v))   // тут v має тип string
    default:
        return fmt.Sprintf("other: %T", v)      // v зберігає свій початковий тип
    }
}

fmt.Println(describe(42))      // output: int: 42
fmt.Println(describe("hi"))    // output: string of len 2
fmt.Println(describe(nil))     // output: nil
fmt.Println(describe(3.14))    // output: other: float64
```

Кілька правил, які варто знати:

- `case nil` збігається з nil-інтерфейсним значенням.
- У випадку з **одним** типом `v` має цей конкретний тип. У випадку з
  **кількома** типами (`case int, int64:`) або в `default` `v` зберігає
  початковий інтерфейсний тип.
- Випадки перевіряються згори вниз; перший збіг перемагає.

## Реальне застосування: дослідження типів помилок

Поширене місце, де це виринає, — дослідження конкретного типу помилки.
Перемикач типів читається чисто:

```go
switch e := err.(type) {
case *os.PathError:
    fmt.Println("path problem:", e.Path)
case nil:
    fmt.Println("no error")
default:
    fmt.Println("some error:", e)
}
```

Для *загорнутих* помилок стандартна `errors.As` краща за голе твердження,
бо вона розкручує ланцюжок. Вона бере вказівник на змінну цільового типу й
заповнює її, якщо будь-яка помилка в ланцюжку збігається:

```go
_, err := os.Open("/nope/nope")

var pe *os.PathError
if errors.As(err, &pe) {
    fmt.Println("path:", pe.Path)   // output: path: /nope/nope
}
```

Механізм під капотом — та сама ідея, що й твердження типу; `errors.As`
просто проходить загорнутий ланцюжок за вас.

> **З погляду Python:** перемикач типів — це ідіоматична заміна драбини
> `isinstance(x, T)`, а comma-ok твердження грають роль захищеної
> перевірки `isinstance` перед використанням значення як конкретного типу.

## Швидка довідка

| Форма | Результат |
|---|---|
| `v := x.(T)` | витягти `T`; **панікує** при розбіжності |
| `v, ok := x.(T)` | витягти `T`; `ok` false (а `v` нульове) при розбіжності |
| `x.(SomeInterface)` | успішне, якщо динамічний тип задовольняє цей інтерфейс |
| `switch v := x.(type) { ... }` | розгалуження за динамічним типом |
| `case nil:` | збігається з nil-інтерфейсним значенням |
| випадок з кількома типами / `default` | `v` зберігає інтерфейсний тип |

## Джерела

- [Type assertions — go.dev/ref/spec#Type_assertions](https://go.dev/ref/spec#Type_assertions)
- [Type switches — go.dev/ref/spec#Type_switches](https://go.dev/ref/spec#Type_switches)
- [errors.As — pkg.go.dev/errors#As](https://pkg.go.dev/errors#As)
- [Effective Go: interface conversions and type assertions — go.dev/doc/effective_go#interface_conversions](https://go.dev/doc/effective_go#interface_conversions)
