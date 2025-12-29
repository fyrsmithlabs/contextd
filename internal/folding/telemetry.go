// Package folding provides context-folding for LLM agent context management.
package folding

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	// InstrumentationName is the name used for OTEL instrumentation.
	InstrumentationName = "github.com/fyrsmithlabs/contextd/internal/folding"
)

// Metrics provides OpenTelemetry metrics for the folding package.
type Metrics struct {
	// Counters
	branchCreatedTotal  metric.Int64Counter
	branchReturnedTotal metric.Int64Counter
	branchTimeoutTotal  metric.Int64Counter
	branchFailedTotal   metric.Int64Counter

	// Gauges (using UpDownCounter for gauge semantics)
	branchActiveCount metric.Int64UpDownCounter

	// Histograms
	branchDuration     metric.Float64Histogram
	budgetConsumed     metric.Int64Histogram
	budgetUtilization  metric.Float64Histogram

	// initialized tracks if metrics were successfully initialized
	initialized bool
}

// NewMetrics creates a new Metrics instance with the provided meter.
// If meter is nil, uses the global meter provider.
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	if meter == nil {
		meter = otel.Meter(InstrumentationName)
	}

	m := &Metrics{}
	var err error

	// Counters
	m.branchCreatedTotal, err = meter.Int64Counter(
		"folding.branch.created.total",
		metric.WithDescription("Total number of branches created"),
		metric.WithUnit("{branch}"),
	)
	if err != nil {
		return nil, err
	}

	m.branchReturnedTotal, err = meter.Int64Counter(
		"folding.branch.returned.total",
		metric.WithDescription("Total number of branches returned successfully"),
		metric.WithUnit("{branch}"),
	)
	if err != nil {
		return nil, err
	}

	m.branchTimeoutTotal, err = meter.Int64Counter(
		"folding.branch.timeout.total",
		metric.WithDescription("Total number of branches that timed out"),
		metric.WithUnit("{branch}"),
	)
	if err != nil {
		return nil, err
	}

	m.branchFailedTotal, err = meter.Int64Counter(
		"folding.branch.failed.total",
		metric.WithDescription("Total number of branches that failed"),
		metric.WithUnit("{branch}"),
	)
	if err != nil {
		return nil, err
	}

	// Gauges
	m.branchActiveCount, err = meter.Int64UpDownCounter(
		"folding.branch.active.count",
		metric.WithDescription("Number of currently active branches"),
		metric.WithUnit("{branch}"),
	)
	if err != nil {
		return nil, err
	}

	// Histograms
	m.branchDuration, err = meter.Float64Histogram(
		"folding.branch.duration.seconds",
		metric.WithDescription("Duration of branch execution in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.5, 1, 5, 10, 30, 60, 120, 300, 600),
	)
	if err != nil {
		return nil, err
	}

	m.budgetConsumed, err = meter.Int64Histogram(
		"folding.budget.consumed.tokens",
		metric.WithDescription("Tokens consumed per branch"),
		metric.WithUnit("{token}"),
		metric.WithExplicitBucketBoundaries(100, 500, 1000, 2000, 4000, 8000, 16000, 32000),
	)
	if err != nil {
		return nil, err
	}

	m.budgetUtilization, err = meter.Float64Histogram(
		"folding.budget.utilization.ratio",
		metric.WithDescription("Budget utilization ratio (tokens used / budget allocated)"),
		metric.WithUnit("1"),
		metric.WithExplicitBucketBoundaries(0.1, 0.2, 0.4, 0.6, 0.8, 0.9, 0.95, 1.0),
	)
	if err != nil {
		return nil, err
	}

	m.initialized = true
	return m, nil
}

// RecordBranchCreated records a branch creation.
// Note: session_id is intentionally omitted from metrics to prevent cardinality explosion (SEC-FOLD-001).
// Session correlation is available via trace context and structured logs.
func (m *Metrics) RecordBranchCreated(ctx context.Context, sessionID string, depth int, budget int) {
	if m == nil || !m.initialized {
		return
	}
	attrs := metric.WithAttributes(
		attribute.Int("depth", depth),
	)
	m.branchCreatedTotal.Add(ctx, 1, attrs)
	m.branchActiveCount.Add(ctx, 1, attrs)
}

// RecordBranchReturned records a successful branch return.
// Note: session_id is intentionally omitted from metrics to prevent cardinality explosion (SEC-FOLD-001).
func (m *Metrics) RecordBranchReturned(ctx context.Context, sessionID string, depth int, tokensUsed int, budget int, duration time.Duration) {
	if m == nil || !m.initialized {
		return
	}
	attrs := metric.WithAttributes(
		attribute.Int("depth", depth),
	)
	m.branchReturnedTotal.Add(ctx, 1, attrs)
	m.branchActiveCount.Add(ctx, -1, attrs)
	m.branchDuration.Record(ctx, duration.Seconds(), attrs)
	m.budgetConsumed.Record(ctx, int64(tokensUsed), attrs)
	if budget > 0 {
		m.budgetUtilization.Record(ctx, float64(tokensUsed)/float64(budget), attrs)
	}
}

// RecordBranchTimeout records a branch timeout.
// Note: session_id is intentionally omitted from metrics to prevent cardinality explosion (SEC-FOLD-001).
func (m *Metrics) RecordBranchTimeout(ctx context.Context, sessionID string, depth int, tokensUsed int, budget int, duration time.Duration) {
	if m == nil || !m.initialized {
		return
	}
	attrs := metric.WithAttributes(
		attribute.Int("depth", depth),
	)
	m.branchTimeoutTotal.Add(ctx, 1, attrs)
	m.branchActiveCount.Add(ctx, -1, attrs)
	m.branchDuration.Record(ctx, duration.Seconds(), attrs)
	m.budgetConsumed.Record(ctx, int64(tokensUsed), attrs)
	if budget > 0 {
		m.budgetUtilization.Record(ctx, float64(tokensUsed)/float64(budget), attrs)
	}
}

// RecordBranchFailed records a branch failure.
// Note: session_id is intentionally omitted from metrics to prevent cardinality explosion (SEC-FOLD-001).
func (m *Metrics) RecordBranchFailed(ctx context.Context, sessionID string, depth int, reason string, tokensUsed int, budget int, duration time.Duration) {
	if m == nil || !m.initialized {
		return
	}
	attrs := metric.WithAttributes(
		attribute.Int("depth", depth),
		attribute.String("failure_reason", reason),
	)
	m.branchFailedTotal.Add(ctx, 1, attrs)
	m.branchActiveCount.Add(ctx, -1, attrs)
	m.branchDuration.Record(ctx, duration.Seconds(), attrs)
	m.budgetConsumed.Record(ctx, int64(tokensUsed), attrs)
	if budget > 0 {
		m.budgetUtilization.Record(ctx, float64(tokensUsed)/float64(budget), attrs)
	}
}

// Tracer returns a tracer for the folding package.
func Tracer() trace.Tracer {
	return otel.Tracer(InstrumentationName)
}

// SpanAttributes returns common span attributes for a branch.
func SpanAttributes(branchID, sessionID string, depth int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("folding.branch_id", branchID),
		attribute.String("folding.session_id", sessionID),
		attribute.Int("folding.depth", depth),
	}
}

// StartSpan starts a new span with branch context.
func StartSpan(ctx context.Context, name string, branchID, sessionID string, depth int, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	attrs := SpanAttributes(branchID, sessionID, depth)
	allOpts := append([]trace.SpanStartOption{trace.WithAttributes(attrs...)}, opts...)
	return Tracer().Start(ctx, name, allOpts...)
}

// SpanFromContext returns the current span from context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// RecordError records an error on the current span.
func RecordError(ctx context.Context, err error, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err, trace.WithAttributes(attrs...))
	}
}

// SetSpanStatus sets the status on the current span.
func SetSpanStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetStatus(code, description)
	}
}
