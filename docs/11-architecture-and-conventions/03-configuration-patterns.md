# Configuration patterns

Configuration scattered as `os.Getenv` calls throughout a codebase has
no single place to read, no validation, and a typo becomes a zero value
rather than an error. One struct, loaded and checked once at startup,
fixes all three.

```go
type Config struct {
    Environment string        `env:"ENVIRONMENT"`
    Port        int           `env:"PORT"`
    Timeout     time.Duration `env:"TIMEOUT"`
    DBHost      string        `env:"DB_HOST"`
}
```

## One struct, loaded once

The rules that make this work:

- **Load in `main`**, before anything else runs.
- **Pass the struct down** as a constructor parameter. Never a package
  global, or you are back to invisible dependencies.
- **Fail at startup**, not on the first request. A missing database host
  should stop the process immediately, while someone is watching.

Give fields real types. `Timeout time.Duration` rather than
`TimeoutSeconds int` means the parsing and the unit live in one place
instead of at every use.

## Defaults first, then overrides

Start from a populated struct so every field has a sensible value, then
let the environment override:

```go
func Load() (Config, error) {
    c := Config{
        Environment: "local",
        Port:        8080,
        Timeout:     5 * time.Second,
    }
    // ... apply environment ...
}
```

Defaults in the struct literal are self-documenting: one place shows
what the program does with no configuration at all.

## Binding with reflection

For a handful of fields, explicit `os.LookupEnv` calls are fine and
clearer. Once there are thirty, the repetition becomes its own bug
source, and walking the struct tags is worth it — the `reflect`
technique from
[XML, CSV and reflection](../06-text-time-and-data/08-xml-csv-and-reflection.md):

```go
v := reflect.ValueOf(&c).Elem()
t := v.Type()

for i := range t.NumField() {
    tag := t.Field(i).Tag.Get("env")
    if tag == "" || tag == "-" {
        continue
    }
    raw, ok := os.LookupEnv(tag)
    if !ok {
        continue          // leave the default in place
    }

    f := v.Field(i)
    switch f.Interface().(type) {
    case string:
        f.SetString(raw)
    case int:
        n, err := strconv.Atoi(raw)
        if err != nil {
            return Config{}, fmt.Errorf("%s: %w", tag, err)
        }
        f.SetInt(int64(n))
    case time.Duration:
        d, err := time.ParseDuration(raw)
        if err != nil {
            return Config{}, fmt.Errorf("%s: %w", tag, err)
        }
        f.SetInt(int64(d))
    }
}
```

`reflect.ValueOf(&c).Elem()` is what makes the fields settable. `ok` from
`LookupEnv` distinguishes "unset" from "set to empty", so an unset
variable leaves the default rather than blanking it.

Note the error includes the variable name:

```go
// PORT=abc
// output: PORT: strconv.Atoi: parsing "abc": invalid syntax
```

`env:"-"` marks a field the loader must skip — typically one derived
from others.

This is exactly what configuration libraries do. Knowing the mechanism
means you can read theirs, and decide whether forty lines beats a
dependency.

## Derived fields

Compute anything assembled from other values once, after loading:

```go
c.DSN = fmt.Sprintf("postgres://%s/%s", c.DBHost, c.DBName)
```

Better still as a method, so it cannot drift out of date:

```go
func (c Config) DSN() string {
    return fmt.Sprintf("postgres://%s/%s", c.DBHost, c.DBName)
}
```

Either way, build it in one place. A DSN assembled at three call sites
will differ at two of them.

## Validate, and report everything at once

Returning on the first problem means fixing one variable per restart.
`errors.Join` collects them:

```go
func (c Config) validate() error {
    var errs []error
    if c.DBHost == "" {
        errs = append(errs, errors.New("DB_HOST is required"))
    }
    if c.Port < 1 || c.Port > 65535 {
        errs = append(errs, fmt.Errorf("PORT %d out of range", c.Port))
    }
    return errors.Join(errs...)
}
```

`errors.Join` returns `nil` when the slice is empty, so the happy path
needs no special case.

```go
// nothing set
// output: DB_HOST is required

// PORT=99999
// output: PORT 99999 out of range
```

Validate ranges and formats too, not just presence. A pool size of zero
or a similarity threshold of 1.5 will otherwise fail somewhere far from
the cause.

## Secrets

Environment variables are the normal transport, and they leak easily.
Two habits:

- **Never log the config struct.** `%+v` on it puts your database
  password in the logs. If you want a startup summary, write a `Redacted()`
  method, or give secret fields a `String()` that returns `"[redacted]"`.
- **Do not put secrets in defaults.** A default password in source is a
  password in your git history.

`.env` files are a local-development convenience. The standard library
does not read them; that is a third-party concern, and production should
use real environment variables or a secret store.

## Flags, environment, or file

| Source | Good for |
|---|---|
| environment | deployment config: hosts, credentials, feature flags |
| flags | per-invocation choices: `--mode`, `--dry-run`, a file path |
| file | large or structured config, checked into the repo |

Environment is the default for services, because that is what
orchestrators supply. A common order is defaults, then file, then
environment, then flags — each overriding the last. Whatever you pick,
document it, because "why is this value not taking effect" is otherwise
unanswerable.

> **From Python:** this is pydantic-settings, hand-rolled. The struct is
> the schema, the tags are the field aliases, and validation is a method
> you write. Less magic, and the whole thing fits in one readable
> function.

## Quick reference

| Concern | Form |
|---|---|
| shape | one struct, real types (`time.Duration`, not `int`) |
| when | once in `main`, passed down as a parameter |
| defaults | a populated struct literal before overrides |
| unset vs empty | `os.LookupEnv`, not `os.Getenv` |
| many fields | walk struct tags with `reflect` |
| skip a field | `env:"-"` |
| derived values | one method, not repeated at call sites |
| validation | collect with `errors.Join`, fail at startup |
| secrets | never log the struct; no secret defaults |

## Sources

- [`os.LookupEnv` — pkg.go.dev/os#LookupEnv](https://pkg.go.dev/os#LookupEnv)
- [`reflect.StructTag` — pkg.go.dev/reflect#StructTag](https://pkg.go.dev/reflect#StructTag)
- [`errors.Join` — pkg.go.dev/errors#Join](https://pkg.go.dev/errors#Join)
- [`time.ParseDuration` — pkg.go.dev/time#ParseDuration](https://pkg.go.dev/time#ParseDuration)
- [`strconv` package reference — pkg.go.dev/strconv](https://pkg.go.dev/strconv)
