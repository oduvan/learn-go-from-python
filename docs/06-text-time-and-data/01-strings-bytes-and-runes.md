# Strings, bytes and runes

The [basic types](../02-language-basics/02-basic-types.md) article
established what a string *is*: an immutable sequence of bytes, usually
holding UTF-8. This one is about the packages you actually reach for —
`strings`, `bytes` and `unicode/utf8` — and the places where the byte/
character distinction bites.

```go
s := "  Name: Ada Lovelace  "
fmt.Printf("%q\n", strings.TrimSpace(s))   // output: "Name: Ada Lovelace"
```

## Searching

```go
fmt.Println(strings.Contains("chicken", "ken"))    // output: true
fmt.Println(strings.HasPrefix("chicken", "chi"))   // output: true
fmt.Println(strings.Index("chicken", "ken"))       // output: 4
fmt.Println(strings.Index("chicken", "zz"))        // output: -1
```

`Index` returns a **byte** offset, and `-1` when there is no match. It
never fails, so there is nothing to handle.

## `Cut` is usually what you want

Splitting "key: value" into two parts is common enough to have its own
function, and it tells you whether the separator was there at all:

```go
key, val, found := strings.Cut("Name: Ada", ": ")
fmt.Printf("%q %q %v\n", key, val, found)   // output: "Name" "Ada" true

_, _, found = strings.Cut("noseparator", ": ")
fmt.Println(found)                          // output: false
```

Without the `found` result you cannot distinguish "separator missing"
from "value was empty" — which is exactly the bug `Cut` exists to
prevent.

## `Split` and `Fields` are not the same

`Split` cuts on an exact separator and keeps empty pieces. `Fields`
splits on runs of whitespace and discards them:

```go
fmt.Printf("%q\n", strings.Split("a,b,,c", ","))   // output: ["a" "b" "" "c"]
fmt.Printf("%q\n", strings.Fields("  a   b \t c\n"))
// output: ["a" "b" "c"]
```

Reach for `Split` when the separator is data (a CSV field, a path) and
`Fields` when it is just spacing. Using `Split(s, " ")` on
human-formatted text gives you a slice full of empty strings:

```go
fmt.Printf("%q\n", strings.Split("  a   b ", " "))
// output: ["" "" "a" "" "" "b" ""]
```

## Building and replacing

```go
fmt.Println(strings.Join([]string{"a", "b", "c"}, "-"))  // output: a-b-c
fmt.Println(strings.Repeat("ab", 3))                     // output: ababab
fmt.Println(strings.ReplaceAll("a.b.c", ".", "/"))       // output: a/b/c
fmt.Println(strings.Replace("a.b.c", ".", "/", 1))       // output: a/b.c
```

For several replacements in one pass, `strings.NewReplacer` beats
chained `ReplaceAll` calls — it walks the input once, and the value is
safe to reuse and to share between goroutines:

```go
r := strings.NewReplacer("<", "&lt;", ">", "&gt;")
fmt.Println(r.Replace("<b>hi</b>"))   // output: &lt;b&gt;hi&lt;/b&gt;
```

## `strings.Builder`

The [operators](../02-language-basics/04-operators.md) article showed why
`s += x` in a loop is O(n²). `Builder` is the fix, and because it
implements `io.Writer` you can also print straight into it:

```go
var b strings.Builder
for i := range 3 {
    fmt.Fprintf(&b, "%d,", i)
}
fmt.Println(b.String())   // output: 0,1,2,
```

Note `&b` — the `Write` methods have pointer receivers, so a `Builder`
must not be copied after first use.

## The `bytes` package mirrors `strings`

Almost every function in `strings` has a `bytes` twin with the same name
that works on `[]byte`. Use it when the data arrives as bytes — from a
file, a socket, a request body — so you avoid converting to `string` and
back, since each conversion copies:

```go
bb := []byte("hello")
fmt.Println(bytes.Contains(bb, []byte("ell")))   // output: true
fmt.Println(string(bytes.ToUpper(bb)))           // output: HELLO
```

The `string(...)` there is not decoration. `bytes.ToUpper` returns a
`[]byte`, and printing one shows the numbers:

```go
fmt.Println(bytes.ToUpper(bb))   // output: [72 69 76 76 79]
```

The one gap is `strings.NewReplacer` — there is no `bytes.NewReplacer`.
It is the only exported function in `strings` with no counterpart, so
for that case convert, or chain `bytes.ReplaceAll`.

`bytes.Buffer` is the `[]byte` counterpart of `strings.Builder`, and it
is both an `io.Writer` and an `io.Reader`:

```go
var buf bytes.Buffer
buf.WriteString("abc")
buf.WriteByte('!')
fmt.Println(buf.String(), buf.Len())   // output: abc! 4
```

## Bytes are not characters

`len` counts bytes. A rune — one Unicode code point — takes one to four
of them:

```go
g := "héllo, 世界"
fmt.Println(len(g), utf8.RuneCountInString(g))   // output: 14 9
```

Ranging over a string decodes runes and gives you the **byte offset** of
each, so the index jumps:

```go
for i, r := range "gö" {
    fmt.Printf("%d:%c(%d) ", i, r, r)
}
// output: 0:g(103) 1:ö(246)
```

There is no index `2`: `ö` occupies bytes 1 and 2. Which is why slicing
a string slices *bytes*, and converting to `[]rune` first is what slices
characters:

```go
fmt.Println(g[:5])                    // output: héll
fmt.Println(string([]rune(g)[:5]))    // output: héllo
```

Converting to `[]rune` allocates a copy, so do it when you genuinely
need character positions, not by reflex.

`unicode/utf8` handles the awkward edges — decoding one rune at a time,
and checking that bytes from outside your program are valid UTF-8 at all:

```go
r, size := utf8.DecodeRuneInString("世界")
fmt.Println(string(r), size)          // output: 世 3

fmt.Println(utf8.ValidString("ok"))                    // output: true
fmt.Println(utf8.ValidString(string([]byte{0xff})))    // output: false
```

## Case folding is not simple

`ToUpper` and `ToLower` map rune by rune, which is not the same as the
casing rules of any particular language:

```go
fmt.Println(strings.ToUpper("größe"))   // output: GRÖßE
fmt.Println(strings.ToLower("ÄPFEL"))   // output: äpfel
```

German `ß` has no single-rune uppercase, so it survives unchanged. For
comparing two strings case-insensitively, do not lowercase both — use
the function built for it:

```go
fmt.Println(strings.EqualFold("Go", "GO"))   // output: true
```

> **From Python:** a Go `string` is Python's `bytes` with a UTF-8
> convention, not Python's `str`. `len()` differs for exactly that
> reason, `s[0]` gives you a byte rather than a one-character string, and
> `[]rune(s)` is the closest thing to what Python hands you when you
> index a `str`.

## Quick reference

| Task | Call |
|---|---|
| substring test / position | `strings.Contains`, `strings.Index` (byte offset, `-1` if absent) |
| split on "key: value" | `strings.Cut` — returns the `found` flag too |
| split on a separator | `strings.Split` (keeps empties) |
| split on whitespace | `strings.Fields` (drops empties) |
| join / repeat | `strings.Join`, `strings.Repeat` |
| many replacements at once | `strings.NewReplacer` |
| build a string in a loop | `strings.Builder`, passed as `&b` |
| same operations on `[]byte` | the `bytes` package, plus `bytes.Buffer` |
| count characters | `utf8.RuneCountInString` (`len` counts bytes) |
| slice by character | `[]rune(s)[a:b]` (allocates) |
| case-insensitive compare | `strings.EqualFold` |
| validate external bytes | `utf8.ValidString` |

## Sources

- [`strings` package reference — pkg.go.dev/strings](https://pkg.go.dev/strings)
- [`bytes` package reference — pkg.go.dev/bytes](https://pkg.go.dev/bytes)
- [`unicode/utf8` package reference — pkg.go.dev/unicode/utf8](https://pkg.go.dev/unicode/utf8)
- [Go blog: strings, bytes, runes and characters — go.dev/blog/strings](https://go.dev/blog/strings)
- [String types — go.dev/ref/spec#String_types](https://go.dev/ref/spec#String_types)
- [For statements with range clause — go.dev/ref/spec#For_range](https://go.dev/ref/spec#For_range)
