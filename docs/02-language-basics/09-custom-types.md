# Custom types

Go lets you define your own named types. There are **two** declaration forms — they look almost identical but mean completely different things.

```go
type Celsius float64       // type definition — a new, distinct type
type Temp    = float64     // type alias    — just another name for float64
```

The single `=` is the entire difference. The consequences are big.

## Why bother creating custom types?

Three reasons, in order of importance:

1. **The compiler stops you from mixing things that shouldn't mix.** A `Celsius` value cannot be added to a `Fahrenheit` value without you explicitly saying so.
2. **You can attach methods.** Only defined types (not aliases) can have methods — covered in a later topic, but it's the main reason to reach for `type Foo Bar`.
3. **Names communicate intent.** `UserID` is more self-documenting than yet another `int64` in a function signature.

## Type definition: `type Foo Bar`

From the Go spec:

> "A type definition creates a new, distinct type with the same underlying type and operations as the given type [...]. It is different from any other type, including the type it is created from."

```go
type Celsius float64
type Fahrenheit float64

var c Celsius = 20
var f Fahrenheit = 68

sum := c + f          // compile error: mismatched types Celsius and Fahrenheit
```

`Celsius` and `Fahrenheit` are *different types*. Both happen to be laid out in memory exactly like a `float64` — that shared layout is their **underlying type** — but the compiler treats them as distinct because they have different names.

You can convert between them with the usual `T(x)` syntax (covered in [03-type-conversions.md](03-type-conversions.md)):

```go
c2 := Celsius((f - 32) * 5 / 9)   // ok — explicit conversion
```

### Common patterns

**Semantic wrappers around primitives:**

```go
type UserID int64
type Email  string
type Money  int64    // store cents to avoid float

func sendEmail(to Email, user UserID) error { /* ... */ }

var id UserID = 42
var addr Email = "ada@example.com"
sendEmail(addr, id)              // ok — argument order matches types

sendEmail(id, addr)              // compile error: order is wrong, compiler catches it
```

This is a free safety net. Without `UserID` as its own type, both args would be `int64`/`string` and the wrong order would compile silently.

**Named function types:**

```go
type Handler func(req string) string

func register(h Handler) { /* ... */ }

register(func(req string) string {
    return "echo: " + req
})
```

**Named slice or map types:**

```go
type Headers map[string]string
type Tags    []string

func process(h Headers, t Tags) { /* ... */ }
```

**Enum-like constants** — typically combined with `iota`:

```go
type Weekday int

const (
    Sunday Weekday = iota   // 0
    Monday                  // 1
    Tuesday                 // 2
    Wednesday               // 3
    Thursday                // 4
    Friday                  // 5
    Saturday                // 6
)

func isWeekend(d Weekday) bool {
    return d == Sunday || d == Saturday
}

isWeekend(Monday)            // ok — type-checked
isWeekend(5)                 // ok — 5 is an untyped int constant, becomes Weekday
isWeekend(int(5))            // compile error — int is not Weekday
```

## Type alias: `type Foo = Bar`

From the spec:

> "An alias declaration binds an identifier to the given type. Within the scope of the identifier, it serves as an _alias_ for the given type."

An alias is **not** a new type — it's a second name for the same type.

```go
type Celsius = float64        // alias

var c Celsius = 20
var f float64 = c             // ok — c IS a float64
var sum = c + f               // ok — same type, no conversion needed
```

Compare to the definition form above: with `type Celsius float64` the compiler would reject `var f float64 = c`. With the alias, it accepts it.

### When to use aliases (rarely)

The vast majority of code wants type definitions, not aliases. Aliases exist mainly for:

1. **Gradual refactoring across packages** — temporarily expose `oldpkg.Foo` as `newpkg.Foo` while callers migrate.
2. **Shortening long type names** — `type list = []*linkedListNode[T]` for readability inside one file.
3. **The standard library uses them sparingly** — `byte = uint8` and `rune = int32` are aliases. That's why `byte` and `uint8` are interchangeable everywhere.

```go
var b byte   = 65
var u uint8  = b              // ok — alias means same type
var c uint32 = uint32(b)      // ok — explicit conversion needed (different types)
```

**Key restriction:** you **cannot attach methods** to an alias. The alias has no separate identity to attach them to.

```go
type Celsius = float64
func (c Celsius) Fmt() string { return "..." }   // compile error
```

If you want methods, use a type definition.

## Methods — the headline feature of type definitions

Once you've defined a type, you can attach **methods** to it — that's
the main reason to reach for `type Foo Bar` in the first place. The
mechanics (value vs. pointer receivers, method sets, embedding) live
in [10-methods.md](10-methods.md). The one rule worth flagging here:
methods can only be defined on types from your own package — never
on `int`, `string`, `time.Duration`, or any other foreign type.

## "Underlying type" — the precise rule

From the spec, paraphrased into a process:

1. Predeclared types (`int`, `string`, `bool`, `float64`, ...) — their underlying type is themselves.
2. Type literals (`[]int`, `map[string]int`, `struct{...}`, `func(...) ...`) — their underlying type is themselves.
3. Any other type — follow the chain of definitions until you reach one of the above.

```go
type A = string         // underlying type: string
type B string           // underlying type: string  (alias chain: B → string)
type C B                // underlying type: string  (chain: C → B → string)
type D struct {         // underlying type: struct{Name string}  (a type literal)
    Name string
}
```

You can check at runtime with `reflect.TypeOf(x).Kind()` — it returns the underlying kind, not the named type.

## Type identity in one paragraph

Two types are **identical** in Go's eyes when:

- They are the same defined type (same name, same package), OR
- They are both type literals with structurally matching underlying types.

A defined type is **never identical** to its underlying type or to any other defined type, even if the layouts match. That's the entire reason `Celsius` and `Fahrenheit` don't mix.

## Quick reference

| Form | New type? | Methods? | Mixes with original? |
|---|---|---|---|
| `type Foo Bar` (definition) | yes | yes | no — needs explicit conversion |
| `type Foo = Bar` (alias) | no | no | yes — same type |

## Sources

- [Type declarations — go.dev/ref/spec#Type_declarations](https://go.dev/ref/spec#Type_declarations)
- [Type definitions — go.dev/ref/spec#Type_definitions](https://go.dev/ref/spec#Type_definitions)
- [Alias declarations — go.dev/ref/spec#Alias_declarations](https://go.dev/ref/spec#Alias_declarations)
- [Underlying types — go.dev/ref/spec#Types](https://go.dev/ref/spec#Types)
- [Type identity — go.dev/ref/spec#Type_identity](https://go.dev/ref/spec#Type_identity)
- [Method declarations — go.dev/ref/spec#Method_declarations](https://go.dev/ref/spec#Method_declarations)
