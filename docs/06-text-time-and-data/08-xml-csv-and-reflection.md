# XML, CSV and reflection

Two more encoders from the standard library, and then the machinery that
makes all of them work. `reflect` is last because the honest advice is to
use it rarely — but you cannot read the encoders, or a config loader,
without knowing what it does.

```go
r := csv.NewReader(strings.NewReader("name,age\nAda,36\n"))
recs, _ := r.ReadAll()
fmt.Println(recs)   // output: [[name age] [Ada 36]]
```

## XML

`encoding/xml` works like `encoding/json` with a richer tag vocabulary,
because XML distinguishes attributes from elements and allows nesting:

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

| Tag | Meaning |
|---|---|
| `xml:"title"` | element name |
| `xml:"id,attr"` | an attribute rather than an element |
| `xml:"tags>tag"` | repeated `<tag>` inside a `<tags>` wrapper |
| `xml:",chardata"` | the element's text content |
| `xml:"-"` | skip |

An `XMLName xml.Name` field names the root element. Decoding is the
mirror image:

```go
var b Book
xml.Unmarshal([]byte(`<book id="9"><title>T</title><tags><tag>x</tag></tags></book>`), &b)
fmt.Println(b.ID, b.Title, b.Tags)   // output: 9 T [x]
```

### Token streaming

For a large document, or one whose shape you do not control, read it as
a stream of tokens. `Token` returns each piece in turn, and a
[type switch](../03-object-oriented-go/03-type-assertions-and-type-switches.md)
sorts them out:

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

Nothing larger than one token is ever in memory. `xml.CharData` is a
`[]byte`, so it needs `string(t)` to print as text — and it is only
valid until the next `Token` call, so copy it if you keep it.

This is also the pattern for pulling text out of formats that are XML
underneath, such as the parts inside an Office document.

## CSV

`encoding/csv` handles quoting, embedded commas and embedded newlines —
all the reasons splitting on `,` yourself is wrong.

```go
r := csv.NewReader(strings.NewReader("name,age\nAda,36\nBo,7\n"))
recs, err := r.ReadAll()
fmt.Println(recs, err)
// output: [[name age] [Ada 36] [Bo 7]] <nil>
```

`ReadAll` holds the whole file. For anything big, read record by record
and stop at `io.EOF`:

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

There is no header handling and no mapping to structs — a record is a
`[]string`, and the first `Read` is the header only because you treated
it that way.

By default the reader enforces that every record has the same number of
fields as the first, which catches truncated files early:

```go
r := csv.NewReader(strings.NewReader("a,b\n1\n"))
_, err := r.ReadAll()
fmt.Println(err)   // output: record on line 2: wrong number of fields
```

Set `r.FieldsPerRecord = -1` to allow ragged rows, and `r.Comma = '\t'`
for TSV.

Writing quotes only what needs it, and **must be flushed**:

```go
var sb strings.Builder
w := csv.NewWriter(&sb)
w.Write([]string{"name", "note"})
w.Write([]string{"Ada", "has, comma"})
w.Flush()

fmt.Printf("%q\n", sb.String())
// output: "name,note\nAda,\"has, comma\"\n"
```

Forgetting `Flush` silently loses buffered rows. Check `w.Error()`
afterwards, since `Write` itself only reports errors from earlier calls.

## Reflection

Reflection is how `encoding/json`, `encoding/xml` and every config
loader read struct tags and walk fields of types they have never seen.
Three ideas cover most of it: `Type` describes a type, `Value` holds a
value, and `Kind` says which category of type it is.

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

`Kind` is the category — `struct`, `int`, `slice`, `ptr` — as distinct
from the named type. `type Celsius float64` has kind `float64` and name
`Celsius`.

### Reading struct tags

This loop is the heart of every tag-driven library:

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

`Tag.Get` returns `""` for a key that is not there, which is why a
misspelled tag fails silently rather than loudly — the point the
[structs](../02-language-basics/10-structs.md) article makes.

`IsExported` is also why unexported fields never appear in JSON: a
library iterating fields can see `skip`, but cannot read or write it.

### Setting values needs a pointer

A `reflect.Value` obtained from a plain value is not addressable — it is
a copy, so writing to it would change nothing and Go refuses rather than
pretending:

```go
c := Cfg{Host: "h", Port: 1}

pv := reflect.ValueOf(&c).Elem()   // Elem() follows the pointer
pv.Field(0).SetString("changed")
fmt.Println(c.Host)                // output: changed

fmt.Println(reflect.ValueOf(c).CanSet(), pv.Field(0).CanSet())
// output: false true
```

`ValueOf(&c).Elem()` is the incantation, and `CanSet` is what to check
before writing. This is exactly why `json.Unmarshal` demands a pointer.

### Two things reflection is genuinely good for

Detecting a typed nil, which `== nil` cannot do — the gotcha from
[interfaces](../03-object-oriented-go/02-interfaces.md):

```go
var p *Cfg
var iface any = p

fmt.Println(iface == nil)                      // output: false
fmt.Println(reflect.ValueOf(iface).IsNil())    // output: true
```

And comparing values that `==` refuses, such as maps and slices:

```go
fmt.Println(reflect.DeepEqual(map[string]int{"a": 1}, map[string]int{"a": 1}))
// output: true
fmt.Println(reflect.DeepEqual([]int{1}, []int{1}))
// output: true
```

`DeepEqual` distinguishes a nil slice from an empty one, which is
usually not what a test means:

```go
var ns []int
fmt.Println(reflect.DeepEqual(ns, []int{}))   // output: false
```

For slices and maps of comparable elements, prefer `slices.Equal` and
`maps.Equal` — they are typed, faster, and treat nil and empty alike.

### Prefer not to

Reflection turns compile-time errors into runtime panics, defeats the
type system, and is slow. Before reaching for it, ask whether an
interface or a generic function would do — both keep the checking at
compile time. The legitimate cases are narrow: implementing a
serialiser, reading struct tags, or writing a test helper. If you are
using it for business logic, there is almost always a better shape.

> **From Python:** reflection is `getattr`, `type()` and `__dict__` —
> ordinary tools there, specialist ones here. `reflect.DeepEqual` is
> `==` on nested structures. The unfamiliar part is addressability: you
> cannot set a field through a copy, so everything mutable starts with a
> pointer and `.Elem()`.

## Quick reference

| Task | Call |
|---|---|
| XML attribute / nested list | `xml:"id,attr"` / `xml:"tags>tag"` |
| stream a large XML document | `xml.NewDecoder(r).Token()` + type switch |
| read a CSV | `csv.NewReader(r)`, `.Read()` until `io.EOF` |
| ragged rows / TSV | `r.FieldsPerRecord = -1` / `r.Comma = '\t'` |
| write a CSV | `csv.NewWriter(w)` — **`Flush()`**, then `Error()` |
| a type's fields | `reflect.TypeOf(v).Field(i)` |
| a struct tag | `f.Tag.Get("json")` — `""` when absent |
| mutate through reflection | `reflect.ValueOf(&v).Elem()`, check `CanSet` |
| detect a typed nil | `reflect.ValueOf(x).IsNil()` |
| compare maps or slices | `slices.Equal` / `maps.Equal`, else `reflect.DeepEqual` |

## Sources

- [`encoding/xml` package reference — pkg.go.dev/encoding/xml](https://pkg.go.dev/encoding/xml)
- [`encoding/csv` package reference — pkg.go.dev/encoding/csv](https://pkg.go.dev/encoding/csv)
- [`reflect` package reference — pkg.go.dev/reflect](https://pkg.go.dev/reflect)
- [`reflect.StructTag` — pkg.go.dev/reflect#StructTag](https://pkg.go.dev/reflect#StructTag)
- [`reflect.DeepEqual` — pkg.go.dev/reflect#DeepEqual](https://pkg.go.dev/reflect#DeepEqual)
- [Go blog: the laws of reflection — go.dev/blog/laws-of-reflection](https://go.dev/blog/laws-of-reflection)
