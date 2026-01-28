package embeddings

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

func TestMetrics_RecordGeneration(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	logger := zap.NewNop()
	m := &Metrics{
		meter:  mp.Meter(embeddingsInstrumentationName),
		logger: logger,
	}
	m.init()

	ctx := context.Background()

	// Test successful batch generation
	m.RecordGeneration(ctx, "BAAI/bge-small-en-v1.5", "embed_documents", 100*time.Millisecond, 10, nil)

	// Test successful query generation
	m.RecordGeneration(ctx, "BAAI/bge-small-en-v1.5", "embed_query", 50*time.Millisecond, 1, nil)

	// Test generation with error
	m.RecordGeneration(ctx, "BAAI/bge-small-en-v1.5", "embed_documents", 25*time.Millisecond, 5, errors.New("generation failed"))

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
	foundBatchSize := false
	foundErrors := false

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch m.Name {
			case "contextd.embedding.generation_duration_seconds":
				foundDuration = true
				if hist, ok := m.Data.(metricdata.Histogram[float64]); ok {
					total := uint64(0)
					for _, dp := range hist.DataPoints {
						total += dp.Count
					}
					if total != 3 {
						t.Errorf("expected 3 duration recordings, got %d", total)
					}
				}
			case "contextd.embedding.batch_size":
				foundBatchSize = true
				if hist, ok := m.Data.(metricdata.Histogram[int64]); ok {
					total := uint64(0)
					for _, dp := range hist.DataPoints {
						total += dp.Count
					}
					if total != 3 {
						t.Errorf("expected 3 batch size recordings, got %d", total)
					}
				}
			case "contextd.embedding.errors_total":
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
	if !foundBatchSize {
		t.Error("batch size histogram not found")
	}
	if !foundErrors {
		t.Error("errors counter not found")
	}
}

func TestMetrics_BatchSizeLabels(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	logger := zap.NewNop()
	m := &Metrics{
		meter:  mp.Meter(embeddingsInstrumentationName),
		logger: logger,
	}
	m.init()

	ctx := context.Background()

	// Record for different models and operations
	m.RecordGeneration(ctx, "BAAI/bge-small-en-v1.5", "embed_documents", 100*time.Millisecond, 10, nil)
	m.RecordGeneration(ctx, "BAAI/bge-base-en-v1.5", "embed_documents", 150*time.Millisecond, 20, nil)
	m.RecordGeneration(ctx, "BAAI/bge-small-en-v1.5", "embed_query", 50*time.Millisecond, 1, nil)

	// Collect metrics
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	// We should have metrics with model and operation attributes
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "contextd.embedding.generation_duration_seconds" {
				if hist, ok := m.Data.(metricdata.Histogram[float64]); ok {
					// Should have data points with different model/operation combinations
					if len(hist.DataPoints) < 2 {
						t.Errorf("expected at least 2 data points for different model/operation combinations, got %d", len(hist.DataPoints))
					}
				}
			}
		}
	}
}
