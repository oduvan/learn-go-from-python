# Вчимо Go з досвідом Python

Особисті конспект-нотатки для Python-розробника, який починає вивчати
Go. Мета — пояснювати Go **його власними категоріями**, з короткими
порівняннями з Python там, де вони допомагають загострити різницю, а не
перетворити пояснення на переклад.

Усі матеріали орієнтовані на актуальний стабільний реліз Go — **Go 1.27.1**.

## Як організовано матеріал

Кожна тема має свою нумеровану теку. Усередині теми конспекти й
прикладні файли коду йдуть однією наскрізною нумерацією, тож порядок
читання та запуску завжди очевидний.

### [Екосистема та встановлення](01-ecosystem-and-installation/01-what-is-go.md)

- [Що таке Go](01-ecosystem-and-installation/01-what-is-go.md) — мова, спільнота, екосистема.
- [Команда `go`](01-ecosystem-and-installation/02-go-subcommands.md) — кожна підкоманда, яку вам справді доведеться використовувати.
- [`go tool trace`](01-ecosystem-and-installation/03-go-tool-trace.md) — трейсер виконання.
- [Типи файлів](01-ecosystem-and-installation/04-go-file-types.md) — `.go`, `_test.go`, `go.mod`, `go.sum`, build-обмеження.
- [Спеціальні каталоги](01-ecosystem-and-installation/05-special-folders.md) — `internal/`, `testdata/`, конвенції.
- [Декілька версій Go](01-ecosystem-and-installation/06-multiple-go-versions.md) — `GOTOOLCHAIN`, директиви `go` й `toolchain`.
- [Встановлення](01-ecosystem-and-installation/07-installation.md) — macOS, Linux, Windows.
- [Додаткові інструменти](01-ecosystem-and-installation/08-additional-tools.md) — `gopls`, `dlv`, `golangci-lint` та інші.
- [Демонстраційний проєкт](01-ecosystem-and-installation/09-demo-project/README.md) — невеликий запускний модуль, який ілюструє все вищесказане.

### [Основи мови](02-language-basics/01-variables-and-constants.md)

- [Змінні та константи](02-language-basics/01-variables-and-constants.md) — `var`, `:=`, `const` та `iota`.
- [Базові типи](02-language-basics/02-basic-types.md) — цілі, дробові, рядки, булеві; без «правдивості».
- [Перетворення типів](02-language-basics/03-type-conversions.md) — явне `T(x)`, `strconv`, без неявного приведення.
- [Оператори](02-language-basics/04-operators.md) — арифметика, переповнення, цілочислове ділення; без тернарного.
- [Керування потоком](02-language-basics/05-control-flow.md) — `if`, `for` (єдиний цикл), `switch`.
- [Функції](02-language-basics/06-functions.md) — кілька значень, що повертаються, іменовані результати, варіативність, функції як значення.
- [Помилки](02-language-basics/07-errors.md) — значення `error`, загортання через `%w`, `errors.Is`/`As`.
- [Вказівники](02-language-basics/08-pointers.md) — `&`/`*`, `nil`, `new`, без арифметики вказівників.
- [Власні типи](02-language-basics/09-custom-types.md) — визначення `type` проти псевдонімів, базові типи.
- [Структури](02-language-basics/10-structs.md) — поля, літерали, нульове значення, вбудовування, теги.
- [Масиви та зрізи](02-language-basics/11-arrays-and-slices.md) — len/cap, `append` і пастка спільного масиву-основи.
- [Map (асоціативні масиви)](02-language-basics/12-maps.md) — пошук за ключем, comma-ok, пастка nil-map, множини.
- [Вибір структури даних](02-language-basics/13-choosing-a-data-structure.md) — зріз проти map проти структури проти власного типу.
- [Defer](02-language-basics/14-defer.md) — відкладені виклики, порядок LIFO, патерни прибирання.
- [Panic та recover](02-language-basics/15-panic-and-recover.md) — коли панікувати, відновлення у відкладених викликах.
- [Імпорти](02-language-basics/16-imports.md) — шляхи імпорту, псевдоніми, порожній та крапковий імпорт.

### [Об'єктноорієнтований Go](03-object-oriented-go/01-methods.md)

- [Методи](03-object-oriented-go/01-methods.md) — отримувачі за значенням і за вказівником, набори методів, підвищення.
- [Інтерфейси](03-object-oriented-go/02-interfaces.md) — неявне задоволення, поліморфізм, порожній інтерфейс / `any`.
- [Твердження типу та перемикачі типів](03-object-oriented-go/03-type-assertions-and-type-switches.md) — повернення конкретного типу під час виконання.
- [Generics](03-object-oriented-go/04-generics.md) — параметри типу та обмеження.
- [Патерни ООП](03-object-oriented-go/05-oop-patterns.md) — інкапсуляція, композиція замість успадкування, поліморфізм.
- [Власні типи помилок](03-object-oriented-go/06-custom-error-types.md) — власні типи `error`, `Unwrap`, `errors.As`, власний `Is`.

### [Пакети та модулі](04-packages-and-modules/01-packages-and-visibility.md)

- [Пакети та видимість](04-packages-and-modules/01-packages-and-visibility.md) — правила пакетів, експортоване проти неекспортованого, `init`.
- [Створення та публікація модуля](04-packages-and-modules/02-creating-and-publishing-a-module.md) — `go.mod`, версіонування, `replace`, публікація.
- [Структура проєкту та робочі простори](04-packages-and-modules/03-project-layout-and-workspaces.md) — `internal/`, `cmd/`, `go.work`.

### [Конкурентність](05-concurrency/01-goroutines.md)

- [Горутини](05-concurrency/01-goroutines.md) — `go`, планування, `WaitGroup`, пастка завершення main.
- [Канали](05-concurrency/02-channels.md) — надсилання/отримання, буферизація, `close`, `range`, взаємні блокування.
- [select](05-concurrency/03-select.md) — мультиплексування, `default`, таймаути, канали done.
- [Синхронізація](05-concurrency/04-synchronization.md) — `Mutex`, `Once`, атоміки, детектор гонитв.
- [Context](05-concurrency/05-context.md) — скасування, дедлайни, поширення.
- [Патерни конкурентності](05-concurrency/06-concurrency-patterns.md) — пули робітників, fan-out/fan-in, конвеєри.
- [Обмежена конкурентність](05-concurrency/07-bounded-concurrency.md) — семафори на каналах, збирання результатів, `sync.Map`.
- [Довгоживучі горутини](05-concurrency/08-long-running-goroutines.md) — `recover` у кожній горутині, тікери, дренаж під час зупинки.

### [Текст, час і дані](06-text-time-and-data/01-strings-bytes-and-runes.md)

- [Рядки, байти та руни](06-text-time-and-data/01-strings-bytes-and-runes.md) — `strings`, `bytes` і чому `len` рахує байти.
- [Форматування через `fmt`](06-text-time-and-data/02-formatting-with-fmt.md) — дієслова, ширина й точність, `Stringer`, `%w`.
- [Регулярні вирази](06-text-time-and-data/03-regular-expressions.md) — `regexp`, іменовані групи і чого немає в RE2.
- [Час](06-text-time-and-data/04-time.md) — еталонний макет, тривалості, часові пояси, `Equal` замість `==`.
- [Сортування](06-text-time-and-data/05-sorting.md) — `slices.SortFunc`, `cmp.Compare`, `cmp.Or`, стабільність.
- [Ітератори](06-text-time-and-data/06-iterators.md) — написання `iter.Seq`, контракт `yield`, `iter.Pull`.
- [Кодування JSON](06-text-time-and-data/07-encoding-json.md) — теги, `omitempty`, `RawMessage`, власна серіалізація.
- [XML, CSV та рефлексія](06-text-time-and-data/08-xml-csv-and-reflection.md) — потік токенів, `csv`, теги структур під час виконання.

### [Операційна система](07-operating-system/01-files-and-paths.md)

- [Файли та шляхи](07-operating-system/01-files-and-paths.md) — `os`, `filepath`, `WalkDir`, перевірка помилок замість шляхів.
- [Читачі та письменники](07-operating-system/02-readers-and-writers.md) — `io.Copy`, `bufio.Scanner` і його ліміт у 64 КБ.
- [`go:embed`](07-operating-system/03-go-embed.md) — файли всередині бінарника, `embed.FS`, `all:`, `fs.Sub`.
- [Прапорці та середовище](07-operating-system/04-flags-and-environment.md) — `flag`, підкоманди, `LookupEnv`, коди виходу.
- [Запуск зовнішніх команд](07-operating-system/05-running-external-commands.md) — `exec.CommandContext`, `ExitError`, без shell.
- [Сигнали та коректна зупинка](07-operating-system/06-signals-and-graceful-shutdown.md) — `NotifyContext`, дренаж у межах бюджету.
- [Хешування та випадкові значення](07-operating-system/07-hashing-and-random-values.md) — sha256, HMAC, `crypto/rand`, base64, gzip.

### [HTTP через `net/http`](08-http-with-net-http/01-http-server.md)

- [HTTP-сервер](08-http-with-net-http/01-http-server.md) — обробники, маршрутизація `ServeMux`, таймаути, зупинка.
- [HTTP-клієнт](08-http-with-net-http/02-http-client.md) — чому 404 не є помилкою, закриття тіла, повтори.
- [Проміжні обробники](08-http-with-net-http/03-middleware.md) — обгортання обробників, відновлення, значення в context.
- [Шаблони](08-http-with-net-http/04-templates.md) — `text/template` проти `html/template` та контекстне екранування.
- [Події від сервера (SSE)](08-http-with-net-http/05-server-sent-events.md) — стрімінг, flush, відкидання повільних клієнтів.

### [Бази даних через `database/sql`](09-database-sql/01-database-sql.md)

- [`database/sql`](09-database-sql/01-database-sql.md) — пул, `Scan`, `ErrNoRows`, NULL, `rows.Err()`.
- [Власні типи колонок](09-database-sql/02-custom-column-types.md) — `driver.Valuer` та `sql.Scanner`.
- [Транзакції](09-database-sql/03-transactions.md) — обгортка на замиканні, відкат при паніці, вкладеність.
- [Патерн репозиторію](09-database-sql/04-the-repository-pattern.md) — пакет контрактів, переклад помилок сховища.

### [Тестування](10-testing/01-the-testing-package.md)

- [Пакет `testing`](10-testing/01-the-testing-package.md) — `TestXxx`, `Errorf` проти `Fatalf`, прапорці `go test`.
- [Табличні тести](10-testing/02-table-driven-tests.md) — зріз випадків, `t.Run`, `t.Parallel`.
- [Хелпери, фікстури та golden-файли](10-testing/03-helpers-fixtures-and-golden-files.md) — `t.Helper`, `t.TempDir`, `testdata/`.
- [Фейки та заглушки](10-testing/04-fakes-and-stubs.md) — заглушки на функційних полях, підміна годинника.
- [Тестування HTTP](10-testing/05-testing-http.md) — рекордери та сервери `httptest`.
- [Бенчмарки, фазинг і детектор гонитв](10-testing/06-benchmarks-fuzzing-and-race.md) — `b.Loop`, `f.Fuzz`, `-race`.

### [Архітектура та домовленості](11-architecture-and-conventions/01-wiring-and-package-structure.md)

- [Зв'язування та структура пакетів](11-architecture-and-conventions/01-wiring-and-package-structure.md) — `cmd/`, `internal/`, інʼєкція через конструктор.
- [Context як носій значень](11-architecture-and-conventions/02-context-as-a-carrier.md) — неекспортовані ключі, `WithoutCancel`, чого туди не класти.
- [Патерни конфігурації](11-architecture-and-conventions/03-configuration-patterns.md) — одна структура, значення за замовчуванням, валідація через `errors.Join`.
- [Напрямок залежностей](11-architecture-and-conventions/04-dependency-direction.md) — який пакет що може імпортувати і чому.
- [Збірка, кодогенерація та cgo](11-architecture-and-conventions/05-build-codegen-and-cgo.md) — теги збірки, `-ldflags`, крос-компіляція, ціна cgo.
- [Домовленості проєкту](11-architecture-and-conventions/06-project-conventions.md) — обгортання помилок, рівні логування, іменування, коментарі.

### [Спостережуваність](12-observability/01-structured-logging-with-slog.md)

- [Структуроване логування через `slog`](12-observability/01-structured-logging-with-slog.md) — хендлери, `With`, `LogValuer`, тестування логів.
- [Профілювання через pprof](12-observability/02-profiling-with-pprof.md) — профілі CPU та heap, flat проти cum, flame-графи.

## Джерела

- Репозиторій з вихідним кодом: <https://github.com/oduvan/learn-go-from-python>.
- Наприкінці кожного конспекту перелічені офіційні джерела, з якими
  було звірено матеріал — як правило, [go.dev](https://go.dev/),
  [pkg.go.dev](https://pkg.go.dev/) або
  [специфікація Go](https://go.dev/ref/spec).
