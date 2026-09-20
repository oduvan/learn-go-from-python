# Prometheus and OpenTelemetry

Two of the three pillars that
[structured logging](../12-observability/01-structured-logging-with-slog.md)
does not cover: metrics, for how the system behaves in aggregate, and
traces, for where one request spent its time.

> **Modules:** `github.com/prometheus/client_golang` and
> `go.opentelemetry.io/otel` with its `sdk` packages.

```go
reqs.WithLabelValues("GET", "200").Inc()
```

## Metrics

Prometheus scrapes an endpoint your process exposes. Four instrument
types cover nearly everything:

| Type | For | Example |
|---|---|---|
| Counter | monotonically increasing | requests served, errors |
| Gauge | a value that goes up and down | in-flight requests, queue depth |
| Histogram | a distribution, bucketed | request duration |
| Summary | client-side quantiles | rarely — prefer a histogram |

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

The `Vec` suffix means the metric is split by labels. `Help` is
mandatory and shows up in the scrape output, so write it for whoever
is paging at 3am.

### Register and serve

Use your own registry rather than the default one, so you control
exactly what is exposed:

```go
reg := prometheus.NewRegistry()
reg.MustRegister(reqs, dur, inflight)

mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
```

`MustRegister` panics on a duplicate name, which is correct: two
metrics with one name is a bug that should stop startup.

### Recording

```go
reqs.WithLabelValues("GET", "200").Inc()
dur.WithLabelValues("/users").Observe(0.012)
inflight.Inc()
defer inflight.Dec()
```

The scrape output:

```
http_requests_total{method="GET",status="200"} 2
http_requests_total{method="POST",status="500"} 1
http_inflight_requests 1
http_request_duration_seconds_bucket{route="/users",le="0.5"} 2
http_request_duration_seconds_sum{route="/users"} 0.412
http_request_duration_seconds_count{route="/users"} 2
```

A histogram produces a bucket series plus `_sum` and `_count`, which is
what lets the server compute quantiles across instances.

### Labels are the thing to get right

Every distinct label combination is a separate time series held in
memory, in your process and in Prometheus. High-cardinality labels are
the standard way to take down a monitoring system:

- **Never** a user id, request id, email, URL with an id in it, or
  timestamp.
- **Do** use method, status code, route *pattern*, queue name, outcome.

Use `/users/{id}`, never `/users/12345`. If a label can take more than
a few dozen values, it belongs in a log line or a trace, not a metric.

Keep names conventional: `_total` for counters, base SI units
(`_seconds`, `_bytes`), lower case with underscores.

### What to measure

Start with the four signals: **rate**, **errors**, **duration** and
**saturation**. In practice that is a request counter by status, a
duration histogram by route, and gauges for whatever is bounded — the
database pool's in-use connections, a queue's depth, in-flight
requests.

Metrics answer "is it broken and how badly". Traces answer "why".

## Tracing

A trace is a tree of spans covering one request across services.

```go
tp := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),
    sdktrace.WithResource(res),
)
otel.SetTracerProvider(tp)
defer tp.Shutdown(ctx)

tr := otel.Tracer("my-service")
```

The provider is set once at startup; `otel.Tracer(name)` anywhere gets
one. **`Shutdown` is not optional** — with a batching exporter, the
spans from your last few seconds are still buffered when the process
exits.

### Spans

```go
ctx, span := tr.Start(ctx, "handle-request",
    trace.WithSpanKind(trace.SpanKindServer),
    trace.WithAttributes(attribute.String("route", "/users")))
defer span.End()
```

`Start` returns a **new context** containing the span. Pass *that* one
down, or child spans attach to nothing:

```go
_, child := tr.Start(ctx, "db-query")
child.SetAttributes(attribute.Int("rows", 3))
child.End()
```

```
span=db-query        kind=internal attrs=[{rows 3}]        parent_set=true
span=handle-request  kind=server   attrs=[{route /users}]  parent_set=false
```

The child's parent is set because it received the context from `Start`.
This is the single most common tracing mistake: passing the original
context and getting a flat list of unrelated spans.

Note the ordering — `db-query` is exported first, because a span is
emitted when it *ends*.

### Errors

```go
span.SetStatus(codes.Error, "boom")
span.RecordError(err)
```

`SetStatus` marks the span failed so it stands out in a UI;
`RecordError` attaches the message as an event. Do both, and only on
the span where the error was handled — marking every ancestor makes
every trace red.

### Cardinality is free here

Unlike metric labels, span attributes are per-request and not
aggregated. A user id, a request id, the specific SQL — all fine. That
is the division of labour: low-cardinality metrics tell you something
is wrong, high-cardinality traces tell you which requests.

### Instrumenting the edges

Write few spans by hand. Most value comes from instrumentation
packages: `otelhttp` for inbound and outbound HTTP, `otelsql` or a
driver's own hooks for the database, and equivalents for message
queues. Those propagate the trace context across process boundaries
via request headers, which is what makes a distributed trace
distributed.

Add manual spans for expensive internal work — an LLM call, a
tree-walk, a batch job.

### Sampling

Tracing everything is expensive. `sdktrace.WithSampler` sets the
policy; a common choice is `ParentBased(TraceIDRatioBased(0.1))`,
which keeps a tenth of traces while respecting an upstream service's
decision so a trace is never half-recorded.

## Testing the instrumentation

Both libraries have in-memory test doubles, so this is ordinary unit
testing:

```go
exp := tracetest.NewInMemoryExporter()
tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
// ... run the code ...
spans := exp.GetSpans()
```

`WithSyncer` exports immediately rather than batching, which is what
you want in a test. For metrics, `prometheus/testutil` has
`CollectAndCompare` and `ToFloat64`.

## OpenTelemetry metrics, or Prometheus?

OTel has its own metrics API and can export to Prometheus. Using it
means one vendor-neutral API for both signals; using `client_golang`
directly means a smaller, more mature dependency. Either is defensible.
Pick one per service — two metrics systems in one process produces two
`/metrics` endpoints and a bad afternoon.

> **From Python:** `client_golang` is `prometheus_client` with the same
> concepts and the same cardinality trap. OTel's API is nearly
> identical across languages, except that here the context is an
> explicit parameter rather than ambient — which is why passing the
> returned `ctx` matters so much.

## Quick reference

| Task | Form |
|---|---|
| counter / gauge / histogram | `NewCounterVec`, `NewGauge`, `NewHistogramVec` |
| register | your own `prometheus.NewRegistry()` + `MustRegister` |
| expose | `promhttp.HandlerFor(reg, ...)` on `/metrics` |
| record | `.WithLabelValues(...).Inc()` / `.Observe(d)` |
| **labels** | low cardinality only — route patterns, never ids |
| naming | `_total`, `_seconds`, `_bytes` |
| tracer provider | once at startup, **`defer tp.Shutdown(ctx)`** |
| a span | `ctx, span := tr.Start(ctx, name)`, `defer span.End()` |
| children | pass the **returned** ctx |
| failure | `span.SetStatus(codes.Error, …)` + `RecordError(err)` |
| cross-service | `otelhttp` and friends propagate the context |
| volume | `ParentBased(TraceIDRatioBased(r))` |
| tests | `tracetest.NewInMemoryExporter`, `prometheus/testutil` |

## Sources

- [`client_golang` — pkg.go.dev/github.com/prometheus/client_golang/prometheus](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus)
- [Metric and label naming — prometheus.io/docs/practices/naming/](https://prometheus.io/docs/practices/naming/)
- [OpenTelemetry Go — opentelemetry.io/docs/languages/go/](https://opentelemetry.io/docs/languages/go/)
- [`otel/trace` — pkg.go.dev/go.opentelemetry.io/otel/trace](https://pkg.go.dev/go.opentelemetry.io/otel/trace)
- [`otelhttp` — pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp)
- [Semantic conventions — opentelemetry.io/docs/specs/semconv/](https://opentelemetry.io/docs/specs/semconv/)
