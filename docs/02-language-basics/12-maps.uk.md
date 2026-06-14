# Map (асоціативні масиви)

**Map** — це вбудована геш-таблиця Go: невпорядкована колекція пар
ключ→значення із пошуком, вставлянням та видаленням у середньому за O(1).
Тип записується як `map[K]V` — `K` це тип ключа, `V` — тип значення.

```go
ages := map[string]int{
    "alice": 30,
    "bob":   25,
}
fmt.Println(ages["alice"])   // output: 30
```

## Ключі мають бути порівнюваними

Тип ключа має підтримувати `==`: це охоплює всі базові типи (рядки, числа,
булеві), вказівники та структури/масиви, поля яких самі порівнювані.
Зрізи, map та функції **не** порівнювані, тож ключами бути не можуть —
спроба робить це помилкою компіляції.

```go
m := map[[]int]string{}   // compile error: invalid map key type []int
```

На типи значень такого обмеження немає — `map[string][]int` цілком
припустимо.

## Створення map

**Літерал**, зокрема порожній літерал `map[K]V{}`:

```go
m := map[string]int{"a": 1, "b": 2}
empty := map[string]int{}      // не-nil, готовий до використання
```

**`make`** — порожня map, готова до запису:

```go
m := make(map[string]int)
m["x"] = 1
fmt.Println(m["x"])    // output: 1
```

### Пастка nil-map

Нульове значення map — `nil`. З nil-map можна **читати** (отримаєте
нульові значення) й брати її `len`, але **запис у nil-map спричиняє
паніку**.

```go
var m map[string]int     // nil
fmt.Println(m["missing"], len(m))   // output: 0 0  — читання припустиме
m["x"] = 1               // panic: assignment to entry in nil map
```

Завжди ініціалізуйте через `make` чи літерал перед записом. Оголосити
`var m map[string]int` і забути зробити `make` — це класична помилка з
map.

## Читання: нульове значення відсутнього ключа

Індексування ключа, якого немає, повертає **нульове значення** типу
значення — не помилку й не паніку:

```go
ages := map[string]int{"alice": 30}
fmt.Println(ages["charlie"])   // output: 0  — відсутній, тож нуль
```

Це неоднозначно: `charlie` відображено в `0` чи його просто немає?
Скористайтеся формою **comma-ok**, щоб їх розрізнити — друге значення є
булевим.

```go
ages := map[string]int{"alice": 30}
v, ok := ages["charlie"]
fmt.Println(v, ok)             // output: 0 false

v, ok = ages["alice"]
fmt.Println(v, ok)             // output: 30 true
```

> **З погляду Python:** індексування відсутнього ключа **не** збуджує
> `KeyError`. Воно тихо повертає нульове значення — ближче до
> `dict.get(key, default)`, ніж до `dict[key]`. Вдавайтеся до comma-ok,
> коли «відсутній» та «присутній, але нульовий» треба розрізняти.

## Оновлення, видалення, розмір

```go
m := map[string]int{"a": 1}
m["a"] = 100          // перезапис
m["b"] = 2            // вставляння
delete(m, "a")        // видалення; нічого не робить, якщо ключа немає, ніколи не панікує
fmt.Println(len(m), m["b"])   // output: 1 2
```

## Порядок ітерування рандомізований

`for range` відвідує кожну пару, але порядок **навмисно
рандомізований** — він відрізняється від запуску до запуску. Ніколи не
покладайтеся на порядок map. Щоб ітерувати у сталому порядку, зберіть
ключі у зріз і відсортуйте його.

```go
m := map[string]int{"a": 1, "b": 2, "c": 3}

keys := make([]string, 0, len(m))
for k := range m {           // одна змінна → лише ключі
    keys = append(keys, k)
}
sort.Strings(keys)
for _, k := range keys {
    fmt.Println(k, m[k])
}
// output:
// a 1
// b 2
// c 3
```

Ітерування з однією змінною дає ключі; з двома — ключі та значення.

## Map поводяться як посилання

Значення map — це невеликий заголовок, що вказує на базову геш-таблицю.
Копіювання map — присвоєння її чи передавання у функцію — копіює цей
заголовок, **а не** дані, тож обидва імені посилаються на ту саму
таблицю. Зміни через одне видно через інше.

```go
func add(m map[string]int) {
    m["new"] = 1          // змінює map викликача
}

m := map[string]int{}
add(m)
fmt.Println(m["new"])     // output: 1
```

Це не так, як зі структурами й масивами (які копіюються цілком).
Вбудованого «скопіювати map» немає — щоб отримати незалежну копію, ви
виділяєте нову map і копіюєте записи в циклі.

## Множина через `map[T]struct{}`

Вбудованого типу множини в Go немає. Ідіома — це map зі значенням-порожньою
структурою `struct{}` на нуль байтів, тож сенс несуть лише ключі:

```go
set := map[string]struct{}{}
set["go"] = struct{}{}
set["go"] = struct{}{}        // ідемпотентно
_, exists := set["go"]
fmt.Println(exists, len(set)) // output: true 1
```

Використання `map[string]bool` — поширена, трохи важча альтернатива, яка
читається дещо природніше (`set["go"] = true`).

## Швидка довідка

| Операція | Код |
|---|---|
| літерал | `map[string]int{"a": 1}` |
| порожня, готова до запису | `make(map[string]int)` або `map[string]int{}` |
| читання (нуль, якщо відсутній) | `v := m[k]` |
| читання з перевіркою наявності | `v, ok := m[k]` |
| вставляння / оновлення | `m[k] = v` |
| видалення | `delete(m, k)` |
| розмір | `len(m)` |
| ітерування (випадковий порядок) | `for k, v := range m` |
| **паніка** | запис у `nil`-map |

## Джерела

- [Map types — go.dev/ref/spec#Map_types](https://go.dev/ref/spec#Map_types)
- [Index expressions — go.dev/ref/spec#Index_expressions](https://go.dev/ref/spec#Index_expressions)
- [The `delete` builtin — go.dev/ref/spec#Deletion_of_map_elements](https://go.dev/ref/spec#Deletion_of_map_elements)
- [For statements with range — go.dev/ref/spec#For_range](https://go.dev/ref/spec#For_range)
- [Go blog: Go maps in action — go.dev/blog/maps](https://go.dev/blog/maps)
