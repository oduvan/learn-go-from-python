# Заплановані задачі

Запустити щось за розкладом — легко. Запустити це точно один раз, коли
три репліки вашого сервісу планують це одночасно — ось справжня
проблема, і стандартна бібліотека не має на неї відповіді.

> **Модуль:** `github.com/go-co-op/gocron/v2`, плюс advisory-блокування
> PostgreSQL для координації.

```go
s, err := gocron.NewScheduler()
s.NewJob(gocron.CronJob("0 3 * * *", false), gocron.NewTask(reindex, ctx))
s.Start()
```

## gocron

```go
s, err := gocron.NewScheduler()
if err != nil {
    return err
}
defer func() { _ = s.Shutdown() }()

j, err := s.NewJob(
    gocron.DurationJob(5*time.Minute),
    gocron.NewTask(pollUpstream, ctx),
    gocron.WithSingletonMode(gocron.LimitModeReschedule),
)

s.Start()
```

`NewJob` бере розклад, задачу й опції. Визначення задач:

| Визначення | Значення |
|---|---|
| `gocron.DurationJob(d)` | кожні `d` |
| `gocron.CronJob("0 3 * * *", false)` | cron-вираз; `true` для секунд |
| `gocron.DailyJob(1, atTimes)` | кожні *n* днів у задані моменти |
| `gocron.OneTimeJob(...)` | один раз, у вказаний час |

Некоректний вираз відхиляється в `NewJob`, а не при першому
спрацюванні — тож одруківка провалюється при старті:

```go
_, err := s2.NewJob(gocron.CronJob("not a cron", false), gocron.NewTask(func() {}))
// err != nil
```

`NewTask(fn, args...)` прив'язує аргументи — саме так ви передаєте
контекст. `Shutdown` чекає завершення запущених задач; викликайте
його, або задача в польоті буде вбита, коли процес завершиться.

### `WithSingletonMode` — в межах одного процесу

```go
gocron.WithSingletonMode(gocron.LimitModeReschedule)
```

Зупиняє перекриття задачі з самою собою, коли запуск триває довше за
інтервал. `LimitModeReschedule` пропускає пропущений тик;
`LimitModeWait` ставить його в чергу.

Це координує лише **всередині одного процесу**. Три репліки мають
власні планувальники й усі спрацюють.

## Точно один раз серед реплік

Загальні рішення — розподілене блокування чи вибори лідера. Якщо у вас
уже є PostgreSQL, його advisory-блокування дають вам одне безкоштовно —
без додаткової інфраструктури.

```go
func LockedRun(ctx context.Context, db *sql.DB, job string, fn func(context.Context) error) (bool, error) {
    conn, err := db.Conn(ctx)
    if err != nil {
        return false, err
    }
    defer conn.Close()

    key := JobLockKey(job)

    var acquired bool
    if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
        return false, err
    }
    if !acquired {
        return false, nil          // someone else has it
    }

    defer func() {
        unlockCtx := context.WithoutCancel(ctx)
        conn.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", key)
    }()

    return true, fn(ctx)
}
```

П'ять горутин, що змагаються за одну й ту саму задачу:

```
ran=1 skipped=4
```

І блокування знімається потім, тож наступний запланований запуск
працює:

```go
ok, err := LockedRun(ctx, db, "nightly-reindex", work)
// true <nil>
```

Три деталі несуть на собі все це, і кожна з них — баг, якщо ви її
пропустите.

**`db.Conn(ctx)`, а не `db`.** Advisory-блокування тримається *сесією*.
Взяти його на одному з'єднанні з пулу й звільнити на іншому залишить
його тримати доти, доки те з'єднання не помре — саме такий баг
вичерпання пулу виглядає так, ніби задача "просто перестала
виконуватись". Це те, про що каже
[`database/sql`](../09-database-sql/01-database-sql.md), резервуючи
з'єднання для стану сесії.

**`pg_try_advisory_lock`, не `pg_advisory_lock`.** Версія `try`
повертає false негайно. Блокувальна версія змушує інші репліки
шикуватися в чергу й виконувати задачу послідовно, що є протилежним
тому, чого ви хотіли.

**`context.WithoutCancel` на знятті блокування.** Якщо контекст
задачі був скасований — завершення, тайм-аут — зняття блокування все
одно має виконатись, інакше блокування залишиться триматись. Це
патерн відв'язки зі статті про
[контекст як носій](../11-architecture-and-conventions/02-context-as-a-carrier.md).

### Ключ має бути числом

Advisory-блокування ключуються `bigint`, тож хешуйте назву задачі.
FNV швидкий і, на відміну від хешування мап у Go, детермінований між
процесами:

```go
func JobLockKey(name string) int64 {
    h := fnv.New64a()
    h.Write([]byte(name))
    return int64(h.Sum64())
}
```

```go
JobLockKey("nightly-reindex") == JobLockKey("nightly-reindex")   // true
JobLockKey("a") != JobLockKey("b")                               // true
```

Детермінованість — весь сенс: кожна репліка мусить обчислити те саме
число для тієї самої назви задачі. Колізія між двома назвами задач
означала б, що одна блокує іншу, тож тримайте назви окремими й
нечисленними.

## Складаючи це разом

```go
s.NewJob(
    gocron.CronJob("0 3 * * *", false),
    gocron.NewTask(func(ctx context.Context) {
        ran, err := LockedRun(ctx, db, "nightly-reindex", reindex)
        switch {
        case err != nil:
            slog.Error("nightly-reindex", "error", err)
        case !ran:
            slog.Debug("nightly-reindex skipped, another replica holds the lock")
        }
    }, ctx),
)
```

Зауважте, "інша репліка виконала це" — `Debug`, не `Warn`. Це
очікуваний випадок на кожній репліці, крім однієї, а логування цього
гучніше змусить кожен деплой виглядати як інцидент.

## Практичні нотатки

- **Відновлюйтесь всередині задачі.** Паніка в запланованій горутині
  забирає весь процес, як пояснює стаття про
  [довгоживучі горутини](../05-concurrency/08-long-running-goroutines.md).
  gocron може вас не врятувати; обгортайте задачу.
- **Давайте кожній задачі тайм-аут.** `context.WithTimeout` навколо
  роботи, або застрягла задача триматиме своє блокування, доки процес
  не перезапуститься.
- **Часові пояси.** Cron-вирази виконуються в локації планувальника;
  встановлюйте її явно через `gocron.WithLocation` замість того, щоб
  успадковувати від контейнера, що зазвичай UTC, а іноді й ні.
- **Робіть задачі ідемпотентними.** Пропущений тик, повторна спроба
  після краху, чи дві репліки, що ненадовго розійшлись у думці щодо
  блокування, не мають нічого зіпсувати.
- **Виставляйте останній успіх.** Gauge з часовою міткою останнього
  запуску дозволяє сповіщати про задачу, що мовчки перестала
  виконуватись — набагато корисніше, ніж сповіщати про провали, бо
  небезпечний випадок — задача, яка взагалі ніколи не запускається.

## Коли використовувати щось інше

Цей патерн підходить для періодичного обслуговування всередині
сервісу: нічний реіндекс, чистка застарілих записів, звіт.

Для роботи, що має пережити перезапуск, повторити спробу з
backoff, або бути спостережуваною як окремі одиниці, вам потрібна
черга задач із довговічним сховищем, а не планувальник. І якщо
платформа вже надає такий — Kubernetes `CronJob`, хмарний планувальник —
окремий бінарник, запущений за розкладом, простіший за координацію
блокувань, бо проблема "точно один раз" стає чужою турботою.

> **З досвіду Python:** gocron — це APScheduler, а `LockedRun` —
> розподілене блокування, за яким ви б інакше пішли в Redis.
> Advisory-блокування Postgres — це еквівалент `SELECT ... FOR UPDATE`,
> використаного як м'ютекс — без потреби в рядку.

## Швидка довідка

| Задача | Форма |
|---|---|
| планувальник | `gocron.NewScheduler()`, `s.Start()`, `defer s.Shutdown()` |
| інтервал | `gocron.DurationJob(d)` |
| cron | `gocron.CronJob("0 3 * * *", false)` |
| передати контекст | `gocron.NewTask(fn, ctx)` |
| без самоперекриття | `WithSingletonMode(LimitModeReschedule)` — лише один процес |
| серед реплік | `pg_try_advisory_lock` на виділеному `db.Conn` |
| ключ | `fnv.New64a()` над назвою задачі |
| зняти | `defer` з `context.WithoutCancel` |
| пропущений запуск | логувати на рівні `Debug` — це нормальний випадок |
| сповіщення | gauge останнього успішного запуску |

## Джерела

- [gocron — pkg.go.dev/github.com/go-co-op/gocron/v2](https://pkg.go.dev/github.com/go-co-op/gocron/v2)
- [PostgreSQL advisory locks — postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS)
- [`sql.DB.Conn` — pkg.go.dev/database/sql#DB.Conn](https://pkg.go.dev/database/sql#DB.Conn)
- [`hash/fnv` — pkg.go.dev/hash/fnv](https://pkg.go.dev/hash/fnv)
- [`context.WithoutCancel` — pkg.go.dev/context#WithoutCancel](https://pkg.go.dev/context#WithoutCancel)
