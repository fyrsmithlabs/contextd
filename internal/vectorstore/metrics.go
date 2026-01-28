// Package vectorstore provides vector storage with metrics instrumentation.
package vectorstore

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

const vectorstoreInstrumentationName = "github.com/fyrsmithlabs/contextd/internal/vectorstore"

// Metrics holds all vectorstore-related metrics.
type Metrics struct {
	meter         metric.Meter
	logger        *zap.Logger
	opDuration    metric.Float64Histogram
	documentsOp   metric.Int64Counter
	searchResults metric.Int64Histogram
	errors        metric.Int64Counter
}

// NewMetrics creates a new Metrics instance for vectorstore.
func NewMetrics(logger *zap.Logger) *Metrics {
	m := &Metrics{
		meter:  otel.Meter(vectorstoreInstrumentationName),
		logger: logger,
	}
	m.init()
	return m
}

func (m *Metrics) init() {
	var err error

	// Operation duration by collection and operation type
	m.opDuration, err = m.meter.Float64Histogram(
		"contextd.vectorstore.operation_duration_seconds",
		metric.WithDescription("Duration of vectorstore operations"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0),
	)
	if err != nil {
		m.logger.Warn("failed to create operation duration histogram", zap.Error(err))
	}

	// Document count for add/delete operations
	m.documentsOp, err = m.meter.Int64Counter(
		"contextd.vectorstore.documents_total",
		metric.WithDescription("Total documents processed by operation type"),
		metric.WithUnit("{document}"),
	)
	if err != nil {
		m.logger.Warn("failed to create documents counter", zap.Error(err))
	}

	// Search results count histogram
	m.searchResults, err = m.meter.Int64Histogram(
		"contextd.vectorstore.search_results",
		metric.WithDescription("Number of results returned by search operations"),
		metric.WithUnit("{result}"),
		metric.WithExplicitBucketBoundaries(0, 1, 5, 10, 25, 50, 100),
	)
	if err != nil {
		m.logger.Warn("failed to create search results histogram", zap.Error(err))
	}

	// Error count by operation
	m.errors, err = m.meter.Int64Counter(
		"contextd.vectorstore.errors_total",
		metric.WithDescription("Total number of vectorstore errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		m.logger.Warn("failed to create errors counter", zap.Error(err))
	}
}

// RecordOperation records a vectorstore operation metric.
func (m *Metrics) RecordOperation(ctx context.Context, op, collection string, duration time.Duration, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("operation", op),
		attribute.String("collection", collection),
	}

	// Record duration
	if m.opDuration != nil {
		m.opDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}

	// Record error if present
	if err != nil && m.errors != nil {
		m.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordDocuments records document count for add/delete operations.
func (m *Metrics) RecordDocuments(ctx context.Context, op, collection string, count int) {
	if m.documentsOp != nil {
		m.documentsOp.Add(ctx, int64(count), metric.WithAttributes(
			attribute.String("operation", op),
			attribute.String("collection", collection),
		))
	}
}

// RecordSearchResults records the number of search results returned.
func (m *Metrics) RecordSearchResults(ctx context.Context, collection string, count int) {
	if m.searchResults != nil {
		m.searchResults.Record(ctx, int64(count), metric.WithAttributes(
			attribute.String("collection", collection),
		))
	}
}

// Global metrics instance for health check functions
var globalMetrics *Metrics

func init() {
	globalMetrics = NewMetrics(zap.NewNop())
}

// RecordHealthCheckResult records whether a health check succeeded or failed.
func RecordHealthCheckResult(success bool) {
	// No-op placeholder for future health check metrics
}

// UpdateHealthMetrics updates metrics based on health check results.
func UpdateHealthMetrics(health *MetadataHealth) {
	// No-op placeholder for future health check metrics
}

// RecordQuarantineResult records whether a quarantine operation succeeded or failed.
func RecordQuarantineResult(success bool) {
	// No-op placeholder for future quarantine metrics
}
