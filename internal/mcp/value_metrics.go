// Package mcp provides value demonstration metrics for contextd.
package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

const valueInstrumentationName = "github.com/fyrsmithlabs/contextd/value"

// ValueMetrics tracks business value metrics for contextd.
// These metrics demonstrate the actual value delivered to users.
type ValueMetrics struct {
	meter  metric.Meter
	logger *zap.Logger

	// Tokens saved via context compression/folding
	tokensSaved metric.Int64Counter

	// Memory outcome tracking
	memoryRetrievalSuccess metric.Int64Counter
	memoryRetrievalFailure metric.Int64Counter

	// Checkpoint utilization tracking
	checkpointCreated metric.Int64Counter
	checkpointResumed metric.Int64Counter

	mu          sync.RWMutex
	initialized bool
}

var (
	globalValueMetrics *ValueMetrics
	valueMetricsOnce   sync.Once
)

// GetValueMetrics returns the global ValueMetrics instance.
func GetValueMetrics(logger *zap.Logger) *ValueMetrics {
	valueMetricsOnce.Do(func() {
		globalValueMetrics = newValueMetrics(logger)
	})
	return globalValueMetrics
}

func newValueMetrics(logger *zap.Logger) *ValueMetrics {
	if logger == nil {
		logger = zap.NewNop()
	}

	m := &ValueMetrics{
		meter:  otel.Meter(valueInstrumentationName),
		logger: logger,
	}
	m.init()
	return m
}

func (m *ValueMetrics) init() {
	var err error

	// Tokens saved by context compression - THE key value metric
	m.tokensSaved, err = m.meter.Int64Counter(
		"contextd.context.tokens_saved_total",
		metric.WithDescription("Total tokens saved via context compression and folding"),
		metric.WithUnit("{token}"),
	)
	if err != nil {
		m.logger.Warn("failed to create tokens saved counter", zap.Error(err))
	}

	// Memory retrieval outcomes - tracks if memories are actually helpful
	m.memoryRetrievalSuccess, err = m.meter.Int64Counter(
		"contextd.memory.retrieval_success_total",
		metric.WithDescription("Total memory retrievals that led to successful outcomes"),
		metric.WithUnit("{retrieval}"),
	)
	if err != nil {
		m.logger.Warn("failed to create memory success counter", zap.Error(err))
	}

	m.memoryRetrievalFailure, err = m.meter.Int64Counter(
		"contextd.memory.retrieval_failure_total",
		metric.WithDescription("Total memory retrievals that did not help"),
		metric.WithUnit("{retrieval}"),
	)
	if err != nil {
		m.logger.Warn("failed to create memory failure counter", zap.Error(err))
	}

	// Checkpoint utilization - tracks if checkpoints are being used
	m.checkpointCreated, err = m.meter.Int64Counter(
		"contextd.checkpoint.created_total",
		metric.WithDescription("Total checkpoints created"),
		metric.WithUnit("{checkpoint}"),
	)
	if err != nil {
		m.logger.Warn("failed to create checkpoint created counter", zap.Error(err))
	}

	m.checkpointResumed, err = m.meter.Int64Counter(
		"contextd.checkpoint.resumed_total",
		metric.WithDescription("Total checkpoints resumed"),
		metric.WithUnit("{checkpoint}"),
	)
	if err != nil {
		m.logger.Warn("failed to create checkpoint resumed counter", zap.Error(err))
	}

	m.mu.Lock()
	m.initialized = true
	m.mu.Unlock()
}

// hashProjectID creates a truncated hash of the project ID to prevent enumeration.
// This provides privacy while still allowing per-project metric aggregation.
func hashProjectID(projectID string) string {
	if projectID == "" {
		return "unknown"
	}
	hash := sha256.Sum256([]byte(projectID))
	// Use first 8 characters of hex-encoded hash for reasonable cardinality
	return hex.EncodeToString(hash[:4])
}

// isInitialized safely checks if the metrics are initialized.
func (m *ValueMetrics) isInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized
}

// RecordTokensSaved records tokens saved via context compression.
// inputTokens is the original size, outputTokens is the compressed size.
func (m *ValueMetrics) RecordTokensSaved(ctx context.Context, inputTokens, outputTokens int) {
	if m == nil || !m.isInitialized() {
		return
	}

	m.mu.RLock()
	counter := m.tokensSaved
	m.mu.RUnlock()

	if counter == nil {
		return
	}

	saved := inputTokens - outputTokens
	if saved > 0 {
		counter.Add(ctx, int64(saved))
	}
}

// RecordMemoryOutcome records whether a memory retrieval was successful.
// Project IDs are hashed before being used as metric labels to prevent enumeration.
func (m *ValueMetrics) RecordMemoryOutcome(ctx context.Context, success bool, projectID string) {
	if m == nil || !m.isInitialized() {
		return
	}

	// Hash project ID to prevent enumeration attacks
	hashedID := hashProjectID(projectID)
	attrs := metric.WithAttributes(
		attribute.String("project_id", hashedID),
	)

	m.mu.RLock()
	successCounter := m.memoryRetrievalSuccess
	failureCounter := m.memoryRetrievalFailure
	m.mu.RUnlock()

	if success {
		if successCounter != nil {
			successCounter.Add(ctx, 1, attrs)
		}
	} else {
		if failureCounter != nil {
			failureCounter.Add(ctx, 1, attrs)
		}
	}
}

// RecordCheckpointCreated records a checkpoint creation.
// Project IDs are hashed before being used as metric labels to prevent enumeration.
func (m *ValueMetrics) RecordCheckpointCreated(ctx context.Context, projectID string, autoCreated bool) {
	if m == nil || !m.isInitialized() {
		return
	}

	m.mu.RLock()
	counter := m.checkpointCreated
	m.mu.RUnlock()

	if counter == nil {
		return
	}

	// Hash project ID to prevent enumeration attacks
	hashedID := hashProjectID(projectID)
	counter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("project_id", hashedID),
		attribute.Bool("auto_created", autoCreated),
	))
}

// RecordCheckpointResumed records a checkpoint resumption.
// Project IDs are hashed before being used as metric labels to prevent enumeration.
func (m *ValueMetrics) RecordCheckpointResumed(ctx context.Context, projectID string, level string) {
	if m == nil || !m.isInitialized() {
		return
	}

	m.mu.RLock()
	counter := m.checkpointResumed
	m.mu.RUnlock()

	if counter == nil {
		return
	}

	// Hash project ID to prevent enumeration attacks
	hashedID := hashProjectID(projectID)
	counter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("project_id", hashedID),
		attribute.String("level", level),
	))
}
