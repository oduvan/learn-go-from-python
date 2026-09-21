# Time

Three things in the `time` package surprise people: layouts are written
as an example date rather than as `%Y-%m-%d`, a duration is a real type
you can do arithmetic on, and `==` is the wrong way to compare two
instants.

```go
t := time.Date(2026, time.September, 20, 15, 4, 5, 0, time.UTC)

fmt.Println(t.Format("2006-01-02 15:04:05"))   // output: 2026-09-20 15:04:05
```

## The reference layout

`2006-01-02 15:04:05` is not a pattern with wildcards. It is a specific
instant — **Mon Jan 2 15:04:05 MST 2006** — and you write it out in the
shape you want. Each component has one number:

| Component | Value | Why |
|---|---|---|
| month | `1` or `01` or `Jan` or `January` | first |
| day | `2` or `02` | second |
| hour | `15` (24h) or `3`/`03` (12h) | third |
| minute | `4` or `04` | fourth |
| second | `5` or `05` | fifth |
| year | `2006` or `06` | sixth |
| timezone | `-0700` or `Z07:00` | seventh |

The mnemonic is the ordering 1, 2, 3, 4, 5, 6, 7. So:

```go
fmt.Println(t.Format("2006-01-02"))              // output: 2026-09-20
fmt.Println(t.Format("02/01/2006"))              // output: 20/09/2026
fmt.Println(t.Format("Jan 2, 2006 at 3:04PM"))   // output: Sep 20, 2026 at 3:04PM
```

Note `15` versus `3`: writing `15` asks for a 24-hour clock, writing `3`
asks for a 12-hour one. There is no separate flag.

The package ships the common layouts as constants — prefer them:

```go
fmt.Println(t.Format(time.RFC3339))   // output: 2026-09-20T15:04:05Z
fmt.Println(t.Format(time.Kitchen))   // output: 3:04PM
```

`RFC3339` is the one to use for anything a machine will read back.

## Parsing uses the same layout

```go
p, err := time.Parse("2006-01-02", "2026-09-20")
fmt.Println(p.Format(time.RFC3339), err)
// output: 2026-09-20T00:00:00Z <nil>
```

Missing components default to zero, so a date-only parse gives you
midnight UTC. When the input does not match, the error names both sides,
which makes layout bugs easy to spot:

```go
_, err = time.Parse("2006-01-02", "20/09/2026")
fmt.Println(err)
// output: parsing time "20/09/2026" as "2006-01-02": cannot parse "20/09/2026" as "2006"
```

Use `time.ParseInLocation` when the input has no offset but you know
which zone it was written in — plain `Parse` assumes UTC.

## Durations are a type, not a number

`time.Duration` is an `int64` count of nanoseconds with a `String`
method, so it prints readably and does arithmetic directly:

```go
d := 90 * time.Minute
fmt.Println(d)                         // output: 1h30m0s
fmt.Println(d.Hours(), d.Minutes())    // output: 1.5 90

fmt.Println(2*time.Hour + 30*time.Minute)   // output: 2h30m0s
```

`90 * time.Minute` works because `90` is an untyped constant, as the
[type conversions](../02-language-basics/03-type-conversions.md) article
explains. A *variable* holding `90` would not compile — you need
`time.Duration(n) * time.Minute`:

```go
fmt.Println(time.Duration(1500) * time.Millisecond)   // output: 1.5s
```

Durations parse from strings too, which is how they arrive from
configuration:

```go
pd, _ := time.ParseDuration("1h30m")
fmt.Println(pd, pd == 90*time.Minute)   // output: 1h30m0s true

fmt.Println(d.Round(time.Hour), d.Truncate(time.Hour))
// output: 2h0m0s 1h0m0s
```

## Arithmetic

`Add` takes a duration; `Sub` returns one; `AddDate` works in calendar
units:

```go
later := t.Add(48 * time.Hour)
fmt.Println(later.Format(time.RFC3339))       // output: 2026-09-22T15:04:05Z
fmt.Println(later.Sub(t))                     // output: 48h0m0s

fmt.Println(t.AddDate(0, 1, 0).Format("2006-01-02"))   // output: 2026-10-20
```

`time.Since(start)` is shorthand for `time.Now().Sub(start)` and is the
usual way to measure how long something took.

## `Add` and `AddDate` differ across a daylight-saving boundary

This is the reason both exist. Adding "one day" and adding "24 hours"
are different questions, and on the night the clocks change they give
different answers:

```go
loc, _ := time.LoadLocation("America/New_York")
before := time.Date(2026, time.March, 7, 12, 0, 0, 0, loc)

fmt.Println(before.AddDate(0, 0, 1).Format(time.RFC3339))
// output: 2026-03-08T12:00:00-04:00

fmt.Println(before.Add(24 * time.Hour).Format(time.RFC3339))
// output: 2026-03-08T13:00:00-04:00
```

`AddDate` keeps the wall-clock time — still noon. `Add` advances the
actual elapsed time, and because that Sunday is an hour short, noon plus
24 hours lands at 1pm. Use `AddDate` for "tomorrow", `Add` for "in 24
hours".

## Time zones

A `time.Time` carries a location. `In` changes how it is *displayed*
without changing the instant:

```go
ny, _ := time.LoadLocation("America/New_York")

fmt.Println(t.In(ny).Format(time.RFC3339))   // output: 2026-09-20T11:04:05-04:00
fmt.Println(t.UTC().Format(time.RFC3339))    // output: 2026-09-20T15:04:05Z
```

`LoadLocation` reads the system zone database, so it can fail on a
minimal container image. `time.UTC` and `time.Local` always work.

## Compare with `Equal`, not `==`

A `time.Time` is a struct holding a wall clock, a monotonic reading and
a location pointer. `==` compares all of that, so the same instant in two
zones is not `==`:

```go
a := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
b := a.In(ny)

fmt.Println(a == b)        // output: false
fmt.Println(a.Equal(b))    // output: true
```

Always use `Equal`, `Before` and `After`. `==` on a `time.Time` compiles
and is almost always a bug — the same trap the
[structs](../02-language-basics/10-structs.md) article warns about for
comparable structs generally.

The zero value is year 1, and `IsZero` is how you spell "not set":

```go
var zero time.Time
fmt.Println(zero.IsZero(), zero.Format(time.RFC3339))
// output: true 0001-01-01T00:00:00Z
```

That is also why a "missing" timestamp is usually a `*time.Time`: the
zero value is a real date, not an absence.

## Monotonic readings

`time.Now()` also records a monotonic clock reading, used automatically
by `Sub`, `Since`, `Before` and `After`. That means measuring elapsed
time stays correct even if the system clock is adjusted underneath you.
The reading is dropped by `Round`, `Truncate`, `UTC` and `In` — so
measure first, convert afterwards.

> **From Python:** `Format`/`Parse` replace `strftime`/`strptime`, but
> the layout is an example rather than `%`-codes. `time.Duration` is
> `timedelta` with arithmetic that reads better. The habit to unlearn is
> `==` on timestamps — Python's `datetime` compares by instant, Go's
> struct equality does not.

## Quick reference

| Task | Call |
|---|---|
| now | `time.Now()` |
| build an instant | `time.Date(y, time.Month, d, h, m, s, ns, loc)` |
| format | `t.Format(time.RFC3339)` or `t.Format("2006-01-02")` |
| parse | `time.Parse(layout, s)`, `time.ParseInLocation` |
| elapsed | `time.Since(start)` |
| add hours | `t.Add(24 * time.Hour)` |
| add calendar days | `t.AddDate(0, 0, 1)` |
| difference | `b.Sub(a)` → `time.Duration` |
| compare | `Equal`, `Before`, `After` — **never `==`** |
| duration from config | `time.ParseDuration("1h30m")` |
| duration from a variable | `time.Duration(n) * time.Second` |
| change display zone | `t.In(loc)`, `t.UTC()` |
| "not set" | `t.IsZero()`, or a `*time.Time` field |

## Sources

- [`time` package reference — pkg.go.dev/time](https://pkg.go.dev/time)
- [`time.Time.Format` — pkg.go.dev/time#Time.Format](https://pkg.go.dev/time#Time.Format)
- [Layout constants — pkg.go.dev/time#pkg-constants](https://pkg.go.dev/time#pkg-constants)
- [`time.Duration` — pkg.go.dev/time#Duration](https://pkg.go.dev/time#Duration)
- [`time.Time.Equal` — pkg.go.dev/time#Time.Equal](https://pkg.go.dev/time#Time.Equal)
- [Monotonic clocks — pkg.go.dev/time#hdr-Monotonic_Clocks](https://pkg.go.dev/time#hdr-Monotonic_Clocks)
