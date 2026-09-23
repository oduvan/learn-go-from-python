# viper

Configuration from environment variables, files, defaults and flags,
merged into one place. [Configuration patterns](../11-architecture-and-conventions/03-configuration-patterns.md)
built this by hand with `reflect`; viper is that, with file formats and
precedence included.

> **Module:** `github.com/spf13/viper`.

```go
v := viper.New()
v.AutomaticEnv()
v.SetDefault("PORT", 8080)

var cfg Config
err := v.Unmarshal(&cfg)
```

## Use an instance, not the package

`viper.Get` and friends operate on a package-level singleton — global
mutable state, shared with every library that also uses viper. Create
your own:

```go
v := viper.New()
```

Then pass the resulting struct around, not the viper instance. Config
should be read once and become a plain value.

## `mapstructure` tags, not `json`

viper decodes through `mapstructure`, so that is the tag it reads:

```go
type Config struct {
    Environment string        `mapstructure:"ENVIRONMENT"`
    Port        int           `mapstructure:"PORT"`
    Timeout     time.Duration `mapstructure:"TIMEOUT"`
    DBHost      string        `mapstructure:"DB_HOST"`
}
```

A `json:` tag here does nothing. Field names are matched
case-insensitively without a tag, but be explicit — `DBHost` and
`DB_HOST` will not match on their own.

Duration fields work: viper's default decode hooks call
`time.ParseDuration`, so `TIMEOUT=45s` lands as a real
`time.Duration`.

## The gotcha: `AutomaticEnv` does not feed `Unmarshal`

This one costs people an afternoon. `AutomaticEnv()` makes `Get`
consult the environment on demand:

```go
v.AutomaticEnv()
fmt.Printf("Get=%q IsSet=%v\n", v.GetString("ONLY_ENV"), v.IsSet("ONLY_ENV"))
// output: Get="zzz" IsSet=true
```

But `Unmarshal` iterates over keys viper *knows about*, and
`AutomaticEnv` registers none — it only intercepts lookups. So:

```go
var c struct {
    OnlyEnv string `mapstructure:"ONLY_ENV"`
}
v.Unmarshal(&c)
// c.OnlyEnv == ""
```

`Get` returns the value and `Unmarshal` leaves the field empty, with no
error either way.

The fix is to make each key known, with `BindEnv` or `SetDefault`:

```go
for _, k := range []string{"ENVIRONMENT", "PORT", "TIMEOUT", "DB_HOST"} {
    v.BindEnv(k)
}

var cfg Config
err := v.Unmarshal(&cfg)
// {Environment:prod Port:7070 Timeout:45s DBHost:db.host} err=<nil>
```

Rather than maintaining that list by hand, generate it from the struct
tags — the same reflection walk as the core article:

```go
func bindEnvs(v *viper.Viper, cfg any) {
    t := reflect.TypeOf(cfg)
    for i := range t.NumField() {
        if tag := t.Field(i).Tag.Get("mapstructure"); tag != "" && tag != "-" {
            _ = v.BindEnv(tag)
        }
    }
}
```

Call it with a value of the struct, not a pointer:

```go
bindEnvs(v, Config{})
```

`reflect.TypeOf` on a pointer gives the pointer type, which has no
fields, so `NumField` would panic.

Now adding a field to the struct is all it takes.

## Defaults and files

```go
v.SetDefault("ENVIRONMENT", "local")
v.SetDefault("PORT", 8080)
v.SetDefault("TIMEOUT", "5s")
```

`SetDefault` also registers the key, so those fields do not need
`BindEnv`.

```go
v.SetConfigFile(".env")
v.SetConfigType("env")
if err := v.ReadInConfig(); err != nil {
    var notFound viper.ConfigFileNotFoundError
    if !errors.As(err, &notFound) {
        return err          // a real parse error
    }
    // absent is fine
}
```

Distinguish "no file" from "broken file". Treating every error as
"absent" means a malformed config silently runs on defaults.

`MergeInConfig` layers a second file over the first, which is how
`.env.local` overrides `.env`.

## Precedence

Highest wins:

1. `v.Set()` — explicit override
2. a bound flag
3. environment
4. config file
5. `SetDefault`

Predictable once you know it, and invisible if you do not — so write it
in your README.

## Is it worth the dependency?

Be honest about the trade.

**For it:** several file formats, live reload, remote backends, flag
integration. If you need two of those, viper earns its place.

**Against it:** a large dependency tree for what is often forty lines,
`Unmarshal` failing silently on unregistered keys, and a
stringly-typed API where a mistyped key is a zero value rather than a
compile error.

For a service reading twenty environment variables, the hand-rolled
loader from the core article is smaller, fully typed, and fails loudly.
Reach for viper when you genuinely need layered files and formats.

Whichever you use, the rules from the core article still apply: load
once at startup, validate with `errors.Join`, fail before serving, and
never log the struct.

> **From Python:** viper is `dynaconf` — layered sources with
> precedence — while the hand-rolled version is closer to
> `pydantic-settings`. The same trade in both languages: flexibility
> against a schema the compiler can check.

## Quick reference

| Task | Form |
|---|---|
| an instance | `viper.New()`, never the package globals |
| struct tags | `mapstructure:"KEY"` |
| read env on `Get` | `v.AutomaticEnv()` |
| **make `Unmarshal` see it** | `v.BindEnv(key)` or `v.SetDefault(key, …)` |
| bind every field | reflect over `mapstructure` tags |
| a file | `SetConfigFile` + `SetConfigType` + `ReadInConfig` |
| absent vs broken | `errors.As(err, &viper.ConfigFileNotFoundError{})` |
| layer files | `MergeInConfig` |
| into a struct | `v.Unmarshal(&cfg)` — durations parse natively |
| afterwards | validate, then pass the struct, not viper |

## Sources

- [viper — pkg.go.dev/github.com/spf13/viper](https://pkg.go.dev/github.com/spf13/viper)
- [viper README — github.com/spf13/viper](https://github.com/spf13/viper)
- [`mapstructure` — pkg.go.dev/github.com/go-viper/mapstructure/v2](https://pkg.go.dev/github.com/go-viper/mapstructure/v2)
- [The twelve-factor app: config — 12factor.net/config](https://12factor.net/config)
