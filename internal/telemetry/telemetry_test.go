package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestNew_DisabledTelemetry(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Enabled = false

	tel, err := New(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, tel)

	// Should return no-op providers
	tracer := tel.Tracer("test")
	assert.NotNil(t, tracer)

	meter := tel.Meter("test")
	assert.NotNil(t, meter)

	// Should report as not enabled
	assert.False(t, tel.IsEnabled())
}

func TestNew_InvalidConfig(t *testing.T) {
	cfg := &Config{
		Enabled:     true,
		Endpoint:    "",
		ServiceName: "",
	}

	tel, err := New(context.Background(), cfg)
	require.Error(t, err)
	assert.Nil(t, tel)
	assert.Contains(t, err.Error(), "invalid telemetry config")
}

func TestTelemetry_Health(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Enabled = false

	tel, err := New(context.Background(), cfg)
	require.NoError(t, err)

	health := tel.Health()
	assert.True(t, health.Healthy)
	assert.False(t, health.Degraded)
}

func TestTelemetry_NilSafe(t *testing.T) {
	var tel *Telemetry = nil

	// All methods should be nil-safe
	assert.NotPanics(t, func() {
		_ = tel.Tracer("test")
		_ = tel.Meter("test")
		_ = tel.LoggerProvider()
		_ = tel.Health()
		_ = tel.IsEnabled()
		_ = tel.Shutdown(context.Background())
		_ = tel.ForceFlush(context.Background())
	})

	// Nil should report unhealthy
	health := tel.Health()
	assert.False(t, health.Healthy)
	assert.True(t, health.Degraded)
}

func TestTelemetry_Shutdown(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Enabled = false

	tel, err := New(context.Background(), cfg)
	require.NoError(t, err)

	// Shutdown should succeed for disabled telemetry
	err = tel.Shutdown(context.Background())
	require.NoError(t, err)

	// Health should be unhealthy after shutdown
	health := tel.Health()
	assert.False(t, health.Healthy)
}

func TestTelemetry_ShutdownWithTimeout(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Enabled = false
	cfg.Shutdown.Timeout = config.Duration(100 * time.Millisecond)

	tel, err := New(context.Background(), cfg)
	require.NoError(t, err)

	// Shutdown with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = tel.Shutdown(ctx)
	require.NoError(t, err)
}

func TestTelemetry_SetLoggerProvider(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Enabled = false

	tel, err := New(context.Background(), cfg)
	require.NoError(t, err)

	// Should be nil initially
	assert.Nil(t, tel.LoggerProvider())

	// SetLoggerProvider on nil should not panic
	var nilTel *Telemetry
	assert.NotPanics(t, func() {
		nilTel.SetLoggerProvider(nil)
	})
}

func TestTestTelemetry_SpanRecording(t *testing.T) {
	tt := NewTestTelemetry()
	require.NotNil(t, tt)

	tracer := tt.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	span.SetAttributes(attribute.String("key", "value"))
	span.End()

	// Verify span was recorded
	tt.AssertSpanExists(t, "test-span")
	tt.AssertSpanAttribute(t, "test-span", "key", "value")

	_ = ctx // Use ctx to avoid unused warning
}

func TestTestTelemetry_SpanNotFound(t *testing.T) {
	tt := NewTestTelemetry()

	span := tt.SpanByName("non-existent")
	assert.Nil(t, span)
}

func TestTestTelemetry_MultipleSpans(t *testing.T) {
	tt := NewTestTelemetry()

	tracer := tt.Tracer("test")

	_, span1 := tracer.Start(context.Background(), "span1")
	span1.SetAttributes(attribute.Int64("count", 1))
	span1.End()

	_, span2 := tracer.Start(context.Background(), "span2")
	span2.SetAttributes(attribute.Int64("count", 2))
	span2.End()

	_, span3 := tracer.Start(context.Background(), "span3")
	span3.SetAttributes(attribute.Bool("done", true))
	span3.End()

	// All spans should be recorded
	assert.Len(t, tt.Spans(), 3)
	tt.AssertSpanExists(t, "span1")
	tt.AssertSpanExists(t, "span2")
	tt.AssertSpanExists(t, "span3")

	// Check attributes
	tt.AssertSpanAttribute(t, "span1", "count", int64(1))
	tt.AssertSpanAttribute(t, "span2", "count", int64(2))
	tt.AssertSpanAttribute(t, "span3", "done", true)
}

func TestTestTelemetry_MeterRecording(t *testing.T) {
	tt := NewTestTelemetry()
	require.NotNil(t, tt)

	meter := tt.Meter("test")
	counter, err := meter.Int64Counter("test.counter")
	require.NoError(t, err)

	counter.Add(context.Background(), 1)
	counter.Add(context.Background(), 2)

	// Force collection
	err = tt.MetricReader.ForceFlush(context.Background())
	require.NoError(t, err)

	// Metrics should be recorded
	metrics := tt.MetricReader.Metrics()
	assert.NotEmpty(t, metrics)
}

func TestTelemetry_ForceFlush_Disabled(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Enabled = false

	tel, err := New(context.Background(), cfg)
	require.NoError(t, err)

	// ForceFlush should succeed for disabled telemetry
	err = tel.ForceFlush(context.Background())
	require.NoError(t, err)
}

func TestTelemetry_ForceFlush_WithTestTelemetry(t *testing.T) {
	tt := NewTestTelemetry()

	// Create a span
	tracer := tt.Tracer("test")
	_, span := tracer.Start(context.Background(), "flush-test")
	span.End()

	// ForceFlush should succeed
	err := tt.ForceFlush(context.Background())
	require.NoError(t, err)
}

func TestTestTelemetry_SpanByName_Exists(t *testing.T) {
	tt := NewTestTelemetry()

	tracer := tt.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")
	span.End()

	// SpanByName should find the span
	found := tt.SpanByName("test-span")
	assert.NotNil(t, found)
	assert.Equal(t, "test-span", found.Name())
}

func TestTestTelemetry_SpanAttributeMatching(t *testing.T) {
	tt := NewTestTelemetry()

	tracer := tt.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")
	span.SetAttributes(attribute.String("key", "value"))
	span.End()

	// Should pass with correct attribute
	tt.AssertSpanAttribute(t, "test-span", "key", "value")
}

func TestTestTelemetry_Reset(t *testing.T) {
	tt := NewTestTelemetry()

	// Create a span
	tracer := tt.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")
	span.End()

	// Should have spans
	assert.NotEmpty(t, tt.Spans())

	// Reset (note: SpanRecorder doesn't have a reset, so this is a no-op)
	tt.Reset()

	// Spans are still there because SpanRecorder.Ended() doesn't clear
}

func TestTestTelemetry_SpanAttributeTypes(t *testing.T) {
	tt := NewTestTelemetry()

	tracer := tt.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")
	span.SetAttributes(
		attribute.String("string-key", "value"),
		attribute.Int64("int-key", 42),
		attribute.Float64("float-key", 3.14),
		attribute.Bool("bool-key", true),
	)
	span.End()

	// Test each attribute type
	tt.AssertSpanAttribute(t, "test-span", "string-key", "value")
	tt.AssertSpanAttribute(t, "test-span", "int-key", int64(42))
	tt.AssertSpanAttribute(t, "test-span", "float-key", 3.14)
	tt.AssertSpanAttribute(t, "test-span", "bool-key", true)
}

func TestTestTelemetry_MetricReaderShutdown(t *testing.T) {
	tt := NewTestTelemetry()

	// Shutdown should succeed
	err := tt.MetricReader.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestTestTelemetry_SpanNames(t *testing.T) {
	tt := NewTestTelemetry()

	tracer := tt.Tracer("test")

	_, span1 := tracer.Start(context.Background(), "alpha")
	span1.End()

	_, span2 := tracer.Start(context.Background(), "beta")
	span2.End()

	// spanNames is private, but we can test via AssertSpanExists
	assert.Len(t, tt.Spans(), 2)
	tt.AssertSpanExists(t, "alpha")
	tt.AssertSpanExists(t, "beta")
}

func TestTelemetry_ShutdownWithProviders(t *testing.T) {
	tt := NewTestTelemetry()

	// Create some spans and metrics
	tracer := tt.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")
	span.End()

	meter := tt.Meter("test")
	counter, _ := meter.Int64Counter("test.counter")
	counter.Add(context.Background(), 1)

	// Shutdown should succeed
	err := tt.Shutdown(context.Background())
	require.NoError(t, err)

	// Health should be unhealthy after shutdown
	health := tt.Health()
	assert.False(t, health.Healthy)
}
