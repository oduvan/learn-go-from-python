# testcontainers

Деякі речі чесно не підробиш. Запит з віконною функцією, обмеження
унікальності, рівень ізоляції транзакцій — єдиний спосіб дізнатись, чи
вони працюють — це запустити їх проти справжньої бази даних.
testcontainers запускає таку прямо з вашого тесту.

> **Модулі:** `github.com/testcontainers/testcontainers-go` та його
> модулі для окремих технологій.

Стаття про [фейки та заглушки](../10-testing/04-fakes-and-stubs.md)
доводила, що тести самого сховища не можуть використовувати
фейкове сховище. Ось що вони використовують натомість.

```go
c, err := postgres.Run(ctx, "postgres:16-alpine",
    postgres.WithDatabase("app"),
    postgres.WithUsername("u"),
    postgres.WithPassword("p"),
)
dsn, err := c.ConnectionString(ctx, "sslmode=disable")
```

## Дочекайтесь готовності

Запущений контейнер — це ще не готова база даних. Без стратегії
очікування ви підключаєтесь до порту, що відкритий, але ще не приймає
запити, і отримуєте тест, що провалюється один раз із двадцяти:

```go
testcontainers.WithWaitStrategy(
    wait.ForLog("database system is ready to accept connections").
        WithOccurrence(2).
        WithStartupTimeout(60*time.Second),
)
```

`WithOccurrence(2)` — не забобон. Образ Postgres запускає сервер один
раз, щоб виконати скрипти ініціалізації, вимикає його й запускає
знову — тож перше входження цього рядка належить одноразовому
екземпляру.

Інші стратегії: `wait.ForListeningPort`, `wait.ForHTTP("/health")`,
`wait.ForSQL`. Надавайте перевагу тій, що доводить придатність
сервісу до використання, а не тій, що доводить лише відкритий порт.

## Запускайте один раз на весь тестовий бінарник

Запуск контейнера домінує над усім іншим. Робіть це один раз, за
`sync.Once`:

```go
var (
    once     sync.Once
    baseDSN  string
    setupErr error
)

func sharedPostgres(t *testing.T) string {
    t.Helper()
    once.Do(func() {
        c, err := postgres.Run(ctx, "postgres:16-alpine", /* ... */)
        if err != nil {
            setupErr = err
            return
        }
        baseDSN, setupErr = c.ConnectionString(ctx, "sslmode=disable")
    })
    if setupErr != nil {
        t.Fatalf("starting postgres: %v", setupErr)
    }
    return baseDSN
}
```

Різниця разюча:

```
--- PASS: TestInsertAndRead (1.52s)            ← started the container
--- PASS: TestSecondUsesSameContainer (0.01s)  ← reused it
```

Зауважте, помилка захоплюється в змінну, а не провалюється всередині
`once.Do`. Виклик `t.Fatalf` там перервав би перший тест, що тримає
блокування, і залишив би `once` позначеним завершеним, тож кожен
подальший тест мовчки отримав би порожній DSN.

## Одна точка входу для тестів

Обгорніть це так, щоб тести казали один рядок:

```go
func SetupTestDB(t *testing.T) *sql.DB {
    t.Helper()

    dsn := sharedPostgres(t)
    db, err := sql.Open("pgx", dsn)
    require.NoError(t, err)
    t.Cleanup(func() { db.Close() })

    // fresh schema for this test
    _, err = db.Exec(`DROP TABLE IF EXISTS users;
                      CREATE TABLE users (id bigserial PRIMARY KEY, name text NOT NULL)`)
    require.NoError(t, err)

    return db
}
```

```go
func TestInsertAndRead(t *testing.T) {
    db := SetupTestDB(t)
    // ...
}
```

Ізоляція тестів один від одного важливіша за ізоляцію від контейнера.
Варіанти, від найдешевшого: очистити (truncate) таблиці, яких торкнувся
тест; запускати кожен тест у транзакції, що завжди відкочується; або
клонувати базу даних з мігрованого шаблону.

Клон із шаблону — той варіант, що масштабується. Мігруйте один раз у
шаблонну базу даних, потім на кожен тестовий бінарник:

```sql
CREATE DATABASE test_7 TEMPLATE app_template;
```

Postgres копіює файли, тож це набагато швидше, ніж повторний запуск
міграцій, і кожен бінарник отримує повну ізоляцію, включно з DDL. Між
тестами всередині одного бінарника зазвичай достатньо очищення
торкнутих таблиць.

### Дайте CI перевикористати сервісний контейнер

CI часто вже надає базу даних, і запуск ще однієї всередині нього
повільний чи неможливий. Спершу перевірте наявність зовнішньо наданого
DSN і повертайтесь до testcontainers лише як до запасного варіанту:

```go
func testDSN(t *testing.T) string {
    if dsn := os.Getenv("TEST_DATABASE_DSN"); dsn != "" {
        return dsn                      // CI service container
    }
    return sharedPostgres(t)            // local Docker
}
```

Один хелпер, два середовища, без build-обмежень. Це часто й є різниця
між тестами бази даних, що запускаються в CI, і тими, що тихо цього не
роблять.

Запускайте свої справжні [міграції](09-goose-migrations.md) проти
контейнера, а не пишіть схему вручну — тоді тести також доводять, що
міграції працюють.

## Прибирання

`testcontainers.CleanupContainer(t, c)` прив'язує тривалість життя
контейнера до тесту. Для спільного контейнера сайдкар-"жнець" Ryuk
прибирає його, коли тестовий процес завершується, включно з ситуацією
після паніки чи `SIGKILL`.

Ryuk потребує доступу до сокета Docker. Під нестандартним рантаймом —
Colima, Podman, віддалений хост — це зазвичай означає встановлення
`TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE` чи `DOCKER_HOST`, інакше
контейнери протікають. Варто прописати це у вашому Makefile, а не
змушувати кожного розробника відкривати це заново.

## Вмикайте під час виконання, ніколи через build-обмеження

Ці тести потребують Docker, тож деякі середовища не можуть їх
запускати. Спокуса — `//go:build integration`. Опирайтеся їй: якщо
ніхто не передає `-tags integration`, тести не запускаються *ніде*, і
при цьому виглядають присутніми в репозиторії весь час.

Пропускайте під час виконання, де пропуск видно у виводі:

```go
if os.Getenv("SKIP_DB_TESTS") != "" {
    t.Skip("SKIP_DB_TESTS set")
}
```

Ще краще — інвертуйте це в CI: зробіть відсутність Docker там провалом,
щоб неправильно налаштований пайплайн був гучним, а не тихо зеленим.
Це той самий аргумент, що й у статті про
[збірку та кодогенерацію](../11-architecture-and-conventions/05-build-codegen-and-cgo.md).

## За межами Postgres

Існують модулі для Redis, Kafka, RabbitMQ, MongoDB, LocalStack і
багатьох інших — усі тієї самої форми. `testcontainers.GenericContainer`
запускає будь-який образ, тож у всього, що має контейнер, є тестовий
дублікат.

## У що це обходиться

- **Секунди, не мілісекунди.** Тримайте ці тести окремо від швидких
  юніт-тестів, щоб `go test ./...` на пакеті лишався швидким.
- **Docker має бути доступний**, включно з CI.
- **Образи треба завантажити**, що повільно вперше й потребує мережі.
  Фіксуйте теги — `postgres:16-alpine`, ніколи `latest` — щоб версія
  не змінилась під вами.
- **Паралелізм потребує обдумування.** Тести, що ділять один контейнер,
  не ізольовані за замовчуванням — саме для цього й потрібне скидання
  на кожен тест.

Використовуйте це для шару сховища й усього, що справді має форму
інтеграційного тесту. Бізнес-логіку все ще варто тестувати проти
інтерфейсів, за мілісекунди.

> **З досвіду Python:** це `pytest-docker` чи `testcontainers-python`,
> де життєвий цикл контейнера керується з тестового бінарника, а не з
> фікстури. `sync.Once` відіграє роль фікстури з областю дії сесії.

## Швидка довідка

| Задача | Форма |
|---|---|
| запустити Postgres | `postgres.Run(ctx, "postgres:16-alpine", opts...)` |
| чекати придатності | `wait.ForLog(...).WithOccurrence(2)` |
| рядок підключення | `c.ConnectionString(ctx, "sslmode=disable")` |
| запустити один раз | `sync.Once`, захоплюючи помилку в змінну |
| точка входу на тест | хелпер `SetupTestDB(t)` з `t.Cleanup` |
| ізоляція | truncate, відкат, або шаблонна база даних |
| схема | запускайте справжні міграції |
| завершення | `CleanupContainer`, або Ryuk для спільного |
| нестандартний Docker | `TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE` |
| пропуск | перевірка змінної середовища під час виконання — **ніколи build-обмеження** |
| образи | зафіксовані теги, ніколи `latest` |

## Джерела

- [testcontainers-go — golang.testcontainers.org](https://golang.testcontainers.org/)
- [Postgres module — golang.testcontainers.org/modules/postgres/](https://golang.testcontainers.org/modules/postgres/)
- [Wait strategies — golang.testcontainers.org/features/wait/introduction/](https://golang.testcontainers.org/features/wait/introduction/)
- [`testcontainers-go` — pkg.go.dev/github.com/testcontainers/testcontainers-go](https://pkg.go.dev/github.com/testcontainers/testcontainers-go)
