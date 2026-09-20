# XML, CSV та рефлексія

Ще два кодери зі стандартної бібліотеки, а тоді — механізм, що робить
можливими їх усіх. `reflect` — насамкінець, бо чесна порада — вживати
рефлексію рідко, але прочитати кодери чи завантажувач конфігурації, не
знаючи, що вона робить, неможливо.

```go
r := csv.NewReader(strings.NewReader("name,age\nAda,36\n"))
recs, _ := r.ReadAll()
fmt.Println(recs)   // output: [[name age] [Ada 36]]
```

## XML

`encoding/xml` працює як `encoding/json`, але з багатшим словником тегів,
бо XML розрізняє атрибути й елементи та дозволяє вкладеність:

```go
type Book struct {
    XMLName xml.Name `xml:"book"`
    ID      string   `xml:"id,attr"`
    Title   string   `xml:"title"`
    Tags    []string `xml:"tags>tag"`
}

b := Book{ID: "7", Title: "Go", Tags: []string{"a", "b"}}
out, _ := xml.MarshalIndent(b, "", "  ")
fmt.Println(string(out))
```

```
<book id="7">
  <title>Go</title>
  <tags>
    <tag>a</tag>
    <tag>b</tag>
  </tags>
</book>
```

| Тег | Значення |
|---|---|
| `xml:"title"` | назва елемента |
| `xml:"id,attr"` | атрибут, а не елемент |
| `xml:"tags>tag"` | повторюваний `<tag>` усередині обгортки `<tags>` |
| `xml:",chardata"` | текстовий вміст елемента |
| `xml:"-"` | пропустити |

Поле `XMLName xml.Name` називає кореневий елемент. Декодування — дзеркальна операція:

```go
var b Book
xml.Unmarshal([]byte(`<book id="9"><title>T</title><tags><tag>x</tag></tags></book>`), &b)
fmt.Println(b.ID, b.Title, b.Tags)   // output: 9 T [x]
```

### Потокове читання токенів

Для великого документа, або такого, чию форму ви не контролюєте,
читайте його як потік токенів. `Token` повертає кожен шматок по черзі, а
[type switch](../03-object-oriented-go/03-type-assertions-and-type-switches.md)
розбирає їх:

```go
dec := xml.NewDecoder(strings.NewReader(`<r><title>One</title><title>Two</title></r>`))

var inTitle bool
for {
    tok, err := dec.Token()
    if err == io.EOF {
        break
    }
    switch t := tok.(type) {
    case xml.StartElement:
        inTitle = t.Name.Local == "title"
    case xml.CharData:
        if inTitle {
            fmt.Printf("%q ", string(t))
        }
    case xml.EndElement:
        inTitle = false
    }
}
// output: "One" "Two"
```

Ніколи в пам'яті не лежить більше одного токена. `xml.CharData` — це
`[]byte`, тож для друку як тексту потрібне `string(t)` — і воно дійсне
лише до наступного виклику `Token`, тож копіюйте його, якщо хочете
зберегти.

Це також патерн для витягування тексту з форматів, які під капотом є
XML, наприклад частин усередині документа Office.

## CSV

`encoding/csv` обробляє лапки, вбудовані коми й вбудовані символи нового
рядка — усі причини, чому розбивати рядок за комою самостійно неправильно.

```go
r := csv.NewReader(strings.NewReader("name,age\nAda,36\nBo,7\n"))
recs, err := r.ReadAll()
fmt.Println(recs, err)
// output: [[name age] [Ada 36] [Bo 7]] <nil>
```

`ReadAll` тримає весь файл у пам'яті. Для чогось великого читайте запис
за записом і зупиняйтесь на `io.EOF`:

```go
r := csv.NewReader(strings.NewReader("name,age\nAda,36\nBo,7\n"))

hdr, _ := r.Read()
fmt.Println(hdr)   // output: [name age]

for {
    rec, err := r.Read()
    if err == io.EOF {
        break
    }
    fmt.Print(rec[0], "=", rec[1], " ")
}
// output: Ada=36 Bo=7
```

Обробки заголовків і зіставлення зі структурами тут немає — запис — це
`[]string`, і перший `Read` є заголовком лише тому, що ви так до нього
поставились.

За замовчуванням читач вимагає, щоб кожен запис мав стільки ж полів,
скільки перший, а це рано ловить обрізані файли:

```go
r := csv.NewReader(strings.NewReader("a,b\n1\n"))
_, err := r.ReadAll()
fmt.Println(err)   // output: record on line 2: wrong number of fields
```

Встановіть `r.FieldsPerRecord = -1`, щоб дозволити нерівні рядки, і
`r.Comma = '\t'` для TSV.

Запис бере в лапки лише те, що цього потребує, і його **обов'язково
треба скинути (flush)**:

```go
var sb strings.Builder
w := csv.NewWriter(&sb)
w.Write([]string{"name", "note"})
w.Write([]string{"Ada", "has, comma"})
w.Flush()

fmt.Printf("%q\n", sb.String())
// output: "name,note\nAda,\"has, comma\"\n"
```

Якщо забути `Flush`, буферизовані рядки мовчки губляться. Перевіряйте
`w.Error()` після цього, бо сам `Write` повідомляє лише про помилки з
попередніх викликів.

## Рефлексія

Рефлексія — це те, як `encoding/json`, `encoding/xml` і будь-який
завантажувач конфігурації читають теги структур і обходять поля типів,
яких раніше не бачили. Три ідеї покривають більшість цього: `Type`
описує тип, `Value` тримає значення, а `Kind` каже, до якої категорії
типів воно належить.

```go
type Cfg struct {
    Host string `json:"host" env:"APP_HOST"`
    Port int    `json:"port" env:"APP_PORT"`
    skip string
}

t := reflect.TypeOf(Cfg{})
fmt.Println(t.Name(), t.Kind(), t.NumField())
// output: Cfg struct 3
```

`Kind` — це категорія — `struct`, `int`, `slice`, `ptr` — на відміну від
іменованого типу. `type Celsius float64` має kind `float64` і назву
`Celsius`.

### Читання тегів структур

Цей цикл — серце кожної бібліотеки, керованої тегами:

```go
for i := range t.NumField() {
    f := t.Field(i)
    fmt.Printf("%s %s json=%q env=%q exported=%v\n",
        f.Name, f.Type, f.Tag.Get("json"), f.Tag.Get("env"), f.IsExported())
}
// output:
// Host string json="host" env="APP_HOST" exported=true
// Port int json="port" env="APP_PORT" exported=true
// skip string json="" env="" exported=false
```

`Tag.Get` повертає `""` для ключа, якого немає, тож помилково написаний
тег мовчки не спрацьовує, а не гучно падає — на це вказує стаття про
[структури](../02-language-basics/10-structs.md).

`IsExported` — також причина, чому неекспортовані поля ніколи не
з'являються в JSON: бібліотека, що обходить поля, бачить `skip`, але не
може його прочитати чи записати.

### Встановлення значень потребує вказівника

`reflect.Value`, отримане зі звичайного значення, не адресоване — це
копія, тож запис у нього нічого б не змінив, і Go радше відмовляється,
ніж прикидається:

```go
c := Cfg{Host: "h", Port: 1}

pv := reflect.ValueOf(&c).Elem()   // Elem() йде за вказівником
pv.Field(0).SetString("changed")
fmt.Println(c.Host)                // output: changed

fmt.Println(reflect.ValueOf(c).CanSet(), pv.Field(0).CanSet())
// output: false true
```

`ValueOf(&c).Elem()` — це формула, а `CanSet` — те, що треба перевіряти
перед записом. Саме тому `json.Unmarshal` вимагає вказівник.

### Дві речі, для яких рефлексія справді придатна

Виявлення типізованого nil, чого не може зробити `== nil` — та ж пастка,
що й у статті про [інтерфейси](../03-object-oriented-go/02-interfaces.md):

```go
var p *Cfg
var iface any = p

fmt.Println(iface == nil)                      // output: false
fmt.Println(reflect.ValueOf(iface).IsNil())    // output: true
```

А також порівняння значень, від яких `==` відмовляється, як-от мапи й
зрізи:

```go
fmt.Println(reflect.DeepEqual(map[string]int{"a": 1}, map[string]int{"a": 1}))
// output: true
fmt.Println(reflect.DeepEqual([]int{1}, []int{1}))
// output: true
```

`DeepEqual` розрізняє нульовий (nil) зріз і порожній, а це зазвичай не
те, що мав на увазі тест:

```go
var ns []int
fmt.Println(reflect.DeepEqual(ns, []int{}))   // output: false
```

Для зрізів і мап з порівнюваних елементів надавайте перевагу
`slices.Equal` і `maps.Equal` — вони типізовані, швидші й ставляться
однаково і до nil, і до порожнього.

### Краще не варто

Рефлексія перетворює помилки часу компіляції на паніки часу виконання,
руйнує систему типів і повільна. Перш ніж сягати по неї, запитайте себе,
чи впорався б інтерфейс або узагальнена функція — обидва тримають
перевірки на етапі компіляції. Законних випадків небагато: реалізація
серіалізатора, читання тегів структур або написання допоміжного коду для
тестів. Якщо ви використовуєте її для бізнес-логіки, майже завжди є
кращий підхід.

> **З досвіду Python:** рефлексія — це `getattr`, `type()` і `__dict__` —
> там звичайні інструменти, тут — вузькоспеціалізовані. `reflect.DeepEqual`
> — це `==` для вкладених структур. Незвичне тут — адресованість: не
> можна встановити поле через копію, тож усе змінюване починається з
> вказівника та `.Elem()`.

## Швидка довідка

| Задача | Виклик |
|---|---|
| атрибут XML / вкладений список | `xml:"id,attr"` / `xml:"tags>tag"` |
| потокове читання великого XML-документа | `xml.NewDecoder(r).Token()` + type switch |
| прочитати CSV | `csv.NewReader(r)`, `.Read()` до `io.EOF` |
| нерівні рядки / TSV | `r.FieldsPerRecord = -1` / `r.Comma = '\t'` |
| записати CSV | `csv.NewWriter(w)` — **`Flush()`**, потім `Error()` |
| поля типу | `reflect.TypeOf(v).Field(i)` |
| тег структури | `f.Tag.Get("json")` — `""`, якщо відсутній |
| змінити через рефлексію | `reflect.ValueOf(&v).Elem()`, перевірити `CanSet` |
| виявити типізований nil | `reflect.ValueOf(x).IsNil()` |
| порівняти мапи чи зрізи | `slices.Equal` / `maps.Equal`, інакше `reflect.DeepEqual` |

## Джерела

- [`encoding/xml` package reference — pkg.go.dev/encoding/xml](https://pkg.go.dev/encoding/xml)
- [`encoding/csv` package reference — pkg.go.dev/encoding/csv](https://pkg.go.dev/encoding/csv)
- [`reflect` package reference — pkg.go.dev/reflect](https://pkg.go.dev/reflect)
- [`reflect.StructTag` — pkg.go.dev/reflect#StructTag](https://pkg.go.dev/reflect#StructTag)
- [`reflect.DeepEqual` — pkg.go.dev/reflect#DeepEqual](https://pkg.go.dev/reflect#DeepEqual)
- [Go blog: the laws of reflection — go.dev/blog/laws-of-reflection](https://go.dev/blog/laws-of-reflection)
