# Вчимо Go з досвідом Python

Особисті конспект-нотатки для Python-розробника, який починає вивчати
Go. Мета — пояснювати Go **його власними категоріями**, з короткими
порівняннями з Python там, де вони допомагають загострити різницю, а не
перетворити пояснення на переклад.

Усі матеріали орієнтовані на актуальний стабільний реліз Go — **Go 1.26.4**.

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

## Джерела

- Репозиторій з вихідним кодом: <https://github.com/oduvan/learn-go-from-python>.
- Наприкінці кожного конспекту перелічені офіційні джерела, з якими
  було звірено матеріал — як правило, [go.dev](https://go.dev/),
  [pkg.go.dev](https://pkg.go.dev/) або
  [специфікація Go](https://go.dev/ref/spec).
