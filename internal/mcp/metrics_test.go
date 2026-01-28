package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

func TestMetrics_RecordInvocation(t *testing.T) {
	// Create a manual reader to collect metrics
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	// Create metrics with test meter
	logger := zap.NewNop()
	m := &Metrics{
		meter:  mp.Meter(instrumentationName),
		logger: logger,
	}
	m.init()

	ctx := context.Background()

	// Test successful invocation
	m.RecordInvocation(ctx, "test_tool", 100*time.Millisecond, nil)

	// Test invocation with error
	m.RecordInvocation(ctx, "test_tool", 50*time.Millisecond, errors.New("validation error"))

	// Collect metrics
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	// Verify we got metrics
	if len(rm.ScopeMetrics) == 0 {
		t.Fatal("expected scope metrics, got none")
	}

	// Check for expected metric names
	foundInvocations := false
	foundDuration := false
	foundErrors := false

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch m.Name {
			case "contextd.mcp.tool.invocations_total":
				foundInvocations = true
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					total := int64(0)
					for _, dp := range sum.DataPoints {
						total += dp.Value
					}
					if total != 2 {
						t.Errorf("expected 2 invocations, got %d", total)
					}
				}
			case "contextd.mcp.tool.duration_seconds":
				foundDuration = true
			case "contextd.mcp.tool.errors_total":
				foundErrors = true
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					total := int64(0)
					for _, dp := range sum.DataPoints {
						total += dp.Value
					}
					if total != 1 {
						t.Errorf("expected 1 error, got %d", total)
					}
				}
			}
		}
	}

	if !foundInvocations {
		t.Error("invocations counter not found")
	}
	if !foundDuration {
		t.Error("duration histogram not found")
	}
	if !foundErrors {
		t.Error("errors counter not found")
	}
}

func TestMetrics_ActiveRequests(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	logger := zap.NewNop()
	m := &Metrics{
		meter:  mp.Meter(instrumentationName),
		logger: logger,
	}
	m.init()

	ctx := context.Background()

	// Increment twice
	m.IncrementActive(ctx, "test_tool")
	m.IncrementActive(ctx, "test_tool")

	// Decrement once
	m.DecrementActive(ctx, "test_tool")

	// Collect metrics
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	// Find active requests metric
	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			if metric.Name == "contextd.mcp.tool.active_requests" {
				if sum, ok := metric.Data.(metricdata.Sum[int64]); ok {
					total := int64(0)
					for _, dp := range sum.DataPoints {
						total += dp.Value
					}
					if total != 1 {
						t.Errorf("expected 1 active request, got %d", total)
					}
				}
				return
			}
		}
	}
	t.Error("active_requests metric not found")
}

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, ""},
		{"tenant error", errors.New("tenant not found"), "tenant_error"},
		{"validation error", errors.New("validation failed"), "validation_error"},
		{"invalid input", errors.New("invalid project_id"), "validation_error"},
		{"not found", errors.New("memory not found"), "not_found"},
		{"timeout", errors.New("operation timeout"), "timeout"},
		{"permission denied", errors.New("permission denied"), "auth_error"},
		{"unauthorized", errors.New("unauthorized access"), "auth_error"},
		{"vectorstore error", errors.New("vectorstore connection failed"), "storage_error"},
		{"embedding error", errors.New("embedding generation failed"), "storage_error"},
		{"generic error", errors.New("something went wrong"), "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeError(tt.err)
			if result != tt.expected {
				t.Errorf("categorizeError(%v) = %q, want %q", tt.err, result, tt.expected)
			}
		})
	}
}
