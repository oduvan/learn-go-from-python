# Демонстраційний проєкт

Невеликий Go-модуль, що охоплює більшість тем з цієї теки:

```
09-demo-project/
├── go.mod                          # маніфест модуля
├── main.go                         # package main — точка входу програми
├── counter/
│   ├── counter.go                  # публічний пакет, доступний для імпорту ззовні
│   ├── counter_test.go             # внутрішні тести + бенчмарк
│   └── testdata/                   # ігнорується збіркою, використовується тестами
│       ├── alice.txt
│       └── lorem.txt
└── internal/
    └── workpool/
        └── workpool.go             # доступний для імпорту лише всередині цього модуля
```

Що демонструє кожна частина:

- **`go.mod`** — шлях модуля `example.com/demo`; створено за допомогою `go mod init`.
- **`main.go`** — `package main` виробляє виконуваний файл. Імпортує як `counter` (суміжний пакет), так і використовує трейсер часу виконання.
- **`counter/`** — звичайний пакет, доступний для імпорту як `example.com/demo/counter`.
- **`counter/counter_test.go`** — тести того самого пакету (можуть бачити неекспортовані символи), плюс `BenchmarkCountConcurrent`.
- **`counter/testdata/`** — інструмент `go` відмовляється компілювати будь-що всередині `testdata/`, тому фікстури безпечно живуть тут.
- **`internal/workpool/`** — компілятор Go забезпечує, що `example.com/demo/internal/workpool` можна імпортувати лише з пакетів, що ростуть з кореня `example.com/demo`. Якщо ви скопіюєте цей модуль за іншим шляхом, зовнішній код не зможе дістатися до `workpool`.

## Запуск

З цієї директорії:

```bash
go run .
```

Очікуваний вивід:

```
Counted 377 words across 2 files (workload amplified to 400 jobs over 4 workers so the trace shows visible parallelism)
```

Реальної роботи мало — два текстові файли в `counter/testdata/`, разом 377
слів. Програма навмисно повторює це крихітне навантаження 200× (→ 400
завдань) і розподіляє його на 4 горутини-робітники з єдиної причини: щоб у
записаному трейсі було достатньо конкурентної активності, аби його варто
було розглядати в `go tool trace`. Також записує `trace.out`.

## Запуск тестів

```bash
go test ./...
```

Щоб запустити бенчмарк:

```bash
go test -bench=. ./counter
```

## Перегляд трейсу

Після того як `go run .` створив `trace.out`:

```bash
go tool trace trace.out
```

Це запускає локальний веб-сервер і виводить URL на кшталт `http://127.0.0.1:NNNNN/...`. Відкрийте його у браузері та досліджуйте:

- **View trace** — часова шкала горутин.
- **Goroutine analysis** — чим кожна горутина витрачала свій час.
- **Sync / scheduler blocking profiles** — де горутини очікували.

Натисніть Ctrl-C у терміналі, щоб зупинити переглядач.

## Файли у прив'язці до конспектів

| Тема | Файл конспекту | Де це видно тут |
|---|---|---|
| Директиви `go.mod` | [04-go-file-types.md](../04-go-file-types.md) | `go.mod` |
| `_test.go`, `testdata/` | [04-go-file-types.md](../04-go-file-types.md) | `counter/counter_test.go`, `counter/testdata/` |
| Правило `internal/` | [05-special-folders.md](../05-special-folders.md) | `internal/workpool/` |
| `runtime/trace` та `go tool trace` | [03-go-tool-trace.md](../03-go-tool-trace.md) | `trace.Start` / `trace.Stop` у `main.go`, а також `trace.out` |
| `go run`, `go test`, `go build` | [02-go-subcommands.md](../02-go-subcommands.md) | команди вище |
