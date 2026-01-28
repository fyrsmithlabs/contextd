package vectorstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

func TestMetrics_RecordOperation(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	logger := zap.NewNop()
	m := &Metrics{
		meter:  mp.Meter(vectorstoreInstrumentationName),
		logger: logger,
	}
	m.init()

	ctx := context.Background()

	// Test successful operation
	m.RecordOperation(ctx, "search", "test_collection", 100*time.Millisecond, nil)

	// Test operation with error
	m.RecordOperation(ctx, "add_documents", "test_collection", 50*time.Millisecond, errors.New("embedding failed"))

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
	foundDuration := false
	foundErrors := false

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch m.Name {
			case "contextd.vectorstore.operation_duration_seconds":
				foundDuration = true
				if hist, ok := m.Data.(metricdata.Histogram[float64]); ok {
					total := uint64(0)
					for _, dp := range hist.DataPoints {
						total += dp.Count
					}
					if total != 2 {
						t.Errorf("expected 2 duration recordings, got %d", total)
					}
				}
			case "contextd.vectorstore.errors_total":
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

	if !foundDuration {
		t.Error("duration histogram not found")
	}
	if !foundErrors {
		t.Error("errors counter not found")
	}
}

func TestMetrics_RecordDocuments(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	logger := zap.NewNop()
	m := &Metrics{
		meter:  mp.Meter(vectorstoreInstrumentationName),
		logger: logger,
	}
	m.init()

	ctx := context.Background()

	// Record document additions
	m.RecordDocuments(ctx, "add", "memories", 10)
	m.RecordDocuments(ctx, "delete", "memories", 5)

	// Collect metrics
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	// Check for documents counter
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "contextd.vectorstore.documents_total" {
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					total := int64(0)
					for _, dp := range sum.DataPoints {
						total += dp.Value
					}
					if total != 15 {
						t.Errorf("expected 15 total documents, got %d", total)
					}
				}
				return
			}
		}
	}
	t.Error("documents counter not found")
}

func TestMetrics_RecordSearchResults(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	logger := zap.NewNop()
	m := &Metrics{
		meter:  mp.Meter(vectorstoreInstrumentationName),
		logger: logger,
	}
	m.init()

	ctx := context.Background()

	// Record search results
	m.RecordSearchResults(ctx, "memories", 5)
	m.RecordSearchResults(ctx, "memories", 10)
	m.RecordSearchResults(ctx, "remediations", 0)

	// Collect metrics
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	// Check for search results histogram
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "contextd.vectorstore.search_results" {
				if hist, ok := m.Data.(metricdata.Histogram[int64]); ok {
					total := uint64(0)
					for _, dp := range hist.DataPoints {
						total += dp.Count
					}
					if total != 3 {
						t.Errorf("expected 3 search result recordings, got %d", total)
					}
				}
				return
			}
		}
	}
	t.Error("search results histogram not found")
}
