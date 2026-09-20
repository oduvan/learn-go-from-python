# Generics: параметри типу та обмеження

**Generics** (узагальнення) дають змогу написати одну функцію чи тип, що
працює з багатьма типами, зберігаючи повну типобезпеку на етапі
компіляції. Там, де інтерфейс абстрагується від *поведінки*, узагальнення
абстрагується від *самого типу* — без `any`, без тверджень типу під час
виконання, без пакування (boxing).

## Параметри типу у функціях

Функція отримує **параметри типу** у квадратних дужках *перед* звичайним
списком параметрів. Кожен параметр типу має **обмеження** (constraint), що
звужує, які типи можна підставити.

```go
func Max[T cmp.Ordered](a, b T) T {
    if a > b {
        return a
    }
    return b
}

fmt.Println(Max(3, 7))         // output: 7
fmt.Println(Max("go", "py"))   // output: py
```

`T` — це параметр типу; `cmp.Ordered` — його обмеження, набір типів, що
підтримують `<`, `>` тощо. Тепер той самий `Max` працює для int, float та
рядків, кожен перевіряється на етапі компіляції.

## Виведення типів

Зазвичай аргумент типу писати не доводиться — компілятор виводить `T` з
аргументів виклику. Ви *можете* вказати його явно, коли виведення
неможливе (або задля ясності):

```go
fmt.Println(Max(3, 7))         // виведено: T = int
fmt.Println(Max[float64](3, 7)) // явно: T = float64 → друкує 7
```

Виведення не обмежується викликами. Скрізь, де узагальнену функцію
**присвоюють змінній відповідного функційного типу** — або перетворюють
до нього, — компілятор з'ясовує аргументи типів саме з цього типу:

```go
func Map[T, U any](s []T, f func(T) U) []U { /* ... */ }

var g func([]int, func(int) string) []string = Map   // виводить T=int, U=string
fmt.Println(g([]int{1, 2}, func(i int) string { return fmt.Sprint(i * 10) }))
// output: [10 20]
```

Тобто узагальнену функцію можна передати просто так усюди, де очікують
конкретний функційний тип — у поле структури, у параметр-колбек, у map
обробників — не виписуючи `Map[int, string]`.

## Обмеження — це інтерфейси

Обмеження — це просто **інтерфейс**, ужитий у позиції параметра типу. Два
вбудовані, які трапляться першими:

- `any` — без обмежень (підходить кожен тип; це буквально `interface{}`)
- `comparable` — типи, що підтримують `==` та `!=`

```go
func Index[T comparable](s []T, target T) int {
    for i, v := range s {
        if v == target {     // == дозволено, бо T є comparable
            return i
        }
    }
    return -1
}

fmt.Println(Index([]string{"a", "b", "c"}, "b"))   // output: 1
```

## Власні обмеження: набори типів та `~`

Інтерфейс-обмеження може перелічити **набір типів** через `|`. Це дозволяє
тілу використовувати оператори, спільні для цих типів. Префікс `~` означає
«будь-який тип, чий *базовий* тип є цим», тож ваші власні визначені типи
теж підходять.

```go
type Number interface {
    ~int | ~int64 | ~float64
}

func Sum[T Number](nums []T) T {
    var total T          // нульове значення T
    for _, n := range nums {
        total += n       // + дозволено: кожен тип у наборі його підтримує
    }
    return total
}

type Celsius float64     // базовий тип — float64
fmt.Println(Sum([]int{1, 2, 3}))            // output: 6
fmt.Println(Sum([]Celsius{1.5, 2.5}))       // output: 4
```

Без `~` `Sum[Celsius]` було б відхилено — `Celsius` не є буквально
`float64`, лише *заснований* на ньому:

```go
type StrictFloat interface{ float64 }   // без ~

func StrictSum[T StrictFloat](xs []T) T { /* ... */ }

StrictSum([]Celsius{1, 2})
// compile error: Celsius does not satisfy StrictFloat
//   (possibly missing ~ for float64 in StrictFloat)
```

Компілятор навіть підказує виправлення. Додайте `~` — і `Celsius` підходить.

## Узагальнені типи

Типи теж приймають параметри типу. Класичний приклад — контейнер, що
тримає елемент будь-якого типу:

```go
type Stack[T any] struct {
    items []T
}

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *Stack[T]) Pop() (T, bool) {
    var zero T
    if len(s.items) == 0 {
        return zero, false
    }
    last := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    return last, true
}

var s Stack[int]
s.Push(1)
s.Push(2)
v, ok := s.Pop()
fmt.Println(v, ok)   // output: 2 true
```

Зверніть увагу на `var zero T` — оскільки ви не знаєте `T`, саме так
отримують його нульове значення. Методи узагальненого типу повторюють
параметр типу в отримувачі: `(s *Stack[T])`.

## Узагальнена множина

Поєднання узагальненого типу з `comparable` дає багаторазову множину —
краще, ніж переписувати `map[T]struct{}` для кожного типу елемента:

```go
type Set[T comparable] map[T]struct{}

func (s Set[T]) Add(v T)      { s[v] = struct{}{} }
func (s Set[T]) Has(v T) bool { _, ok := s[v]; return ok }

s := Set[string]{}
s.Add("go")
fmt.Println(s.Has("go"), s.Has("py"))   // output: true false
```

## Узагальнені методи

Метод може оголошувати **власні параметри типів**, окремі від тих, що
несе отримувач. Це важить щоразу, коли операція мусить *змінити* тип
елемента: множина ID користувачів перетворюється на множину імен, кеш із
одним ключем перекладається на інший. `T` отримувача зафіксовано тим
значенням, на якому ви кличете метод, тож новому типові потрібен власний
параметр.

```go
type Set[T comparable] map[T]struct{}

func (s Set[T]) Add(v T)      { s[v] = struct{}{} }
func (s Set[T]) Has(v T) bool { _, ok := s[v]; return ok }

// MapTo оголошує U для себе — T надходить від отримувача.
func (s Set[T]) MapTo[U comparable](f func(T) U) Set[U] {
    out := Set[U]{}
    for v := range s {
        out.Add(f(v))
    }
    return out
}
```

Виклик виводить `U` із переданої функції — так само, як для узагальненої
функції:

```go
ids := Set[int]{}
ids.Add(1)
ids.Add(2)

names := ids.MapTo(func(id int) string { return fmt.Sprintf("user-%d", id) })
fmt.Println(names.Has("user-1"), names.Has("user-9"))   // output: true false
```

Метод можна інстанціювати явно, коли виведення не зарадить; це заразом
дає багаторазове значення методу:

```go
toString := ids.MapTo[string]
fmt.Println(toString(func(id int) string { return fmt.Sprint(id) }).Has("2"))
// output: true
```

Виграш — у просторі імен. Без власного параметра типу такий метод
довелося б робити функцією рівня пакета — `MapSet`, `MapStack`,
`MapList` — по одній на контейнер, і всі змагалися б за імена в пакеті.
Як метод він живе на тому типі, якому належить.

### Інтерфейси лишаються неузагальненими

Єдине тверде обмеження: **метод інтерфейсу не може оголошувати параметри
типів**, а узагальнений метод не може реалізувати неузагальнений.

```go
type Doer interface {
    Do[T any](T) T   // compile error: interface method must have no type parameters
}
```

Узагальнений метод не має єдиної сталої сигнатури, тож він не задовольняє
метод, який інтерфейс зафіксував:

```go
type Doer interface{ Do(int) int }

type T struct{}
func (T) Do[U any](u U) U { return u }

var _ Doer = T{}
// compile error: T does not implement Doer (wrong type for method Do)
//   have Do[U any](U) U
//   want Do(int) int
```

Динамічній диспетчеризації потрібна одна конкретна сигнатура на метод, а
узагальнений метод — це ціла їх родина. Тримайте методи інтерфейсів
конкретними, а узагальнену роботу кладіть на тип, що їх реалізує.

## Коли не варто вдаватися до узагальнень

Узагальнення сяють для **контейнерів та алгоритмів**, що однакові для
різних типів елементів (колекції, `Map`/`Filter`/`Reduce`, min/max). Вони
*не* заміна інтерфейсам: коли ви хочете, щоб різні типи постачали різну
поведінку за однією абстракцією, це робота інтерфейсу. Емпіричне правило —
якщо єдине, що змінюється, це *тип*, беріть узагальнення; якщо змінюється
*поведінка*, беріть інтерфейс.

> **З погляду Python:** це територія `typing.TypeVar` / `Generic[T]`, але
> забезпечена компілятором, а не необов'язковим перевіряльником — і з
> нульовою вартістю під час виконання, бо типи визначаються на етапі
> збирання.

## Швидка довідка

| Форма | Значення |
|---|---|
| `func F[T any](x T)` | функція з параметром типу |
| `[T cmp.Ordered]` | обмеження, що дозволяє `<`, `>` |
| `[T comparable]` | обмеження, що дозволяє `==`, `!=` |
| `interface{ ~int \| ~float64 }` | обмеження-набір типів; `~` = базовий тип |
| `type Box[T any] struct{ v T }` | узагальнений тип |
| `func (b Box[T]) Get() T` | метод узагальненого типу |
| `func (b Box[T]) To[U any](...)` | узагальнений метод — власний параметр типу |
| `var f func(int) string = G` | виведення з присвоєння функційному типу |
| `var zero T` | нульове значення параметра типу |

## Джерела

- [Type parameters — go.dev/ref/spec#Type_parameter_declarations](https://go.dev/ref/spec#Type_parameter_declarations)
- [Type constraints — go.dev/ref/spec#Type_constraints](https://go.dev/ref/spec#Type_constraints)
- [The `comparable` constraint — go.dev/ref/spec#Comparison_operators](https://go.dev/ref/spec#Comparison_operators)
- [cmp.Ordered — pkg.go.dev/cmp#Ordered](https://pkg.go.dev/cmp#Ordered)
- [Go blog: an introduction to generics — go.dev/blog/intro-generics](https://go.dev/blog/intro-generics)
- [Tutorial: getting started with generics — go.dev/doc/tutorial/generics](https://go.dev/doc/tutorial/generics)
- [Method declarations — go.dev/ref/spec#Method_declarations](https://go.dev/ref/spec#Method_declarations)
- [Type inference — go.dev/ref/spec#Type_inference](https://go.dev/ref/spec#Type_inference)
