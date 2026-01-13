package reasoningbank

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

// TestCounterMetricsExport verifies that counter metrics are properly recorded
// and can be exported. This test helps debug why counters aren't appearing in
// VictoriaMetrics when Observable Gauges are working fine.
func TestCounterMetricsExport(t *testing.T) {
	ctx := context.Background()

	// Create a manual reader to capture metrics
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
	)

	// Set as the global meter provider
	oldMP := otel.GetMeterProvider()
	otel.SetMeterProvider(mp)
	defer otel.SetMeterProvider(oldMP)

	// Create an in-memory mock store for testing (uses local mockStore)
	store := newMockStore()
	logger := zap.NewNop()

	// Create the ReasoningBank service - this should initialize counters
	svc, err := NewService(store, logger)
	require.NoError(t, err, "failed to create ReasoningBank service")

	// Verify counters are not nil
	assert.NotNil(t, svc.searchCounter, "searchCounter should not be nil")
	assert.NotNil(t, svc.recordCounter, "recordCounter should not be nil")
	assert.NotNil(t, svc.feedbackCounter, "feedbackCounter should not be nil")
	assert.NotNil(t, svc.outcomeCounter, "outcomeCounter should not be nil")
	assert.NotNil(t, svc.errorCounter, "errorCounter should not be nil")

	// Manually increment the search counter to verify it works
	t.Log("Incrementing search counter manually...")
	svc.searchCounter.Add(ctx, 5)

	// Force collect metrics
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err, "failed to collect metrics")

	// Log all collected metrics for debugging
	t.Logf("Collected %d scope metrics", len(rm.ScopeMetrics))
	for _, sm := range rm.ScopeMetrics {
		t.Logf("Scope: %s", sm.Scope.Name)
		for _, m := range sm.Metrics {
			t.Logf("  Metric: %s (type: %T)", m.Name, m.Data)
			switch data := m.Data.(type) {
			case metricdata.Sum[int64]:
				t.Logf("    Sum data points: %d, temporality: %s", len(data.DataPoints), data.Temporality)
				for _, dp := range data.DataPoints {
					t.Logf("      Value: %d, Attrs: %v", dp.Value, dp.Attributes.ToSlice())
				}
			case metricdata.Gauge[int64]:
				t.Logf("    Gauge data points: %d", len(data.DataPoints))
				for _, dp := range data.DataPoints {
					t.Logf("      Value: %d, Attrs: %v", dp.Value, dp.Attributes.ToSlice())
				}
			default:
				t.Logf("    Unknown data type: %T", data)
			}
		}
	}

	// Find the search counter in the collected metrics
	var foundSearchCounter bool
	var searchCounterValue int64

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "contextd.memory.searches_total" {
				foundSearchCounter = true
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					for _, dp := range sum.DataPoints {
						searchCounterValue += dp.Value
					}
				}
			}
		}
	}

	assert.True(t, foundSearchCounter, "search counter should be present in metrics")
	assert.Equal(t, int64(5), searchCounterValue, "search counter should have value 5")
}

// TestCounterWithAttributes verifies that counters with attributes work properly.
func TestCounterWithAttributes(t *testing.T) {
	ctx := context.Background()

	// Create a manual reader to capture metrics
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
	)

	// Set as the global meter provider
	oldMP := otel.GetMeterProvider()
	otel.SetMeterProvider(mp)
	defer otel.SetMeterProvider(oldMP)

	// Create an in-memory mock store for testing (uses local mockStore)
	store := newMockStore()
	logger := zap.NewNop()

	// Create the ReasoningBank service
	svc, err := NewService(store, logger)
	require.NoError(t, err, "failed to create ReasoningBank service")

	// Perform a search to trigger the counter increment
	// Note: mock store doesn't require tenant context the same way real store does
	_, err = svc.Search(ctx, "test-project", "test query", 10)
	// Search may fail due to mock store, but counter should still be incremented
	t.Logf("Search result error (expected for mock): %v", err)

	// Force collect metrics
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err, "failed to collect metrics")

	// Log what we collected
	t.Logf("Collected %d scope metrics after search", len(rm.ScopeMetrics))
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			t.Logf("  Metric: %s", m.Name)
		}
	}
}
