# Сортування

Сортування живе у двох пакетах. `slices` — той, яким варто користуватись;
`sort` — старіше API, яке ви ще зустрінете в наявному коді. Помічники для
порівняння в `cmp` — те, що робить новий пакет приємним у користуванні.

```go
s := []int{3, 1, 2}
slices.Sort(s)
fmt.Println(s)   // output: [1 2 3]
```

## `slices.Sort` для впорядкованих типів

`slices.Sort` працює з будь-яким зрізом, елементи якого мають природний
порядок — числа, рядки і все, чий базовий тип один з них. Він сортує на
місці й нічого не повертає:

```go
w := []string{"pear", "Apple", "fig"}
slices.Sort(w)
fmt.Println(w)   // output: [Apple fig pear]
```

`Apple` сортується першим, бо порівняння побайтове, а `A` — це 65, тоді
як `f` — 102. Сортування рядків — це не алфавітний порядок, і воно не
враховує локаль.

## `slices.SortFunc` повертає `int`, а не `bool`

Для всього іншого ви надаєте функцію порівняння. Вона повертає
**від'ємне число, нуль або додатне число** — не булеве значення:

```go
type user struct {
    Name string
    Age  int
}

us := []user{{"Bo", 30}, {"Ada", 25}, {"Cy", 30}}
slices.SortFunc(us, func(a, b user) int {
    return cmp.Compare(a.Age, b.Age)
})
fmt.Println(us)   // output: [{Ada 25} {Bo 30} {Cy 30}]
```

`cmp.Compare` дає саме такий трійковий результат, тож порівняння вручну
пишуть рідко:

```go
fmt.Println(cmp.Compare(1, 2), cmp.Compare(2, 2), cmp.Compare(3, 2))
// output: -1 0 1
```

Щоб відсортувати за спаданням, поміняйте аргументи місцями, а не
заперечуйте результат — заперечення веде себе неправильно на межі
діапазону цілого числа:

```go
d := []int{3, 1, 2}
slices.SortFunc(d, func(a, b int) int { return cmp.Compare(b, a) })
fmt.Println(d)   // output: [3 2 1]
```

Сортування без урахування регістру — це порівняння на перетворених
значеннях:

```go
w := []string{"pear", "Apple", "fig"}
slices.SortFunc(w, func(a, b string) int {
    return cmp.Compare(strings.ToLower(a), strings.ToLower(b))
})
fmt.Println(w)   // output: [Apple fig pear]
```

## `cmp.Or` для другого ключа сортування

`cmp.Or` повертає свій перший ненульовий аргумент. Оскільки «рівні» —
це нуль, вона прямо виражає «сортувати за віком, потім за іменем»:

```go
slices.SortFunc(us, func(a, b user) int {
    return cmp.Or(
        cmp.Compare(a.Age, b.Age),
        cmp.Compare(a.Name, b.Name),
    )
})
fmt.Println(us)   // output: [{Ada 25} {Bo 30} {Cy 30}]
```

Вона не специфічна для сортування — це загальний помічник «перше
ненульове значення», який також добре читається для значень за
замовчуванням:

```go
fmt.Println(cmp.Or(0, 0, -1, 5))      // output: -1
fmt.Println(cmp.Or("", "fallback"))   // output: fallback
```

## Стабільне сортування

`slices.Sort` і `SortFunc` можуть переставити елементи, які ваше
порівняння вважає рівними. `SortStableFunc` зберігає їхній початковий
відносний порядок:

```go
us := []user{{"Bo", 30}, {"Ada", 25}, {"Cy", 30}}
slices.SortStableFunc(us, func(a, b user) int { return cmp.Compare(a.Age, b.Age) })
fmt.Println(us)   // output: [{Ada 25} {Bo 30} {Cy 30}]
```

Bo лишається попереду Cy, бо так було спочатку. Стабільність коштує
продуктивності, тож беріть її лише тоді, коли вона потрібна — наприклад,
коли сортуєте за одним стовпцем поверх наявного порядку. Альтернатива —
зробити порівняння повним за допомогою `cmp.Or`, що зазвичай зрозуміліше.

## Пошук у відсортованому зрізі

```go
fmt.Println(slices.IsSorted([]int{1, 2, 3}))    // output: true
fmt.Println(slices.BinarySearch([]int{1, 3, 5}, 3))   // output: 1 true
```

`BinarySearchFunc` приймає *ціль*, яка не мусить бути типом елемента, тож
можна шукати за одним полем. Її функція порівняння отримує елемент
першим, а ціль — другою:

```go
fmt.Println(slices.BinarySearchFunc(us, 30, func(u user, age int) int {
    return cmp.Compare(u.Age, age)
}))
// output: 1 true
```

Зріз має бути вже відсортований за тим самим порядком, інакше результат
не має сенсу — це ніколи не перевіряється за вас.

## Мінімум і максимум

`min` і `max` — вбудовані функції, що приймають будь-яку кількість
аргументів. Версії з `slices` приймають зріз і панікують на порожньому:

```go
fmt.Println(max(3, 1, 2), min(3, 1, 2))                 // output: 3 1
fmt.Println(slices.Max([]int{3, 1, 2}), slices.Min([]int{3, 1, 2}))
// output: 3 1
```

## Старіший пакет `sort`

`sort` існував ще до дженериків. Ви зустрінете `sort.Slice`, яка приймає
функцію-**компаратор** (less) над індексами, що повертає bool:

```go
l := []int{3, 1, 2}
sort.Slice(l, func(i, j int) bool { return l[i] < l[j] })
fmt.Println(l)   // output: [1 2 3]

ss := []string{"b", "a"}
sort.Strings(ss)
fmt.Println(ss)   // output: [a b]
```

Зверніть увагу, що вона захоплює зріз через замикання, а не отримує
елементи, тож легко помилитися після переприсвоєння. Є ще й
`sort.Interface`, що вимагає методи `Len`, `Less` і `Swap` на іменованому
типі.

Надавайте перевагу `slices` у новому коді. Ці два підходи відрізняються
трьома важливими речами: `slices` порівнює **елементи**, `sort` порівнює
**індекси**; `slices` хоче трійковий `int`, `sort` хоче `bool`; і `slices`
типобезпечний без типу-обгортки.

> **З досвіду Python:** `slices.Sort` — це `list.sort()`. Головна різниця
> в тому, що в Go немає `key=` — немає `sort(key=lambda u: u.age)`, тож ви
> пишете порівняння самостійно, а `cmp.Or` — це те, як виразити те, що в
> Python зробив би кортежний ключ. Зверніть увагу на напрямок:
> `cmp.Compare` повертає трійковий результат, як компаратор, який ви
> передали б у `functools.cmp_to_key`, тоді як старіша `sort.Slice` хоче
> звичайний bool «чи a менше за b».

## Швидка довідка

| Задача | Виклик |
|---|---|
| відсортувати числа чи рядки | `slices.Sort(s)` |
| відсортувати за полем | `slices.SortFunc(s, func(a, b T) int { ... })` |
| трійкове порівняння | `cmp.Compare(a, b)` → `-1`, `0`, `1` |
| за спаданням | `cmp.Compare(b, a)` — поміняти місцями, не заперечувати |
| другий ключ сортування | `cmp.Or(cmp.Compare(...), cmp.Compare(...))` |
| зберегти порядок рівних елементів | `slices.SortStableFunc` |
| чи відсортовано | `slices.IsSorted`, `slices.IsSortedFunc` |
| знайти у відсортованому зрізі | `slices.BinarySearch`, `BinarySearchFunc` |
| найбільше / найменше | `max(a, b, c)`, `slices.Max(s)` |
| старіший код | `sort.Slice` (bool над індексами), `sort.Interface` |

## Джерела

- [`slices` package reference — pkg.go.dev/slices](https://pkg.go.dev/slices)
- [`cmp` package reference — pkg.go.dev/cmp](https://pkg.go.dev/cmp)
- [`sort` package reference — pkg.go.dev/sort](https://pkg.go.dev/sort)
- [`slices.SortFunc` — pkg.go.dev/slices#SortFunc](https://pkg.go.dev/slices#SortFunc)
- [`cmp.Or` — pkg.go.dev/cmp#Or](https://pkg.go.dev/cmp#Or)
