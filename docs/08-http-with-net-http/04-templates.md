# Templates

Two packages share one syntax. `text/template` renders any text;
`html/template` renders HTML and escapes automatically. Choosing
correctly between them is a security decision, not a style one.

```go
t := template.Must(template.New("t").Parse("Hello {{.Name}}\n"))
t.Execute(os.Stdout, Item{Name: "Ada"})
// output: Hello Ada
```

## Parsing and executing

`Parse` compiles, `Execute` renders to an `io.Writer`. `template.Must`
wraps a parse and panics on failure:

```go
var tmpl = template.Must(template.New("page").Parse(src))
```

Parse **once**, at startup, into a package-level variable. Parsing per
request wastes work and turns a template typo into a runtime error on
one unlucky path rather than a crash at boot. `Must` is right here for
the same reason `regexp.MustCompile` is: a template in your source that
does not parse is a bug that should stop the program.

`ParseFS` reads templates from an `embed.FS`, which is how you ship them
inside the binary — see [`go:embed`](../07-operating-system/03-go-embed.md).

## The syntax

`{{.}}` is the current value; `{{.Field}}` walks into it.

```
{{range .Tags}}[{{.}}]{{else}}none{{end}}
{{if gt .Price 10.0}}expensive{{else}}cheap{{end}}
{{with .Name}}name={{.}}{{end}}
```

With `Tags: ["x","y"]`, `Price: 5`, `Name: "Bo"`:

```
[x][y]
cheap
name=Bo
```

With no tags and `Price: 50`:

```
none
expensive
name=Cy
```

Inside `range` and `with`, `.` is rebound to the element — `$` keeps a
reference to the original top-level value. `range` takes an `{{else}}`
for the empty case, which is neater than a separate `if`. `with` skips
its block entirely when the value is empty.

For example, with the first data set above,
`{{range .Tags}}[{{$.Name}}:{{.}}]{{end}}` prints `[Bo:x][Bo:y]`.
Inside `range`, `.` is each tag, and `$.Name` still reaches the
top-level name.

Comparisons are functions, not operators: `eq`, `ne`, `lt`, `le`, `gt`,
`ge`, plus `and`, `or`, `not`. There is no arithmetic — compute in Go
and pass the result in.

Trim surrounding whitespace with `{{-` and `-}}`:

```go
template.Must(template.New("t").Parse("a{{- if true}} b {{- end}}c\n"))
// output: a bc
```

## Custom functions

`Funcs` must be called **before** `Parse`, since parsing resolves the
names:

```go
fm := template.FuncMap{
    "upper": strings.ToUpper,
    "money": func(f float64) string { return fmt.Sprintf("$%.2f", f) },
}

t := template.Must(template.New("t").Funcs(fm).Parse("{{upper .Name}} {{money .Price}}\n"))
// output: ADA $3.50
```

Keep these to formatting. Logic belongs in Go, where it can be tested.

## Missing data behaves differently by type

A missing **struct field** is an execution error:

```go
t := template.Must(template.New("t").Parse("{{.Nope}}"))
err := t.Execute(os.Stdout, Item{})
// err: template: t:1:2: executing "t" at <.Nope>:
//      can't evaluate field Nope in type main.Item
```

A missing **map key** is not — it renders as `<no value>`:

```go
t := template.Must(template.New("t").Parse("[{{.missing}}]\n"))
t.Execute(os.Stdout, map[string]string{})
// output: [<no value>]
```

Prefer structs for template data. You get the typo caught, and `Option`
handling (`template.Option("missingkey=error")`) is not needed.

`Execute` may have written part of the output before failing, so render
into a `bytes.Buffer` first and copy it to the response only on
success — otherwise a mid-template error leaves a half-written page
with a `200` already sent.

## `html/template` escapes; `text/template` does not

Same syntax, different behaviour, and this is the whole reason both
exist:

```go
// html/template
h := template.Must(template.New("h").Parse("<p>{{.}}</p>\n"))
h.Execute(os.Stdout, `<script>alert("x")</script>`)
// output: <p>&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;</p>
```

```go
// text/template — identical code, no escaping
t := template.Must(template.New("t").Parse("<p>{{.}}</p>\n"))
t.Execute(os.Stdout, `<script>alert("x")</script>`)
// output: <p><script>alert("x")</script></p>
```

The import path is the only difference at the call site, so **importing
the wrong one is an XSS hole that compiles cleanly**. If the output is
HTML, import `html/template`.

## The escaping is context-aware

`html/template` parses the HTML and escapes according to where the value
lands. In an attribute it escapes quotes:

```go
h := template.Must(template.New("h").Parse(`<div x="{{.}}"></div>` + "\n"))
h.Execute(os.Stdout, `a" onclick="evil()`)
// output: <div x="a&#34; onclick=&#34;evil()"></div>
```

In an `href` it recognises a dangerous scheme and replaces the whole
value:

```go
h := template.Must(template.New("h").Parse(`<a href="{{.}}">link</a>` + "\n"))
h.Execute(os.Stdout, `javascript:alert(1)`)
// output: <a href="#ZgotmplZ">link</a>
```

`ZgotmplZ` is a deliberate marker meaning "a value was rejected here".
Seeing it in a page means unsafe content reached a URL position — fix
the data, not the template.

Types like `template.HTML` and `template.JS` opt a value out of
escaping. They mean "I promise this is safe", so they must never hold
anything derived from user input.

## Composition

`define` names a block; `template` invokes one. That is how layouts work:

```go
layout := `{{define "page"}}<h1>{{.Name}}</h1>{{template "body" .}}{{end}}` +
          `{{define "body"}}<p>{{.Price}}</p>{{end}}`

h := template.Must(template.New("l").Parse(layout))
h.ExecuteTemplate(os.Stdout, "page", Item{Name: "Ada", Price: 9})
// output: <h1>Ada</h1><p>9</p>
```

The trailing `.` in `{{template "body" .}}` passes the data down —
omit it and the nested template gets `nil`. With several templates
defined, use `ExecuteTemplate` to pick one by name.

`block` defines a default that a later template can override, which
gives you a base layout with replaceable sections. `Clone` copies the
base, so each page can replace the section without touching it:

```go
base := template.Must(template.New("base").Parse(
    `<main>{{block "content" .}}default{{end}}</main>`))
page := template.Must(template.Must(base.Clone()).Parse(
    `{{define "content"}}hello{{end}}`))

base.Execute(os.Stdout, nil)   // output: <main>default</main>
page.Execute(os.Stdout, nil)   // output: <main>hello</main>
```

Call `Clone` before the first `Execute`: `html/template` refuses to
clone a template that has already run.

> **From Python:** this is Jinja with far less in it — no arithmetic, no
> filters with `|` (functions are prefix), no template inheritance
> beyond `define`/`block`. The autoescaping is stronger than Jinja's,
> because it knows whether the value is landing in an attribute, a URL
> or a script. The trap Jinja does not have is that the non-escaping
> package is one import line away.

## Quick reference

| Task | Form |
|---|---|
| HTML output | `html/template` — always |
| any other text | `text/template` |
| parse at startup | `template.Must(template.New(n).Parse(src))` |
| from an embedded FS | `template.ParseFS(fsys, "tpl/*.tmpl")` |
| render | `t.Execute(w, data)` / `t.ExecuteTemplate(w, name, data)` |
| field / current value | `{{.Field}}` / `{{.}}` |
| loop | `{{range .Xs}}...{{else}}empty{{end}}` |
| condition | `{{if gt .N 3}}...{{end}}` — functions, not operators |
| trim whitespace | `{{-` and `-}}` |
| helpers | `.Funcs(FuncMap{...})` **before** `Parse` |
| layouts | `{{define "x"}}`, `{{template "x" .}}`, `{{block}}` |
| pass data down | the trailing `.` in `{{template "x" .}}` |
| escaping rejected a value | `ZgotmplZ` in the output |

## Sources

- [`text/template` package reference — pkg.go.dev/text/template](https://pkg.go.dev/text/template)
- [`html/template` package reference — pkg.go.dev/html/template](https://pkg.go.dev/html/template)
- [Template actions — pkg.go.dev/text/template#hdr-Actions](https://pkg.go.dev/text/template#hdr-Actions)
- [Contextual autoescaping — pkg.go.dev/html/template#hdr-Contexts](https://pkg.go.dev/html/template#hdr-Contexts)
- [`template.FuncMap` — pkg.go.dev/text/template#FuncMap](https://pkg.go.dev/text/template#FuncMap)
