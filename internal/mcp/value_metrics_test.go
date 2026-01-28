package mcp

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

func TestHashProjectID(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		wantLen   int
	}{
		{"empty string returns unknown", "", 7}, // "unknown" is 7 chars
		{"normal project ID", "my-project", 8},  // 8 hex chars
		{"same input gives same output", "test-project", 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hashProjectID(tt.projectID)
			if len(result) != tt.wantLen {
				t.Errorf("hashProjectID(%q) = %q (len %d), want len %d", tt.projectID, result, len(result), tt.wantLen)
			}
		})
	}

	// Test determinism
	hash1 := hashProjectID("my-project")
	hash2 := hashProjectID("my-project")
	if hash1 != hash2 {
		t.Errorf("hashProjectID is not deterministic: %q != %q", hash1, hash2)
	}

	// Test different inputs give different outputs
	hashA := hashProjectID("project-a")
	hashB := hashProjectID("project-b")
	if hashA == hashB {
		t.Errorf("different inputs should give different hashes: %q == %q", hashA, hashB)
	}
}

func TestValueMetrics_RecordTokensSaved(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	logger := zap.NewNop()
	m := &ValueMetrics{
		meter:  mp.Meter(valueInstrumentationName),
		logger: logger,
	}
	m.init()

	ctx := context.Background()

	// Record tokens saved
	m.RecordTokensSaved(ctx, 1000, 200) // Saved 800 tokens
	m.RecordTokensSaved(ctx, 500, 100)  // Saved 400 tokens
	m.RecordTokensSaved(ctx, 100, 150)  // Negative (no save)

	// Collect metrics
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	// Find tokens saved counter
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "contextd.context.tokens_saved_total" {
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					total := int64(0)
					for _, dp := range sum.DataPoints {
						total += dp.Value
					}
					// 800 + 400 = 1200 (negative not counted)
					if total != 1200 {
						t.Errorf("expected 1200 tokens saved, got %d", total)
					}
				}
				return
			}
		}
	}
	t.Error("tokens saved counter not found")
}

func TestValueMetrics_RecordMemoryOutcome(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	logger := zap.NewNop()
	m := &ValueMetrics{
		meter:  mp.Meter(valueInstrumentationName),
		logger: logger,
	}
	m.init()

	ctx := context.Background()

	// Record outcomes
	m.RecordMemoryOutcome(ctx, true, "project1")
	m.RecordMemoryOutcome(ctx, true, "project1")
	m.RecordMemoryOutcome(ctx, false, "project1")

	// Collect metrics
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	foundSuccess := false
	foundFailure := false

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch m.Name {
			case "contextd.memory.retrieval_success_total":
				foundSuccess = true
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					total := int64(0)
					for _, dp := range sum.DataPoints {
						total += dp.Value
					}
					if total != 2 {
						t.Errorf("expected 2 successes, got %d", total)
					}
				}
			case "contextd.memory.retrieval_failure_total":
				foundFailure = true
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					total := int64(0)
					for _, dp := range sum.DataPoints {
						total += dp.Value
					}
					if total != 1 {
						t.Errorf("expected 1 failure, got %d", total)
					}
				}
			}
		}
	}

	if !foundSuccess {
		t.Error("success counter not found")
	}
	if !foundFailure {
		t.Error("failure counter not found")
	}
}

func TestValueMetrics_RecordCheckpointUtilization(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	logger := zap.NewNop()
	m := &ValueMetrics{
		meter:  mp.Meter(valueInstrumentationName),
		logger: logger,
	}
	m.init()

	ctx := context.Background()

	// Record checkpoint events
	m.RecordCheckpointCreated(ctx, "project1", false)
	m.RecordCheckpointCreated(ctx, "project1", true)
	m.RecordCheckpointResumed(ctx, "project1", "summary")

	// Collect metrics
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	foundCreated := false
	foundResumed := false

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch m.Name {
			case "contextd.checkpoint.created_total":
				foundCreated = true
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					total := int64(0)
					for _, dp := range sum.DataPoints {
						total += dp.Value
					}
					if total != 2 {
						t.Errorf("expected 2 created, got %d", total)
					}
				}
			case "contextd.checkpoint.resumed_total":
				foundResumed = true
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					total := int64(0)
					for _, dp := range sum.DataPoints {
						total += dp.Value
					}
					if total != 1 {
						t.Errorf("expected 1 resumed, got %d", total)
					}
				}
			}
		}
	}

	if !foundCreated {
		t.Error("created counter not found")
	}
	if !foundResumed {
		t.Error("resumed counter not found")
	}
}
