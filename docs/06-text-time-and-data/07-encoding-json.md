# Encoding JSON

`encoding/json` maps Go values to JSON using struct tags and reflection.
The [structs](../02-language-basics/10-structs.md) article introduced the
tag syntax; this one is about the behaviour, which has a handful of sharp
edges worth meeting deliberately.

```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email,omitempty"`
    Age   int    `json:"age"`
}

b, _ := json.Marshal(User{Name: "Ada", Age: 36})
fmt.Println(string(b))   // output: {"name":"Ada","age":36}
```

`Marshal` returns `[]byte`, so printing it needs `string(...)`.

## Only exported fields travel

Unexported fields are invisible to the encoder — it uses reflection, and
reflection cannot read them. There is no tag that changes this. If a
field must be serialised, export it.

## Tag options

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

| Option | Effect |
|---|---|
| `json:"name"` | rename the key |
| `json:",omitempty"` | drop the key when the value is zero |
| `json:"-"` | never encode or decode this field |
| `json:",string"` | encode a number as a JSON string |

`,string` exists because JSON numbers are float64 in many consumers, so
an `int64` beyond 2^53 loses precision in transit. Sending it as a
string avoids that.

Use `json.MarshalIndent` when a human will read the output:

```go
b, _ := json.MarshalIndent(u, "", "  ")
```

## `omitempty` means zero, not absent

`omitempty` drops a field whose value is the zero value — `0`, `""`,
`false`, `nil`, or an empty slice or map. It cannot tell "the user set
this to zero" from "the user said nothing", because in Go those are the
same value.

When that distinction matters, use a pointer. `nil` is then "absent" and
`&""` is "explicitly empty":

```go
var a, b, c User
json.Unmarshal([]byte(`{"name":"x"}`), &a)
json.Unmarshal([]byte(`{"name":"x","nick":null}`), &b)
json.Unmarshal([]byte(`{"name":"x","nick":""}`), &c)

fmt.Println(a.Nick == nil, b.Nick == nil, *c.Nick == "")
// output: true true true
```

Missing and explicit `null` both leave the pointer `nil`; only a real
value allocates one.

## A nil slice encodes as `null`, not `[]`

This is the one that causes bug reports from front-end developers:

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

A nil slice and an empty slice behave identically everywhere in Go —
`len` is 0, `range` does nothing, `append` works. They differ only here.
If a client will iterate the field, initialise it to an empty slice
rather than leaving it nil.

## Decoding

`Unmarshal` needs a pointer, because it must write into your value:

```go
var u User
err := json.Unmarshal([]byte(`{"name":"Bo","age":7,"id":"99"}`), &u)
```

Key matching is case-insensitive and unknown keys are ignored by
default, so a payload with extra fields decodes fine. Fields absent from
the JSON are left at their current value — `Unmarshal` does not reset
the target first, which matters when you reuse a variable in a loop.

Forget the `&` and you get a clear error rather than silence:

```go
err := json.Unmarshal(data, u)
fmt.Println(err)   // output: json: Unmarshal(non-pointer main.User)
```

## Decoding into `any`

Without a struct, JSON decodes into a fixed set of Go types:

```go
var v any
json.Unmarshal([]byte(`{"n":1,"s":"x","b":true,"arr":[1,2],"o":{"k":1}}`), &v)

m := v.(map[string]any)
fmt.Printf("%T %T %T %T %T\n", m["n"], m["s"], m["b"], m["arr"], m["o"])
// output: float64 string bool []interface {} map[string]interface {}
```

| JSON | Go |
|---|---|
| number | `float64` |
| string | `string` |
| `true`/`false` | `bool` |
| array | `[]any` |
| object | `map[string]any` |
| `null` | `nil` |

**Every number becomes a `float64`**, even one that looks like an
integer. `m["n"].(int)` panics. Either define a struct, or convert:
`int(m["n"].(float64))`.

## Errors worth handling

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

`*json.UnmarshalTypeError` carries the field name and what was found, so
an API can tell a caller which field was wrong — a good use of
[errors.As](../03-object-oriented-go/06-custom-error-types.md).
Malformed input gives a different error entirely:

```go
err = json.Unmarshal([]byte(`{`), &a)
fmt.Println(err)   // output: unexpected end of JSON input
```

## `json.RawMessage` defers decoding

When the shape of one field depends on another, keep it as raw bytes and
decode it once you know what it is:

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

`RawMessage` is just `[]byte` that encodes itself verbatim, so it also
works for passing JSON through untouched.

## Custom encoding

A type controls its own representation with `MarshalJSON` and
`UnmarshalJSON`:

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

`MarshalJSON` takes a value receiver, `UnmarshalJSON` a pointer receiver
— it has to write to the receiver. Getting this wrong is silent: a
pointer-receiver `MarshalJSON` is simply not found when you marshal a
value.

### The `type plain T` trick

Inside `MarshalJSON`, calling `json.Marshal(t)` on your own type calls
`MarshalJSON` again — infinite recursion. The way out relies on a rule
worth stating plainly: **a defined type starts with an empty method
set.** `type plain Wrapper` has all of `Wrapper`'s fields and none of its
methods, so marshalling a `plain` cannot re-enter `MarshalJSON`:

```go
func (w Wrapper) MarshalJSON() ([]byte, error) {
    type plain Wrapper       // same fields, no methods
    p := plain(w)
    p.Kind = strings.ToUpper(p.Kind)
    return json.Marshal(p)
}

b, _ := json.Marshal(Wrapper{Name: "n", Kind: "abc"})
fmt.Println(string(b))   // output: {"name":"n","kind":"ABC"}
```

Struct tags survive the conversion, so the keys are unchanged.

## Streaming with `Decoder` and `Encoder`

`Unmarshal` needs the whole document in memory. `json.NewDecoder` reads
from an `io.Reader` and can decode a stream of values one at a time:

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

Breaking on any error conflates "end of input" with "bad input"; real
code compares against `io.EOF`. Decoders also support strict mode:

```go
dec := json.NewDecoder(strings.NewReader(`{"nope":1}`))
dec.DisallowUnknownFields()
fmt.Println(dec.Decode(&u))   // output: json: unknown field "nope"
```

`json.NewEncoder(w)` is the mirror image and writes straight to an
`io.Writer`, adding a newline after each value.

> **From Python:** `Marshal`/`Unmarshal` are `json.dumps`/`json.loads`,
> but typed. The three things with no Python equivalent: unexported
> fields never serialise, a nil slice becomes `null` rather than `[]`,
> and decoding into `any` makes every number a `float64`.

## Quick reference

| Task | Call |
|---|---|
| encode | `json.Marshal(v)` → `[]byte` |
| encode readably | `json.MarshalIndent(v, "", "  ")` |
| decode | `json.Unmarshal(b, &v)` — pointer required |
| rename / omit / skip | `json:"name,omitempty"`, `json:"-"` |
| big integers | `json:",string"` |
| distinguish absent from zero | a pointer field |
| avoid `null` for a list | initialise to `[]T{}` |
| defer decoding | `json.RawMessage` |
| custom form | `MarshalJSON` (value), `UnmarshalJSON` (pointer) |
| avoid recursion in those | `type plain T` |
| stream | `json.NewDecoder(r)` / `json.NewEncoder(w)` |
| reject unknown keys | `dec.DisallowUnknownFields()` |

## Sources

- [`encoding/json` package reference — pkg.go.dev/encoding/json](https://pkg.go.dev/encoding/json)
- [`json.Marshal` — pkg.go.dev/encoding/json#Marshal](https://pkg.go.dev/encoding/json#Marshal)
- [`json.Unmarshal` — pkg.go.dev/encoding/json#Unmarshal](https://pkg.go.dev/encoding/json#Unmarshal)
- [`json.RawMessage` — pkg.go.dev/encoding/json#RawMessage](https://pkg.go.dev/encoding/json#RawMessage)
- [`json.Decoder` — pkg.go.dev/encoding/json#Decoder](https://pkg.go.dev/encoding/json#Decoder)
- [Go blog: JSON and Go — go.dev/blog/json](https://go.dev/blog/json)
