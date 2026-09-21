# Prometheus та OpenTelemetry

Два з трьох стовпів, які [структуроване логування](../12-observability/01-structured-logging-with-slog.md)
не покриває: метрики — для того, як система поводиться в сукупності, і
трейси — для того, де конкретний запит витратив свій час.

> **Модулі:** `github.com/prometheus/client_golang` та
> `go.opentelemetry.io/otel` з його пакетами `sdk`.

```go
reqs.WithLabelValues("GET", "200").Inc()
```

## Метрики

Prometheus зчитує ендпоінт, який виставляє ваш процес. Чотири типи
інструментів покривають майже все:

| Тип | Для | Приклад |
|---|---|---|
| Counter | монотонно зростаюче | оброблені запити, помилки |
| Gauge | значення, що росте й падає | запити в польоті, глибина черги |
| Histogram | розподіл, за кошиками | тривалість запиту |
| Summary | квантилі на боці клієнта | рідко — надавайте перевагу histogram |

```go
var (
    reqs = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "http_requests_total",
        Help: "Total HTTP requests.",
    }, []string{"method", "status"})

    dur = prometheus.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "http_request_duration_seconds",
        Help:    "Request duration.",
        Buckets: prometheus.DefBuckets,
    }, []string{"route"})

    inflight = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "http_inflight_requests",
        Help: "In-flight requests.",
    })
)
```

Суфікс `Vec` означає, що метрика розбита за мітками (labels). `Help`
обов'язковий і показується у виводі зчитування, тож пишіть його для
того, хто отримає сповіщення о третій ранку.

### Реєстрація та обслуговування

Використовуйте власний реєстр замість стандартного, щоб контролювати
точно, що виставляється:

```go
reg := prometheus.NewRegistry()
reg.MustRegister(reqs, dur, inflight)

mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
```

`MustRegister` панікує на дублікаті імені, і це правильно: дві метрики
з однією назвою — це баг, що має зупинити старт.

### Запис значень

```go
reqs.WithLabelValues("GET", "200").Inc()
dur.WithLabelValues("/users").Observe(0.012)
inflight.Inc()
defer inflight.Dec()
```

Вивід зчитування:

```
http_requests_total{method="GET",status="200"} 2
http_requests_total{method="POST",status="500"} 1
http_inflight_requests 1
http_request_duration_seconds_bucket{route="/users",le="0.5"} 2
http_request_duration_seconds_sum{route="/users"} 0.412
http_request_duration_seconds_count{route="/users"} 2
```

Histogram видає серію кошиків плюс `_sum` і `_count`, що й дозволяє
серверу обчислювати квантилі між екземплярами.

### Мітки — те, що потрібно зробити правильно

Кожна унікальна комбінація міток — окремий часовий ряд, що тримається
в пам'яті, і у вашому процесі, і в Prometheus. Мітки з високою
кардинальністю — стандартний спосіб покласти систему моніторингу:

- **Ніколи** id користувача, id запиту, email, URL з id всередині чи
  часову мітку.
- **Так** метод, код статусу, *шаблон* маршруту, назву черги, результат.

Використовуйте `/users/{id}`, ніколи `/users/12345`. Якщо мітка може
приймати більш ніж кілька десятків значень, їй місце в рядку логу чи
трейсі, а не в метриці.

Тримайте назви за домовленістю: `_total` для лічильників, базові
одиниці SI (`_seconds`, `_bytes`), нижній регістр з підкресленнями.

### Що вимірювати

Почніть із чотирьох сигналів: **швидкість (rate)**, **помилки
(errors)**, **тривалість (duration)** і **насиченість (saturation)**.
На практиці це лічильник запитів за статусом, histogram тривалості за
маршрутом, і gauge для всього обмеженого — кількості зайнятих
з'єднань у пулі бази даних, глибини черги, кількості запитів у
польоті.

Метрики відповідають на "чи все зламано і наскільки сильно". Трейси
відповідають на "чому".

## Трейсинг

Трейс — це дерево спанів, що покриває один запит через кілька
сервісів.

```go
tp := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),
    sdktrace.WithResource(res),
)
otel.SetTracerProvider(tp)
defer tp.Shutdown(ctx)

tr := otel.Tracer("my-service")
```

Провайдер встановлюється один раз при старті; `otel.Tracer(name)`
будь-де отримує один такий. **`Shutdown` не опціональний** — з
батчевим експортером спани останніх кількох секунд ще буферизовані,
коли процес завершується.

### Спани

```go
ctx, span := tr.Start(ctx, "handle-request",
    trace.WithSpanKind(trace.SpanKindServer),
    trace.WithAttributes(attribute.String("route", "/users")))
defer span.End()
```

`Start` повертає **новий контекст**, що містить спан. Передавайте
*саме його* далі, інакше дочірні спани не приєднуються ні до чого:

```go
_, child := tr.Start(ctx, "db-query")
child.SetAttributes(attribute.Int("rows", 3))
child.End()
```

```
span=db-query        kind=internal attrs=[{rows 3}]        parent_set=true
span=handle-request  kind=server   attrs=[{route /users}]  parent_set=false
```

Батько дочірнього спану встановлений, бо той отримав контекст від
`Start`. Це найпоширеніша помилка трейсингу: передати початковий
контекст і отримати плаский список непов'язаних спанів.

Зауважте порядок — `db-query` експортується першим, бо спан
випромінюється, коли він *завершується*.

### Помилки

```go
span.SetStatus(codes.Error, "boom")
span.RecordError(err)
```

`SetStatus` позначає спан як провалений, тож він виділяється в
інтерфейсі; `RecordError` додає повідомлення як подію. Робіть обидва,
і лише на тому спані, де помилку обробили — позначення кожного предка
робить кожен трейс червоним.

### Тут кардинальність безкоштовна

На відміну від міток метрик, атрибути спанів — на кожен запит окремо
й не агрегуються. Id користувача, id запиту, конкретний SQL — усе
годиться. У цьому й полягає розподіл праці: метрики з низькою
кардинальністю кажуть, що щось не так, трейси з високою
кардинальністю кажуть, які саме запити.

### Інструментація країв системи

Пишіть мало спанів вручну. Найбільшу цінність дають пакети
інструментації: `otelhttp` для вхідного й вихідного HTTP, `otelsql`
чи власні хуки драйвера для бази даних, і еквіваленти для черг
повідомлень. Вони поширюють контекст трейсу через межі процесів через
заголовки запиту — саме це й робить розподілений трейс розподіленим.

Додавайте ручні спани для дорогої внутрішньої роботи — виклику LLM,
обходу дерева, пакетного завдання.

### Семплінг

Трейсити все — дорого. `sdktrace.WithSampler` встановлює політику;
поширений вибір — `ParentBased(TraceIDRatioBased(0.1))`, що зберігає
десяту частину трейсів, поважаючи при цьому рішення вищого сервісу,
щоб трейс ніколи не був записаний наполовину.

## Тестування інструментації

Обидві бібліотеки мають тестові дублікати в пам'яті, тож це звичайне
юніт-тестування:

```go
exp := tracetest.NewInMemoryExporter()
tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
// ... run the code ...
spans := exp.GetSpans()
```

`WithSyncer` експортує одразу, а не пакетами, що й потрібно в тесті.
Для метрик у `prometheus/testutil` є `CollectAndCompare` і
`ToFloat64`.

## Метрики OpenTelemetry чи Prometheus?

OTel має власний API метрик і може експортувати в Prometheus. Його
використання означає один вендор-незалежний API для обох сигналів;
використання `client_golang` напряму означає меншу, зрілішу
залежність. Обидва варіанти захищені. Оберіть один на сервіс — дві
системи метрик в одному процесі дають два ендпоінти `/metrics` і
поганий вечір.

> **З досвіду Python:** `client_golang` — це `prometheus_client` з тими
> самими концепціями й тією самою пасткою кардинальності. API OTel
> майже ідентичний у різних мовах, окрім того, що тут контекст —
> явний параметр, а не неявний — саме тому передача поверненого `ctx`
> має таке значення.

## Швидка довідка

| Задача | Форма |
|---|---|
| counter / gauge / histogram | `NewCounterVec`, `NewGauge`, `NewHistogramVec` |
| зареєструвати | власний `prometheus.NewRegistry()` + `MustRegister` |
| виставити | `promhttp.HandlerFor(reg, ...)` на `/metrics` |
| записати | `.WithLabelValues(...).Inc()` / `.Observe(d)` |
| **мітки** | лише низька кардинальність — шаблони маршрутів, ніколи id |
| іменування | `_total`, `_seconds`, `_bytes` |
| провайдер трейсера | один раз при старті, **`defer tp.Shutdown(ctx)`** |
| спан | `ctx, span := tr.Start(ctx, name)`, `defer span.End()` |
| дочірні | передавайте **повернений** ctx |
| провал | `span.SetStatus(codes.Error, …)` + `RecordError(err)` |
| між сервісами | `otelhttp` та подібні поширюють контекст |
| обсяг | `ParentBased(TraceIDRatioBased(r))` |
| тести | `tracetest.NewInMemoryExporter`, `prometheus/testutil` |

## Джерела

- [`client_golang` — pkg.go.dev/github.com/prometheus/client_golang/prometheus](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus)
- [Metric and label naming — prometheus.io/docs/practices/naming/](https://prometheus.io/docs/practices/naming/)
- [OpenTelemetry Go — opentelemetry.io/docs/languages/go/](https://opentelemetry.io/docs/languages/go/)
- [`otel/trace` — pkg.go.dev/go.opentelemetry.io/otel/trace](https://pkg.go.dev/go.opentelemetry.io/otel/trace)
- [`otelhttp` — pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp)
- [Semantic conventions — opentelemetry.io/docs/specs/semconv/](https://opentelemetry.io/docs/specs/semconv/)
