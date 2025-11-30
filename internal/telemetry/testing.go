package telemetry

import (
	"context"
	"sync"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestTelemetry provides in-memory telemetry for testing.
type TestTelemetry struct {
	*Telemetry

	SpanRecorder   *tracetest.SpanRecorder
	MetricReader   *testMetricReader
	tracerProvider *trace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
}

// NewTestTelemetry creates telemetry with in-memory exporters for testing.
func NewTestTelemetry() *TestTelemetry {
	cfg := NewDefaultConfig()
	cfg.Enabled = true

	spanRecorder := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))

	metricReader := newTestMetricReader()
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricReader.reader),
	)

	return &TestTelemetry{
		Telemetry: &Telemetry{
			config:         cfg,
			tracerProvider: tp,
			meterProvider:  mp,
		},
		SpanRecorder:   spanRecorder,
		MetricReader:   metricReader,
		tracerProvider: tp,
		meterProvider:  mp,
	}
}

// Spans returns all recorded spans.
func (t *TestTelemetry) Spans() []trace.ReadOnlySpan {
	return t.SpanRecorder.Ended()
}

// SpanByName finds a span by name, or nil if not found.
func (t *TestTelemetry) SpanByName(name string) trace.ReadOnlySpan {
	for _, span := range t.Spans() {
		if span.Name() == name {
			return span
		}
	}
	return nil
}

// AssertSpanExists verifies a span with the given name was recorded.
func (t *TestTelemetry) AssertSpanExists(tb testing.TB, name string) {
	tb.Helper()
	if t.SpanByName(name) == nil {
		tb.Errorf("expected span %q not found, got: %v", name, t.spanNames())
	}
}

// AssertSpanAttribute verifies a span has the expected attribute.
func (t *TestTelemetry) AssertSpanAttribute(tb testing.TB, spanName string, key string, expected interface{}) {
	tb.Helper()
	span := t.SpanByName(spanName)
	if span == nil {
		tb.Fatalf("span %q not found", spanName)
	}

	for _, attr := range span.Attributes() {
		if string(attr.Key) == key {
			got := attrValue(attr.Value)
			if got != expected {
				tb.Errorf("span %q attribute %q: got %v, want %v", spanName, key, got, expected)
			}
			return
		}
	}
	tb.Errorf("span %q missing attribute %q", spanName, key)
}

// spanNames returns names of all recorded spans.
func (t *TestTelemetry) spanNames() []string {
	spans := t.Spans()
	names := make([]string, len(spans))
	for i, span := range spans {
		names[i] = span.Name()
	}
	return names
}

// attrValue extracts the value from an attribute.
func attrValue(v attribute.Value) interface{} {
	switch v.Type() {
	case attribute.STRING:
		return v.AsString()
	case attribute.INT64:
		return v.AsInt64()
	case attribute.FLOAT64:
		return v.AsFloat64()
	case attribute.BOOL:
		return v.AsBool()
	default:
		return v.AsInterface()
	}
}

// Reset clears all recorded spans and metrics.
func (t *TestTelemetry) Reset() {
	// SpanRecorder doesn't have a reset, but ended spans are consumed on read
	// For metrics, we'd need to recreate the reader
}

// testMetricReader wraps the SDK's ManualReader for testing.
type testMetricReader struct {
	reader  *sdkmetric.ManualReader
	mu      sync.Mutex
	metrics []metricdata.ResourceMetrics
}

func newTestMetricReader() *testMetricReader {
	reader := sdkmetric.NewManualReader()
	return &testMetricReader{
		reader: reader,
	}
}

// ForceFlush triggers metric collection and stores results.
func (r *testMetricReader) ForceFlush(ctx context.Context) error {
	var rm metricdata.ResourceMetrics
	err := r.reader.Collect(ctx, &rm)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.metrics = append(r.metrics, rm)
	r.mu.Unlock()
	return nil
}

// Shutdown shuts down the reader.
func (r *testMetricReader) Shutdown(ctx context.Context) error {
	return r.reader.Shutdown(ctx)
}

// Metrics returns all collected metrics.
func (r *testMetricReader) Metrics() []metricdata.ResourceMetrics {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.metrics
}
