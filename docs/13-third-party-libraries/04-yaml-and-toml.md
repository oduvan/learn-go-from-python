# YAML and TOML

The standard library has JSON and XML but no YAML or TOML. Both are
common in configuration, and both work the way
[encoding JSON](../06-text-time-and-data/07-encoding-json.md) taught —
struct tags and `Marshal`/`Unmarshal`.

> **Modules:** `gopkg.in/yaml.v3` and `github.com/pelletier/go-toml/v2`.

```go
var doc Doc
err := yaml.Unmarshal(data, &doc)
```

## Several tag namespaces on one struct

A type read from a config file and served as JSON carries both tags —
the mechanic from [structs](../02-language-basics/10-structs.md), where
each library reads only its own key:

```go
type Server struct {
    Host    string        `yaml:"host" json:"host" toml:"host"`
    Port    int           `yaml:"port" json:"port" toml:"port"`
    Timeout time.Duration `yaml:"timeout" json:"timeout" toml:"timeout"`
    Tags    []string      `yaml:"tags,omitempty" json:"tags,omitempty"`
    Secret  string        `yaml:"-" json:"-"`
}
```

`omitempty` and `-` mean what they do in `encoding/json`. The default
key differs, though: `encoding/json` uses the Go field name, while
`yaml.v3` lower-cases it. Write the tag explicitly and the question
does not arise.

## YAML

```go
src := `
server:
  host: example.com
  port: 8080
  timeout: 30s
  tags: [a, b]
extra:
  k: v
`

var d Doc
err := yaml.Unmarshal([]byte(src), &d)
// {Server:{Host:example.com Port:8080 Timeout:30s Tags:[a b]} Extra:map[k:v]}
```

**`yaml.v3` parses `time.Duration` from a string.** `30s` becomes a
real duration with no custom unmarshaller — worth knowing, because it
is the main reason config structs can use proper types.

Marshalling indents four spaces and quotes only when needed:

```go
out, _ := yaml.Marshal(Doc{Server: Server{Host: "h", Port: 1, Timeout: time.Second}})
```

```yaml
server:
    host: h
    port: 1
    timeout: 1s
extra: {}
```

### Reject unknown fields

By default an unrecognised key is silently ignored, so a typo in a
config file does nothing and the default stays. `KnownFields` turns
that into an error:

```go
dec := yaml.NewDecoder(r)
dec.KnownFields(true)
err := dec.Decode(&d)
// yaml: unmarshal errors:
//   line 2: field nope not found in type main.Server
```

Turn this on for anything a human edits. A mistyped `tiemout` that
leaves the default in place is a bad afternoon.

Type errors already report the line:

```go
yaml.Unmarshal([]byte("server:\n  port: notanumber\n"), &d)
// yaml: unmarshal errors:
//   line 2: cannot unmarshal !!str `notanumber` into int
```

### YAML has sharp edges

They belong to the format, not the library:

- **Indentation is significant**, and tabs are a syntax error.
- **Unquoted values get guessed.** `yes`, `no`, `on`, `off` become
  booleans; a leading zero can become octal. Quote anything that must
  stay a string — version numbers and country codes especially.
- **Anchors and aliases** (`&name`, `*name`) exist and are expanded on
  read, which is occasionally useful and frequently surprising.
- **Untrusted YAML is not safe to parse into `any`.** Prefer a struct,
  and set a size limit on the input.

## TOML

Flatter, no significant whitespace, unambiguous scalars:

```toml
[server]
host = "t.example"
port = 9090
```

```go
var d Doc
err := toml.Unmarshal([]byte(src), &d)
// {Host:t.example Port:9090}
```

Errors name the field and the types involved:

```go
toml.Unmarshal([]byte("[server]\nport = \"nope\"\n"), &d)
// toml: cannot decode TOML string into struct field main.Server.Port of type int
```

### TOML does not parse a duration

This is the difference that will catch you moving a struct between the
two formats:

```go
// timeout = "5s"
// toml: cannot decode TOML string into struct field
//       main.Server.Timeout of type time.Duration
```

And marshalling a duration produces a raw integer of nanoseconds rather
than `5s`. If a struct is shared between YAML and TOML, either declare
the field as a `string` and parse it yourself with
`time.ParseDuration`, or give the type its own
`UnmarshalText`/`MarshalText`. Both libraries call those methods when a
type has them, so the second option looks like this:

```go
type Duration struct{ time.Duration }

func (d *Duration) UnmarshalText(b []byte) error {
    v, err := time.ParseDuration(string(b))
    d.Duration = v
    return err
}

func (d Duration) MarshalText() ([]byte, error) {
    return []byte(d.String()), nil
}
```

With `Timeout Duration` in the struct, `timeout = "5s"` decodes in TOML
and `timeout: 5s` in YAML, and marshalling writes `5s` back instead of
nanoseconds.

TOML does have native dates and times, which YAML only approximates.

## Which to use

| Use | Format |
|---|---|
| Kubernetes, CI, anything in that ecosystem | YAML — no real choice |
| a tool's own config file | TOML — fewer ways to be wrong |
| an API | JSON — it is in the standard library |

TOML's advantage is that it has no ambiguous scalars and no
indentation rules, so a hand-edited file is harder to break. YAML's
advantage is that everything else already speaks it.

Whichever you pick: decode into a **struct**, not a
`map[string]any`, and validate after decoding. Both formats will happily
give you a syntactically valid document that means nothing.

> **From Python:** `yaml.Unmarshal` is `yaml.safe_load` into a
> dataclass, and there is no unsafe loader to accidentally reach for.
> `KnownFields(true)` is the strictness you would get from pydantic's
> `extra="forbid"`, and it is off by default here too.

## Quick reference

| Task | Form |
|---|---|
| decode YAML | `yaml.Unmarshal(b, &v)` |
| encode YAML | `yaml.Marshal(v)` — four-space indent |
| reject typos | `dec := yaml.NewDecoder(r); dec.KnownFields(true)` |
| durations in YAML | work natively from `30s` |
| decode TOML | `toml.Unmarshal(b, &v)` |
| durations in TOML | **not supported** — use a string, or `UnmarshalText` |
| tags | `yaml:"name,omitempty"`, `toml:"name"`, `-` to skip |
| several formats | stack the tags on one field |
| always | decode into a struct, then validate |

## Sources

- [`gopkg.in/yaml.v3` — pkg.go.dev/gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3)
- [`go-toml/v2` — pkg.go.dev/github.com/pelletier/go-toml/v2](https://pkg.go.dev/github.com/pelletier/go-toml/v2)
- [YAML 1.2 specification — yaml.org/spec/1.2.2/](https://yaml.org/spec/1.2.2/)
- [TOML specification — toml.io/en/v1.0.0](https://toml.io/en/v1.0.0)
- [`time.ParseDuration` — pkg.go.dev/time#ParseDuration](https://pkg.go.dev/time#ParseDuration)
