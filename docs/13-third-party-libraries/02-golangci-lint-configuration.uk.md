# Налаштування golangci-lint

Стаття про [напрямок залежностей](../11-architecture-and-conventions/04-dependency-direction.md)
доводила, що те, який пакет може імпортувати який — це архітектурне
рішення, варте захисту. Ось як перестати захищати його вручну.

> **Модуль:** `github.com/golangci/golangci-lint` — інструмент, не
> імпорт. Встановлюєте бінарник; нічого не потрапляє у ваш `go.mod`.

```bash
golangci-lint run ./...
```

## Що це таке

Раннер, що виконує багато лінтерів над однією розпарсеною програмою,
тож ви платите за перевірку типів лише один раз. `go vet` включений;
також `staticcheck`, `errcheck`, `ineffassign` і `unused`.

Зафіксуйте версію в CI. Лінтери отримують нові перевірки між релізами,
і незафіксований інструмент означає, що збірка, яка пройшла вчора,
провалюється сьогодні без жодної зміни коду.

## Файл конфігурації

`.golangci.yml` у корені репозиторію. Версія 2 формату:

```yaml
version: "2"

linters:
  enable:
    - depguard
    - forbidigo
```

Стандартний набір — `errcheck`, `govet`, `ineffassign`, `staticcheck`,
`unused` — вже увімкнений. Додавання записів у `enable` розширює його.

Утримайтеся від увімкнення всього поспіль. Лінтер, чиї знахідки ви
регулярно придушуєте, гірший за той, який ви взагалі не вмикали, бо він
привчає людей додавати `//nolint` без читання.

## `depguard`: який пакет може що імпортувати

Це лінтер, що робить архітектурне правило реальним. Правило нижче каже,
що лише шар зберігання може торкатися `database/sql`:

```yaml
linters:
  settings:
    depguard:
      rules:
        upper-layers:
          files:
            - "$all"
            - "!$test"
            - "!**/internal/storage/**"
          deny:
            - pkg: database/sql
              desc: "only internal/storage may hold a database handle"
```

Обробник, що імпортує його тепер, провалює збірку з причиною, доданою
до повідомлення:

```
internal/web/handler.go:5:2: import 'database/sql' is not allowed from
list 'upper-layers': only internal/storage may hold a database handle (depguard)
```

### Область через виключення, а не включення

Список `files` — це несуча частина. `"$all"` мінус пакети, яким
*дозволено* означає, що **новий пакет охоплений з дня, коли хтось його
створить**. Перелічіть натомість дозволені пакети — і кожен новий
залишиться незахищеним, доки хтось не згадає його додати, а саме тоді
правило й перестає працювати.

`$all` і `$test` — вбудовані селектори: `$all` збігається з кожним
Go-файлом, а `$test` — лише з файлами `_test.go`. Провідний `!`
виключає, тож `"!$test"` не включає тести.

Завжди пишіть `desc`. Він стає повідомленням про помилку, і правило, що
пояснює само себе, дотримуються, а не обходять.

Той самий підхід захищає пакет контрактів:

```yaml
services:
  files: ["**/internal/services/**"]
  deny:
    - pkg: gorm.io/gorm
      desc: "internal/services declares store interfaces; it does not implement them"
```

## `forbidigo`: заборона ідентифікатора

Там, де `depguard` забороняє імпорт, `forbidigo` забороняє ім'я:

```yaml
forbidigo:
  analyze-types: true
  forbid:
    - pattern: '^fmt\.Print.*$'
      msg: "use log/slog, not fmt.Print"
```

```
internal/web/extra.go:5:16: use of `fmt.Println` forbidden because
"use log/slog, not fmt.Print" (forbidigo)
```

**`analyze-types: true` не опціональний**, коли ви зіставляєте
кваліфіковані імена. Без нього зіставлення текстове: воно бачить
ідентифікатор так, як він написаний, тож псевдонім імпорту
проскакує, а локальна змінна зі збіжним ім'ям дає хибне спрацювання.
Увімкнення робить зіставлення чутливим до типів.

Так ви дозволяєте бібліотечний сентинел помилки, забороняючи водночас
її типи — дозволяючи `gorm.ErrRecordNotFound` всюди, оскільки саме так
сховища повідомляють про відсутність запису, забороняючи при цьому
`gorm.DB` за межами шару зберігання.

## `//nolint` потребує причини

```go
//nolint:forbidigo // CLI output, not a log line
func Shout() { fmt.Println("hello") }
```

Назвіть конкретний лінтер і напишіть причину після `//`. Голий
`//nolint` вимикає все на цьому рядку, що ховає й наступний баг.

Це можна вимагати:

```yaml
linters:
  settings:
    nolintlint:
      require-explanation: true
      require-specific: true
      allow-unused: false
```

`allow-unused: false` також позначає директиви, що більше нічого не
придушують, тож їх видаляють, а не накопичують.

## Згенерований код і шляхи

```yaml
linters:
  exclusions:
    generated: lax
    paths:
      - ".*_templ\\.go$"
```

`generated: lax` пропускає файли з міткою `// Code generated ... DO NOT
EDIT.` зі статті про
[збірку та кодогенерацію](../11-architecture-and-conventions/05-build-codegen-and-cgo.md).
Немає сенсу повідомляти про стильові проблеми у виводі, який ніхто не
редагує.

`paths` виключає файли за іменем, а не за міткою. Кожен запис — це
регулярний вираз, що зіставляється зі шляхом до файлу; файли, що
збігаються, усе одно аналізуються, але про їхні проблеми не
повідомляється. Тут вказано вихідні файли templ — `_templ.go`.

## Форматування

```yaml
formatters:
  enable:
    - gofmt
    - goimports
```

```bash
golangci-lint fmt ./...
```

У v2 форматери — окрема секція від лінтерів, і `fmt` застосовує їх.
Дайте інструменту зробити це; форматування не варте коментаря в код-рев'ю.

## У CI

```yaml
- name: Lint
  run: golangci-lint run --timeout=5m ./...
```

Підвищуйте таймаут на великій кодовій базі — стандартний закороткий,
щоб пройти холодний кеш. Кешуйте `~/.cache/golangci-lint` між
запусками.

Впровадження інструменту в наявну кодову базу видає тисячі знахідок.
`--new-from-rev=origin/main` повідомляє лише про те, що привніс ваш
патч, що дозволяє впровадити його без спринту на прибирання.

## Що він не може

Він не може перевірити домовленості зі статті про
[конвенції проєкту](../11-architecture-and-conventions/06-project-conventions.md):
чи корисне повідомлення про помилку, чи пояснює коментар "чому", чи
чесна назва. Це залишається за рецензентами. Сенс добре налаштованого
лінтера в тому, що рецензенти можуть витрачати увагу саме на це.

> **З досвіду Python:** це `ruff` — один швидкий раннер над багатьма
> перевірками — з тим доповненням, що `depguard` і `forbidigo`
> дозволяють закодувати архітектурні правила, які в Python окремо
> робить `import-linter`.

## Швидка довідка

| Задача | Форма |
|---|---|
| конфіг | `.golangci.yml`, `version: "2"` |
| запуск | `golangci-lint run ./...`, зафіксована версія в CI |
| обмежити імпорти | `depguard`, з `desc` на кожному правилі |
| автоматично охопити нові пакети | область через `"$all"` мінус винятки |
| заборонити ідентифікатор | `forbidigo` з **`analyze-types: true`** |
| придушити | `//nolint:linter // причина` |
| вимагати цього | `nolintlint` з `require-explanation` |
| пропустити згенеровані файли | `exclusions.generated: lax` |
| форматувати | секція `formatters`, `golangci-lint fmt ./...` |
| впроваджувати поступово | `--new-from-rev=origin/main` |

## Джерела

- [golangci-lint — golangci-lint.run](https://golangci-lint.run/)
- [Configuration reference — golangci-lint.run/docs/configuration/file/](https://golangci-lint.run/docs/configuration/file/)
- [depguard — github.com/OpenPeeDeeP/depguard](https://github.com/OpenPeeDeeP/depguard)
- [forbidigo — github.com/ashanbrown/forbidigo](https://github.com/ashanbrown/forbidigo)
- [`go vet` — pkg.go.dev/cmd/vet](https://pkg.go.dev/cmd/vet)
