# Додаткові інструменти Go

Команда `go` охоплює збирання, тестування, форматування, vet та керування залежностями. Крім цього, спільнота Go (і сама команда Go) підтримує невеликий набір інструментів, з якими ви майже одразу зустрінетеся в реальних проєктах. У цій статті перераховано ті, що варто знати.

Усі вони встановлюються однаково:

```bash
go install <import-path>@latest
```

Бінарний файл потрапляє у `$GOBIN` (типово `~/go/bin`), який має бути у вашому `$PATH`.

## Підтримка редактора / IDE

### `gopls` — офіційний мовний сервер

Офіційна реалізація Language Server Protocol для Go, яку підтримує команда Go. Забезпечує автодоповнення, перехід до визначення, пошук посилань, вбудовану діагностику, рефакторинг і перейменування — для **будь-якого** LSP-сумісного редактора.

```bash
go install golang.org/x/tools/gopls@latest
```

Редактори, що використовують його «з коробки»: VS Code (розширення Go), Neovim (через вбудований LSP), JetBrains GoLand (використовує власний рушій, не gopls), Sublime, Emacs, Helix.

> **З досвіду Python:** ≈ `pyright` або `python-lsp-server` — але `gopls` є *офіційним*, підтримується поруч з компілятором, тому фрагментації немає.

## Відладчик

### `dlv` (Delve)

Відладчик Go. Встановлює точки зупину, виконує покроковий прохід, інспектує змінні та горутини (goroutine), приєднується до запущених процесів, відлагоджує дампи пам'яті.

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

Поширені виклики:

```bash
dlv debug ./cmd/server          # build with debug info and start
dlv test ./mypkg                # debug tests
dlv attach <pid>                # attach to running process
dlv exec ./mybinary             # debug an existing binary
```

Розширення Go для VS Code та GoLand обидва використовують Delve «під капотом» — зазвичай ви ніколи не викликаєте `dlv` напряму, щойно ваш редактор налаштовано.

> **З досвіду Python:** ≈ `pdb` або `debugpy` — але з підтримкою скомпільованих бінарних файлів та вбудованою інспекцією горутин.

## Лінтери

### `golangci-lint` — мета-лінтер

Запускає **десятки лінтерів паралельно** з єдиним конфігураційним файлом (`.golangci.yml`), кешуванням та інтеграціями для IDE і CI. Де-факто стандарт для Go CI-конвеєрів.

```bash
# preferred install method per the project: dedicated installer, not `go install`
brew install golangci-lint
# or
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin latest
```

Запуск:

```bash
golangci-lint run ./...
```

Лінтери, які він агрегує: `govet`, `staticcheck`, `errcheck`, `ineffassign`, `unused`, `gosimple`, `revive`, `gosec` та багато інших. Більшість команд вмикають налаштований підмножинний набір, а не всі 100+.

> **З досвіду Python:** ≈ `ruff` — один швидкий агрегатор, що замінює стек окремих лінтерів. `golangci-lint` з'явився на кілька років раніше за `ruff`.

### `staticcheck` — поглиблений статичний аналіз

Якщо `go vet` — це консервативний вбудований інструмент, то `staticcheck` — глибший: знаходить мертвий код, підозрілі патерни, неефективні присвоєння, пастки продуктивності. Входить як один аналізатор у `golangci-lint`, але також можна запускати окремо.

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

> **З досвіду Python:** ≈ `pylint` (без крутого порогу налаштування). Досить консервативний, щоб використовувати без великих налаштувань.

## Форматери поверх `gofmt`

### `goimports` — `gofmt` плюс керування імпортами

Замінник `gofmt`, що **також автоматично додає відсутні імпорти та видаляє невикористані**.

```bash
go install golang.org/x/tools/cmd/goimports@latest
goimports -w .
```

Налаштуйте редактор на запуск `goimports` при збереженні (більшість роблять це типово для Go).

> **З досвіду Python:** ≈ `isort` + `black` в одному інструменті.

### `gofumpt` — суворіший `gofmt`

Надмножина `gofmt`, що застосовує додаткові правила стилю, яких `gofmt` не вимагає (наприклад, жодних порожніх рядків на початку/кінці блоків, послідовне групування). Категоричний і незмінний — у тому ж дусі, що й сам `gofmt`.

```bash
go install mvdan.cc/gofumpt@latest
gofumpt -w .
```

Більшість проєктів використовують або `gofmt` (типово), або `gofumpt` (невелика, але зростаюча меншість). Головне — послідовність у межах одного коду.

## Безпека

### `govulncheck` — офіційний сканер вразливостей

Підтримується командою безпеки Go. Сканує ваш код (або скомпільований бінарний файл) на **відомі CVE у залежностях, які ви фактично викликаєте** — не просто залежностях, що ви підтягнули. Ця обізнаність про граф викликів різко зменшує кількість хибних спрацьовувань.

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

Працює на базі бази даних вразливостей Go за адресою [vuln.go.dev](https://vuln.go.dev), переглядати можна на [pkg.go.dev/vuln](https://pkg.go.dev/vuln). Інтегровано у розширення Go для VS Code та доступно як GitHub Action.

> **З досвіду Python:** ≈ `pip-audit` або `safety` — але `govulncheck` розуміє, до яких вразливих функцій ваш код фактично досягає.

## Генератори коду

Go заохочує генерацію коду для повторюваних патернів (enum-подібні типи, моки, згенерований код для БД). Генератори викликаються через директиви `//go:generate` або напряму.

### `stringer` — генерує методи `String()` для enum-подібних типів

За наявності блоку `const` з типізованими цілочисельними значеннями генерує метод `String()`. (Синтаксис Go у прикладі нижче — визначення власного типу `Pill` та використання `iota` для автоматичної нумерації констант — розглядається у темі «основи мови»; тут показано лише як виглядає код, що `stringer` споживає.)

```bash
go install golang.org/x/tools/cmd/stringer@latest
```

```go
//go:generate stringer -type=Pill
type Pill int
const (
    Placebo Pill = iota
    Aspirin
    Ibuprofen
)
```

Потім `go generate ./...` регенерує файл `_string.go`.

### `mockgen` — генерує моки для інтерфейсів

З [форку Uber `gomock`](https://github.com/uber-go/mock), що є тепер канонічною версією.

```bash
go install go.uber.org/mock/mockgen@latest
mockgen -source=foo.go -destination=mocks/foo_mock.go
```

> **З досвіду Python:** еквівалент `unittest.mock` — але статична типізація Go означає, що моки *генеруються* за сигнатурою інтерфейсу, а не збираються динамічно.

### `sqlc` — SQL у типізований Go-код

Пишіть SQL-запити у файлах `.sql`; `sqlc` генерує повністю типізовані Go-функції для їх виклику. Уникає як ORM, так і SQL з форматованими рядками.

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
sqlc generate
```

### `protoc-gen-go` — генератор коду для protobuf

Для робочих процесів gRPC / Protobuf.

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Використовується `protoc` (компілятором protobuf) або, що частіше сьогодні, [`buf`](https://buf.build/).

## Профілювання та продуктивність

### `go tool pprof` — вбудований профілювальник

Вже постачається з toolchain. Зчитує профілі CPU, пам'яті, блокувань, м'ютексів та горутин. Режим UI:

```bash
go tool pprof -http=:8080 cpu.prof
```

Профілі надходять з `runtime/pprof` (у коді), `go test -cpuprofile=cpu.prof` або кінцевої точки сервера `net/http/pprof`.

### `benchstat` — порівняння результатів бенчмарків

```bash
go install golang.org/x/perf/cmd/benchstat@latest

go test -bench=. -count=10 > old.txt
# ... make changes ...
go test -bench=. -count=10 > new.txt
benchstat old.txt new.txt
```

Виводить статистично значущі дельти — необхідно для будь-якого питання «чи це насправді швидше?».

## Робочий процес розробки

### `air` — живе перезавантаження під час розробки

Стежить за деревом вихідних файлів, перезбирає і перезапускає при змінах.

```bash
go install github.com/air-verse/air@latest
air                                  # uses .air.toml in the current dir
```

> **З досвіду Python:** ≈ `watchmedo auto-restart` або те, що роблять dev-сервери Django/Flask.

### `mage` — автоматизація збирання на Go

Альтернатива `Make`, де кроки збирання написані на Go.

```bash
go install github.com/magefile/mage@latest
```

Ви пишете `magefile.go` з експортованими функціями; `mage <target>` їх запускає. Корисно, коли shell-скрипти стають надто громіздкими.

> **З досвіду Python:** ≈ `invoke` або `nox` — автоматизація задач мовою хоста.

## Варто знати про існування

- **`yaegi`** — інтерпретатор Go з підтримкою REPL. Обмежений (без cgo, часткова stdlib), але корисний для дослідження. `go install github.com/traefik/yaegi/cmd/yaegi@latest`.
- **`gore`** — ще один REPL для Go, у схожому дусі.
- **`go-callvis`** — візуалізує граф викликів вашої програми у вигляді діаграми.
- **`gopium`** — аналізує та оптимізує розкладку полів структури для ущільнення пам'яті.

Ці інструменти корисно знати, але вони не є щоденними.

## Рекомендований стартовий набір

Для типового дня роботи з Go вам потрібно:

| Інструмент | Навіщо |
|---|---|
| `gopls` | Функції IDE у будь-якому редакторі. |
| `dlv` | Відладка, коли операторів print недостатньо. |
| `goimports` | Форматування при збереженні з автоматичним виправленням імпортів. |
| `golangci-lint` | Одна команда, що відловлює широкий спектр проблем. |
| `govulncheck` | Періодичне сканування безпеки. |

Додавайте `mockgen`, `stringer`, `sqlc` або `protoc-gen-go` залежно від потреб проєкту.

## Джерела

- [`gopls` — pkg.go.dev/golang.org/x/tools/gopls](https://pkg.go.dev/golang.org/x/tools/gopls) та [go.dev/gopls](https://go.dev/gopls)
- [Delve — github.com/go-delve/delve](https://github.com/go-delve/delve)
- [`golangci-lint` — golangci-lint.run](https://golangci-lint.run/)
- [`staticcheck` — staticcheck.dev](https://staticcheck.dev/)
- [`goimports` — pkg.go.dev/golang.org/x/tools/cmd/goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)
- [`gofumpt` — github.com/mvdan/gofumpt](https://github.com/mvdan/gofumpt)
- [`govulncheck` — go.dev/blog/govulncheck](https://go.dev/blog/govulncheck) та [pkg.go.dev/vuln](https://pkg.go.dev/vuln)
- [`stringer` — pkg.go.dev/golang.org/x/tools/cmd/stringer](https://pkg.go.dev/golang.org/x/tools/cmd/stringer)
- [`mockgen` — github.com/uber-go/mock](https://github.com/uber-go/mock)
- [`sqlc` — sqlc.dev](https://sqlc.dev/)
- [`protoc-gen-go` — pkg.go.dev/google.golang.org/protobuf](https://pkg.go.dev/google.golang.org/protobuf)
- [`benchstat` — pkg.go.dev/golang.org/x/perf/cmd/benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [`pprof` — github.com/google/pprof](https://github.com/google/pprof)
- [`air` — github.com/air-verse/air](https://github.com/air-verse/air)
- [`mage` — magefile.org](https://magefile.org/)
