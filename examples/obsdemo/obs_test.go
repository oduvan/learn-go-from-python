// Verifies docs/13-third-party-libraries/14-prometheus-and-opentelemetry.md
package obsdemo

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func metrics(t *testing.T) (*prometheus.Registry, *prometheus.CounterVec, *prometheus.HistogramVec, prometheus.Gauge) {
	t.Helper()
	reqs := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total", Help: "Total HTTP requests.",
	}, []string{"method", "status"})
	dur := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds", Help: "Request duration.",
		Buckets: prometheus.DefBuckets,
	}, []string{"route"})
	inflight := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_inflight_requests", Help: "In-flight requests.",
	})

	reg := prometheus.NewRegistry()
	reg.MustRegister(reqs, dur, inflight)
	return reg, reqs, dur, inflight
}

func scrape(t *testing.T, reg *prometheus.Registry) string {
	t.Helper()
	rec := httptest.NewRecorder()
	promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).
		ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	return rec.Body.String()
}

func TestScrapeOutput(t *testing.T) {
	reg, reqs, dur, inflight := metrics(t)

	reqs.WithLabelValues("GET", "200").Inc()
	reqs.WithLabelValues("GET", "200").Inc()
	reqs.WithLabelValues("POST", "500").Inc()
	dur.WithLabelValues("/users").Observe(0.012)
	dur.WithLabelValues("/users").Observe(0.4)
	inflight.Inc()
	inflight.Inc()
	inflight.Dec()

	out := scrape(t, reg)
	for _, want := range []string{
		`http_requests_total{method="GET",status="200"} 2`,
		`http_requests_total{method="POST",status="500"} 1`,
		`http_inflight_requests 1`,
		`http_request_duration_seconds_count{route="/users"} 2`,
		`http_request_duration_seconds_bucket{route="/users",le="0.5"} 2`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("scrape missing %q", want)
		}
	}
}

// A histogram emits _bucket, _sum and _count — that triple is what lets
// the server compute quantiles across instances.
func TestHistogramEmitsSumAndCount(t *testing.T) {
	reg, _, dur, _ := metrics(t)
	dur.WithLabelValues("/x").Observe(0.25)
	out := scrape(t, reg)
	for _, suffix := range []string{"_bucket", "_sum", "_count"} {
		if !strings.Contains(out, "http_request_duration_seconds"+suffix) {
			t.Errorf("missing %s series", suffix)
		}
	}
}

func TestMustRegisterPanicsOnDuplicate(t *testing.T) {
	reg := prometheus.NewRegistry()
	c := func() prometheus.Counter {
		return prometheus.NewCounter(prometheus.CounterOpts{Name: "dupe_total", Help: "h"})
	}
	reg.MustRegister(c())

	defer func() {
		if recover() == nil {
			t.Error("MustRegister did not panic on a duplicate name")
		}
	}()
	reg.MustRegister(c())
}

// Child spans attach only if you pass the context Start returned.
func TestSpanParenting(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	tr := otel.Tracer("demo")

	ctx, span := tr.Start(context.Background(), "handle-request",
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attribute.String("route", "/users")))

	_, child := tr.Start(ctx, "db-query")
	child.SetAttributes(attribute.Int("rows", 3))
	child.End()

	span.SetStatus(codes.Error, "boom")
	span.RecordError(errors.New("disk full"))
	span.End()

	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatal(err)
	}

	spans := exp.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("exported %d spans, want 2", len(spans))
	}
	// A span is exported when it ends, so the child comes first.
	kid, root := spans[0], spans[1]
	if kid.Name != "db-query" || root.Name != "handle-request" {
		t.Fatalf("names = %q, %q", kid.Name, root.Name)
	}
	if !kid.Parent.IsValid() {
		t.Error("child has no parent — was the returned ctx passed to Start?")
	}
	if root.Parent.IsValid() {
		t.Error("root span has a parent")
	}
	if root.SpanKind != trace.SpanKindServer {
		t.Errorf("root kind = %v, want server", root.SpanKind)
	}
	if root.Status.Code != codes.Error {
		t.Errorf("root status = %v, want Error", root.Status.Code)
	}
	if len(root.Events) != 1 {
		t.Errorf("RecordError produced %d events, want 1", len(root.Events))
	}
}

// The failure mode the article warns about: reuse the original context
// and the spans come out flat and unrelated.
func TestWrongContextLosesParenting(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	tr := tp.Tracer("demo")

	outer := context.Background()
	_, span := tr.Start(outer, "root")
	_, child := tr.Start(outer, "orphan") // outer, not the returned ctx
	child.End()
	span.End()
	tp.ForceFlush(context.Background())

	for _, s := range exp.GetSpans() {
		if s.Name == "orphan" && s.Parent.IsValid() {
			t.Error("orphan unexpectedly has a parent")
		}
	}
}
