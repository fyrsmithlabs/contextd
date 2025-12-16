package folding

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewMetrics(t *testing.T) {
	// Set up a test meter provider
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
	)
	meter := provider.Meter(InstrumentationName)

	metrics, err := NewMetrics(meter)
	require.NoError(t, err, "NewMetrics should not error")
	require.NotNil(t, metrics, "metrics should not be nil")
	assert.True(t, metrics.initialized, "metrics should be initialized")
}

func TestNewMetrics_NilMeter(t *testing.T) {
	// Should use global meter provider when nil
	metrics, err := NewMetrics(nil)
	require.NoError(t, err, "NewMetrics with nil meter should not error")
	require.NotNil(t, metrics, "metrics should not be nil")
	assert.True(t, metrics.initialized, "metrics should be initialized")
}

func TestMetrics_RecordBranchCreated(t *testing.T) {
	// Set up a test meter provider with a reader
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
	)
	meter := provider.Meter(InstrumentationName)

	metrics, err := NewMetrics(meter)
	require.NoError(t, err)

	ctx := context.Background()

	// Record two branch creations
	metrics.RecordBranchCreated(ctx, "sess_001", 0, 8192)
	metrics.RecordBranchCreated(ctx, "sess_001", 1, 4096)

	// Collect metrics
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err, "should collect metrics without error")

	// Verify branch created counter
	foundCreated := false
	foundActive := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "folding.branch.created.total" {
				foundCreated = true
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					total := int64(0)
					for _, dp := range sum.DataPoints {
						total += dp.Value
					}
					assert.Equal(t, int64(2), total, "created counter should be 2")
				}
			}
			if m.Name == "folding.branch.active.count" {
				foundActive = true
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					total := int64(0)
					for _, dp := range sum.DataPoints {
						total += dp.Value
					}
					assert.Equal(t, int64(2), total, "active count should be 2")
				}
			}
		}
	}
	assert.True(t, foundCreated, "should find branch.created.total counter")
	assert.True(t, foundActive, "should find branch.active.count gauge")
}

func TestMetrics_NilReceiver(t *testing.T) {
	var metrics *Metrics
	ctx := context.Background()

	// Should not panic with nil receiver
	assert.NotPanics(t, func() {
		metrics.RecordBranchCreated(ctx, "sess_001", 0, 8192)
		metrics.RecordBranchReturned(ctx, "sess_001", 0, 5000, 8192, 30*time.Second)
		metrics.RecordBranchTimeout(ctx, "sess_001", 0, 3000, 8192, 300*time.Second)
		metrics.RecordBranchFailed(ctx, "sess_001", 0, "test", 100, 8192, 10*time.Second)
	})
}

func TestTracer(t *testing.T) {
	tracer := Tracer()
	assert.NotNil(t, tracer, "tracer should not be nil")

	// Verify we can start a span
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	defer span.End()

	assert.NotNil(t, span, "span should not be nil")
}

func TestSpanAttributes(t *testing.T) {
	attrs := SpanAttributes("br_123", "sess_456", 2)
	require.Len(t, attrs, 3, "should have 3 attributes")

	// Verify attributes
	assert.Equal(t, attribute.String("folding.branch_id", "br_123"), attrs[0])
	assert.Equal(t, attribute.String("folding.session_id", "sess_456"), attrs[1])
	assert.Equal(t, attribute.Int("folding.depth", 2), attrs[2])
}

func TestStartSpan(t *testing.T) {
	// Set up a test tracer provider with a recorder
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
	)
	otel.SetTracerProvider(provider)
	defer otel.SetTracerProvider(sdktrace.NewTracerProvider())

	ctx := context.Background()
	spanCtx, span := StartSpan(ctx, "test.operation", "br_123", "sess_456", 1)

	assert.NotNil(t, spanCtx, "span context should not be nil")
	assert.NotNil(t, span, "span should not be nil")

	span.End()

	// Verify span was recorded with correct attributes
	spans := recorder.Ended()
	require.Len(t, spans, 1, "should record one span")

	recordedSpan := spans[0]
	assert.Equal(t, "test.operation", recordedSpan.Name(), "span name should match")

	// Check attributes
	attrs := recordedSpan.Attributes()
	hasBranchID := false
	hasSessionID := false
	hasDepth := false
	for _, attr := range attrs {
		if attr.Key == "folding.branch_id" && attr.Value.AsString() == "br_123" {
			hasBranchID = true
		}
		if attr.Key == "folding.session_id" && attr.Value.AsString() == "sess_456" {
			hasSessionID = true
		}
		if attr.Key == "folding.depth" && attr.Value.AsInt64() == 1 {
			hasDepth = true
		}
	}
	assert.True(t, hasBranchID, "should have branch_id attribute")
	assert.True(t, hasSessionID, "should have session_id attribute")
	assert.True(t, hasDepth, "should have depth attribute")
}

func TestRecordError(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
	)
	otel.SetTracerProvider(provider)
	defer otel.SetTracerProvider(sdktrace.NewTracerProvider())

	ctx := context.Background()
	spanCtx, span := StartSpan(ctx, "test.operation", "br_123", "sess_456", 0)

	testErr := assert.AnError
	RecordError(spanCtx, testErr, attribute.String("error.type", "test"))

	span.End()

	// Verify error was recorded
	spans := recorder.Ended()
	require.Len(t, spans, 1)
	events := spans[0].Events()
	require.NotEmpty(t, events, "should have at least one event")

	// Check for exception event
	hasException := false
	for _, event := range events {
		if event.Name == "exception" {
			hasException = true
			break
		}
	}
	assert.True(t, hasException, "should have exception event")
}

func TestSetSpanStatus(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
	)
	otel.SetTracerProvider(provider)
	defer otel.SetTracerProvider(sdktrace.NewTracerProvider())

	ctx := context.Background()
	spanCtx, span := StartSpan(ctx, "test.operation", "br_123", "sess_456", 0)

	SetSpanStatus(spanCtx, codes.Error, "test error")

	span.End()

	// Verify status was set
	spans := recorder.Ended()
	require.Len(t, spans, 1)
	status := spans[0].Status()
	assert.Equal(t, "test error", status.Description, "status description should match")
	assert.Equal(t, codes.Error, status.Code, "status code should be Error")
}
