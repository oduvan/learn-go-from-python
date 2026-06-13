# Спеціальні директорії в Go-проекті

Існують рівно **дві** назви директорій, які інструмент `go` обробляє особливим чином. Все інше — конвенція спільноти.

## `internal/` — примусова видимість

Компілятор застосовує правило щодо `internal/`. З довідника `cmd/go`:

> An import of a path containing the element "internal" is disallowed if the importing code is outside the tree rooted at the parent of the "internal" directory.

На практиці, якщо ваш модуль — `example.com/app`:

```
app/
├── go.mod
├── api/
│   └── handler.go        # може імпортувати .../internal/auth — той самий модуль
└── internal/
    └── auth/
        └── token.go      # лише пакети під app/... можуть імпортувати це
```

*Інший* модуль, що спробує виконати `import "example.com/app/internal/auth"`, не скомпілюється.

Саме так бібліотеки оголошують «це деталь реалізації — я не обіцяю зберігати її стабільною». Це найсильніший механізм видимості в Go.

## `testdata/` — ігнорується збиранням

З довідника `cmd/go`:

> "The go tool will ignore a directory named 'testdata', making it available to hold ancillary data needed by the tests."

```
foo/
├── foo.go
├── foo_test.go
└── testdata/
    ├── input1.json
    └── golden/
        └── expected.txt
```

Все, що знаходиться всередині `testdata/`, є невидимим для збирання: Go не намагатиметься це скомпілювати, не скаржитиметься на не-Go файли всередині, не включатиме в графи залежностей. Це стандартне місце для тестових фікстур, «золотих» файлів (golden files), некоректних вхідних даних для fuzz-тестів тощо.

## Конвенції (не примусові) — все ж варто знати

| Директорія | Значення |
|---|---|
| `cmd/<name>/` | Кожна піддиректорія містить пакет `main`, що виробляє один бінарний файл (`cmd/server`, `cmd/cli`). Стандартне розташування, коли репозиторій містить кілька виконуваних файлів. |
| `pkg/` | Стара конвенція для бібліотечних пакетів. Сучасний Go її не потребує — розміщуйте пакети в корені модуля. Не додавайте її лише тому, що бачили десь іще. |
| `vendor/` | Якщо присутня, `go build` використовує її замість `$GOMODCACHE`. Створюється командою `go mod vendor`. Для ізольованих або повністю відтворюваних збирань. |
| `api/`, `web/`, `configs/` тощо | З неофіційного репозиторію «golang-standards/project-layout». **Не схвалено командою Go.** Сприймайте як думку однієї команди. |

## Файли та директорії, що ігноруються інструментом `go`

На додачу до `testdata/`, інструмент ігнорує:

- Будь-який файл або директорію, ім'я якої починається з `_` (наприклад, `_scratch.go`, `_drafts/`).
- Будь-який файл або директорію, ім'я якої починається з `.` (наприклад, `.idea/`).

Їх можна використовувати для чернеткової роботи, яка повинна залишатися поруч із кодом, але не брати участі у збираннях.

## Поширене, рекомендоване розташування

Для проекту з кількома виконуваними файлами та спільним внутрішнім кодом:

```
myapp/
├── go.mod
├── go.sum
├── README.md
├── cmd/
│   ├── api-server/
│   │   └── main.go
│   └── worker/
│       └── main.go
├── internal/
│   ├── auth/
│   │   └── auth.go
│   └── metrics/
│       └── metrics.go
├── api.go                 # публічний пакет у корені модуля
└── api_test.go
```

Для бібліотеки з одним призначенням підходить найпростіша форма:

```
greetings/
├── go.mod
├── greetings.go
└── greetings_test.go
```

## Джерела

- [Організація Go-модуля — go.dev/doc/modules/layout](https://go.dev/doc/modules/layout)
- [Правило пакетів `internal/` — pkg.go.dev/cmd/go#hdr-Internal_Directories](https://pkg.go.dev/cmd/go#hdr-Internal_Directories)
- [`testdata` та ігноровані файли — pkg.go.dev/cmd/go#hdr-Test_packages](https://pkg.go.dev/cmd/go#hdr-Test_packages)
