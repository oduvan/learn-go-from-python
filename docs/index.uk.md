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

- [Змінні та константи](02-language-basics/01-variables-and-constants.md)
- [Базові типи](02-language-basics/02-basic-types.md)
- [Перетворення типів](02-language-basics/03-type-conversions.md)
- [Оператори](02-language-basics/04-operators.md)
- [Керування потоком](02-language-basics/05-control-flow.md)
- [Функції](02-language-basics/06-functions.md)
- [Помилки](02-language-basics/07-errors.md)
- [Вказівники](02-language-basics/08-pointers.md)
- [Власні типи](02-language-basics/09-custom-types.md)
- [Структури](02-language-basics/10-structs.md)
- [Масиви та зрізи](02-language-basics/11-arrays-and-slices.md)
- [Map (асоціативні масиви)](02-language-basics/12-maps.md)
- [Вибір структури даних](02-language-basics/13-choosing-a-data-structure.md)
- [Методи](02-language-basics/14-methods.md)
- [Defer](02-language-basics/15-defer.md)
- [Panic та recover](02-language-basics/16-panic-and-recover.md)
- [Імпорти](02-language-basics/17-imports.md)
- [Інтерфейси](02-language-basics/18-interfaces.md)

## Джерела

- Репозиторій з вихідним кодом: <https://github.com/oduvan/learn-go-from-python>.
- Наприкінці кожного конспекту перелічені офіційні джерела, з якими
  було звірено матеріал — як правило, [go.dev](https://go.dev/),
  [pkg.go.dev](https://pkg.go.dev/) або
  [специфікація Go](https://go.dev/ref/spec).
