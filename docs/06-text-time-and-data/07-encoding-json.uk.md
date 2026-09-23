# Кодування JSON

`encoding/json` перетворює значення Go в JSON, використовуючи теги
структур і рефлексію: огляд типу та полів значення під час виконання
програми. (Стаття [XML, CSV та рефлексія](08-xml-csv-and-reflection.md)
розповідає про саму рефлексію.) Стаття про
[структури](../02-language-basics/10-structs.md) уже познайомила з
синтаксисом тегів; ця стаття — про поведінку, у якої є кілька гострих
країв, з якими варто зустрітися свідомо.

```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email,omitempty"`
    Age   int    `json:"age"`
}

b, _ := json.Marshal(User{Name: "Ada", Age: 36})
fmt.Println(string(b))   // output: {"name":"Ada","age":36}
```

`Marshal` повертає `[]byte`, тож для друку потрібне `string(...)`.

## Подорожують лише експортовані поля

Неекспортовані поля невидимі для кодера — він працює через рефлексію, а
рефлексія не може їх прочитати. Немає тега, який це змінює. Якщо поле
має серіалізуватися, зробіть його експортованим.

## Опції тегів

```go
type User struct {
    Name  string  `json:"name"`
    Email string  `json:"email,omitempty"`
    Nick  *string `json:"nick"`
    Admin bool    `json:"-"`
    ID    int64   `json:"id,string"`
}

u := User{Name: "Ada", ID: 7, Admin: true}
b, _ := json.Marshal(u)
fmt.Println(string(b))
// output: {"name":"Ada","nick":null,"id":"7"}
```

| Опція | Ефект |
|---|---|
| `json:"name"` | перейменувати ключ |
| `json:",omitempty"` | пропустити ключ, якщо значення нульове |
| `json:"-"` | ніколи не кодувати й не декодувати це поле |
| `json:",string"` | закодувати число як рядок JSON |

`,string` існує, бо в багатьох споживачів числа JSON — це float64, тож
`int64` понад 2^53 втрачає точність при передачі. Надсилання його як
рядка цього уникає.

Використовуйте `json.MarshalIndent`, коли результат читатиме людина:

```go
b, _ := json.MarshalIndent(u, "", "  ")
```

## `omitempty` означає нуль, а не відсутність

`omitempty` прибирає поле, значення якого — нульове значення: `0`, `""`,
`false`, `nil` або порожній зріз чи мапа. Вона не може відрізнити
«користувач встановив нуль» від «користувач нічого не сказав», бо в Go
це те саме значення.

Коли ця відмінність важлива, використовуйте вказівник. Тоді `nil` — це
«відсутнє», а `&""` — «явно порожнє»:

```go
var a, b, c User
json.Unmarshal([]byte(`{"name":"x"}`), &a)
json.Unmarshal([]byte(`{"name":"x","nick":null}`), &b)
json.Unmarshal([]byte(`{"name":"x","nick":""}`), &c)

fmt.Println(a.Nick == nil, b.Nick == nil, *c.Nick == "")
// output: true true true
```

Відсутність поля й явний `null` однаково лишають вказівник `nil`; лише
справжнє значення виділяє пам'ять під нього.

## Нульовий (nil) зріз кодується як `null`, а не `[]`

Саме це спричиняє звіти про баги від фронтенд-розробників:

```go
var cfg Config
b, _ := json.Marshal(cfg)
fmt.Println(string(b))
// output: {"debug":false,"plugins":null,"extra":null}

cfg = Config{Plugins: []string{}, Extra: map[string]any{}}
b, _ = json.Marshal(cfg)
fmt.Println(string(b))
// output: {"debug":false,"plugins":[],"extra":{}}
```

Нульовий (nil) зріз і порожній зріз поводяться однаково всюди в Go —
`len` дорівнює 0, `range` нічого не робить, `append` працює. Вони
відрізняються лише тут. Якщо клієнт перебиратиме це поле, ініціалізуйте
його як порожній зріз, а не лишайте `nil`.

## Декодування

`Unmarshal` потребує вказівник, бо мусить записати результат у ваше
значення:

```go
var u User
err := json.Unmarshal([]byte(`{"name":"Bo","age":7,"id":"99"}`), &u)
```

Зіставлення ключів не враховує регістр, і невідомі ключі за замовчуванням
ігноруються, тож корисне навантаження із зайвими полями декодується без
проблем. Поля, яких немає в JSON, залишаються при своєму поточному
значенні — `Unmarshal` не скидає ціль спочатку, і це важливо, коли ви
повторно використовуєте змінну в циклі.

Забудьте `&`, і ви отримаєте зрозумілу помилку замість мовчання:

```go
err := json.Unmarshal(data, u)
fmt.Println(err)   // output: json: Unmarshal(non-pointer main.User)
```

## Декодування в `any`

Без структури JSON декодується у фіксований набір типів Go:

```go
var v any
json.Unmarshal([]byte(`{"n":1,"s":"x","b":true,"arr":[1,2],"o":{"k":1}}`), &v)

m := v.(map[string]any)
fmt.Printf("%T %T %T %T %T\n", m["n"], m["s"], m["b"], m["arr"], m["o"])
// output: float64 string bool []interface {} map[string]interface {}
```

| JSON | Go |
|---|---|
| число | `float64` |
| рядок | `string` |
| `true`/`false` | `bool` |
| масив | `[]any` |
| об'єкт | `map[string]any` |
| `null` | `nil` |

**Кожне число стає `float64`**, навіть те, що виглядає як ціле. `m["n"].(int)`
панікує. Або визначте структуру, або конвертуйте: `int(m["n"].(float64))`.

## Помилки, з якими варто рахуватись

```go
type Account struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

var a Account
err := json.Unmarshal([]byte(`{"age":"old"}`), &a)
fmt.Println(err)
// output: json: cannot unmarshal string into Go struct field Account.age of type int

var ute *json.UnmarshalTypeError
fmt.Println(errors.As(err, &ute), ute.Field, ute.Value)
// output: true age string
```

`*json.UnmarshalTypeError` несе назву поля й те, що було знайдено, тож
API може повідомити викликачу, яке саме поле неправильне — гарне
використання [errors.As](../03-object-oriented-go/06-custom-error-types.md).
Зіпсовані вхідні дані дають зовсім іншу помилку:

```go
err = json.Unmarshal([]byte(`{`), &a)
fmt.Println(err)   // output: unexpected end of JSON input
```

## `json.RawMessage` відкладає декодування

Коли форма одного поля залежить від іншого, лишіть його як сирі байти й
декодуйте, щойно дізнаєтесь, що це:

```go
type Envelope struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`
}

var e Envelope
json.Unmarshal([]byte(`{"type":"user","payload":{"name":"Cy"}}`), &e)
fmt.Println(e.Type, string(e.Payload))
// output: user {"name":"Cy"}

var inner User
json.Unmarshal(e.Payload, &inner)
fmt.Println(inner.Name)   // output: Cy
```

`RawMessage` — це просто `[]byte`, що кодує себе дослівно, тож він також
підходить для передачі JSON без змін.

## Власне кодування

Тип керує власним представленням через `MarshalJSON` і `UnmarshalJSON`:

```go
type Temp float64

func (t Temp) MarshalJSON() ([]byte, error) {
    return json.Marshal(fmt.Sprintf("%.1fC", float64(t)))
}

func (t *Temp) UnmarshalJSON(b []byte) error {
    var s string
    if err := json.Unmarshal(b, &s); err != nil {
        return err
    }
    var f float64
    if _, err := fmt.Sscanf(s, "%fC", &f); err != nil {
        return err
    }
    *t = Temp(f)
    return nil
}
```

```go
b, _ := json.Marshal(Temp(21.456))
fmt.Println(string(b))   // output: "21.5C"

var t Temp
json.Unmarshal([]byte(`"30.5C"`), &t)
fmt.Println(float64(t))  // output: 30.5
```

`MarshalJSON` приймає отримувача-значення, `UnmarshalJSON` —
отримувача-вказівник — вона мусить писати в отримувача. Помилитися тут
легко й непомітно: `MarshalJSON` з отримувачем-вказівником просто не
знаходиться, коли ви кодуєте значення.

### Трюк `type plain T`

Усередині `MarshalJSON` виклик `json.Marshal(t)` для вашого власного типу
знову викликає `MarshalJSON` — нескінченна рекурсія. Вихід спирається на
правило, яке варто сформулювати прямо: **визначений тип починається з
порожнього набору методів.** `type plain Wrapper` має всі поля
`Wrapper` і жодного з його методів, тож серіалізація `plain` не може
повторно увійти в `MarshalJSON`:

```go
func (w Wrapper) MarshalJSON() ([]byte, error) {
    type plain Wrapper       // ті самі поля, без методів
    p := plain(w)
    p.Kind = strings.ToUpper(p.Kind)
    return json.Marshal(p)
}

b, _ := json.Marshal(Wrapper{Name: "n", Kind: "abc"})
fmt.Println(string(b))   // output: {"name":"n","kind":"ABC"}
```

Теги структур переживають конвертацію, тож ключі не змінюються.

## Потокова обробка через `Decoder` і `Encoder`

`Unmarshal` потребує весь документ у пам'яті. `json.NewDecoder` читає з
`io.Reader` і вміє декодувати потік значень по одному:

```go
dec := json.NewDecoder(strings.NewReader(`{"name":"A"} {"name":"B"}`))
for {
    var u User
    if err := dec.Decode(&u); err != nil {
        break
    }
    fmt.Print(u.Name, " ")
}
// output: A B
```

Переривання циклу на будь-якій помилці не розрізняє «кінець вхідних
даних» і «погані вхідні дані». У реальному коді помилку порівнюють з
`io.EOF` — значенням з пакета `io`, яке означає «більше немає вхідних
даних». Стаття
[читачі та письменники](../07-operating-system/02-readers-and-writers.md)
розповідає про це. Декодери
також підтримують суворий режим:

```go
dec := json.NewDecoder(strings.NewReader(`{"nope":1}`))
dec.DisallowUnknownFields()
fmt.Println(dec.Decode(&u))   // output: json: unknown field "nope"
```

`json.NewEncoder(w)` — дзеркальне відображення, воно пише прямо в
`io.Writer`, додаючи символ нового рядка після кожного значення.

> **З досвіду Python:** `Marshal`/`Unmarshal` — це `json.dumps`/`json.loads`,
> але типізовані. Три речі без аналога в Python: неекспортовані поля
> ніколи не серіалізуються, нульовий (nil) зріз стає `null`, а не `[]`, і
> декодування в `any` робить кожне число `float64`.

## Швидка довідка

| Задача | Виклик |
|---|---|
| закодувати | `json.Marshal(v)` → `[]byte` |
| закодувати читабельно | `json.MarshalIndent(v, "", "  ")` |
| декодувати | `json.Unmarshal(b, &v)` — потрібен вказівник |
| перейменувати / пропустити / виключити | `json:"name,omitempty"`, `json:"-"` |
| великі цілі числа | `json:",string"` |
| відрізнити відсутність від нуля | поле-вказівник |
| уникнути `null` для списку | ініціалізувати як `[]T{}` |
| відкласти декодування | `json.RawMessage` |
| власна форма | `MarshalJSON` (значення), `UnmarshalJSON` (вказівник) |
| уникнути рекурсії в них | `type plain T` |
| потокова обробка | `json.NewDecoder(r)` / `json.NewEncoder(w)` |
| відхиляти невідомі ключі | `dec.DisallowUnknownFields()` |

## Джерела

- [`encoding/json` package reference — pkg.go.dev/encoding/json](https://pkg.go.dev/encoding/json)
- [`json.Marshal` — pkg.go.dev/encoding/json#Marshal](https://pkg.go.dev/encoding/json#Marshal)
- [`json.Unmarshal` — pkg.go.dev/encoding/json#Unmarshal](https://pkg.go.dev/encoding/json#Unmarshal)
- [`json.RawMessage` — pkg.go.dev/encoding/json#RawMessage](https://pkg.go.dev/encoding/json#RawMessage)
- [`json.Decoder` — pkg.go.dev/encoding/json#Decoder](https://pkg.go.dev/encoding/json#Decoder)
- [Go blog: JSON and Go — go.dev/blog/json](https://go.dev/blog/json)
