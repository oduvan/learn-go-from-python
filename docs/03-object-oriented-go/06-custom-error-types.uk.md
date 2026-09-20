# Власні типи помилок

`error` — це інтерфейс з одним-єдиним методом. Усе, що має метод
`Error() string`, є помилкою, а отже, ви можете визначати власні типи
помилок і додавати їм будь-які поля:

```go
type ValidationError struct {
    Field string
    Msg   string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Msg)
}

var err error = &ValidationError{Field: "email", Msg: "must contain @"}
fmt.Println(err)   // output: email: must contain @
```

Жодної реєстрації, жодного базового класу. [Стаття про
помилки](../02-language-basics/07-errors.md) обіцяла це, щойно методи та
інтерфейси стануть на місце, — і ось воно.

## Дані, а не лише текст

Сигнальна помилка (sentinel error) на кшталт `errors.New("invalid field")`
— це фіксований рядок. Щойно виклику потрібно щось *зробити* з деталлю —
підсвітити конкретне поле форми, повторити спробу лише при 503, повідомити
номер рядка — рядок змушує його розбирати текст назад. Тип передає ці дані
напряму:

```go
func checkAge(s string) error {
    return &ValidationError{Field: "age", Msg: "not a number"}
}
```

Виклик отримує поля, а не речення.

## `errors.As` повертає ваш тип

`errors.As` проходить ланцюжок помилок у пошуку значення, яке можна
присвоїти цілі, і присвоює його, якщо знаходить:

```go
err := checkAge("x")

var ve *ValidationError
if errors.As(err, &ve) {
    fmt.Println(ve.Field, "|", ve.Msg)   // output: age | not a number
}
```

Другий аргумент — це **вказівник на** змінну вашого типу помилки: `&ve`,
де `ve` вже є `*ValidationError`. Саме через цю подвійну непряму адресацію
`As` записує результат назад.

## `Unwrap` вбудовує ваш тип у ланцюжок

Тип помилки, що обгортає іншу помилку, має відкривати її через метод
`Unwrap() error`. Саме цей один метод дозволяє `errors.Is` та `errors.As`
бачити крізь ваш тип те, що лежить під ним:

```go
var ErrPermission = errors.New("permission denied")

type QueryError struct {
    Query string
    Err   error
}

func (e *QueryError) Error() string { return e.Query + ": " + e.Err.Error() }
func (e *QueryError) Unwrap() error { return e.Err }

qe := &QueryError{Query: "SELECT 1", Err: ErrPermission}
fmt.Println(qe)                            // output: SELECT 1: permission denied
fmt.Println(errors.Is(qe, ErrPermission))  // output: true
```

Ваш тип так само компонується з обгортанням через `fmt.Errorf`. Ланцюжок
може складатися частково із сигнальних помилок, частково з власних типів,
і обидва види пошуку однаково працюють ззовні:

```go
wrapped := fmt.Errorf("loading user: %w", qe)
fmt.Println(wrapped)
// output: loading user: SELECT 1: permission denied

fmt.Println(errors.Is(wrapped, ErrPermission))   // output: true

var q *QueryError
fmt.Println(errors.As(wrapped, &q), q.Query)     // output: true SELECT 1
```

## Отримувач за вказівником чи за значенням — обирайте вказівник

Визначайте `Error()` з отримувачем за вказівником і завжди повертайте
`&T{...}`. На це є дві причини, і друга з них боляче кусається.

Набори методів: з `func (e *ValidationError) Error() string` інтерфейс
`error` задовольняє лише `*ValidationError` — звичайний `ValidationError`
ні. Змішайте їх, і `errors.As` не просто промахнеться — вона панікує:

```go
var ve ValidationError          // value, not pointer
errors.As(err, &ve)
// panic: errors: *target must be interface or implement error
```

Ідентичність: два окремі значення `&T{}` — це різні вказівники, тож
порівняння лишаються передбачуваними. Два однакові структурні *значення*
порівнялися б як рівні, і це непомітно перетворило б непов'язані помилки
на одну й ту саму.

## Власний метод `Is`

За замовчуванням `errors.Is` порівнює через `==`, тож два різні вказівники
ніколи не збігаються, навіть якщо їхній вміст однаковий. Якщо вам потрібне
порівняння за значенням, заявіть про це методом `Is`:

```go
type HTTPError struct{ Code int }

func (e *HTTPError) Error() string { return fmt.Sprintf("http %d", e.Code) }

func (e *HTTPError) Is(target error) bool {
    t, ok := target.(*HTTPError)
    return ok && t.Code == e.Code
}

fmt.Println(errors.Is(&HTTPError{404}, &HTTPError{404}))   // output: true
fmt.Println(errors.Is(&HTTPError{500}, &HTTPError{404}))   // output: false
```

`errors.Is` викликає ваш `Is` на кожному кроці ланцюжка, тож це працює і
крізь обгортання.

## Пастка типізованого nil

Це [пастка типізованого nil](02-interfaces.md), перевдягнена в костюм
помилки, — і це найпоширеніший спосіб випадково відвантажити баг разом із
власним типом помилки:

```go
func find() error {
    var e *ValidationError   // nil pointer
    return e                 // ...but not a nil error
}

fmt.Println(find() == nil)   // output: false
```

Повернений інтерфейс тримає тип `*ValidationError` та значення `nil`, а
інтерфейс є `nil`, лише коли `nil` **обидві** його половини. Оголошуйте
змінну результату як `error`, а не як ваш конкретний тип, і повертайте
буквальний `nil` на успішному шляху.

## Сигнальна помилка чи тип помилки?

| Що використати | Коли |
|---|---|
| сигнальна — `var ErrX = errors.New(...)` | виклику треба знати лише, *яка саме* сталася невдача |
| тип помилки | виклику потрібні дані з цієї невдачі |
| тип з `Unwrap` | ви додаєте контекст навколо чужої помилки |
| тип з `Is` | два екземпляри мають вважатися рівними за значенням |

Починайте із сигнальної помилки. Сягайте по тип, коли інакше виклику
довелося б читати текст повідомлення.

> **З погляду Python:** це те саме, що `class ValidationError(Exception)`
> з полями, а `errors.As` — це `except ValidationError as e:`, що зв'язує
> її з ім'ям. Різниця в тому, що ніщо не розкручує стек за вас, а
> ланцюжок явний — `Unwrap` це як `__cause__`, тільки писати його
> доводиться самому.

## Швидка довідка

| Форма | Значення |
|---|---|
| `func (e *T) Error() string` | робить `*T` типом `error` |
| `func (e *T) Unwrap() error` | відкриває обгорнуту помилку для `Is`/`As` |
| `func (e *T) Is(target error) bool` | власна рівність для `errors.Is` |
| `var t *T; errors.As(err, &t)` | дістати ваш тип із ланцюжка |
| `return &T{...}` | завжди вказівник; ніколи не повертайте nil `*T` як `error` |

## Джерела

- [Довідник пакету `errors` — pkg.go.dev/errors](https://pkg.go.dev/errors)
- [`errors.As` — pkg.go.dev/errors#As](https://pkg.go.dev/errors#As)
- [Go blog: working with errors in Go 1.13 — go.dev/blog/go1.13-errors](https://go.dev/blog/go1.13-errors)
- [Errors are values — go.dev/blog/errors-are-values](https://go.dev/blog/errors-are-values)
- [Method sets — go.dev/ref/spec#Method_sets](https://go.dev/ref/spec#Method_sets)
- [Why is my nil error value not equal to nil? — go.dev/doc/faq#nil_error](https://go.dev/doc/faq#nil_error)
