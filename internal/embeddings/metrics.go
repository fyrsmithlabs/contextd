// Package embeddings provides embedding generation with metrics instrumentation.
package embeddings

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

const embeddingsInstrumentationName = "github.com/fyrsmithlabs/contextd/internal/embeddings"

// Metrics holds all embedding-related metrics.
type Metrics struct {
	meter     metric.Meter
	logger    *zap.Logger
	duration  metric.Float64Histogram
	batchSize metric.Int64Histogram
	errors    metric.Int64Counter
}

// NewMetrics creates a new Metrics instance for embeddings.
func NewMetrics(logger *zap.Logger) *Metrics {
	m := &Metrics{
		meter:  otel.Meter(embeddingsInstrumentationName),
		logger: logger,
	}
	m.init()
	return m
}

func (m *Metrics) init() {
	var err error

	// Embedding generation duration by model and operation
	m.duration, err = m.meter.Float64Histogram(
		"contextd.embedding.generation_duration_seconds",
		metric.WithDescription("Duration of embedding generation in seconds, labeled by model (e.g., all-MiniLM-L6-v2) and operation (embed, batch_embed)"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0),
	)
	if err != nil {
		m.logger.Warn("failed to create duration histogram", zap.Error(err))
	}

	// Batch size histogram to understand embedding workload patterns
	m.batchSize, err = m.meter.Int64Histogram(
		"contextd.embedding.batch_size",
		metric.WithDescription("Number of texts per embedding batch request. Useful for optimizing batch sizes: too small wastes overhead, too large increases latency."),
		metric.WithUnit("{text}"),
		metric.WithExplicitBucketBoundaries(1, 2, 5, 10, 25, 50, 100, 250, 500),
	)
	if err != nil {
		m.logger.Warn("failed to create batch size histogram", zap.Error(err))
	}

	// Error count by model and operation
	m.errors, err = m.meter.Int64Counter(
		"contextd.embedding.errors_total",
		metric.WithDescription("Total embedding generation errors by model and operation. Includes model loading failures, ONNX runtime errors, and batch processing failures."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		m.logger.Warn("failed to create errors counter", zap.Error(err))
	}
}

// RecordGeneration records embedding generation metrics.
func (m *Metrics) RecordGeneration(ctx context.Context, model, operation string, duration time.Duration, batchSize int, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("model", model),
		attribute.String("operation", operation),
	}

	// Record duration
	if m.duration != nil {
		m.duration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}

	// Record batch size for batch operations
	if batchSize > 0 && m.batchSize != nil {
		m.batchSize.Record(ctx, int64(batchSize), metric.WithAttributes(attrs...))
	}

	// Record error if present
	if err != nil && m.errors != nil {
		m.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}
