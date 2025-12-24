// Package framework provides the integration test harness for contextd.
package framework

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/fyrsmithlabs/contextd/test/integration/framework"

// TestMetrics provides observability for integration tests.
type TestMetrics struct {
	tracer trace.Tracer
	meter  metric.Meter

	// Counters
	testPassCounter metric.Int64Counter
	testFailCounter metric.Int64Counter

	// Histograms
	suiteDuration          metric.Float64Histogram
	memorySearchLatency    metric.Float64Histogram
	checkpointSaveLatency  metric.Float64Histogram
	checkpointLoadLatency  metric.Float64Histogram

	// Gauges (using callbacks for real-time values)
	mu                sync.RWMutex
	memoryHitCount    int64
	memoryMissCount   int64
	checkpointSuccess int64
	checkpointFailure int64
	crossDevSearches  int64
	totalSearches     int64
}

// NewTestMetrics creates a new TestMetrics instance.
func NewTestMetrics() (*TestMetrics, error) {
	tracer := otel.Tracer(instrumentationName)
	meter := otel.Meter(instrumentationName)

	m := &TestMetrics{
		tracer: tracer,
		meter:  meter,
	}

	var err error

	// Test pass/fail counters
	m.testPassCounter, err = meter.Int64Counter(
		"contextd.test.pass_total",
		metric.WithDescription("Total number of passed tests"),
		metric.WithUnit("{test}"),
	)
	if err != nil {
		return nil, err
	}

	m.testFailCounter, err = meter.Int64Counter(
		"contextd.test.fail_total",
		metric.WithDescription("Total number of failed tests"),
		metric.WithUnit("{test}"),
	)
	if err != nil {
		return nil, err
	}

	// Duration histograms
	m.suiteDuration, err = meter.Float64Histogram(
		"contextd.test.suite_duration_seconds",
		metric.WithDescription("Time to complete each test suite"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	m.memorySearchLatency, err = meter.Float64Histogram(
		"contextd.test.memory_search_latency_ms",
		metric.WithDescription("Time to search for relevant memories"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	m.checkpointSaveLatency, err = meter.Float64Histogram(
		"contextd.test.checkpoint_save_latency_ms",
		metric.WithDescription("Time to save a checkpoint"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	m.checkpointLoadLatency, err = meter.Float64Histogram(
		"contextd.test.checkpoint_load_latency_ms",
		metric.WithDescription("Time to load from a checkpoint"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	// Register gauge callbacks
	_, err = meter.Float64ObservableGauge(
		"contextd.test.memory_hit_rate",
		metric.WithDescription("Percentage of memory searches that return useful results"),
		metric.WithUnit("1"),
		metric.WithFloat64Callback(m.memoryHitRateCallback),
	)
	if err != nil {
		return nil, err
	}

	_, err = meter.Float64ObservableGauge(
		"contextd.test.checkpoint_success_rate",
		metric.WithDescription("Percentage of successful checkpoint loads"),
		metric.WithUnit("1"),
		metric.WithFloat64Callback(m.checkpointSuccessRateCallback),
	)
	if err != nil {
		return nil, err
	}

	_, err = meter.Float64ObservableGauge(
		"contextd.test.cross_dev_search_rate",
		metric.WithDescription("How often Dev B finds Dev A's knowledge"),
		metric.WithUnit("1"),
		metric.WithFloat64Callback(m.crossDevSearchRateCallback),
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// memoryHitRateCallback calculates the memory hit rate.
func (m *TestMetrics) memoryHitRateCallback(_ context.Context, observer metric.Float64Observer) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.memoryHitCount + m.memoryMissCount
	if total == 0 {
		observer.Observe(0)
		return nil
	}
	rate := float64(m.memoryHitCount) / float64(total)
	observer.Observe(rate)
	return nil
}

// checkpointSuccessRateCallback calculates the checkpoint success rate.
func (m *TestMetrics) checkpointSuccessRateCallback(_ context.Context, observer metric.Float64Observer) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.checkpointSuccess + m.checkpointFailure
	if total == 0 {
		observer.Observe(0)
		return nil
	}
	rate := float64(m.checkpointSuccess) / float64(total)
	observer.Observe(rate)
	return nil
}

// crossDevSearchRateCallback calculates the cross-developer search rate.
func (m *TestMetrics) crossDevSearchRateCallback(_ context.Context, observer metric.Float64Observer) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.totalSearches == 0 {
		observer.Observe(0)
		return nil
	}
	rate := float64(m.crossDevSearches) / float64(m.totalSearches)
	observer.Observe(rate)
	return nil
}

// StartSuiteSpan starts a trace span for a test suite.
func (m *TestMetrics) StartSuiteSpan(ctx context.Context, suiteName string) (context.Context, trace.Span) {
	return m.tracer.Start(ctx, "suite_execution",
		trace.WithAttributes(attribute.String("suite", suiteName)),
	)
}

// StartTestSpan starts a trace span for a single test.
func (m *TestMetrics) StartTestSpan(ctx context.Context, testName string) (context.Context, trace.Span) {
	return m.tracer.Start(ctx, "test_execution",
		trace.WithAttributes(attribute.String("test", testName)),
	)
}

// StartPhaseSpan starts a trace span for a test phase.
func (m *TestMetrics) StartPhaseSpan(ctx context.Context, phase string) (context.Context, trace.Span) {
	return m.tracer.Start(ctx, phase+"_phase")
}

// RecordTestPass records a passed test.
func (m *TestMetrics) RecordTestPass(ctx context.Context, suite, testName string) {
	m.testPassCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("suite", suite),
			attribute.String("test", testName),
		),
	)
}

// RecordTestFail records a failed test.
func (m *TestMetrics) RecordTestFail(ctx context.Context, suite, testName string) {
	m.testFailCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("suite", suite),
			attribute.String("test", testName),
		),
	)
}

// RecordSuiteDuration records the duration of a test suite.
func (m *TestMetrics) RecordSuiteDuration(ctx context.Context, suiteName string, duration time.Duration) {
	m.suiteDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(attribute.String("suite", suiteName)),
	)
}

// RecordMemorySearch records a memory search operation.
func (m *TestMetrics) RecordMemorySearch(ctx context.Context, latency time.Duration, hit bool, crossDev bool) {
	m.memorySearchLatency.Record(ctx, float64(latency.Milliseconds()))

	m.mu.Lock()
	defer m.mu.Unlock()

	if hit {
		m.memoryHitCount++
	} else {
		m.memoryMissCount++
	}

	m.totalSearches++
	if crossDev {
		m.crossDevSearches++
	}
}

// RecordCheckpointSave records a checkpoint save operation.
func (m *TestMetrics) RecordCheckpointSave(ctx context.Context, latency time.Duration) {
	m.checkpointSaveLatency.Record(ctx, float64(latency.Milliseconds()))
}

// RecordCheckpointLoad records a checkpoint load operation.
func (m *TestMetrics) RecordCheckpointLoad(ctx context.Context, latency time.Duration, success bool) {
	m.checkpointLoadLatency.Record(ctx, float64(latency.Milliseconds()))

	m.mu.Lock()
	defer m.mu.Unlock()

	if success {
		m.checkpointSuccess++
	} else {
		m.checkpointFailure++
	}
}

// RecordConfidenceScore records a confidence score observation.
func (m *TestMetrics) RecordConfidenceScore(ctx context.Context, span trace.Span, score float64) {
	span.SetAttributes(attribute.Float64("confidence_score", score))
}

// GetStats returns current metric statistics (for testing/debugging).
func (m *TestMetrics) GetStats() MetricStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hitRate := float64(0)
	if total := m.memoryHitCount + m.memoryMissCount; total > 0 {
		hitRate = float64(m.memoryHitCount) / float64(total)
	}

	checkpointRate := float64(0)
	if total := m.checkpointSuccess + m.checkpointFailure; total > 0 {
		checkpointRate = float64(m.checkpointSuccess) / float64(total)
	}

	crossDevRate := float64(0)
	if m.totalSearches > 0 {
		crossDevRate = float64(m.crossDevSearches) / float64(m.totalSearches)
	}

	return MetricStats{
		MemoryHitCount:         m.memoryHitCount,
		MemoryMissCount:        m.memoryMissCount,
		MemoryHitRate:          hitRate,
		CheckpointSuccessCount: m.checkpointSuccess,
		CheckpointFailureCount: m.checkpointFailure,
		CheckpointSuccessRate:  checkpointRate,
		CrossDevSearchCount:    m.crossDevSearches,
		TotalSearchCount:       m.totalSearches,
		CrossDevSearchRate:     crossDevRate,
	}
}

// MetricStats holds a snapshot of metric statistics.
type MetricStats struct {
	MemoryHitCount         int64
	MemoryMissCount        int64
	MemoryHitRate          float64
	CheckpointSuccessCount int64
	CheckpointFailureCount int64
	CheckpointSuccessRate  float64
	CrossDevSearchCount    int64
	TotalSearchCount       int64
	CrossDevSearchRate     float64
}

// Reset resets all counters (useful for testing).
func (m *TestMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.memoryHitCount = 0
	m.memoryMissCount = 0
	m.checkpointSuccess = 0
	m.checkpointFailure = 0
	m.crossDevSearches = 0
	m.totalSearches = 0
}
