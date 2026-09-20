# Ітератори

Стаття про [керування потоком виконання](../02-language-basics/05-control-flow.md)
показала `range` над функцією та пообіцяла розглянути продукуючу сторону
пізніше. Ось вона. Ітератор у Go — це просто функція, по якій можна
робити `range`, і написати таку функцію — менше механіки, ніж здається.

```go
for v := range Countdown(3) {
    fmt.Print(v, " ")
}
// output: 3 2 1
```

## Форма

Два псевдоніми типів у пакеті `iter` називають ці сигнатури:

```go
type Seq[V any]     func(yield func(V) bool)
type Seq2[K, V any] func(yield func(K, V) bool)
```

Ітератор — це функція, яка *приймає* функцію. `range` надає `yield`, ваш
код викликає її по одному разу на елемент, а тіло циклу — це те, що
виконується всередині `yield`:

```go
func Countdown(n int) iter.Seq[int] {
    return func(yield func(int) bool) {
        for i := n; i > 0; i-- {
            if !yield(i) {
                return
            }
        }
    }
}
```

Напрямок тут перевернутий порівняно з більшістю мов: цикл не витягує
значення з вас, а ви заштовхуєте значення в цикл.

## `yield` повертає false, коли цикл зупиняється

Оце `if !yield(i) { return }` — увесь контракт. `break`, `return` чи
`panic` у тілі циклу змушує `yield` повернути `false`, і ви мусите
зупинитися й повернутися:

```go
for v := range Countdown(10) {
    if v < 8 {
        break
    }
    fmt.Print(v, " ")
}
// output: 10 9 8
```

Ігнорування цього результату — єдиний справжній баг, який тут можна
написати, і час виконання його ловить:

```go
func Bad(n int) iter.Seq[int] {
    return func(yield func(int) bool) {
        for i := n; i > 0; i-- {
            yield(i)   // результат ігнорується
        }
    }
}
// panic: runtime error: range function continued iteration after
//        function for loop body returned false
```

Наслідок: тіло циклу може завершитись у будь-який момент, тож ресурс, що
відкритий до виклику yield, все одно потребує `defer`, щоб звільнитися,
коли викликач вийде достроково.

## Дві пари значень: `Seq2`

`Seq2` — та сама ідея з парою, яку `range` вже дає вам над мапою чи
зрізом:

```go
func Enumerate[T any](s []T) iter.Seq2[int, T] {
    return func(yield func(int, T) bool) {
        for i, v := range s {
            if !yield(i, v) {
                return
            }
        }
    }
}

for i, v := range Enumerate([]string{"a", "b"}) {
    fmt.Printf("%d=%s ", i, v)
}
// output: 0=a 1=b
```

## Ітератори компонуються

Оскільки ітератор — це звичайне значення, функція може приймати один
ітератор і повертати інший. Нічого не буферизується — значення й далі
течуть по одному:

```go
func Filter[T any](seq iter.Seq[T], keep func(T) bool) iter.Seq[T] {
    return func(yield func(T) bool) {
        for v := range seq {
            if keep(v) && !yield(v) {
                return
            }
        }
    }
}

even := Filter(Countdown(6), func(n int) bool { return n%2 == 0 })
fmt.Println(slices.Collect(even))   // output: [6 4 2]
```

## Де вони окупаються: рекурсивні структури

Раніше, щоб показати вміст дерева назовні, потрібно було будувати зріз
або приймати колбек. Ітератор дає викликачам звичайний цикл `for` над
структурою, яку незручно обходити:

```go
type Tree struct {
    Val         int
    Left, Right *Tree
}

func (t *Tree) All() iter.Seq[int] {
    return func(yield func(int) bool) {
        t.walk(yield)
    }
}

func (t *Tree) walk(yield func(int) bool) bool {
    if t == nil {
        return true
    }
    return t.Left.walk(yield) && yield(t.Val) && t.Right.walk(yield)
}
```

```go
tr := &Tree{Val: 2, Left: &Tree{Val: 1}, Right: &Tree{Val: 3}}
fmt.Println(slices.Collect(tr.All()))   // output: [1 2 3]
```

Ланцюжок `&&` виконує подвійну роботу: він задає послідовність
ліворуч-себе-праворуч і одразу ж зупиняється, щойно `yield` повертає
`false`. Зверніть увагу на виклики методів на `nil` `*Tree` — це
безпечно, як пояснює стаття про [методи](../03-object-oriented-go/01-methods.md).

## Стандартна бібліотека і продукує, і споживає їх

Ви вже користувалися цим. `maps.Keys` повертає ітератор, тому вона й
поєднується зі `slices.Sorted`:

```go
m := map[string]int{"b": 2, "a": 1}
fmt.Println(slices.Sorted(maps.Keys(m)))   // output: [a b]
```

| Продюсер | Дає |
|---|---|
| `slices.Values(s)` | кожен елемент |
| `slices.All(s)` | індекс і елемент |
| `slices.Backward(s)` | індекс і елемент, від останнього до першого |
| `maps.Keys(m)` / `maps.Values(m)` | ключі / значення |
| `maps.All(m)` | ключ і значення |
| `strings.SplitSeq(s, sep)` | шматки, без виділення зрізу |

| Споживач | Дає |
|---|---|
| `slices.Collect(seq)` | `[]T` |
| `slices.Sorted(seq)` | відсортований `[]T` |
| `maps.Collect(seq2)` | `map[K]V` |

```go
fmt.Println(slices.Collect(slices.Values([]int{1, 2, 3})))     // output: [1 2 3]
fmt.Println(slices.Collect(strings.SplitSeq("a,b,c", ",")))    // output: [a b c]

for i, v := range slices.Backward([]int{1, 2, 3}) {
    fmt.Printf("%d:%d ", i, v)
}
// output: 2:3 1:2 0:1
```

`SplitSeq` — це вся суть цієї можливості в мініатюрі: `strings.Split`
виділяє зріз, який потім викидають, тоді як `SplitSeq` віддає шматки
одразу, як їх знаходить.

## `iter.Pull`, коли потрібно керувати самому

Іноді циклом `for` не обійтись — треба просувати дві послідовності
синхронно. `iter.Pull` перетворює push-ітератор на функцію `next`:

```go
next, stop := iter.Pull(slices.Values([]int{1, 2, 3}))
defer stop()

for {
    v, ok := next()
    if !ok {
        break
    }
    fmt.Print(v, " ")
}
// output: 1 2 3
```

`stop` треба викликати завжди — звідси й `defer` — тому що `Pull` запускає
ітератор в окремій горутині, а `stop` саме і звільняє її. З двома такими
можна робити зіп:

```go
a, stopA := iter.Pull(slices.Values([]string{"x", "y"}))
defer stopA()
b, stopB := iter.Pull(slices.Values([]int{10, 20}))
defer stopB()

for {
    s, ok1 := a()
    n, ok2 := b()
    if !ok1 || !ok2 {
        break
    }
    fmt.Printf("%s=%d ", s, n)
}
// output: x=10 y=20
```

`Pull` коштує дорожче за пряме перебирання через `range`, тож
використовуйте його лише тоді, коли керування потоком дійсно цього
вимагає.

## Коли не варто писати ітератор

Якщо у вас уже є зріз, поверніть зріз. Ітератор виправдовує себе, коли
послідовність коштовна, необмежена або незручна для матеріалізації —
обхід дерева, посторінковий API, рядки великого файлу. Для жменьки
значень у пам'яті це непотрібна непряма робота.

> **З досвіду Python:** це генератор, але побудований навпаки. `yield` у
> Python призупиняє вашу функцію; `yield` у Go — це колбек, який дає вам
> цикл, і повернення `false` — те саме, що робить `GeneratorExit`.
> `iter.Pull` — найближчий аналог утримування об'єкта-генератора й
> самостійного виклику `next()`.

## Швидка довідка

| Форма | Значення |
|---|---|
| `iter.Seq[V]` | `func(yield func(V) bool)` |
| `iter.Seq2[K, V]` | `func(yield func(K, V) bool)` |
| `if !yield(v) { return }` | зупинитись, коли тіло циклу перериває — обов'язково |
| `slices.Collect(seq)` | вилити в зріз |
| `slices.Sorted(maps.Keys(m))` | відсортовані ключі мапи |
| `iter.Pull(seq)` | `next, stop` — завжди `defer stop()` |

## Джерела

- [`iter` package reference — pkg.go.dev/iter](https://pkg.go.dev/iter)
- [`slices` iterator functions — pkg.go.dev/slices](https://pkg.go.dev/slices)
- [`maps` iterator functions — pkg.go.dev/maps](https://pkg.go.dev/maps)
- [For statements with range clause — go.dev/ref/spec#For_range](https://go.dev/ref/spec#For_range)
- [Go blog: range over function types — go.dev/blog/range-functions](https://go.dev/blog/range-functions)
