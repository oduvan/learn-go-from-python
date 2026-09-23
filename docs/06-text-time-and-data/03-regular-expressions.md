# Regular expressions

Go's `regexp` package implements RE2, not the Perl-compatible dialect
Python uses. Most patterns you already know work unchanged; two features
are missing on purpose, and that is the part worth knowing up front.

```go
var semver = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

fmt.Println(semver.MatchString("v1.22.3"))   // output: true
fmt.Println(semver.MatchString("1.22.3"))    // output: false
```

## Compile once, at package level

Compiling a pattern is expensive; matching with it is cheap. A compiled
`*regexp.Regexp` is safe for concurrent use, so the idiom is a
package-level variable built with `MustCompile`, which panics on a bad
pattern:

```go
var semver = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)
```

Panicking is the right behaviour here: the pattern is a constant in your
source, so a broken one is a bug that should stop the program at startup
rather than at the first request. Use `regexp.Compile` and handle the
error only when the pattern comes from outside the program:

```go
_, err := regexp.Compile(`[a-`)
fmt.Println(err)
// output: error parsing regexp: missing closing ]: `[a-`
```

Note the **backquoted** string: a
[raw string literal](../02-language-basics/02-basic-types.md), where a
backslash is just a backslash. In a regular `"..."` literal you would
have to write `\\d` for every `\d`, so raw strings are all but mandatory
for patterns.

## Matching and extracting

`MatchString` answers yes or no. `FindStringSubmatch` returns the whole
match at index 0 and each capture group after it — or `nil` when there
is no match:

```go
m := semver.FindStringSubmatch("v1.22.3")
fmt.Printf("%q\n", m)   // output: ["v1.22.3" "1" "22" "3"]

fmt.Println(semver.FindStringSubmatch("nope") == nil)   // output: true
```

There is no `ok` second result, so the `nil` check *is* the match check.
Indexing `m[1]` without it panics on no match.

Named groups use `(?P<name>...)`, and `SubexpIndex` turns the name back
into a position:

```go
named := regexp.MustCompile(`(?P<key>\w+)=(?P<val>\w+)`)
m := named.FindStringSubmatch("mode=fast")

fmt.Println(named.SubexpIndex("val"), m[named.SubexpIndex("val")])
// output: 2 fast
```

That is clumsier than Python's `m.group("val")`, and it is the main
ergonomic cost of the package.

## Finding every match

The `All` variants take a count, where `-1` means "no limit":

```go
word := regexp.MustCompile(`\w+`)

fmt.Printf("%q\n", word.FindAllString("a bb ccc", -1))  // output: ["a" "bb" "ccc"]
fmt.Printf("%q\n", word.FindAllString("a bb ccc", 2))   // output: ["a" "bb"]

fmt.Println(word.FindString("  hi there"))              // output: hi
fmt.Println(word.FindStringIndex("  hi there"))         // output: [2 4]
```

`FindString` returns `""` for no match, which is ambiguous when the
pattern can match an empty string — `FindStringIndex` returning `nil` is
the unambiguous form.

## Replacing

In the replacement string, `$1` and `${name}` refer to capture groups:

```go
date := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)

fmt.Println(date.ReplaceAllString("on 2026-09-20 ok", "$3/$2/$1"))
// output: on 20/09/2026 ok
```

Use the braces whenever a digit could run into following text. `$3x`
would be read as the group named `3x`, which does not exist and expands
to nothing:

```go
fmt.Println(date.ReplaceAllString("on 2026-09-20 ok", "${3}x"))
// output: on 20x ok
```

When the replacement needs real logic, `ReplaceAllStringFunc` hands you
each match:

```go
fmt.Println(word.ReplaceAllStringFunc("go rocks", func(s string) string {
    return "<" + s + ">"
}))
// output: <go> <rocks>
```

Splitting takes a count for the same reason the `All` functions do:

```go
sp := regexp.MustCompile(`\s*,\s*`)
fmt.Printf("%q\n", sp.Split("a ,b,  c", -1))   // output: ["a" "b" "c"]
```

## Flags go inside the pattern

There is no flags argument. `(?i)`, `(?m)` and `(?s)` go at the start of
the pattern instead:

```go
fmt.Println(regexp.MustCompile(`(?i)^go$`).MatchString("GO"))   // output: true

fmt.Printf("%q\n", regexp.MustCompile(`(?m)^\w+`).FindAllString("one\ntwo", -1))
// output: ["one" "two"]

fmt.Println(regexp.MustCompile(`(?s)a.b`).MatchString("a\nb"))  // output: true
```

`(?i)` is case-insensitive, `(?m)` makes `^`/`$` match at line
boundaries, `(?s)` lets `.` match a newline.

## What RE2 will not do

Backreferences and lookaround are absent, and they are absent by design.
RE2 guarantees matching in time linear in the input length, which rules
out the constructs that make a regex able to blow up exponentially. A
pattern that takes a hostile input and hangs the process is not possible
here.

```go
_, err := regexp.Compile(`(\w)\1`)
fmt.Println(err)
// output: error parsing regexp: invalid escape sequence: `\1`

_, err = regexp.Compile(`(?=foo)`)
fmt.Println(err)
// output: error parsing regexp: invalid or unsupported Perl syntax: `(?=`
```

If you need them, the answer is usually to match something broader and
then check the rest in ordinary Go code — which is clearer than the
lookahead would have been anyway.

## Escaping a literal

When part of a pattern comes from data, escape it:

```go
fmt.Println(regexp.QuoteMeta("a.b*c"))   // output: a\.b\*c
```

## Reach for `strings` first

A regex is slower and harder to read than a direct call. If
`strings.Contains`, `HasPrefix`, `Cut` or `Fields` will do the job, use
those — see [strings, bytes and runes](01-strings-bytes-and-runes.md).

> **From Python:** `MustCompile` is `re.compile` at import time,
> `MatchString` is `re.search` returning a bool, and
> `FindStringSubmatch` is `m.groups()` with the full match prepended.
> The differences that will catch you: flags live inside the pattern,
> there is no match object so you check for `nil`, named groups need
> `SubexpIndex`, and `\1` and `(?=...)` simply do not exist.

## Quick reference

| Task | Call |
|---|---|
| compile a constant pattern | ``regexp.MustCompile(`...`)`` at package level |
| compile untrusted input | `regexp.Compile`, handle the error |
| yes/no | `MatchString` |
| capture groups | `FindStringSubmatch` — `nil` means no match |
| named group | `SubexpIndex("name")` |
| every match | `FindAllString(s, -1)` |
| replace with groups | `ReplaceAllString(s, "${1}")` |
| replace with logic | `ReplaceAllStringFunc` |
| split | `Split(s, -1)` |
| case-insensitive / multiline / dotall | `(?i)` / `(?m)` / `(?s)` |
| escape literal text | `regexp.QuoteMeta` |

## Sources

- [`regexp` package reference — pkg.go.dev/regexp](https://pkg.go.dev/regexp)
- [`regexp/syntax` — pkg.go.dev/regexp/syntax](https://pkg.go.dev/regexp/syntax)
- [RE2 syntax — github.com/google/re2/wiki/Syntax](https://github.com/google/re2/wiki/Syntax)
- [Go blog: regular expression matching can be simple and fast — swtch.com/~rsc/regexp/regexp1.html](https://swtch.com/~rsc/regexp/regexp1.html)
